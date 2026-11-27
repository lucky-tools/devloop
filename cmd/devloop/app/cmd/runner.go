/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	sErrors "github.com/lucky-tools/devloop/pkg/devloop/errors"
	"github.com/lucky-tools/devloop/pkg/devloop/event"
	"github.com/lucky-tools/devloop/pkg/devloop/hooks"
	"github.com/lucky-tools/devloop/pkg/devloop/instrumentation"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/parser"
	"github.com/lucky-tools/devloop/pkg/devloop/runner"
	"github.com/lucky-tools/devloop/pkg/devloop/runner/runcontext"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/defaults"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/util"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/validation"
	"github.com/lucky-tools/devloop/pkg/devloop/update"
	"github.com/lucky-tools/devloop/proto/v1"
)

// For tests
var createRunner = createNewRunner

func withRunner(ctx context.Context, out io.Writer, action func(runner.Runner, []util.VersionedConfig) error) error {
	runner, config, runCtx, err := createRunner(ctx, out, opts)
	if err != nil {
		return err
	}

	err = action(runner, config)

	return alwaysSucceedWhenCancelled(ctx, runCtx, err)
}

// createNewRunner creates a Runner and returns the DevloopConfig associated with it.
func createNewRunner(ctx context.Context, out io.Writer, opts config.DevloopOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
	runCtx, configs, err := runContext(ctx, out, opts)
	if err != nil {
		return nil, nil, nil, err
	}

	instrumentation.InitTraceFromEnvVar()
	hooks.SetupStaticEnvOptions(runCtx)
	runner, err := runner.NewForConfig(ctx, runCtx)
	if err != nil {
		event.InititializationFailed(err)
		return nil, nil, nil, fmt.Errorf("creating runner: %w", err)
	}
	return runner, configs, runCtx, nil
}

func runContext(ctx context.Context, out io.Writer, opts config.DevloopOptions) (*runcontext.RunContext, []util.VersionedConfig, error) {
	cfgSet, err := withFallbackConfig(ctx, out, opts, parser.GetConfigSet)
	if err != nil {
		return nil, nil, err
	}

	// DOCKER_HOST_SYNC (from the process environment or from the config's `env`
	// field, which withFallbackConfig has just applied) lets `devloop sync`
	// target a different Docker daemon than the other commands. It must be
	// applied here, before the first Docker client is created below in
	// validation.ProcessWithRunContext.
	if err := applySyncDockerHost(opts.Command); err != nil {
		return nil, nil, err
	}
	setDefaultRendererAndDeployer(cfgSet)

	if err := validation.Process(cfgSet, validation.GetValidationOpts(opts)); err != nil {
		return nil, nil, fmt.Errorf("invalid devloop config: %w", err)
	}
	var configs []util.VersionedConfig
	for _, cfg := range cfgSet {
		configs = append(configs, cfg.DevloopConfig)
	}

	// Re-resolve flags from the environment now that the config's `env` field has
	// been applied, so that variables such as `DEVLOOP_IMAGES` can bind to flags
	// such as `--images`. Flags already set on the command line are left untouched.
	if devloopRootCmd != nil {
		setFlagsFromEnvVariables(devloopRootCmd, true)
	}

	runCtx, err := runcontext.GetRunContext(ctx, opts, configs)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	if err := validation.ProcessWithRunContext(ctx, runCtx); err != nil {
		return nil, nil, fmt.Errorf("invalid devloop config: %w", err)
	}

	return runCtx, configs, nil
}

// applySyncDockerHost points the process at DOCKER_HOST_SYNC when the sync
// command runs, so a single config file can build on one Docker host and sync
// into containers on another. Other commands ignore the variable.
//
// It must be called after the config's `env` field has been applied and before
// any Docker client is created: the API client (docker.NewAPIClient) and the
// `docker exec` subprocesses used by sync both read DOCKER_HOST from the
// process environment.
func applySyncDockerHost(command string) error {
	if command != "sync" {
		return nil
	}

	host := os.Getenv("DOCKER_HOST_SYNC")
	if host == "" {
		return nil
	}

	if err := os.Setenv("DOCKER_HOST", host); err != nil {
		return fmt.Errorf("setting DOCKER_HOST from DOCKER_HOST_SYNC: %w", err)
	}
	// The Docker CLI gives DOCKER_CONTEXT precedence over DOCKER_HOST. Clear it
	// so the `docker exec` subprocesses used by sync reach the same daemon as
	// the API client, which only reads DOCKER_HOST.
	if err := os.Unsetenv("DOCKER_CONTEXT"); err != nil {
		return fmt.Errorf("clearing DOCKER_CONTEXT for DOCKER_HOST_SYNC: %w", err)
	}
	log.Entry(context.TODO()).Infof("Using DOCKER_HOST_SYNC for the sync command")

	return nil
}

// withFallbackConfig will try to automatically generate a config if root `devloop.yaml` file does not exist.
func withFallbackConfig(ctx context.Context, _ io.Writer, opts config.DevloopOptions, getCfgs func(context.Context, config.DevloopOptions) (parser.DevloopConfigSet, error)) (parser.DevloopConfigSet, error) {
	configs, err := getCfgs(ctx, opts)
	if err == nil {
		return configs, nil
	}
	var e sErrors.Error
	if errors.As(err, &e) && e.StatusCode() == proto.StatusCode_CONFIG_FILE_NOT_FOUND_ERR {
		return nil, fmt.Errorf("devloop config file %s not found - check your current working directory", opts.ConfigurationFile)
	}

	// If the error is NOT that the file doesn't exist, then we warn the user
	// that maybe they are using an outdated version of Devloop that's unable to read
	// the configuration.
	warnIfUpdateIsAvailable()
	return nil, fmt.Errorf("parsing devloop config: %w", err)
}

func warnIfUpdateIsAvailable() {
	warning, err := update.CheckVersionOnError(opts.GlobalConfig)
	if err != nil {
		log.Entry(context.TODO()).Infof("update check failed: %s", err)
		return
	}
	if warning != "" {
		log.Entry(context.TODO()).Warn(warning)
	}
}

func setDefaultRendererAndDeployer(configs parser.DevloopConfigSet) {
	// do not set a default deployer or renderer in a multi-config application.
	if len(configs) > 1 {
		return
	}
	// there always exists at least one config
	defaults.SetDefaultRenderer(configs[0].DevloopConfig)
	defaults.SetDefaultDeployer(configs[0].DevloopConfig)
}
