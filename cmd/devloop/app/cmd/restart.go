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
	"io"

	"github.com/spf13/cobra"

	"github.com/lucky-tools/devloop/pkg/devloop/runner"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/util"
)

// NewCmdRestart describes the CLI command to update image tags and restart workloads.
func NewCmdRestart() *cobra.Command {
	return NewCmd("restart").
		WithDescription("Update image tags and rolling-restart workloads without re-applying manifests").
		WithExample("Update my-app to tag v2 and rolling-restart its workloads", "restart --images my-app=v2").
		WithCommonFlags().
		WithHouseKeepingMessages().
		NoArgs(doRestart)
}

func doRestart(ctx context.Context, out io.Writer) error {
	return withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
		var artifacts []*latest.Artifact
		for _, cfg := range configs {
			artifacts = append(artifacts, cfg.(*latest.DevloopConfig).Build.Artifacts...)
		}
		buildArtifacts, err := getBuildArtifactsAndSetTagsWithTagPolicy(ctx, artifacts, r.ApplyDefaultRepo, r.TagFor)
		if err != nil {
			return err
		}
		return r.Restart(ctx, out, buildArtifacts)
	})
}
