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

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky-tools/devloop/integration/devloop"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/testutil"
)

func TestHelmDeploy(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	ns, client := SetupNamespace(t)

	// To fix #1823, we make use of env variable templating for release name
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	devloop.Deploy("--images", "us-central1-docker.pkg.dev/k8s-devloop/testing/devloop-helm").InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)

	dep := client.GetDeployment("devloop-helm-" + ns.Name)
	testutil.CheckDeepEqual(t, dep.Name, dep.ObjectMeta.Labels["release"])

	devloop.Delete().InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)
}

func TestHelmDeployWithHook(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, client := SetupNamespace(t)

	// To fix #1823, we make use of env variable templating for release name
	replicas := 5
	env := []string{fmt.Sprintf("REPLICAS=%d", replicas), fmt.Sprintf("TEST_NS=%s", ns.Name)}
	devloop.Deploy("--images", "us-central1-docker.pkg.dev/k8s-devloop/testing/devloop-helm", "-p", "helm-hook").InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)

	dep := client.GetDeployment("devloop-helm-" + ns.Name)
	testutil.CheckDeepEqual(t, dep.Spec.Replicas, util.Ptr(int32(replicas)))

	devloop.Delete().InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)
}

func TestRunHelmMultiConfig(t *testing.T) {
	tests := []struct {
		description  string
		dir          string
		args         []string
		deployments  []string
		pods         []string
		env          []string
		targetLogOne string
		targetLogTwo string
	}{
		{
			description:  "helm-multi-config",
			dir:          "testdata/helm-multi-config/devloop",
			deployments:  []string{"app1", "app2"},
			targetLogOne: "app1",
			targetLogTwo: "app2",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			if test.targetLogOne == "" || test.targetLogTwo == "" {
				t.SkipNow()
			}
			if test.dir == emptydir {
				err := os.MkdirAll(filepath.Join(test.dir, "emptydir"), 0o755)
				t.Log("Creating empty directory")
				if err != nil {
					t.Errorf("Error creating empty dir: %s", err)
				}
			}

			ns, _ := SetupNamespace(t)

			devloop.Run(test.args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFailOutput(t)

			out := devloop.Run(test.args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunLive(t)
			defer devloop.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).Run(t)

			WaitForLogs(t, out, test.targetLogOne)
			WaitForLogs(t, out, test.targetLogTwo)
		})
	}
}

func TestRunHelmStatefulSet(t *testing.T) {
	tests := []struct {
		description string
		dir         string
		args        []string
		pods        []string
		env         []string
		targetLog   string
	}{
		{
			description: "helm-statefulset-v1-schema",
			dir:         "testdata/helm-statefulset-v1-schema",
			targetLog:   "statefulset/devloop-helm is ready",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			if test.targetLog == "" {
				t.SkipNow()
			}
			if test.dir == emptydir {
				err := os.MkdirAll(filepath.Join(test.dir, "emptydir"), 0o755)
				t.Log("Creating empty directory")
				if err != nil {
					t.Errorf("Error creating empty dir: %s", err)
				}
			}

			ns, _ := SetupNamespace(t)

			out := devloop.Run(test.args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFailOutput(t)
			defer devloop.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).Run(t)

			testutil.CheckContains(t, test.targetLog, string(out))
		})
	}
}

func TestHelmRenderWithOCIRegistry(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	devloopConfig := fmt.Sprintf(`apiVersion: devloop/v1
kind: Config

deploy:
  helm:
    releases:
    - name: devloop-helm-chart-oci
      remoteChart: oci://%s/devloop-helm-chart
      setValues:
        image: devloop-helm`, devloop.DefaultRepo)

	expectedOutput := `---
# Source: devloop-helm-chart/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: devloop-helm-chart
  labels:
    app: devloop-helm-chart
spec:
  selector:
    matchLabels:
      app: devloop-helm-chart
  replicas: 2
  template:
    metadata:
      labels:
        app: devloop-helm-chart
    spec:
      containers:
      - name: devloop-helm-chart
        image: devloop-helm
`

	tmpDir := testutil.NewTempDir(t)
	tmpDir.Write("devloop.yaml", devloopConfig)
	tmpDir.Chdir()

	devloop.Render("--output", "rendered.yaml").RunOrFail(t)
	fileContent, err := os.ReadFile("rendered.yaml")

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, expectedOutput, string(fileContent))
}

func TestHelmDeployWithGlobalFlags(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	ns, _ := SetupNamespace(t)
	// To fix #1823, we make use of env variable templating for release name
	env := []string{fmt.Sprintf("TEST_NS=%s", ns.Name)}
	devloop.Deploy("--images", "us-central1-docker.pkg.dev/k8s-devloop/testing/devloop-helm", "-p", "helm-with-global-flags").InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)
	devloop.Delete().InDir("testdata/helm").InNs(ns.Name).WithEnv(env).RunOrFail(t)
}
