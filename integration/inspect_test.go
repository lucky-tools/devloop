/*
Copyright 2021 The Skaffold Authors

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

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky-tools/devloop/integration/devloop"
	"github.com/lucky-tools/devloop/testutil"
)

func TestInspectBuildEnv(t *testing.T) {
	clusterParams := []string{
		"--concurrency", "2",
		"--pullSecretName", "kaniko-secret2",
		"--randomDockerConfigSecret=true",
		"--randomPullSecret=true",
	}

	tests := []struct {
		description        string
		inputConfigFile    string
		args               []string
		expectedConfigFile string
		expectedOut        string
	}{
		{
			description:        "add new cluster build env definition in default pipeline",
			inputConfigFile:    "cluster/devloop.local.yaml",
			expectedConfigFile: "cluster/devloop.add.default.yaml",
			args:               append([]string{"build-env", "add", "cluster"}, clusterParams...),
		},
		{
			description:        "add new cluster build env definition in new profile",
			inputConfigFile:    "cluster/devloop.cluster.yaml",
			expectedConfigFile: "cluster/devloop.add.profile.yaml",
			args:               append([]string{"build-env", "add", "cluster", "--profile", "cluster"}, clusterParams...),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			tmpDir := t.NewTempDir()
			configContents, err := os.ReadFile(filepath.Join("testdata/inspect", test.inputConfigFile))
			t.CheckNoError(err)
			tmpDir.Write("devloop.yaml", string(configContents))
			args := append(test.args, fmt.Sprintf("-f=%s", tmpDir.Path("devloop.yaml")))
			out := devloop.Inspect(args...).InDir(tmpDir.Root()).RunOrFailOutput(t.T)
			if test.expectedOut != "" {
				t.CheckDeepEqual(test.expectedOut, string(out))
			}
			if test.expectedConfigFile != "" {
				expectedConfig, err := os.ReadFile(filepath.Join("testdata/inspect", test.expectedConfigFile))
				t.CheckNoError(err)
				actualConfig, err := os.ReadFile(tmpDir.Path("devloop.yaml"))
				t.CheckNoError(err)
				t.CheckDeepEqual(string(expectedConfig), string(actualConfig), testutil.YamlObj(t.T))
			}
		})
	}
}
