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
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/lucky-tools/devloop/integration/devloop"
	"github.com/lucky-tools/devloop/testutil"
)

// run on GCP as this test requires a load balancer
func TestBuildKanikoInsecureRegistry(t *testing.T) {
	// TODO: This test shall pass once render v2 is completed.
	t.SkipNow()

	MarkIntegrationTest(t, NeedsGcp)

	ns, client := SetupNamespace(t)

	dir := "testdata/kaniko-insecure-registry"
	devloop.Run("-p", "deploy-insecure-registry").InDir(dir).InNs(ns.Name).RunOrFailOutput(t)

	ip := client.ExternalIP("registry")
	registry := fmt.Sprintf("%s:5000", ip)

	devloop.Build("--insecure-registry", registry, "-p", "build-artifact").WithRepo(registry).InDir(dir).InNs(ns.Name).RunOrFailOutput(t)
}

func TestBuildKanikoWithExplicitRepo(t *testing.T) {
	// TODO: This test shall pass once render v2 is completed.
	t.SkipNow()

	MarkIntegrationTest(t, NeedsGcp)

	// Other integration tests run with the --default-repo option.
	// This one explicitly specifies the full image name.
	devloop.Build().WithRepo("").InDir("testdata/kaniko-explicit-repo").RunOrFail(t)
}

// see integration/testdata/README.md for details
func TestBuildInCluster(t *testing.T) {
	// TODO: This test shall pass once render v2 is completed.
	t.SkipNow()

	MarkIntegrationTest(t, NeedsGcp)

	testutil.Run(t, "", func(t *testutil.T) {
		ns, client := SetupNamespace(t.T)

		// this workaround is to ensure there is no overlap between testcases on kokoro
		// see https://github.com/lucky-tools/devloop/issues/2781#issuecomment-527770537
		project, err := filepath.Abs("testdata/devloop-in-cluster")
		t.CheckNoError(err)

		// copy the devloop binary to the test case folder
		// this is geared towards the in-docker setup: the fresh built binary is here
		// for manual testing, we can override this temporarily
		devloopSrc, err := exec.LookPath("devloop")
		t.CheckNoError(err)

		t.NewTempDir().Chdir()
		copyDir(t, project, ".")
		copyFile(t, devloopSrc, "devloop")

		// TODO: until https://github.com/lucky-tools/devloop/issues/2757 is resolved,
		// this is the simplest way to override the build.cluster.namespace
		replaceNamespace(t, "devloop.yaml", ns)
		replaceNamespace(t, "build-step/kustomization.yaml", ns)

		// we have to copy the e2esecret from default ns -> temporary namespace for kaniko
		client.CreateSecretFrom("default", "e2esecret")

		devloop.Run("-p", "create-build-step").InNs(ns.Name).RunOrFail(t.T)

		client.WaitForPodsInPhase(v1.PodSucceeded, "devloop-in-cluster")
	})
}

func replaceNamespace(t *testutil.T, fileName string, ns *v1.Namespace) {
	origDevloopYaml, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("failed reading %s: %s", fileName, err)
	}

	namespacedYaml := strings.ReplaceAll(string(origDevloopYaml), "VAR_CLUSTER_NAMESPACE", ns.Name)

	if err := os.WriteFile(fileName, []byte(namespacedYaml), 0o666); err != nil {
		t.Fatalf("failed to write %s: %s", fileName, err)
	}
}

func copyFile(t *testutil.T, src, dst string) {
	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("can't read source file: %s: %s", src, err)
	}

	if err := os.WriteFile(dst, content, 0o666); err != nil {
		t.Fatalf("failed to copy file %s to %s: %s", src, dst, err)
	}
}

func copyDir(t *testutil.T, src string, dst string) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		t.Fatalf("failed to copy dir %s->%s: %s ", src, dst, err)
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		t.Fatalf("failed to copy dir %s->%s: %s ", src, dst, err)
	}

	files, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to copy dir %s->%s: %s ", src, dst, err)
	}

	for _, f := range files {
		srcfp := path.Join(src, f.Name())
		dstfp := path.Join(dst, f.Name())

		if f.IsDir() {
			copyDir(t, srcfp, dstfp)
		} else {
			copyFile(t, srcfp, dstfp)
		}
	}
}
