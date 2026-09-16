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
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/testutil"
)

const Windows = "windows"

func TestBuildHooks(t *testing.T) {
	workDir, _ := filepath.Abs("./foo")
	tests := []struct {
		description       string
		artifact          latest.Artifact
		image             string
		pushImage         bool
		requiresWindowsOS bool
		preHookOut        string
		postHookOut       string
	}{
		{
			description: "linux/darwin build hook on matching host",
			artifact: latest.Artifact{
				ImageName: "img1",
				Workspace: "./foo",
				LifecycleHooks: latest.BuildHooks{
					PreHooks: []latest.HostHook{
						{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo pre-hook running with DEVLOOP_IMAGE=$DEVLOOP_IMAGE,DEVLOOP_PUSH_IMAGE=$DEVLOOP_PUSH_IMAGE,DEVLOOP_IMAGE_REPO=$DEVLOOP_IMAGE_REPO,DEVLOOP_IMAGE_TAG=$DEVLOOP_IMAGE_TAG,DEVLOOP_BUILD_CONTEXT=$DEVLOOP_BUILD_CONTEXT"},
						},
					},
					PostHooks: []latest.HostHook{
						{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo post-hook running with DEVLOOP_IMAGE=$DEVLOOP_IMAGE,DEVLOOP_PUSH_IMAGE=$DEVLOOP_PUSH_IMAGE,DEVLOOP_IMAGE_REPO=$DEVLOOP_IMAGE_REPO,DEVLOOP_IMAGE_TAG=$DEVLOOP_IMAGE_TAG,DEVLOOP_BUILD_CONTEXT=$DEVLOOP_BUILD_CONTEXT"},
						},
					},
				},
			},
			image:       "gcr.io/foo/img1:latest",
			pushImage:   true,
			preHookOut:  fmt.Sprintf("pre-hook running with DEVLOOP_IMAGE=gcr.io/foo/img1:latest,DEVLOOP_PUSH_IMAGE=true,DEVLOOP_IMAGE_REPO=gcr.io/foo,DEVLOOP_IMAGE_TAG=latest,DEVLOOP_BUILD_CONTEXT=%s\n", workDir),
			postHookOut: fmt.Sprintf("post-hook running with DEVLOOP_IMAGE=gcr.io/foo/img1:latest,DEVLOOP_PUSH_IMAGE=true,DEVLOOP_IMAGE_REPO=gcr.io/foo,DEVLOOP_IMAGE_TAG=latest,DEVLOOP_BUILD_CONTEXT=%s\n", workDir),
		},
		{
			description:       "linux/darwin build hook on non-matching host",
			requiresWindowsOS: true,
			artifact: latest.Artifact{
				ImageName: "img1",
				Workspace: "./foo",
				LifecycleHooks: latest.BuildHooks{
					PreHooks: []latest.HostHook{
						{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo pre-hook running with DEVLOOP_IMAGE=$DEVLOOP_IMAGE,DEVLOOP_PUSH_IMAGE=$DEVLOOP_PUSH_IMAGE,DEVLOOP_IMAGE_REPO=$DEVLOOP_IMAGE_REPO,DEVLOOP_IMAGE_TAG=$DEVLOOP_IMAGE_TAG,DEVLOOP_BUILD_CONTEXT=$DEVLOOP_BUILD_CONTEXT"},
						},
					},
					PostHooks: []latest.HostHook{
						{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo post-hook running with DEVLOOP_IMAGE=$DEVLOOP_IMAGE,DEVLOOP_PUSH_IMAGE=$DEVLOOP_PUSH_IMAGE,DEVLOOP_IMAGE_REPO=$DEVLOOP_IMAGE_REPO,DEVLOOP_IMAGE_TAG=$DEVLOOP_IMAGE_TAG,DEVLOOP_BUILD_CONTEXT=$DEVLOOP_BUILD_CONTEXT"},
						},
					},
				},
			},
			image:     "gcr.io/foo/img1:latest",
			pushImage: true,
		},
		{
			description:       "windows build hook on matching host",
			requiresWindowsOS: true,
			artifact: latest.Artifact{
				ImageName: "img1",
				Workspace: "./foo",
				LifecycleHooks: latest.BuildHooks{
					PreHooks: []latest.HostHook{
						{
							OS:      []string{Windows},
							Command: []string{"cmd.exe", "/C", "echo pre-hook running with DEVLOOP_IMAGE=%DEVLOOP_IMAGE%,DEVLOOP_PUSH_IMAGE=%DEVLOOP_PUSH_IMAGE%,DEVLOOP_IMAGE_REPO=%DEVLOOP_IMAGE_REPO%,DEVLOOP_IMAGE_TAG=%DEVLOOP_IMAGE_TAG%,DEVLOOP_BUILD_CONTEXT=%DEVLOOP_BUILD_CONTEXT%"},
						},
					},
					PostHooks: []latest.HostHook{
						{
							OS:      []string{Windows},
							Command: []string{"cmd.exe", "/C", "echo post-hook running with DEVLOOP_IMAGE=%DEVLOOP_IMAGE%,DEVLOOP_PUSH_IMAGE=%DEVLOOP_PUSH_IMAGE%,DEVLOOP_IMAGE_REPO=%DEVLOOP_IMAGE_REPO%,DEVLOOP_IMAGE_TAG=%DEVLOOP_IMAGE_TAG%,DEVLOOP_BUILD_CONTEXT=%DEVLOOP_BUILD_CONTEXT%"},
						},
					},
				},
			},
			image:       "gcr.io/foo/img1:latest",
			pushImage:   true,
			preHookOut:  fmt.Sprintf("pre-hook running with DEVLOOP_IMAGE=gcr.io/foo/img1:latest,DEVLOOP_PUSH_IMAGE=true,DEVLOOP_IMAGE_REPO=gcr.io/foo,DEVLOOP_IMAGE_TAG=latest,DEVLOOP_BUILD_CONTEXT=%s\r\n", workDir),
			postHookOut: fmt.Sprintf("post-hook running with DEVLOOP_IMAGE=gcr.io/foo/img1:latest,DEVLOOP_PUSH_IMAGE=true,DEVLOOP_IMAGE_REPO=gcr.io/foo,DEVLOOP_IMAGE_TAG=latest,DEVLOOP_BUILD_CONTEXT=%s\r\n", workDir),
		},
		{
			description: "windows build hook on non-matching host",
			artifact: latest.Artifact{
				ImageName: "img1",
				Workspace: "./foo",
				LifecycleHooks: latest.BuildHooks{
					PreHooks: []latest.HostHook{
						{
							OS:      []string{Windows},
							Command: []string{"cmd.exe", "/C", "echo pre-hook running with DEVLOOP_IMAGE=%DEVLOOP_IMAGE%,DEVLOOP_PUSH_IMAGE=%DEVLOOP_PUSH_IMAGE%,DEVLOOP_IMAGE_REPO=%DEVLOOP_IMAGE_REPO%,DEVLOOP_IMAGE_TAG=%DEVLOOP_IMAGE_TAG%,DEVLOOP_BUILD_CONTEXT=%DEVLOOP_BUILD_CONTEXT%"},
						},
					},
					PostHooks: []latest.HostHook{
						{
							OS:      []string{Windows},
							Command: []string{"cmd.exe", "/C", "echo post-hook running with DEVLOOP_IMAGE=%DEVLOOP_IMAGE%,DEVLOOP_PUSH_IMAGE=%DEVLOOP_PUSH_IMAGE%,DEVLOOP_IMAGE_REPO=%DEVLOOP_IMAGE_REPO%,DEVLOOP_IMAGE_TAG=%DEVLOOP_IMAGE_TAG%,DEVLOOP_BUILD_CONTEXT=%DEVLOOP_BUILD_CONTEXT%"},
						},
					},
				},
			},
			image:     "gcr.io/foo/img1:latest",
			pushImage: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.requiresWindowsOS != (runtime.GOOS == Windows) {
				t.Skip()
			}
			opts, err := NewBuildEnvOpts(&test.artifact, test.image, test.pushImage)
			t.CheckNoError(err)
			runner := BuildRunner(test.artifact.LifecycleHooks, opts)
			var preOut, postOut bytes.Buffer
			err = runner.RunPreHooks(context.Background(), &preOut)
			t.CheckNoError(err)
			t.CheckContains(test.preHookOut, preOut.String())
			err = runner.RunPostHooks(context.Background(), &postOut)
			t.CheckNoError(err)
			t.CheckContains(test.postHookOut, postOut.String())
		})
	}
}
