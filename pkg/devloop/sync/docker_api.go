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
	"fmt"
	"io"
	"time"

	"github.com/moby/moby/client"

	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// dockerClient is the subset of the Docker Engine API used for API-backed sync.
// It is satisfied by client.APIClient (and docker.LocalDaemon.RawClient()).
type dockerClient interface {
	ContainerInspect(ctx context.Context, container string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	CopyToContainer(ctx context.Context, container string, options client.CopyToContainerOptions) (client.CopyToContainerResult, error)
	ExecCreate(ctx context.Context, container string, options client.ExecCreateOptions) (client.ExecCreateResult, error)
	ExecStart(ctx context.Context, execID string, options client.ExecStartOptions) (client.ExecStartResult, error)
	ExecInspect(ctx context.Context, execID string, options client.ExecInspectOptions) (client.ExecInspectResult, error)
	ContainerRestart(ctx context.Context, container string, options client.ContainerRestartOptions) (client.ContainerRestartResult, error)
}

func (s *ContainerSyncer) copyAPI(ctx context.Context, containerName string, files syncMap) error {
	containerID, err := s.containerID(ctx, containerName)
	if err != nil {
		return err
	}

	reader, writer := io.Pipe()
	go func() {
		if err := util.CreateMappedTar(ctx, writer, "/", files); err != nil {
			writer.CloseWithError(err)
		} else {
			writer.Close()
		}
	}()

	_, err = s.client.CopyToContainer(ctx, containerID, client.CopyToContainerOptions{
		DestinationPath: "/",
		Content:         reader,
	})
	return err
}

func (s *ContainerSyncer) deleteAPI(ctx context.Context, containerName string, files syncMap) error {
	containerID, err := s.containerID(ctx, containerName)
	if err != nil {
		return err
	}

	args := []string{"rm", "-rf", "--"}
	for _, dsts := range files {
		args = append(args, dsts...)
	}

	exec, err := s.client.ExecCreate(ctx, containerID, client.ExecCreateOptions{Cmd: args})
	if err != nil {
		return fmt.Errorf("creating rm exec: %w", err)
	}

	if _, err := s.client.ExecStart(ctx, exec.ID, client.ExecStartOptions{Detach: true}); err != nil {
		return fmt.Errorf("starting rm exec: %w", err)
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		inspect, err := s.client.ExecInspect(ctx, exec.ID, client.ExecInspectOptions{})
		if err != nil {
			return fmt.Errorf("inspecting rm exec: %w", err)
		}
		if !inspect.Running {
			if inspect.ExitCode != 0 {
				return fmt.Errorf("rm exited with code %d", inspect.ExitCode)
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *ContainerSyncer) restartAPI(ctx context.Context, containerName string) error {
	containerID, err := s.containerID(ctx, containerName)
	if err != nil {
		return err
	}
	_, err = s.client.ContainerRestart(ctx, containerID, client.ContainerRestartOptions{})
	return err
}

func (s *ContainerSyncer) containerID(ctx context.Context, name string) (string, error) {
	inspect, err := s.client.ContainerInspect(ctx, name, client.ContainerInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("inspecting container %q: %w", name, err)
	}
	return inspect.Container.ID, nil
}
