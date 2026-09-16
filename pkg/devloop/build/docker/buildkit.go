/*
Copyright 2026 The Skaffold Authors

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
	"path/filepath"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/session"
	"github.com/tonistiigi/fsutil"

	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/platform"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// BuildKitBuilder builds Docker artifacts against a buildkitd daemon via the
// moby/buildkit Go client. It pushes the built image directly to the registry
// (there is no load into a local Docker daemon), so it is only useful in push
// mode and returns the image digest.
type BuildKitBuilder struct {
	cfg                docker.Config
	buildKitHost       string
	artifacts          ArtifactResolver
	sourceDependencies TransitiveSourceDependenciesResolver
}

// NewBuildKitBuilder returns a new instance of a buildkit-backed docker artifact builder.
func NewBuildKitBuilder(cfg docker.Config, buildKitHost string, ar ArtifactResolver, dr TransitiveSourceDependenciesResolver) *BuildKitBuilder {
	return &BuildKitBuilder{
		cfg:                cfg,
		buildKitHost:       buildKitHost,
		artifacts:          ar,
		sourceDependencies: dr,
	}
}

// SupportedPlatforms reports that the buildkit backend builds for any platform.
func (b *BuildKitBuilder) SupportedPlatforms() platform.Matcher {
	return platform.All
}

// Build builds a Docker artifact with buildkit and pushes it to the registry.
func (b *BuildKitBuilder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {
	var pls []v1.Platform
	for _, plat := range matcher.Platforms {
		pls = append(pls, util.ConvertToV1Platform(plat))
	}

	a = adjustCache(ctx, a, tag, true, "", true)

	dockerfile, err := validateDockerfile(a)
	if err != nil {
		return "", err
	}

	solveOpt, err := b.buildSolveOpt(a, tag, dockerfile, pls)
	if err != nil {
		return "", err
	}

	cli, err := client.New(ctx, b.buildKitHost)
	if err != nil {
		return "", newBuildError(fmt.Errorf("creating buildkit client: %w", err), b.cfg)
	}
	defer cli.Close()

	statusChan := make(chan *client.SolveStatus)
	statusDone := make(chan struct{})
	go func() {
		defer close(statusDone)
		for status := range statusChan {
			writeSolveStatus(out, status)
		}
	}()

	resp, err := cli.Solve(ctx, nil, *solveOpt, statusChan)
	<-statusDone
	if err != nil {
		return "", newBuildError(fmt.Errorf("buildkit solve: %w", err), b.cfg)
	}

	digest := resp.ExporterResponse[exptypes.ExporterImageDigestKey]
	if digest == "" {
		return "", newBuildError(fmt.Errorf("buildkit did not return an image digest for %q", tag), b.cfg)
	}
	return digest, nil
}

// buildSolveOpt assembles the buildkit solve request for a dockerfile build.
func (b *BuildKitBuilder) buildSolveOpt(a *latest.Artifact, tag string, dockerfile string, pls []v1.Platform) (*client.SolveOpt, error) {
	workspace := a.Workspace
	if !filepath.IsAbs(workspace) {
		if abs, err := filepath.Abs(workspace); err == nil {
			workspace = abs
		}
	}

	imageInfoEnv, err := docker.EnvTags(tag)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse image tag: %w", err)
	}
	buildArgs, err := docker.EvalBuildArgsWithEnv(b.cfg.Mode(), workspace, a.DockerArtifact.DockerfilePath, a.DockerArtifact.BuildArgs, docker.ResolveDependencyImages(a.Dependencies, b.artifacts, true), imageInfoEnv)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate build args: %w", err)
	}

	frontendAttrs := map[string]string{
		"filename": filepath.Base(dockerfile),
	}
	if a.DockerArtifact.Target != "" {
		frontendAttrs["target"] = a.DockerArtifact.Target
	}
	if a.DockerArtifact.NoCache {
		frontendAttrs["no-cache"] = ""
	}
	for k, v := range buildArgs {
		if v != nil {
			frontendAttrs["build-arg:"+k] = *v
		} else {
			frontendAttrs["build-arg:"+k] = ""
		}
	}
	var platforms []string
	for _, pl := range pls {
		if pl.String() != "" {
			platforms = append(platforms, pl.String())
		}
	}
	if len(platforms) > 0 {
		frontendAttrs["platform"] = strings.Join(platforms, ",")
	}

	contextMount, err := fsutil.NewFS(workspace)
	if err != nil {
		return nil, fmt.Errorf("invalid build context %q: %w", workspace, err)
	}
	dockerfileMount, err := fsutil.NewFS(filepath.Dir(dockerfile))
	if err != nil {
		return nil, fmt.Errorf("invalid dockerfile dir %q: %w", filepath.Dir(dockerfile), err)
	}

	authProvider, err := docker.BuildKitAuthProvider()
	if err != nil {
		return nil, err
	}

	return &client.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,
		LocalMounts: map[string]fsutil.FS{
			"context":    contextMount,
			"dockerfile": dockerfileMount,
		},
		Exports: []client.ExportEntry{
			{
				Type:  "image",
				Attrs: map[string]string{"name": tag, "push": "true"},
			},
		},
		CacheImports: parseCacheImports(a.DockerArtifact.CacheFrom),
		CacheExports: parseCacheExports(a.DockerArtifact.CacheTo),
		Session:      []session.Attachable{authProvider},
	}, nil
}

// writeSolveStatus forwards buildkit build logs to out.
func writeSolveStatus(out io.Writer, status *client.SolveStatus) {
	for _, l := range status.Logs {
		if _, err := out.Write(l.Data); err != nil {
			return
		}
	}
}

// parseCacheImports converts cacheFrom image references into buildkit registry
// cache imports.
func parseCacheImports(cacheFrom []string) []client.CacheOptionsEntry {
	var imports []client.CacheOptionsEntry
	for _, c := range cacheFrom {
		imports = append(imports, parseCacheEntry(c, "registry"))
	}
	return imports
}

// parseCacheExports converts cacheTo references into buildkit cache exports.
func parseCacheExports(cacheTo []string) []client.CacheOptionsEntry {
	var exports []client.CacheOptionsEntry
	for _, c := range cacheTo {
		exports = append(exports, parseCacheEntry(c, ""))
	}
	return exports
}

// parseCacheEntry parses a cache reference in the buildkit `type=...,k=v,...`
// syntax into a buildkit CacheOptionsEntry. A plain image reference (no `type=`)
// is treated as a registry cache of the given default type.
func parseCacheEntry(s, defaultType string) client.CacheOptionsEntry {
	entry := client.CacheOptionsEntry{Type: defaultType, Attrs: map[string]string{}}
	if !strings.Contains(s, "type=") {
		if defaultType == "registry" {
			entry.Attrs["ref"] = s
		}
		return entry
	}
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "type" {
			entry.Type = kv[1]
		} else {
			entry.Attrs[kv[0]] = kv[1]
		}
	}
	return entry
}
