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
	"fmt"
	"io"

	"github.com/lucky-tools/devloop/pkg/devloop/build"
	"github.com/lucky-tools/devloop/pkg/devloop/build/buildpacks"
	"github.com/lucky-tools/devloop/pkg/devloop/build/custom"
	dockerbuilder "github.com/lucky-tools/devloop/pkg/devloop/build/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/build/jib"
	"github.com/lucky-tools/devloop/pkg/devloop/build/ko"
	"github.com/lucky-tools/devloop/pkg/devloop/build/misc"
	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/platform"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// Builder uses the host docker daemon to build and tag the image.
type Builder struct {
	local latest.LocalBuild

	cfg                docker.Config
	localDocker        docker.LocalDaemon
	dockerDaemon       docker.LocalDaemon
	localCluster       bool
	pushImages         bool
	deleteLocalImage   bool
	tryImportMissing   bool
	prune              bool
	pruneChildren      bool
	skipTests          bool
	mode               config.RunMode
	kubeContext        string
	buildkit           bool
	builtImages        []string
	insecureRegistries map[string]bool
	muted              build.Muted
	localPruner        *pruner
	artifactStore      build.ArtifactStore
	sourceDependencies graph.SourceDependenciesCache
}

type Config interface {
	docker.Config

	GlobalConfig() string
	GetKubeContext() string
	GetCluster() config.Cluster
	SkipTests() bool
	Mode() config.RunMode
	NoPruneChildren() bool
	Muted() config.Muted
	PushImages() config.BoolOrUndefined
}

type BuilderContext interface {
	Config
	ArtifactStore() build.ArtifactStore
	SourceDependenciesResolver() graph.SourceDependenciesCache
}

// NewBuilder returns an new instance of a local Builder.
func NewBuilder(ctx context.Context, bCtx BuilderContext, buildCfg *latest.LocalBuild) (*Builder, error) {
	localDocker, err := docker.NewAPIClient(ctx, bCtx)
	if err != nil {
		return nil, fmt.Errorf("getting docker client: %w", err)
	}

	// The daemon used to build docker artifacts. For the API backend it is
	// determined by `build.local.api.host` (or the local socket), never by
	// DOCKER_HOST / docker context / minikube detection. For the CLI backend it
	// is the default daemon, used only for metadata (image ID, extra env).
	var dockerDaemon docker.LocalDaemon
	if buildCfg.CLI == nil {
		host := ""
		if buildCfg.API != nil {
			host = buildCfg.API.Host
		}
		dockerDaemon, err = docker.NewLocalDaemonFromHost(host, bCtx)
		if err != nil {
			return nil, fmt.Errorf("getting docker client for host %q: %w", host, err)
		}
	} else {
		dockerDaemon = localDocker
	}

	cluster := bCtx.GetCluster()
	pushFlag := bCtx.PushImages()

	var pushImages bool
	switch {
	case pushFlag.Value() != nil:
		pushImages = *pushFlag.Value()
		log.Entry(context.TODO()).Debugf("push value set via devloop build --push flag, --push=%t", *pushFlag.Value())
	case buildCfg.Push == nil:
		pushImages = cluster.PushImages
		log.Entry(context.TODO()).Debugf("push value not present in NewBuilder, defaulting to %t because cluster.PushImages is %t", pushImages, cluster.PushImages)
	default:
		pushImages = *buildCfg.Push
	}

	tryImportMissing := buildCfg.TryImportMissing

	// delete the local image after it has been pushed to a registry
	deleteLocalImage := buildCfg.Delete != nil && *buildCfg.Delete

	// buildkit is enabled directly from the devloop.yaml via the `cli` backend's `useBuildKit: true`.
	buildkit := buildCfg.CLI != nil && buildCfg.CLI.UseBuildKit

	return &Builder{
		local:              *buildCfg,
		cfg:                bCtx,
		kubeContext:        bCtx.GetKubeContext(),
		localDocker:        localDocker,
		dockerDaemon:       dockerDaemon,
		localCluster:       cluster.Local,
		pushImages:         pushImages,
		deleteLocalImage:   deleteLocalImage,
		tryImportMissing:   tryImportMissing,
		buildkit:           buildkit,
		skipTests:          bCtx.SkipTests(),
		mode:               bCtx.Mode(),
		prune:              bCtx.Prune(),
		pruneChildren:      !bCtx.NoPruneChildren(),
		localPruner:        newPruner(localDocker, !bCtx.NoPruneChildren()),
		insecureRegistries: bCtx.GetInsecureRegistries(),
		muted:              bCtx.Muted(),
		artifactStore:      bCtx.ArtifactStore(),
		sourceDependencies: bCtx.SourceDependenciesResolver(),
	}, nil
}

// artifactBuilder represents a per artifact builder interface
type artifactBuilder interface {
	Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, platforms platform.Matcher) (string, error)
	SupportedPlatforms() platform.Matcher
}

// newPerArtifactBuilder returns an instance of `artifactBuilder`
func newPerArtifactBuilder(b *Builder, a *latest.Artifact) (artifactBuilder, error) {
	switch {
	case a.DockerArtifact != nil:
		if b.local.API != nil && b.local.API.UseBuildKit {
			if !b.pushImages {
				return nil, fmt.Errorf("`api.useBuildKit` builds and pushes directly to a registry via buildkit; it requires push to be enabled (set `build.local.push: true` or use `--push`)")
			}
			return dockerbuilder.NewBuildKitBuilder(b.cfg, b.local.API.BuildKitHost, b.artifactStore, b.sourceDependencies), nil
		}
		if b.local.CLI != nil {
			return dockerbuilder.NewCLIBuilder(b.localDocker, b.cfg, b.pushImages, b.buildkit, b.local.CLI.BuildKitBuilder, b.local.CLI.CacheTag, b.artifactStore, b.sourceDependencies), nil
		}
		return dockerbuilder.NewAPIBuilder(b.dockerDaemon, b.cfg, b.pushImages, b.artifactStore, b.sourceDependencies), nil

	case a.JibArtifact != nil:
		return jib.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages, b.skipTests, b.artifactStore), nil

	case a.CustomArtifact != nil:
		// required artifacts as environment variables
		dependencies := util.EnvPtrMapToSlice(docker.ResolveDependencyImages(a.Dependencies, b.artifactStore, true), "=")
		return custom.NewArtifactBuilder(b.localDocker, b.cfg, b.pushImages, b.skipTests, append(b.retrieveExtraEnv(), dependencies...)), nil

	case a.BuildpackArtifact != nil:
		return buildpacks.NewArtifactBuilder(b.localDocker, b.pushImages, b.mode, b.artifactStore), nil

	case a.KoArtifact != nil:
		return ko.NewArtifactBuilder(b.localDocker, b.pushImages, b.mode, b.insecureRegistries), nil

	default:
		return nil, fmt.Errorf("unexpected type %q for local artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}
