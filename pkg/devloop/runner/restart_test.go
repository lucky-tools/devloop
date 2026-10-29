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
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

func TestPodContainers(t *testing.T) {
	w := unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{"name": "app", "image": "my-app:v2"},
						map[string]interface{}{"name": "sidecar", "image": "nginx:1.23"},
					},
				},
			},
		},
	}}

	want := []containerRef{
		{name: "app", image: "my-app:v2"},
		{name: "sidecar", image: "nginx:1.23"},
	}
	if got := podContainers(w); !reflect.DeepEqual(got, want) {
		t.Fatalf("podContainers() = %v, want %v", got, want)
	}
}

func TestWorkloadSelector(t *testing.T) {
	tests := []struct {
		group, kind string
		want        bool
	}{
		{"apps", "Deployment", true},
		{"apps", "StatefulSet", true},
		{"apps", "DaemonSet", true},
		{"extensions", "Deployment", true},
		{"apps", "Service", false},
		{"", "Service", false},
		{"", "ConfigMap", false},
	}
	for _, tt := range tests {
		if got := workloadSelector.Matches(tt.group, tt.kind); got != tt.want {
			t.Errorf("workloadSelector.Matches(%q, %q) = %v, want %v", tt.group, tt.kind, got, tt.want)
		}
	}
}

type fakeTagger struct {
	tag string
	err error
}

func (t *fakeTagger) GenerateTag(context.Context, latest.Artifact) (string, error) {
	return t.tag, t.err
}

func TestBuilderTagFor(t *testing.T) {
	b := &Builder{tagger: &fakeTagger{tag: "v1"}}
	tag, err := b.TagFor(context.Background(), latest.Artifact{ImageName: "my-app"})
	if err != nil {
		t.Fatalf("TagFor() error = %v", err)
	}
	if want := "my-app:v1"; tag != want {
		t.Fatalf("TagFor() = %q, want %q", tag, want)
	}
}

func TestHasDockerDeploy(t *testing.T) {
	tests := []struct {
		name string
		cfgs []latest.DeployConfig
		want bool
	}{
		{name: "docker", cfgs: []latest.DeployConfig{{DockerDeploy: &latest.DockerDeploy{}}}, want: true},
		{name: "kubectl", cfgs: []latest.DeployConfig{{KubectlDeploy: &latest.KubectlDeploy{}}}, want: false},
		{name: "kpt", cfgs: []latest.DeployConfig{{KptDeploy: &latest.KptDeploy{}}}, want: false},
		{name: "none", cfgs: nil, want: false},
		{name: "docker among several", cfgs: []latest.DeployConfig{{KubectlDeploy: &latest.KubectlDeploy{}}, {DockerDeploy: &latest.DockerDeploy{}}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasDockerDeploy(tt.cfgs); got != tt.want {
				t.Fatalf("hasDockerDeploy() = %v, want %v", got, tt.want)
			}
		})
	}
}
