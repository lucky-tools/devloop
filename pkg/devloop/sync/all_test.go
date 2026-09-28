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

package sync

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/testutil"
)

func TestAllItems(t *testing.T) {
	ctx := context.Background()
	absWorkspace, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		description  string
		artifact     *latest.Artifact
		builds       []graph.Artifact
		dependencies map[string][]string
		shouldErr    bool
		expected     *Item
	}{
		{
			description: "nil sync is a no-op",
			artifact:    &latest.Artifact{ImageName: "test", Workspace: "."},
			builds:      []graph.Artifact{{ImageName: "test", Tag: "test:123"}},
			expected:    nil,
		},
		{
			description: "no tag errors",
			artifact:    &latest.Artifact{ImageName: "test", Workspace: ".", Sync: &latest.Sync{Infer: []string{"*.html"}}},
			builds:      []graph.Artifact{{ImageName: "other", Tag: "other:123"}},
			shouldErr:   true,
		},
		{
			description: "auto sync is not supported",
			artifact:    &latest.Artifact{ImageName: "test", Workspace: ".", Sync: &latest.Sync{Auto: util.Ptr(true)}},
			builds:      []graph.Artifact{{ImageName: "test", Tag: "test:123"}},
			shouldErr:   true,
		},
		{
			description: "infer syncs all files matching the glob",
			artifact:    &latest.Artifact{ImageName: "test", Workspace: ".", Sync: &latest.Sync{Infer: []string{"*.html"}}},
			builds:      []graph.Artifact{{ImageName: "test", Tag: "test:123"}},
			dependencies: map[string][]string{
				"index.html": {"/app/index.html"},
				"server.js":  {"/app/server.js"},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join(absWorkspace, "index.html"): {"/app/index.html"},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&SyncMap, func(context.Context, *latest.Artifact, docker.Config) (map[string][]string, error) {
				return test.dependencies, nil
			})

			actual, err := AllItems(ctx, test.artifact, test.builds, &mockConfig{})
			if test.expected != nil {
				test.expected.Artifact = test.artifact
			}
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}

func TestAllItemsManual(t *testing.T) {
	ctx := context.Background()

	testutil.Run(t, "manual syncs all files matching the rules", func(t *testutil.T) {
		t.Override(&WorkingDir, func(context.Context, string, docker.Config) (string, error) { return "/", nil })
		dir := t.NewTempDir().WriteFiles(map[string]string{"index.html": "", "main.go": ""})

		artifact := &latest.Artifact{
			ImageName: "test",
			Workspace: dir.Root(),
			Sync:      &latest.Sync{Manual: []*latest.SyncRule{{Src: "*.html", Dest: "/app"}}},
		}
		builds := []graph.Artifact{{ImageName: "test", Tag: "test:123"}}

		actual, err := AllItems(ctx, artifact, builds, &mockConfig{})

		expected := &Item{
			Image:    "test:123",
			Artifact: artifact,
			Copy: map[string][]string{
				filepath.Join(dir.Root(), "index.html"): {"/app/index.html"},
			},
		}
		t.CheckErrorAndDeepEqual(false, err, expected, actual)
	})
}
