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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
)

var (
	pullOutputFlag   string
	pullInsecureFlag bool
	pullUsernameFlag string
	pullPasswordFlag string
)

// NewCmdPull describes the CLI command to pull an image from a registry.
func NewCmdPull() *cobra.Command {
	return NewCmd("pull").
		WithDescription("Pull an image from a registry and save it as a tarball").
		WithExample("Pull from a local HTTP registry", "pull localhost:5000/app:latest --insecure").
		WithExample("Pull with credentials and a custom output path", "pull localhost:5000/app:latest -o app.tar --username admin --password secret").
		WithFlagAdder(func(f *pflag.FlagSet) {
			f.StringVarP(&opts.GlobalConfig, "config", "c", "", "File for global configurations (defaults to $HOME/.devloop/config)")
		}).
		WithFlags([]*Flag{
			{Value: &pullOutputFlag, Name: "output", Shorthand: "o", DefValue: "", Usage: "Path to write the image tarball to (defaults to a name derived from the image tag)"},
			{Value: &pullInsecureFlag, Name: "insecure", DefValue: false, Usage: "Allow pulling from an insecure registry (plain HTTP and/or untrusted TLS)", IsEnum: true},
			{Value: &pullUsernameFlag, Name: "username", DefValue: "", Usage: "Registry username for basic authentication"},
			{Value: &pullPasswordFlag, Name: "password", DefValue: "", Usage: "Registry password for basic authentication"},
		}).
		ExactArgs(1, doPull)
}

func doPull(ctx context.Context, out io.Writer, args []string) error {
	identifier := args[0]

	outputPath := pullOutputFlag
	if outputPath == "" {
		path, err := docker.PullOutputPath(identifier)
		if err != nil {
			return err
		}
		outputPath = path
	}

	insecureRegistries := map[string]bool{}
	if regs, err := config.GetInsecureRegistries(opts.GlobalConfig); err == nil {
		for _, r := range regs {
			insecureRegistries[r] = true
		}
	} else {
		log.Entry(ctx).Warnf("error retrieving insecure registries from global config: %v", err)
	}

	if err := docker.Pull(ctx, identifier, outputPath, docker.PullOptions{
		Insecure:           pullInsecureFlag,
		InsecureRegistries: insecureRegistries,
		Username:           pullUsernameFlag,
		Password:           pullPasswordFlag,
	}); err != nil {
		return err
	}

	fmt.Fprintf(out, "Pulled %s to %s\n", identifier, outputPath)
	return nil
}
