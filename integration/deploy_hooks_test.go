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

package integration

import (
	"fmt"
	"testing"

	"github.com/lucky-tools/devloop/integration/devloop"
	"github.com/lucky-tools/devloop/testutil"
)

func TestPostDeployHooksNamespaces(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description  string
		dir          string
		configFile   string
		resourceFile string
	}{
		{
			description: "set DEVLOOP_NAMESPACES from manifests with kubectl deployer",
			configFile: `apiVersion: devloop/v1
kind: Config
manifests:
  rawYaml:
  - pod.yaml

deploy:
  kubectl:
    hooks:
      after:
        - host:
            command: ["sh", "-c", "echo $DEVLOOP_NAMESPACES"]
`,
			resourceFile: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: %v
spec:
  containers:
  - name: getting-started
    image: us-central1-docker.pkg.dev/k8s-devloop/testing/devloop-example
`,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ns, _ := SetupNamespace(t)
			expectedOutput := fmt.Sprintf("Starting post-deploy hooks...\ndefault,%v", ns.Name)
			resourceWithNs := fmt.Sprintf(test.resourceFile, ns.Name)

			dir := testutil.NewTempDir(t)
			dir.Write("devloop.yaml", test.configFile)
			dir.Write("pod.yaml", resourceWithNs)
			out, err := devloop.Run().InDir(dir.Root()).RunWithCombinedOutput(t)
			testutil.CheckError(t, false, err)
			testutil.CheckContains(t, expectedOutput, string(out))
		})
	}
}
