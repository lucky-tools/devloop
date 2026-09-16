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
	"io"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/instrumentation"
	"github.com/lucky-tools/devloop/pkg/devloop/platform"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// Build builds a Docker artifact against the Docker Engine HTTP API.
func (b *APIBuilder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {
	var pls []v1.Platform
	if len(matcher.Platforms) > 0 {
		for _, plat := range matcher.Platforms {
			pls = append(pls, util.ConvertToV1Platform(plat))
		}
	} else {
		pls = append(pls, v1.Platform{})
	}
	a = adjustCache(ctx, a, tag, false, "", b.pushImages)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "docker",
		"Context":     instrumentation.PII(a.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	if _, err := validateDockerfile(a); err != nil {
		return "", err
	}

	for _, pl := range pls {
		if err := pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact, pl, b.daemon, false); err != nil {
			return "", cacheFromPullErr(err, a.ImageName)
		}
	}
	opts := docker.BuildOptions{Tag: tag, Mode: b.cfg.Mode(), ExtraBuildArgs: docker.ResolveDependencyImages(a.Dependencies, b.artifacts, true)}

	imageID, err := b.daemon.Build(ctx, out, a.Workspace, a.ImageName, a.ArtifactType.DockerArtifact, opts)
	if err != nil {
		return "", newBuildError(err, b.cfg)
	}

	if b.pushImages {
		return b.daemon.Push(ctx, out, tag)
	}

	return imageID, nil
}

// SupportedPlatforms reports that the API backend only builds for the host platform.
func (b *APIBuilder) SupportedPlatforms() platform.Matcher {
	return platform.Host
}
