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

package hooks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/lucky-tools/devloop/pkg/devloop/kubectl"
	kubernetesclient "github.com/lucky-tools/devloop/pkg/devloop/kubernetes/client"
	"github.com/lucky-tools/devloop/pkg/devloop/log"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/testutil"
)

func TestSyncHooks(t *testing.T) {
	testutil.Run(t, "TestSyncHooks", func(t *testutil.T) {
		workDir, _ := filepath.Abs("./foo")
		hooks := latest.SyncHooks{
			PreHooks: []latest.SyncHookItem{
				{
					HostHook: &latest.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo pre-hook running with DEVLOOP_IMAGE=$DEVLOOP_IMAGE,DEVLOOP_BUILD_CONTEXT=$DEVLOOP_BUILD_CONTEXT,DEVLOOP_FILES_ADDED_OR_MODIFIED=$DEVLOOP_FILES_ADDED_OR_MODIFIED,DEVLOOP_FILES_DELETED=$DEVLOOP_FILES_DELETED,DEVLOOP_KUBE_CONTEXT=$DEVLOOP_KUBE_CONTEXT,DEVLOOP_NAMESPACES=$DEVLOOP_NAMESPACES"},
					},
				},
				{
					HostHook: &latest.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo pre-hook running with DEVLOOP_IMAGE=%DEVLOOP_IMAGE%,DEVLOOP_BUILD_CONTEXT=%DEVLOOP_BUILD_CONTEXT%,DEVLOOP_FILES_ADDED_OR_MODIFIED=%DEVLOOP_FILES_ADDED_OR_MODIFIED%,DEVLOOP_FILES_DELETED=%DEVLOOP_FILES_DELETED%,DEVLOOP_KUBE_CONTEXT=%DEVLOOP_KUBE_CONTEXT%,DEVLOOP_NAMESPACES=%DEVLOOP_NAMESPACES%"},
					},
				},
				{
					ContainerHook: &latest.ContainerHook{
						Command: []string{"foo", "pre-hook"},
					},
				},
			},
			PostHooks: []latest.SyncHookItem{
				{
					HostHook: &latest.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo post-hook running with DEVLOOP_IMAGE=$DEVLOOP_IMAGE,DEVLOOP_BUILD_CONTEXT=$DEVLOOP_BUILD_CONTEXT,DEVLOOP_FILES_ADDED_OR_MODIFIED=$DEVLOOP_FILES_ADDED_OR_MODIFIED,DEVLOOP_FILES_DELETED=$DEVLOOP_FILES_DELETED,DEVLOOP_KUBE_CONTEXT=$DEVLOOP_KUBE_CONTEXT,DEVLOOP_NAMESPACES=$DEVLOOP_NAMESPACES"},
					},
				},
				{
					HostHook: &latest.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo post-hook running with DEVLOOP_IMAGE=%DEVLOOP_IMAGE%,DEVLOOP_BUILD_CONTEXT=%DEVLOOP_BUILD_CONTEXT%,DEVLOOP_FILES_ADDED_OR_MODIFIED=%DEVLOOP_FILES_ADDED_OR_MODIFIED%,DEVLOOP_FILES_DELETED=%DEVLOOP_FILES_DELETED%,DEVLOOP_KUBE_CONTEXT=%DEVLOOP_KUBE_CONTEXT%,DEVLOOP_NAMESPACES=%DEVLOOP_NAMESPACES%"},
					},
				},
				{
					ContainerHook: &latest.ContainerHook{
						Command: []string{"foo", "post-hook"},
					},
				},
			},
		}
		preHostHookOut := fmt.Sprintf("pre-hook running with DEVLOOP_IMAGE=gcr.io/foo/img1:latest,DEVLOOP_BUILD_CONTEXT=%s,DEVLOOP_FILES_ADDED_OR_MODIFIED=foo1;bar1,DEVLOOP_FILES_DELETED=foo2;bar2,DEVLOOP_KUBE_CONTEXT=context1,DEVLOOP_NAMESPACES=np1,np2", workDir)
		preContainerHookOut := "container pre-hook succeeded"
		postHostHookOut := fmt.Sprintf("post-hook running with DEVLOOP_IMAGE=gcr.io/foo/img1:latest,DEVLOOP_BUILD_CONTEXT=%s,DEVLOOP_FILES_ADDED_OR_MODIFIED=foo1;bar1,DEVLOOP_FILES_DELETED=foo2;bar2,DEVLOOP_KUBE_CONTEXT=context1,DEVLOOP_NAMESPACES=np1,np2", workDir)
		postContainerHookOut := "container post-hook succeeded"

		artifact := &latest.Artifact{
			ImageName: "img1",
			Workspace: workDir,
			Sync: &latest.Sync{
				LifecycleHooks: hooks,
			},
		}
		image := "gcr.io/foo/img1:latest"
		namespaces := []string{"np1", "np2"}
		kubeContext := "context1"
		opts, err := NewSyncEnvOpts(artifact, image, []string{"foo1", "bar1"}, []string{"foo2", "bar2"}, namespaces, kubeContext)
		t.CheckNoError(err)
		formatter := func(corev1.Pod, corev1.ContainerStatus, func() bool) log.Formatter { return mockLogFormatter{} }
		runner := NewSyncRunner(&kubectl.CLI{KubeContext: kubeContext}, artifact.ImageName, image, namespaces, formatter, artifact.Sync.LifecycleHooks, opts)

		t.Override(&util.DefaultExecCommand,
			testutil.CmdRunWithOutput("kubectl --context context1 exec pod1 --namespace np1 -c container1 -- foo pre-hook", preContainerHookOut).
				AndRunWithOutput("kubectl --context context1 exec pod1 --namespace np1 -c container1 -- foo post-hook", postContainerHookOut))
		t.Override(&kubernetesclient.Client, fakeKubernetesClient)
		var preOut, postOut bytes.Buffer
		err = runner.RunPreHooks(context.Background(), &preOut)
		t.CheckNoError(err)
		t.CheckContains(preHostHookOut, preOut.String())
		t.CheckContains(preContainerHookOut, strings.TrimRight(preOut.String(), "\r\n"))
		err = runner.RunPostHooks(context.Background(), &postOut)
		t.CheckNoError(err)
		t.CheckContains(postHostHookOut, postOut.String())
		t.CheckContains(postContainerHookOut, postOut.String())
	})
}

func fakeKubernetesClient(string) (kubernetes.Interface, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "np1",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "rs",
					Kind: "ReplicaSet",
				},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/foo/img1:latest",
			},
		}},
	}
	return fakeclient.NewSimpleClientset(pod), nil
}

type mockLogFormatter struct{}

func (mockLogFormatter) Name() string { return "" }

func (mockLogFormatter) PrintLine(w io.Writer, s string) { fmt.Fprint(w, s) }
