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
	"os/exec"

	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// ContainerSyncer syncs files into containers running in a local Docker daemon.
//
// It supports two transfer backends, mirroring the `build.local` CLI/API split:
//   - CLI (default): spawns `docker exec` to pipe a tar archive into the container.
//   - API: uses the Docker Engine API (`PUT /containers/{id}/archive`), which
//     requires no `tar` binary inside the container.
type ContainerSyncer struct {
	useAPI bool
	client dockerClient
}

func NewContainerSyncer(client dockerClient, useAPI bool) *ContainerSyncer {
	return &ContainerSyncer{client: client, useAPI: useAPI}
}

func (s *ContainerSyncer) Sync(ctx context.Context, _ io.Writer, item *Item) error {
	if len(item.Copy) > 0 {
		log.Entry(ctx).Info("Copying files:", item.Copy, "to", item.Image)
		if err := s.copy(ctx, item.Artifact.ImageName, item.Copy); err != nil {
			return fmt.Errorf("copying files: %w", err)
		}
	}

	if len(item.Delete) > 0 {
		log.Entry(ctx).Info("Deleting files:", item.Delete, "from", item.Image)
		if err := s.delete(ctx, item.Artifact.ImageName, item.Delete); err != nil {
			return fmt.Errorf("deleting files: %w", err)
		}
	}

	if item.Artifact.Sync != nil && item.Artifact.Sync.Restart && item.HasChanges() {
		log.Entry(ctx).Info("Restarting container:", item.Artifact.ImageName)
		if err := s.restart(ctx, item.Artifact.ImageName); err != nil {
			return fmt.Errorf("restarting container: %w", err)
		}
	}

	return nil
}

func (s *ContainerSyncer) restart(ctx context.Context, containerName string) error {
	if s.useAPI {
		return s.restartAPI(ctx, containerName)
	}
	_, err := util.RunCmdOut(ctx, exec.CommandContext(ctx, "docker", "restart", containerName))
	return err
}

func (s *ContainerSyncer) copy(ctx context.Context, containerName string, files syncMap) error {
	if s.useAPI {
		return s.copyAPI(ctx, containerName, files)
	}
	_, err := util.RunCmdOut(ctx, s.copyFileFn(ctx, containerName, files))
	return err
}

func (s *ContainerSyncer) delete(ctx context.Context, containerName string, files syncMap) error {
	if s.useAPI {
		return s.deleteAPI(ctx, containerName, files)
	}
	_, err := util.RunCmdOut(ctx, s.deleteFileFn(ctx, containerName, files))
	return err
}

func (s *ContainerSyncer) deleteFileFn(ctx context.Context, containerName string, files syncMap) *exec.Cmd {
	var args []string
	args = append(args, "exec", "-i", containerName, "rm", "-rf", "--")
	for _, dsts := range files {
		args = append(args, dsts...)
	}
	return exec.CommandContext(ctx, "docker", args...)
}

func (s *ContainerSyncer) copyFileFn(ctx context.Context, containerName string, files syncMap) *exec.Cmd {
	// Use "m" flag to touch the files as they are copied.
	reader, writer := io.Pipe()
	go func() {
		if err := util.CreateMappedTar(ctx, writer, "/", files); err != nil {
			writer.CloseWithError(err)
		} else {
			writer.Close()
		}
	}()

	copyCmd := exec.CommandContext(ctx, "docker", "exec", "-i", containerName, "tar", "xmf", "-", "-C", "/", "--no-same-owner")
	copyCmd.Stdin = reader
	return copyCmd
}
