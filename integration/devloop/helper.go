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

package devloop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/lucky-tools/devloop/cmd/devloop/app/cmd"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	timeutil "github.com/lucky-tools/devloop/pkg/devloop/util/time"
)

// RunBuilder is used to build a command line to run `devloop`.
type RunBuilder struct {
	command    string
	configFile string
	dir        string
	ns         string
	repo       string
	profiles   []string
	args       []string
	env        []string
	stdin      []byte
}

const DefaultRepo = "us-central1-docker.pkg.dev/k8s-devloop/testing"

// Apply runs `devloop apply` with the given arguments.
func Apply(args ...string) *RunBuilder {
	return withDefaults("apply", args)
}

// Dev runs `devloop dev` with the given arguments.
func Dev(args ...string) *RunBuilder {
	return withDefaults("dev", args)
}

// Fix runs `devloop fix` with the given arguments.
func Fix(args ...string) *RunBuilder {
	return withDefaults("fix", args)
}

// Build runs `devloop build` with the given arguments.
func Build(args ...string) *RunBuilder {
	return withDefaults("build", args)
}

// Test runs `devloop test` with the given arguments.
func Test(args ...string) *RunBuilder {
	return withDefaults("test", args)
}

// Deploy runs `devloop deploy` with the given arguments.
func Deploy(args ...string) *RunBuilder {
	return withDefaults("deploy", args)
}

// Verify runs `devloop verify` with the given arguments.
func Verify(args ...string) *RunBuilder {
	return withDefaults("verify", args)
}

// Exec runs `devloop exec` with the given arguments.
func Exec(args ...string) *RunBuilder {
	return withDefaults("exec", args)
}

// Debug runs `devloop debug` with the given arguments.
func Debug(args ...string) *RunBuilder {
	return withDefaults("debug", args)
}

// Run runs `devloop run` with the given arguments.
func Run(args ...string) *RunBuilder {
	return withDefaults("run", args)
}

// Delete runs `devloop delete` with the given arguments.
func Delete(args ...string) *RunBuilder {
	return withDefaults("delete", args)
}

// Config runs `devloop config` with the given arguments.
func Config(args ...string) *RunBuilder {
	return withDefaults("config", args)
}

// Init runs `devloop init` with the given arguments.
func Init(args ...string) *RunBuilder {
	return withDefaults("init", args)
}

// Diagnose runs `devloop diagnose` with the given arguments.
func Diagnose(args ...string) *RunBuilder {
	return withDefaults("diagnose", args)
}

// Schema runs `devloop schema` with the given arguments.
func Schema(args ...string) *RunBuilder {
	return &RunBuilder{command: "schema", args: args}
}

// Credits runs `devloop credits` with the given arguments.
func Credits(args ...string) *RunBuilder {
	return &RunBuilder{command: "credits", args: args}
}

// Render runs `devloop render` with the given arguments.
func Render(args ...string) *RunBuilder {
	return withDefaults("render", args)
}

// Inspect runs `devloop inspect` with the given arguments.
func Inspect(args ...string) *RunBuilder {
	return &RunBuilder{command: "inspect", args: args}
}

// Filter runs `devloop filter` with the given arguments.
func Filter(args ...string) *RunBuilder {
	return withDefaults("filter", args)
}

func GeneratePipeline(args ...string) *RunBuilder {
	return withDefaults("generate-pipeline", args)
}

func withDefaults(command string, args []string) *RunBuilder {
	repo := os.Getenv("DEFAULT_REPO")
	if repo == "" {
		repo = DefaultRepo
	}
	return &RunBuilder{command: command, args: args, repo: repo}
}

// InDir sets the directory in which devloop is running.
func (b *RunBuilder) InDir(dir string) *RunBuilder {
	b.dir = dir
	return b
}

// WithConfig sets the config file to be used by devloop.
func (b *RunBuilder) WithConfig(configFile string) *RunBuilder {
	b.configFile = configFile
	return b
}

// WithRepo sets the default repository to be used by devloop.
func (b *RunBuilder) WithRepo(repo string) *RunBuilder {
	b.repo = repo
	return b
}

// WithStdin sets the stdin.
func (b *RunBuilder) WithStdin(input []byte) *RunBuilder {
	b.stdin = input
	return b
}

// InNs sets the Kubernetes namespace in which devloop deploys.
func (b *RunBuilder) InNs(ns string) *RunBuilder {
	b.ns = ns
	return b
}

// WithEnv sets environment variables.
func (b *RunBuilder) WithEnv(env []string) *RunBuilder {
	b.env = env
	return b
}

// WithProfiles sets profiles.
func (b *RunBuilder) WithProfiles(profiles []string) *RunBuilder {
	b.profiles = profiles
	return b
}

// RunBackground runs the devloop command in the background.  The Devloop output
// is accumulated and logged on test failure.
func (b *RunBuilder) RunBackground(t *testing.T) {
	t.Helper()
	out := bytes.Buffer{}
	b.runForked(t, &out)

	t.Cleanup(func() {
		if t.Failed() {
			t.Log("Devloop log:\n", strings.ReplaceAll(out.String(), "\n", "\n> "))
		}
	})
}

// RunLive runs the devloop command in the background with live output.
// !!Warning!! RunLive blocks the devloop command until the caller reads from
// the returned `PipeReader`. Please use `WaitForLogs` or similar to read
// continuously from the returned `PipeReader`.
func (b *RunBuilder) RunLive(t *testing.T) io.ReadCloser {
	t.Helper()
	pr, pw := io.Pipe()
	b.runForked(t, pw)
	t.Cleanup(func() {
		pr.Close()
	})
	return pr
}

// RunInBackgroundWithOutput runs the devloop command in the background.
func (b *RunBuilder) RunInBackgroundWithOutput(t *testing.T, out io.Writer) {
	t.Helper()
	b.runForked(t, out)
}

// runForked runs the devloop command in the background with stdout sent to the provided writer.
func (b *RunBuilder) runForked(t *testing.T, out io.Writer) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	cmd := b.cmd(ctx)
	cmd.Stdout = out
	t.Logf("Running %s in %s", cmd.Args, cmd.Dir)

	if err := cmd.Start(); err != nil {
		t.Fatalf("devloop %s: %v", b.command, err)
	}

	waitAndTriggerStacktrace(ctx, t, cmd.Process)

	t.Cleanup(func() {
		cancel()
		cmd.Wait()
	})
}

// RunOrFail runs the devloop command and fails the test
// if the command returns an error.
func (b *RunBuilder) RunOrFail(t *testing.T) {
	b.RunOrFailOutput(t)
}

// Run runs the devloop command.
func (b *RunBuilder) Run(t *testing.T) error {
	t.Helper()

	cmd := b.cmd(context.Background())
	t.Logf("Running %s in %s", cmd.Args, cmd.Dir)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("devloop %q: %w", b.command, err)
	}
	return nil
}

// StartWithProcess starts the devloop command and returns the process id and error.
func (b *RunBuilder) StartWithProcess(t *testing.T) (*os.Process, error) {
	t.Helper()

	cmd := b.cmd(context.Background())
	t.Logf("Running %s in %s", cmd.Args, cmd.Dir)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("devloop %q: %w", b.command, err)
	}
	return cmd.Process, nil
}

// RunWithCombinedOutput runs the devloop command and returns the combined standard output and error.
func (b *RunBuilder) RunWithCombinedOutput(t *testing.T) ([]byte, error) {
	t.Helper()

	cmd := b.cmd(context.Background())
	cmd.Stdout, cmd.Stderr = nil, nil
	t.Logf("Running %s in %s", cmd.Args, cmd.Dir)

	start := time.Now()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("devloop %q: %w", b.command, err)
	}
	t.Logf("Ran %s in %v", cmd.Args, timeutil.Humanize(time.Since(start)))
	return out, nil
}

// RunOrFailOutput runs the devloop command and fails the test
// if the command returns an error.
// It only returns the standard output.
func (b *RunBuilder) RunOrFailOutput(t *testing.T) []byte {
	t.Helper()

	cmd := b.cmd(context.Background())
	cmd.Stdout, cmd.Stderr = nil, nil
	t.Logf("Running %s in %s", cmd.Args, cmd.Dir)

	start := time.Now()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			defer t.Error(string(ee.Stderr))
		}
		t.Fatalf("devloop %s: %v, %s", b.command, err, out)
	}
	t.Logf("Ran %s in %v", cmd.Args, timeutil.Humanize(time.Since(start)))
	return out
}

// RunWithStdoutAndStderrOrFail runs the Devloop command copying the stdout and stderr to the given writers.
// Fails if there is an error.
func (b *RunBuilder) RunWithStdoutAndStderrOrFail(t *testing.T, stdout, stderr io.Writer) {
	t.Helper()

	cmd := b.cmd(context.Background())
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	t.Logf("Running %s in %s", cmd.Args, cmd.Dir)

	start := time.Now()
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			defer t.Error(string(ee.Stderr))
		}
		t.Fatalf("devloop %s: %v, %s", b.command, err, stderr)
	}

	t.Logf("Ran %s in %v", cmd.Args, timeutil.Humanize(time.Since(start)))
}

func (b *RunBuilder) cmd(ctx context.Context) *exec.Cmd {
	args := []string{b.command}
	command := b.getCobraCommand()

	if b.ns != "" && command.Flags().Lookup("namespace") != nil {
		args = append(args, "--namespace", b.ns)
	}
	if b.configFile != "" && command.Flags().ShorthandLookup("f") != nil {
		args = append(args, "-f", b.configFile)
	}
	if b.repo != "" && command.Flags().Lookup("default-repo") != nil {
		args = append(args, "--default-repo", b.repo)
	}
	if len(b.profiles) > 0 && command.Flags().Lookup("profile") != nil {
		args = append(args, "--profile", strings.Join(b.profiles, ","))
	}
	args = append(args, b.args...)

	devloopBinary := "devloop"
	if value, found := os.LookupEnv("DEVLOOP_BINARY"); found {
		devloopBinary = value
	}
	cmd := exec.CommandContext(ctx, devloopBinary, args...)
	cmd.Env = append(removeDevloopEnvVariables(util.OSEnviron()), b.env...)
	if b.stdin != nil {
		cmd.Stdin = bytes.NewReader(b.stdin)
	}
	if b.dir != "" {
		cmd.Dir = b.dir
	}

	// If the test is killed by a timeout, go test will wait for
	// os.Stderr and os.Stdout to close as a result.
	//
	// However, the `cmd` will still run in the background
	// and hold those descriptors open.
	// As a result, go test will hang forever.
	//
	// Avoid that by wrapping stderr and stdout, breaking the short
	// circuit and forcing cmd.Run to use another pipe and goroutine
	// to pass along stderr and stdout.
	// See https://github.com/golang/go/issues/23019
	cmd.Stdout = struct{ io.Writer }{os.Stdout}
	cmd.Stderr = struct{ io.Writer }{os.Stderr}

	return cmd
}

// getCobraCommand returns the matching cobra command for the command
// in b, or return a dummy without flags
func (b *RunBuilder) getCobraCommand() *cobra.Command {
	c := cmd.NewDevloopCommand(os.Stdout, os.Stderr)
	for _, comm := range c.Commands() {
		if comm.Name() == b.command {
			return comm
		}
	}
	return &cobra.Command{}
}

// removeDevloopEnvVariables makes sure Devloop runs without
// any env variable that might change its behavior, such as
// enabling caching.
func removeDevloopEnvVariables(env []string) []string {
	var clean []string

	for _, value := range env {
		if !strings.HasPrefix(value, "DEVLOOP_") {
			clean = append(clean, value)
		}
	}

	// Disable update check
	return append(clean, "DEVLOOP_UPDATE_CHECK=false")
}
