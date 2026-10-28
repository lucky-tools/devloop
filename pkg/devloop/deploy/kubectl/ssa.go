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

package kubectl

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	k8syaml "sigs.k8s.io/yaml"

	deployutil "github.com/lucky-tools/devloop/pkg/devloop/deploy/util"
	kubernetesclient "github.com/lucky-tools/devloop/pkg/devloop/kubernetes/client"
	"github.com/lucky-tools/devloop/pkg/devloop/kubernetes/manifest"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
)

// applyFieldManager is the field manager used for server-side apply patches.
// It replaces `kubectl apply`'s client-side 3-way merge and lets the API
// server track which fields devloop owns. The value must stay stable so later
// applies do not conflict with fields managed by other tools.
const applyFieldManager = "devloop"

// ssaClients bundles the client-go clients used to apply/delete resources
// without shelling out to the `kubectl` binary.
type ssaClients struct {
	dyn   dynamic.Interface
	disco discovery.DiscoveryInterface
}

// newSSAClients builds the dynamic + discovery clients for the configured
// kubeContext. Both already go through the kubeconfig loaded by devloop
// (including a `kubectl proxy` HTTP endpoint), so no kubectl binary is needed.
func (c *CLI) newSSAClients() (*ssaClients, error) {
	dyn, err := kubernetesclient.DynamicClient(c.KubeContext)
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes dynamic client: %w", err)
	}
	client, err := kubernetesclient.Client(c.KubeContext)
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}
	return &ssaClients{dyn: dyn, disco: client.Discovery()}, nil
}

// toUnstructured decodes each YAML document in a ManifestList into an
// unstructured object, reusing the same decoding path as manifest.Filter.
func toUnstructured(manifests manifest.ManifestList) ([]unstructured.Unstructured, error) {
	var objs []unstructured.Unstructured
	for _, y := range manifests {
		// Skip empty documents (e.g. a trailing `---` separator).
		if strings.TrimSpace(string(y)) == "" {
			continue
		}
		j, err := k8syaml.YAMLToJSON(y)
		if err != nil {
			return nil, fmt.Errorf("yaml to json error: %w", err)
		}
		var obj unstructured.Unstructured
		if err := obj.UnmarshalJSON(j); err != nil {
			return nil, fmt.Errorf("unmarshaling config: %w", err)
		}
		// Expand `kind: List` into its items, and skip documents without an
		// apiVersion/kind, matching `kubectl apply`'s behaviour.
		if obj.IsList() {
			list, err := obj.ToList()
			if err != nil {
				return nil, fmt.Errorf("converting list: %w", err)
			}
			objs = append(objs, list.Items...)
			continue
		}
		if obj.GetAPIVersion() == "" || obj.GetKind() == "" {
			continue
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

// resourceInterface resolves a manifest object to a dynamic ResourceInterface
// for the correct group/version/resource and namespace. Namespaced resources
// without an explicit namespace fall back to the deployer's default namespace
// (matching `kubectl apply`'s behaviour).
func (c *CLI) resourceInterface(clients *ssaClients, obj *unstructured.Unstructured) (dynamic.ResourceInterface, string, error) {
	gvk := obj.GroupVersionKind()
	namespaced, gvr, err := deployutil.GroupVersionResource(clients.disco, gvk)
	if err != nil {
		return nil, "", fmt.Errorf("resolving %s: %w", gvk.String(), err)
	}
	if !namespaced {
		return clients.dyn.Resource(gvr), "", nil
	}
	ns := obj.GetNamespace()
	if ns == "" {
		ns = c.Namespace
		if ns == "" {
			ns = "default"
		}
	}
	return clients.dyn.Resource(gvr).Namespace(ns), ns, nil
}

// resourceName returns kubectl's `<resource>.<group>` display form, e.g.
// "deployment.apps" or "configmap" for core resources.
func resourceName(gvk schema.GroupVersionKind) string {
	kind := strings.ToLower(gvk.Kind)
	if gvk.Group == "" {
		return kind
	}
	return kind + "." + gvk.Group
}

// applySSA applies each manifest using a server-side apply patch, the client-go
// equivalent of `kubectl apply -f -`.
func (c *CLI) applySSA(ctx context.Context, out io.Writer, manifests manifest.ManifestList) error {
	objs, err := toUnstructured(manifests)
	if err != nil {
		return err
	}
	clients, err := c.newSSAClients()
	if err != nil {
		return err
	}
	for i := range objs {
		obj := &objs[i]
		ri, _, err := c.resourceInterface(clients, obj)
		if err != nil {
			return err
		}

		// Match kubectl apply's "created"/"configured" output.
		action := "configured"
		if _, err := ri.Get(ctx, obj.GetName(), metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				action = "created"
			} else {
				return fmt.Errorf("getting %s %q: %w", obj.GetKind(), obj.GetName(), err)
			}
		}

		data, err := obj.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshaling %s %q: %w", obj.GetKind(), obj.GetName(), err)
		}
		force := true
		if _, err := ri.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: applyFieldManager,
			Force:        &force,
		}); err != nil {
			return fmt.Errorf("applying %s %q: %w", obj.GetKind(), obj.GetName(), err)
		}

		fmt.Fprintf(out, "%s/%s %s\n", resourceName(obj.GroupVersionKind()), obj.GetName(), action)
	}
	return nil
}

// deleteSSA deletes each manifest using the dynamic client, the client-go
// equivalent of `kubectl delete --ignore-not-found --wait=false -f -`.
func (c *CLI) deleteSSA(ctx context.Context, manifests manifest.ManifestList) error {
	objs, err := toUnstructured(manifests)
	if err != nil {
		return err
	}
	clients, err := c.newSSAClients()
	if err != nil {
		return err
	}
	for i := range objs {
		obj := &objs[i]
		ri, _, err := c.resourceInterface(clients, obj)
		if err != nil {
			return err
		}
		if err := ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("deleting %s %q: %w", obj.GetKind(), obj.GetName(), err)
		}
	}
	return nil
}

// waitForDeletionsSSA polls each manifest until none is marked for deletion,
// the client-go equivalent of `kubectl get -f - --ignore-not-found -ojson`.
func (c *CLI) waitForDeletionsSSA(ctx context.Context, out io.Writer, manifests manifest.ManifestList) error {
	objs, err := toUnstructured(manifests)
	if err != nil {
		return err
	}
	clients, err := c.newSSAClients()
	if err != nil {
		return err
	}

	// Resolve each object to its ResourceInterface once, before the polling
	// loop, so discovery is not re-queried on every iteration.
	type target struct {
		ri   dynamic.ResourceInterface
		name string
	}
	targets := make([]target, 0, len(objs))
	for i := range objs {
		obj := &objs[i]
		ri, _, err := c.resourceInterface(clients, obj)
		if err != nil {
			return err
		}
		targets = append(targets, target{ri: ri, name: obj.GetName()})
	}

	previousList := ""
	previousCount := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%d resources failed to complete their deletion before a new deployment: %s", previousCount, previousList)
		default:
		}

		var marked []string
		for _, t := range targets {
			got, err := t.ri.Get(ctx, t.name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if got.GetDeletionTimestamp() != nil {
				marked = append(marked, t.name)
			}
		}

		if len(marked) == 0 {
			return nil
		}

		list := `"` + strings.Join(marked, `", "`) + `"`
		log.Entry(ctx).Debug("Resources are marked for deletion: ", list)
		if list != previousList {
			if len(marked) == 1 {
				fmt.Fprintf(out, "%s is marked for deletion, waiting for completion\n", list)
			} else {
				fmt.Fprintf(out, "%d resources are marked for deletion, waiting for completion: %s\n", len(marked), list)
			}
			previousList = list
			previousCount = len(marked)
		}

		select {
		case <-ctx.Done():
		case <-time.After(c.waitForDeletions.Delay):
		}
	}
}
