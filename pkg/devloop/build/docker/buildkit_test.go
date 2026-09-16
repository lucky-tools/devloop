/*
Copyright 2026 The Skaffold Authors

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

package docker

import (
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/testutil"
)

func TestParseCacheEntry(t *testing.T) {
	tests := []struct {
		description string
		input       string
		defaultType string
		expected    client.CacheOptionsEntry
	}{
		{
			description: "plain ref treated as registry cache",
			input:       "gcr.io/foo/cache:tag",
			defaultType: "registry",
			expected:    client.CacheOptionsEntry{Type: "registry", Attrs: map[string]string{"ref": "gcr.io/foo/cache:tag"}},
		},
		{
			description: "explicit type with attrs",
			input:       "type=registry,ref=gcr.io/foo/cache:tag,mode=max",
			defaultType: "",
			expected:    client.CacheOptionsEntry{Type: "registry", Attrs: map[string]string{"ref": "gcr.io/foo/cache:tag", "mode": "max"}},
		},
		{
			description: "inline cache type",
			input:       "type=inline",
			defaultType: "",
			expected:    client.CacheOptionsEntry{Type: "inline", Attrs: map[string]string{}},
		},
		{
			description: "no type with empty default",
			input:       "gcr.io/foo/cache:tag",
			defaultType: "",
			expected:    client.CacheOptionsEntry{Type: "", Attrs: map[string]string{}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, parseCacheEntry(test.input, test.defaultType))
		})
	}
}

func TestBuildKitSolveOpt(t *testing.T) {
	testutil.Run(t, "dockerfile frontend with platform, target and build args", func(t *testutil.T) {
		t.NewTempDir().Touch("Dockerfile").Chdir()
		dockerfilePath, _ := filepath.Abs("Dockerfile")

		t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, _ map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
			return map[string]*string{"FOO": util.Ptr("bar")}, nil
		})
		t.Override(&docker.BuildKitAuthProvider, func() (session.Attachable, error) { return nil, nil })

		builder := NewBuildKitBuilder(mockConfig{runMode: config.RunModes.Dev}, "", mockArtifactResolver{make(map[string]string)}, nil)

		artifact := &latest.Artifact{
			Workspace: ".",
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: "Dockerfile",
					Target:         "build",
					NoCache:        true,
				},
			},
		}

		opt, err := builder.buildSolveOpt(artifact, "gcr.io/foo/bar:tag", dockerfilePath, []v1.Platform{
			{OS: "linux", Architecture: "amd64"},
			{OS: "linux", Architecture: "arm64"},
		})
		t.CheckNoError(err)
		t.CheckDeepEqual("dockerfile.v0", opt.Frontend)
		t.CheckDeepEqual("Dockerfile", opt.FrontendAttrs["filename"])
		t.CheckDeepEqual("build", opt.FrontendAttrs["target"])
		t.CheckDeepEqual("", opt.FrontendAttrs["no-cache"])
		t.CheckDeepEqual("bar", opt.FrontendAttrs["build-arg:FOO"])
		t.CheckDeepEqual("linux/amd64,linux/arm64", opt.FrontendAttrs["platform"])
		t.CheckDeepEqual([]client.ExportEntry{{Type: "image", Attrs: map[string]string{"name": "gcr.io/foo/bar:tag", "push": "true"}}}, opt.Exports)

		_, hasContext := opt.LocalMounts["context"]
		t.CheckTrue(hasContext)
		_, hasDockerfile := opt.LocalMounts["dockerfile"]
		t.CheckTrue(hasDockerfile)
	})
}
