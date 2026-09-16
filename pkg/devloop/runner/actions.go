/*
Copyright 2023 The Skaffold Authors

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

package runner

import (
	"context"
	"fmt"
	"io"

	"github.com/lucky-tools/devloop/pkg/devloop/constants"
	eventV2 "github.com/lucky-tools/devloop/pkg/devloop/event/v2"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/instrumentation"
	"github.com/lucky-tools/devloop/pkg/devloop/output"
)

func (r *DevloopRunner) Exec(ctx context.Context, out io.Writer, artifacts []graph.Artifact, action string) error {
	out, ctx = output.WithEventContext(ctx, out, constants.Exec, constants.SubtaskIDNone)

	if len(artifacts) > 0 {
		output.Default.Fprintln(out, "Tags used in execution:")
		for _, artifact := range artifacts {
			output.Default.Fprintf(out, " - %s -> ", artifact.ImageName)
			fmt.Fprintln(out, artifact.Tag)
		}
	}

	eventV2.TaskInProgress(constants.Exec, fmt.Sprintf("Executing custom action %v", action))
	ctx, endTrace := instrumentation.StartTrace(ctx, "Exec_Executing")

	lm, err := localImages(r, artifacts)
	if err != nil {
		return err
	}

	err = r.actionsRunner.Exec(ctx, out, artifacts, lm, action)
	if err != nil {
		eventV2.TaskFailed(constants.Exec, err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	eventV2.TaskSucceeded(constants.Exec)
	endTrace()
	return nil
}

func localImages(r *DevloopRunner, artifacts []graph.Artifact) ([]graph.Artifact, error) {
	var localImgs []graph.Artifact
	for _, a := range artifacts {
		if isLocal, err := r.isLocalImage(a.ImageName); err != nil {
			return nil, err
		} else if isLocal {
			localImgs = append(localImgs, a)
		}
	}
	// We assume all the localImgs were build by Devloop, so we want to load them all.
	return localImgs, nil
}
