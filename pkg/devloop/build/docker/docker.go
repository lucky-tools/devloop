/*
Copyright 2020 The Skaffold Authors

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

package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util/stringslice"
	"github.com/lucky-tools/devloop/pkg/devloop/warnings"
)

// validateDockerfile normalizes the Dockerfile path and fails fast if it can't be found.
func validateDockerfile(a *latest.Artifact) (string, error) {
	dockerfile, err := docker.NormalizeDockerfilePath(a.Workspace, a.DockerArtifact.DockerfilePath)
	if err != nil {
		return "", dockerfileNotFound(fmt.Errorf("normalizing dockerfile path: %w", err), a.ImageName)
	}
	if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
		return "", dockerfileNotFound(err, a.ImageName)
	}
	return dockerfile, nil
}

// pullCacheFromImages pulls any configured cacheFrom images that are not present locally.
func pullCacheFromImages(ctx context.Context, out io.Writer, a *latest.DockerArtifact, pl v1.Platform, daemon docker.LocalDaemon, buildkit bool) error {
	// when using buildkit, avoid pulling as the builder not necessarily uses the local docker daemon
	if len(a.CacheFrom) == 0 || buildkit {
		return nil
	}

	for _, image := range a.CacheFrom {
		imageID, err := daemon.ImageID(ctx, image)
		if err != nil {
			return fmt.Errorf("getting imageID for %q: %w", image, err)
		}
		if imageID != "" {
			// already pulled
			continue
		}

		if err := daemon.Pull(ctx, out, image, pl); err != nil {
			warnings.Printf("cacheFrom image %q couldn't be pulled for platform %q\n", image, pl)
		}
	}

	return nil
}

// adjustCache returns an artifact where any cache references from the artifactImage is changed to the tagged built image name instead.
// Under buildkit, if cacheTag is configured, it will be used instead of the generated artifact tag (registry preserved)
// if no cacheTo was specified in the devloop yaml, it will add a tagged destination using the same cache source reference.
func adjustCache(ctx context.Context, a *latest.Artifact, artifactTag string, buildkit bool, buildKitCacheTag string, pushImages bool) *latest.Artifact {
	cacheRef := a.ImageName // lookup value to be replaced
	cacheTag := artifactTag // full reference to be used
	if os.Getenv("DEVLOOP_DISABLE_DOCKER_CACHE_ADJUSTMENT") != "" {
		// allow this behaviour to be disabled
		return a
	}
	if buildkit {
		// compute the full cache reference (including registry, preserving tag)
		tag := buildKitCacheTag
		imgRef, err := docker.ParseReference(artifactTag)
		if err != nil {
			log.Entry(ctx).Errorf("couldn't parse image tag: %v", err)
		} else if tag != "" {
			cacheTag = fmt.Sprintf("%s:%s", imgRef.BaseName, tag)
		}
	}
	if !stringslice.Contains(a.DockerArtifact.CacheFrom, cacheRef) {
		return a
	}

	cf := make([]string, 0, len(a.DockerArtifact.CacheFrom))
	ct := make([]string, 0, len(a.DockerArtifact.CacheTo))
	for _, image := range a.DockerArtifact.CacheFrom {
		if image == cacheRef {
			// change cache reference to to the tagged image name (built or given, including registry)
			log.Entry(ctx).Debugf("Adjusting cache source image ref: %s", cacheTag)
			cf = append(cf, cacheTag)
			if buildkit {
				// add cache destination reference, only if we're pushing to a registry and not given in config
				if len(a.DockerArtifact.CacheTo) == 0 && pushImages {
					log.Entry(ctx).Debugf("Adjusting cache destination image ref: %s", cacheTag)
					ct = append(ct, fmt.Sprintf("type=registry,ref=%s,mode=max", cacheTag))
				}
			}
		} else {
			cf = append(cf, image)
		}
	}
	// just copy any other cache destination given in the config file:
	for _, image := range a.DockerArtifact.CacheTo {
		ct = append(ct, image)
	}
	copy := *a
	copy.DockerArtifact.CacheFrom = cf
	copy.DockerArtifact.CacheTo = ct
	return &copy
}
