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

package sync

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/testutil"
)

func TestDockerSync(t *testing.T) {
	tests := []struct {
		description string
		item        *Item
		expected    []string
	}{
		{
			description: "additions are added via tar",
			item: &Item{
				Image: "image:123",
				Artifact: &latest.Artifact{
					ImageName: "image",
				},
				Copy: syncMap{"test.go": {"/test.go"}},
			},
			expected: []string{"docker exec -i image tar xmf - -C / --no-same-owner"},
		},
		{
			description: "one deletion",
			item: &Item{
				Image: "image:123",
				Artifact: &latest.Artifact{
					ImageName: "image",
				},
				Delete: syncMap{"test.go": {"/test.go"}},
			},
			expected: []string{"docker exec -i image rm -rf -- /test.go"},
		},
		{
			description: "two deletions",
			item: &Item{
				Image: "image:123",
				Artifact: &latest.Artifact{
					ImageName: "image",
				},
				Delete: syncMap{"test.go": {"/test.go"}, "foobar.js": {"/dev/js/foobar.js"}},
			},
			expected: []string{"docker exec -i image rm -rf -- /dev/js/foobar.js /test.go"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cmdRecord := &TestCmdRecorder{}

			t.Override(&util.DefaultExecCommand, cmdRecord)
			NewContainerSyncer(nil, false).Sync(context.Background(), nil, test.item)

			// sync maps are unordered, but we can split the resulting command strings and compare elements
			t.CheckElementsMatch(strings.Split(test.expected[0], " "), strings.Split(cmdRecord.cmds[0], " "))
		})
	}
}

func TestDockerSyncRestart(t *testing.T) {
	tests := []struct {
		description string
		item        *Item
		wantCmds    []string
	}{
		{
			description: "restart after copy",
			item: &Item{
				Image: "image:123",
				Artifact: &latest.Artifact{
					ImageName: "image",
					Sync:      &latest.Sync{Restart: true},
				},
				Copy: syncMap{"test.go": {"/test.go"}},
			},
			wantCmds: []string{
				"docker exec -i image tar xmf - -C / --no-same-owner",
				"docker restart image",
			},
		},
		{
			description: "no restart when flag unset",
			item: &Item{
				Image:    "image:123",
				Artifact: &latest.Artifact{ImageName: "image"},
				Copy:     syncMap{"test.go": {"/test.go"}},
			},
			wantCmds: []string{"docker exec -i image tar xmf - -C / --no-same-owner"},
		},
		{
			description: "no restart when nothing changed",
			item: &Item{
				Image: "image:123",
				Artifact: &latest.Artifact{
					ImageName: "image",
					Sync:      &latest.Sync{Restart: true},
				},
			},
			wantCmds: nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cmdRecord := &TestCmdRecorder{}

			t.Override(&util.DefaultExecCommand, cmdRecord)
			NewContainerSyncer(nil, false).Sync(context.Background(), nil, test.item)

			t.CheckDeepEqual(test.wantCmds, cmdRecord.cmds)
		})
	}
}

func TestDockerAPISync(t *testing.T) {
	tests := []struct {
		description string
		item        *Item
		fake        *fakeDockerClient
		shouldErr   bool
		check       func(t *testutil.T, fake *fakeDockerClient)
	}{
		{
			description: "additions are copied via the archive API",
			item: &Item{
				Image:    "image:123",
				Artifact: &latest.Artifact{ImageName: "image"},
				Copy:     syncMap{"test.go": {"/test.go"}},
			},
			fake: &fakeDockerClient{containerID: "container-abc"},
			check: func(t *testutil.T, fake *fakeDockerClient) {
				t.CheckDeepEqual("container-abc", fake.copiedTo)
				t.CheckDeepEqual("/", fake.copyDest)
			},
		},
		{
			description: "deletions are run via exec",
			item: &Item{
				Image:    "image:123",
				Artifact: &latest.Artifact{ImageName: "image"},
				Delete:   syncMap{"test.go": {"/test.go"}},
			},
			fake: &fakeDockerClient{containerID: "container-abc"},
			check: func(t *testutil.T, fake *fakeDockerClient) {
				t.CheckTrue(fake.execRan)
				t.CheckDeepEqual([]string{"rm", "-rf", "--", "/test.go"}, fake.execCmd)
			},
		},
		{
			description: "inspect error propagates",
			item: &Item{
				Image:    "image:123",
				Artifact: &latest.Artifact{ImageName: "image"},
				Copy:     syncMap{"test.go": {"/test.go"}},
			},
			fake:      &fakeDockerClient{inspectErr: errors.New("no such container")},
			shouldErr: true,
		},
		{
			description: "restart after copy when enabled",
			item: &Item{
				Image:    "image:123",
				Artifact: &latest.Artifact{ImageName: "image", Sync: &latest.Sync{Restart: true}},
				Copy:     syncMap{"test.go": {"/test.go"}},
			},
			fake: &fakeDockerClient{containerID: "container-abc"},
			check: func(t *testutil.T, fake *fakeDockerClient) {
				t.CheckDeepEqual("container-abc", fake.restarted)
			},
		},
		{
			description: "no restart when flag unset",
			item: &Item{
				Image:    "image:123",
				Artifact: &latest.Artifact{ImageName: "image"},
				Copy:     syncMap{"test.go": {"/test.go"}},
			},
			fake: &fakeDockerClient{containerID: "container-abc"},
			check: func(t *testutil.T, fake *fakeDockerClient) {
				t.CheckDeepEqual("", fake.restarted)
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := NewContainerSyncer(test.fake, true).Sync(context.Background(), nil, test.item)
			t.CheckError(test.shouldErr, err)
			if test.check != nil {
				test.check(t, test.fake)
			}
		})
	}
}

type fakeDockerClient struct {
	containerID string
	inspectErr  error

	copiedTo  string
	copyDest  string
	execCmd   []string
	execRan   bool
	exitCode  int
	restarted string
}

func (f *fakeDockerClient) ContainerInspect(_ context.Context, _ string, _ client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	if f.inspectErr != nil {
		return client.ContainerInspectResult{}, f.inspectErr
	}
	return client.ContainerInspectResult{Container: container.InspectResponse{ID: f.containerID}}, nil
}

func (f *fakeDockerClient) CopyToContainer(_ context.Context, c string, opts client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
	f.copiedTo = c
	f.copyDest = opts.DestinationPath
	if opts.Content != nil {
		// Drain the tar so the writer goroutine can finish.
		_, _ = io.Copy(io.Discard, opts.Content)
	}
	return client.CopyToContainerResult{}, nil
}

func (f *fakeDockerClient) ExecCreate(_ context.Context, _ string, opts client.ExecCreateOptions) (client.ExecCreateResult, error) {
	f.execCmd = opts.Cmd
	return client.ExecCreateResult{ID: "exec-id"}, nil
}

func (f *fakeDockerClient) ExecStart(_ context.Context, _ string, _ client.ExecStartOptions) (client.ExecStartResult, error) {
	f.execRan = true
	return client.ExecStartResult{}, nil
}

func (f *fakeDockerClient) ExecInspect(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
	return client.ExecInspectResult{Running: false, ExitCode: f.exitCode}, nil
}

func (f *fakeDockerClient) ContainerRestart(_ context.Context, c string, _ client.ContainerRestartOptions) (client.ContainerRestartResult, error) {
	f.restarted = c
	return client.ContainerRestartResult{}, nil
}
