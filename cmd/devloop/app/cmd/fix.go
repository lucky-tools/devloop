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
	"fmt"
	"io"
	"os"

	util2 "github.com/containerd/containerd/pkg/cri/util"
	"github.com/spf13/cobra"

	"github.com/lucky-tools/devloop/pkg/devloop/output"
	"github.com/lucky-tools/devloop/pkg/devloop/parser"
	"github.com/lucky-tools/devloop/pkg/devloop/schema"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/defaults"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/util"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/validation"
	"github.com/lucky-tools/devloop/pkg/devloop/yaml"
)

var (
	toVersion     string
	fixOutputPath string
)

func NewCmdFix() *cobra.Command {
	return NewCmd("fix").
		WithDescription("Update old configuration to a newer schema version").
		WithExample("Update \"devloop.yaml\" in the current folder to the latest version", "fix").
		WithExample("Update \"devloop.yaml\" in the current folder to version \"devloop/v1\"", "fix --version devloop/v1").
		WithExample("Update \"devloop.yaml\" in the current folder in-place", "fix --overwrite").
		WithExample("Update \"devloop.yaml\" and write the output to a new file", "fix --output devloop.new.yaml").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &overwrite, Name: "overwrite", DefValue: false, Usage: "Overwrite original config with fixed config"},
			{Value: &toVersion, Name: "version", DefValue: latest.Version, Usage: "Target schema version to upgrade to"},
			{Value: &fixOutputPath, Name: "output", Shorthand: "o", DefValue: "", Usage: "File to write the changed config (instead of standard output)"},
		}).
		NoArgs(doFix)
}

func doFix(_ context.Context, out io.Writer) error {
	if overwrite && fixOutputPath != "" {
		return fmt.Errorf("--overwrite and --output/-o cannot be used together")
	}
	var toFile string
	if fixOutputPath != "" {
		toFile = fixOutputPath
	} else if overwrite {
		toFile = opts.ConfigurationFile
	}
	return fix(out, opts.ConfigurationFile, toFile, toVersion, overwrite)
}

func fix(out io.Writer, configFile, outFile string, toVersion string, overwrite bool) error {
	parsedCfgs, err := schema.ParseConfig(configFile)
	if err != nil {
		return err
	}
	needsUpdate := false
	for _, cfg := range parsedCfgs {
		if cfg.GetVersion() != toVersion {
			needsUpdate = true
			break
		}
	}
	if !needsUpdate {
		output.Default.Fprintln(out, "config is already version", toVersion)
		return nil
	}

	versionedCfgs, err := schema.ParseConfig(configFile)
	if err != nil {
		return err
	}
	var upgraded []util.VersionedConfig
	if upgraded, err = schema.UpgradeTo(versionedCfgs, toVersion); err != nil {
		return err
	}

	// TODO(dgageot): We should be able run validations on any schema version
	// but that's not the case. They can only run on the latest version for now.
	if toVersion == latest.Version {
		var cfgs parser.DevloopConfigSet
		for _, cfg := range upgraded {
			cpCfg := latest.NewDevloopConfig()
			if err = util2.DeepCopy(cpCfg, cfg); err != nil {
				cpCfg = cfg
			}
			defaults.Set(cpCfg.(*latest.DevloopConfig))
			cfgs = append(cfgs, &parser.DevloopConfigEntry{
				DevloopConfig: cpCfg.(*latest.DevloopConfig),
				SourceFile:    configFile,
				IsRootConfig:  true,
			})
		}
		if err := validation.Process(cfgs, validation.GetValidationOpts(opts)); err != nil {
			return fmt.Errorf("validating upgraded config: %w", err)
		}
	}
	newCfg, err := yaml.MarshalWithSeparator(upgraded)
	if err != nil {
		return fmt.Errorf("marshaling new config: %w", err)
	}
	if outFile != "" {
		var writeErr error
		if overwrite {
			oldCfg, readErr := os.ReadFile(configFile)
			if readErr != nil {
				return fmt.Errorf("reading config file: %w", readErr)
			}
			newFile := fmt.Sprintf("%s.v2", outFile)

			writeErr = os.WriteFile(newFile, oldCfg, 0o644)
			if writeErr == nil {
				output.Default.Fprintln(out, "Backed up previous devloop.yaml at ", newFile)
			}
		}
		if err := os.WriteFile(outFile, newCfg, 0o644); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}
		output.Default.Fprintf(out, "New config at version %s generated and written to %s\n", toVersion, outFile)
		if writeErr != nil {
			output.Yellow.Fprintln(out, "Error moving old config. Dumping old v2 config on stdout:")
			output.Default.Fprintln(out, getOldConfigYaml(versionedCfgs))
		}
	} else {
		out.Write(newCfg)
	}
	return nil
}

func getOldConfigYaml(cfgs []util.VersionedConfig) string {
	yamlStr, err := yaml.MarshalWithSeparator(cfgs)
	if err != nil {
		return fmt.Sprintf("marshaling old config: %v", err)
	}
	return string(yamlStr)
}
