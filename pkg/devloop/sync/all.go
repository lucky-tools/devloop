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

package sync

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar"

	"github.com/lucky-tools/devloop/pkg/devloop/build/list"
	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

// AllItems builds a full-sync Item that copies every file currently matching an
// artifact's sync rules into its running containers. Unlike NewItem, which only
// syncs files that changed since the last watch cycle, this syncs the entire
// current state. It is the appropriate entry point for a one-shot `devloop sync`.
func AllItems(ctx context.Context, a *latest.Artifact, builds []graph.Artifact, cfg docker.Config) (*Item, error) {
	if a.Sync == nil {
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find tag for image %q; run `devloop build`/`devloop deploy` first or pass --build-artifacts/--images", a.ImageName)
	}

	switch {
	case len(a.Sync.Manual) > 0:
		return allManualItems(ctx, a, tag, cfg)

	case a.Sync.Auto != nil:
		return nil, fmt.Errorf("full sync is not supported for `sync.auto` (jib/buildpacks); use `sync.manual` or `sync.infer`")

	case len(a.Sync.Infer) > 0:
		return allInferredItems(ctx, a, tag, cfg)

	default:
		return nil, nil
	}
}

func allManualItems(ctx context.Context, a *latest.Artifact, tag string, cfg docker.Config) (*Item, error) {
	workspace, err := filepath.Abs(a.Workspace)
	if err != nil {
		return nil, fmt.Errorf("getting absolute workspace for %q: %w", a.ImageName, err)
	}

	containerWd, err := WorkingDir(ctx, tag, cfg)
	if err != nil {
		return nil, fmt.Errorf("retrieving working dir for %q: %w", tag, err)
	}

	var srcs []string
	for _, r := range a.Sync.Manual {
		srcs = append(srcs, r.Src)
	}

	relPaths, err := list.Files(workspace, srcs, nil)
	if err != nil {
		return nil, fmt.Errorf("listing files to sync for %q: %w", a.ImageName, err)
	}

	toCopy := make(map[string][]string)
	for _, relPath := range relPaths {
		dsts, err := matchSyncRules(a.Sync.Manual, relPath, containerWd)
		if err != nil {
			return nil, fmt.Errorf("matching sync rules for %q: %w", relPath, err)
		}
		if len(dsts) == 0 {
			continue
		}
		toCopy[filepath.Join(workspace, relPath)] = dsts
	}

	return &Item{Image: tag, Artifact: a, Copy: toCopy}, nil
}

func allInferredItems(ctx context.Context, a *latest.Artifact, tag string, cfg docker.Config) (*Item, error) {
	workspace, err := filepath.Abs(a.Workspace)
	if err != nil {
		return nil, fmt.Errorf("getting absolute workspace for %q: %w", a.ImageName, err)
	}

	syncMap, err := SyncMap(ctx, a, cfg)
	if err != nil {
		return nil, fmt.Errorf("inferring sync map for image %q: %w", a.ImageName, err)
	}

	toCopy := make(map[string][]string)
	for relPath, dsts := range syncMap {
		matches := false
		for _, p := range a.Sync.Infer {
			ok, err := doublestar.PathMatch(filepath.FromSlash(p), relPath)
			if err != nil {
				return nil, fmt.Errorf("pattern error for %q: %w", relPath, err)
			}
			if ok {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		toCopy[filepath.Join(workspace, relPath)] = dsts
	}

	return &Item{Image: tag, Artifact: a, Copy: toCopy}, nil
}
