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
	"testing"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/runner"
	"github.com/lucky-tools/devloop/pkg/devloop/runner/runcontext"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/util"
	"github.com/lucky-tools/devloop/testutil"
)

type mockSyncRunner struct {
	runner.Runner
	syncedArtifacts []*latest.Artifact
	syncedBuilds    []graph.Artifact
}

func (r *mockSyncRunner) ApplyDefaultRepo(tag string) (string, error) { return tag, nil }

func (r *mockSyncRunner) Sync(ctx context.Context, out io.Writer, artifacts []*latest.Artifact, builds []graph.Artifact) error {
	r.syncedArtifacts = artifacts
	r.syncedBuilds = builds
	return nil
}

func TestSync(t *testing.T) {
	artifacts := []*latest.Artifact{{ImageName: "gcr.io/devloop/example"}}

	mock := &mockSyncRunner{}
	mockCreateRunner := func(context.Context, io.Writer, config.DevloopOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
		return mock, []util.VersionedConfig{&latest.DevloopConfig{
			Pipeline: latest.Pipeline{
				Build: latest.BuildConfig{Artifacts: artifacts},
			},
		}}, nil, nil
	}

	testutil.Run(t, "sync runs with artifacts and resolved tags", func(t *testutil.T) {
		t.Override(&createRunner, mockCreateRunner)
		t.Override(&opts.CustomTag, "tag")

		err := doSync(context.Background(), io.Discard)

		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(mock.syncedArtifacts))
		t.CheckDeepEqual("gcr.io/devloop/example", mock.syncedArtifacts[0].ImageName)
		t.CheckDeepEqual(1, len(mock.syncedBuilds))
		t.CheckDeepEqual("gcr.io/devloop/example:tag", mock.syncedBuilds[0].Tag)
	})
}
