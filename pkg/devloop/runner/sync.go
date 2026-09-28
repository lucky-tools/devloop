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

package runner

import (
	"context"
	"io"

	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/sync"
)

// Sync copies the files matching each artifact's sync configuration into its
// running containers. It is a one-shot operation used by the `devloop sync`
// command and, unlike the file sync performed during `devloop dev`, copies the
// full current state of the syncable files rather than only the files that
// changed since the last watch cycle.
func (r *DevloopRunner) Sync(ctx context.Context, out io.Writer, artifacts []*latest.Artifact, builds []graph.Artifact) error {
	for _, a := range artifacts {
		if a.Sync == nil {
			continue
		}

		item, err := sync.AllItems(ctx, a, builds, r.runCtx)
		if err != nil {
			return err
		}
		if item == nil {
			continue
		}

		if err := r.deployer.GetSyncer().Sync(ctx, out, item); err != nil {
			return err
		}
	}
	return nil
}
