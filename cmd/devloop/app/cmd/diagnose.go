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

	"github.com/spf13/cobra"

	deployutil "github.com/lucky-tools/devloop/pkg/devloop/deploy/util"
	"github.com/lucky-tools/devloop/pkg/devloop/diagnose"
	"github.com/lucky-tools/devloop/pkg/devloop/output"
	"github.com/lucky-tools/devloop/pkg/devloop/parser"
	"github.com/lucky-tools/devloop/pkg/devloop/runner/runcontext"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	schemaUtil "github.com/lucky-tools/devloop/pkg/devloop/schema/util"
	"github.com/lucky-tools/devloop/pkg/devloop/tag"
	"github.com/lucky-tools/devloop/pkg/devloop/tags"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/pkg/devloop/version"
	"github.com/lucky-tools/devloop/pkg/devloop/yaml"
)

var (
	yamlOnly   bool
	outputFile string

	enableTemplating bool
	// for testing
	getRunContext = runcontext.GetRunContext
	getCfgs       = parser.GetAllConfigs
)

// NewCmdDiagnose describes the CLI command to diagnose devloop.
func NewCmdDiagnose() *cobra.Command {
	return NewCmd("diagnose").
		WithDescription("Run a diagnostic on Devloop").
		WithExample("Search for configuration issues and print the effective configuration", "diagnose").
		WithExample("Print the effective devloop.yaml configuration for given profile", "diagnose --yaml-only --profile PROFILE").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &yamlOnly, Name: "yaml-only", DefValue: false, Usage: "Only prints the effective devloop.yaml configuration"},
			{Value: &enableTemplating, Name: "enable-templating", DefValue: false, Usage: "Render supported templated fields with golang template engine"},
			{Value: &outputFile, Name: "output", Shorthand: "o", DefValue: "", Usage: "File to write diagnose result"},
		}).
		NoArgs(doDiagnose)
}

func doDiagnose(ctx context.Context, out io.Writer) error {
	// force absolute path resolution during diagnose
	opts.MakePathsAbsolute = util.Ptr(true)
	configs, err := getCfgs(ctx, opts)
	if err != nil {
		return err
	}
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	if !yamlOnly {
		if err := printArtifactDiagnostics(ctx, out, configs); err != nil {
			return err
		}
	}
	// remove the dependency config references since they have already been imported and will be marshalled together.
	for i := range configs {
		configs[i].(*latest.DevloopConfig).Dependencies = nil
	}
	if enableTemplating {
		if err := tags.ApplyTemplates(configs); err != nil {
			return err
		}
	}
	buf, err := yaml.MarshalWithSeparator(configs)
	if err != nil {
		return fmt.Errorf("marshalling configuration: %w", err)
	}

	out.Write(buf)

	return nil
}

func printArtifactDiagnostics(ctx context.Context, out io.Writer, configs []schemaUtil.VersionedConfig) error {
	runCtx, err := getRunContext(ctx, opts, configs)
	if err != nil {
		return fmt.Errorf("getting run context: %w", err)
	}
	tagger, err := tag.NewTaggerMux(runCtx)
	if err != nil {
		return fmt.Errorf("getting tagger: %w", err)
	}
	imageTags, err := deployutil.ImageTags(ctx, runCtx, tagger, out, runCtx.Artifacts())
	if err != nil {
		return fmt.Errorf("getting tags: %w", err)
	}
	for _, c := range configs {
		config := c.(*latest.DevloopConfig)
		fmt.Fprintln(out, "Devloop version:", version.Get().GitCommit)
		fmt.Fprintln(out, "Configuration version:", config.APIVersion)
		fmt.Fprintln(out, "Number of artifacts:", len(config.Build.Artifacts))
		if err := diagnose.CheckArtifacts(ctx, runCtx, imageTags, out); err != nil {
			return fmt.Errorf("running diagnostic on artifacts: %w", err)
		}

		output.Blue.Fprintln(out, "\nConfiguration")
	}
	return nil
}
