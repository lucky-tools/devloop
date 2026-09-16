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

package local

import (
	"context"
	"io"

	"github.com/moby/moby/client"

	"github.com/lucky-tools/devloop/pkg/devloop/build"
	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/output"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/platform"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (b *Builder) Build(ctx context.Context, out io.Writer, a *latest.Artifact) build.ArtifactBuilder {
	if b.prune {
		b.localPruner.asynchronousCleanupOldImages(ctx, []string{a.ImageName})
	}
	builder := build.WithLogFile(b.buildArtifact, b.muted)
	return builder
}

func (b *Builder) PreBuild(_ context.Context, out io.Writer) error {
	if b.localCluster {
		output.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.kubeContext)
	}
	return nil
}

func (b *Builder) PostBuild(ctx context.Context, _ io.Writer) error {
	defer b.localDocker.Close()
	if b.dockerDaemon != b.localDocker {
		defer b.dockerDaemon.Close()
	}
	if b.prune {
		if b.mode == config.RunModes.Build {
			b.localPruner.synchronousCleanupOldImages(ctx, b.builtImages)
		} else {
			b.localPruner.asynchronousCleanupOldImages(ctx, b.builtImages)
		}
	}
	return nil
}

// Prune uses the docker API client to remove all images built with Devloop
func (b *Builder) Prune(ctx context.Context, _ io.Writer) error {
	var toPrune []string
	seen := make(map[string]bool)

	for _, img := range b.builtImages {
		if !seen[img] && !b.localPruner.isPruned(img) {
			toPrune = append(toPrune, img)
			seen[img] = true
		}
	}
	_, err := b.localDocker.Prune(ctx, toPrune, b.pruneChildren)
	return err
}

func (b *Builder) Concurrency() *int { return b.local.Concurrency }

func (b *Builder) PushImages() bool {
	return b.pushImages
}

func (b *Builder) SupportedPlatforms() platform.Matcher { return platform.All }

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, platforms platform.Matcher) (string, error) {
	digestOrImageID, err := b.runBuildForArtifact(ctx, out, a, tag, platforms)
	if err != nil {
		return "", err
	}

	if b.pushImages {
		// delete the local image after it has been pushed to the registry
		if b.deleteLocalImage {
			b.deleteLocalImageAfterPush(ctx, out, tag, b.daemonFor(a))
		}

		// only track images for pruning when building with docker
		// if the local image was deleted after push, there's nothing left to track for pruning
		if a.DockerArtifact != nil && !b.deleteLocalImage {
			imageID, err := b.getImageIDForTag(ctx, tag, b.daemonFor(a))
			if err != nil && !b.buildkit {
				log.Entry(ctx).Warn("unable to inspect image: built images may not be cleaned up correctly by devloop")
			}
			if imageID != "" {
				b.builtImages = append(b.builtImages, imageID)
			}
		}

		digest := digestOrImageID
		return build.TagWithDigest(tag, digest), nil
	}

	imageID := digestOrImageID
	if b.mode == config.RunModes.Dev {
		artifacts, err := b.artifactStore.GetArtifacts([]*latest.Artifact{a})
		if err != nil {
			log.Entry(ctx).Debugf("failed to get artifacts from store, err: %v", err)
		}
		// delete previous built images asynchronously
		go func() {
			if len(artifacts) > 0 {
				bgCtx := context.Background()
				id, err := b.getImageIDForTag(bgCtx, artifacts[0].Tag, b.daemonFor(a))
				if err != nil {
					log.Entry(bgCtx).Debugf("failed to get image ID for tag %s, err: %v", artifacts[0].Tag, err)
					return
				}
				// only prune if the image changed
				if id != imageID {
					b.localPruner.runPrune(bgCtx, []string{id})
				}
			}
		}()
	}
	b.builtImages = append(b.builtImages, imageID)
	return build.TagWithImageID(ctx, tag, imageID, b.daemonFor(a))
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, platforms platform.Matcher) (string, error) {
	if !b.buildkit && !b.pushImages {
		// Most builders will rely on a local Docker (except when using a remote buildkit daemon):
		// + Either to build the image,
		// + Or to docker load it.
		// Let's fail fast if Docker is not available
		if _, err := b.localDocker.ServerVersion(ctx); err != nil {
			return "", err
		}
	}

	builder, err := newPerArtifactBuilder(b, a)
	if err != nil {
		return "", err
	}
	if platforms.IsNotEmpty() {
		supported := builder.SupportedPlatforms()
		if p := platforms.Intersect(supported); p.IsNotEmpty() {
			platforms = p
		} else {
			log.Entry(ctx).Warnf("builder for artifact %q doesn't support building for target platforms: %q. Building for supported platforms %q instead.", a.ImageName, platforms, supported)
			platforms = supported
		}
	}
	return builder.Build(ctx, out, a, tag, platforms)
}

func (b *Builder) getImageIDForTag(ctx context.Context, tag string, daemon docker.LocalDaemon) (string, error) {
	insp, _, err := daemon.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		return "", err
	}
	return insp.ID, nil
}

// daemonFor returns the docker daemon that the given artifact is built against.
// Docker artifacts may target a different daemon (build.local.api.host); other
// builders always use the default daemon.
func (b *Builder) daemonFor(a *latest.Artifact) docker.LocalDaemon {
	if a.DockerArtifact != nil {
		return b.dockerDaemon
	}
	return b.localDocker
}

// deleteLocalImageAfterPush removes the local image for the given tag once it
// has been pushed to a registry. The image may not exist locally (e.g. when
// building directly to a registry via buildkit), in which case the error
// is ignored.
func (b *Builder) deleteLocalImageAfterPush(ctx context.Context, out io.Writer, tag string, daemon docker.LocalDaemon) {
	if _, err := daemon.ImageRemove(ctx, tag, client.ImageRemoveOptions{}); err != nil {
		log.Entry(ctx).Debugf("unable to delete local image %q after push: %v", tag, err)
		return
	}
	output.Default.Fprintf(out, "imageRemove: %s\n", tag)
}

func (b *Builder) retrieveExtraEnv() []string {
	return b.localDocker.ExtraEnv()
}
