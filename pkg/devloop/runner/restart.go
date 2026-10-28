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
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lucky-tools/devloop/pkg/devloop/deploy"
	"github.com/lucky-tools/devloop/pkg/devloop/deploy/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/kubectl"
	"github.com/lucky-tools/devloop/pkg/devloop/kubernetes/manifest"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

// workloadSelector matches the Kubernetes workload kinds that support a rolling restart.
var workloadSelector = &manifest.WildcardGroupKind{
	Group: regexp.MustCompile(`^(apps|extensions)$`),
	Kind:  regexp.MustCompile(`^(Deployment|StatefulSet|DaemonSet)$`),
}

type containerRef struct {
	name  string
	image string
}

// Restart updates the image of devloop-managed containers and restarts the workloads
// that run them. The target is chosen the same way the deployer is selected: a
// `deploy.docker` config restarts the local containers, otherwise the Kubernetes
// workloads are updated and rolling-restarted.
func (r *DevloopRunner) Restart(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	if hasDockerDeploy(r.runCtx.DeployConfigs()) {
		return r.restartDocker(ctx, out, artifacts)
	}
	return r.restartKubernetes(ctx, out, artifacts)
}

// hasDockerDeploy reports whether any deploy config targets a local Docker daemon.
func hasDockerDeploy(deployConfigs []latest.DeployConfig) bool {
	for _, d := range deployConfigs {
		if d.DockerDeploy != nil {
			return true
		}
	}
	return false
}

// restartDocker restarts the containers created for the managed artifacts.
func (r *DevloopRunner) restartDocker(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	if mux, ok := r.deployer.(deploy.DeployerMux); ok {
		for _, d := range mux.GetDeployers() {
			if dd, ok := d.(*docker.Deployer); ok {
				return dd.Restart(ctx, out, artifacts)
			}
		}
	}
	return errors.New("docker deployer not found")
}

// restartKubernetes updates the image of devloop-managed containers and rolling-restarts
// the workloads that run them. It deliberately avoids `kubectl apply` so that other
// fields that may have been changed directly on the cluster are left untouched.
func (r *DevloopRunner) restartKubernetes(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	manifests, err := r.Render(ctx, out, artifacts, false)
	if err != nil {
		return fmt.Errorf("rendering manifests: %w", err)
	}

	managed := make(map[string]bool, len(artifacts))
	for _, a := range artifacts {
		managed[a.Tag] = true
	}

	cli := kubectl.NewCLI(r.runCtx, "")
	// Resolve the kube context the same way the deployer does: honour
	// deploy.kubeContext from the config when no CLI/env override is present.
	if r.runCtx.IsDefaultKubeContext() {
		for _, p := range r.runCtx.GetPipelines() {
			if p.Deploy.KubeContext != "" {
				cli.KubeContext = p.Deploy.KubeContext
				break
			}
		}
	}

	list := manifests.GetForConfig(r.deployer.ConfigName())
	workloads, err := list.SelectResources(workloadSelector)
	if err != nil {
		return err
	}

	for _, w := range workloads {
		kind := strings.ToLower(w.GetKind())
		name := w.GetName()
		ns := w.GetNamespace()
		if ns == "" {
			ns = r.runCtx.GetNamespace()
		}

		var imageArgs []string
		for _, c := range podContainers(w) {
			if managed[c.image] {
				imageArgs = append(imageArgs, c.name+"="+c.image)
			}
		}
		if len(imageArgs) == 0 {
			continue
		}

		setArgs := append([]string{"image", kind + "/" + name}, imageArgs...)
		if err := cli.RunInNamespace(ctx, nil, out, "set", ns, setArgs...); err != nil {
			return fmt.Errorf("setting image for %s/%s: %w", kind, name, err)
		}
		if err := cli.RunInNamespace(ctx, nil, out, "rollout", ns, "restart", kind+"/"+name); err != nil {
			return fmt.Errorf("restarting %s/%s: %w", kind, name, err)
		}
	}
	return nil
}

// podContainers returns the containers (name + image) declared in the pod template
// of a workload manifest.
func podContainers(w unstructured.Unstructured) []containerRef {
	raw, found, _ := unstructured.NestedSlice(w.Object, "spec", "template", "spec", "containers")
	if !found {
		return nil
	}
	var containers []containerRef
	for _, c := range raw {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		image, _ := m["image"].(string)
		if name == "" || image == "" {
			continue
		}
		containers = append(containers, containerRef{name: name, image: image})
	}
	return containers
}
