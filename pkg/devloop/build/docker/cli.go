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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/instrumentation"
	"github.com/lucky-tools/devloop/pkg/devloop/output"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/platform"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// Build builds a Docker artifact via the docker command-line interface.
func (b *CLIBuilder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {
	var pls []v1.Platform
	if len(matcher.Platforms) > 0 {
		for _, plat := range matcher.Platforms {
			pls = append(pls, util.ConvertToV1Platform(plat))
		}
	} else {
		pls = append(pls, v1.Platform{})
	}
	a = adjustCache(ctx, a, tag, b.buildkit, b.cacheTag, b.pushImages)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "docker",
		"Context":     instrumentation.PII(a.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	dockerfile, err := validateDockerfile(a)
	if err != nil {
		return "", err
	}

	for _, pl := range pls {
		if err := pullCacheFromImages(ctx, out, a.ArtifactType.DockerArtifact, pl, b.daemon, b.buildkit); err != nil {
			return "", cacheFromPullErr(err, a.ImageName)
		}
	}
	opts := docker.BuildOptions{Tag: tag, Mode: b.cfg.Mode(), ExtraBuildArgs: docker.ResolveDependencyImages(a.Dependencies, b.artifacts, true)}

	imageID, err := b.dockerCLIBuild(ctx, output.GetUnderlyingWriter(out), a.ImageName, a.Workspace, dockerfile, a.ArtifactType.DockerArtifact, opts, pls)
	if err != nil {
		return "", newBuildError(err, b.cfg)
	}

	if b.pushImages && !b.buildkit {
		return b.daemon.Push(ctx, out, opts.Tag)
	}

	return imageID, nil
}

// SupportedPlatforms reports that the CLI backend can build for any platform (via buildkit).
func (b *CLIBuilder) SupportedPlatforms() platform.Matcher {
	return platform.All
}

func (b *CLIBuilder) dockerCLIBuild(ctx context.Context, out io.Writer, name string, workspace string, dockerfilePath string, a *latest.DockerArtifact, opts docker.BuildOptions, pls []v1.Platform) (string, error) {
	args := []string{}
	buildCmd := "build"
	if b.buildkit {
		buildCmd = "buildx"
	}
	args = append(args, buildCmd, workspace, "--file", dockerfilePath, "-t", opts.Tag)
	imageInfoEnv, err := docker.EnvTags(opts.Tag)
	if err != nil {
		return "", fmt.Errorf("couldn't parse image tag: %w", err)
	}
	ba, err := docker.EvalBuildArgsWithEnv(b.cfg.Mode(), workspace, a.DockerfilePath, a.BuildArgs, opts.ExtraBuildArgs, imageInfoEnv)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate build args: %w", err)
	}
	cliArgs, err := docker.ToCLIBuildArgs(a, ba, imageInfoEnv)
	if err != nil {
		return "", fmt.Errorf("getting docker build args: %w", err)
	}
	args = append(args, cliArgs...)

	if b.cfg.Prune() {
		args = append(args, "--force-rm")
	}

	var platforms []string
	for _, pl := range pls {
		if pl.String() != "" {
			platforms = append(platforms, pl.String())
		}
	}
	if len(platforms) > 0 {
		args = append(args, "--platform", strings.Join(platforms, ","))
	}

	if b.buildkit {
		if b.pushImages {
			// with buildkit, push the image directly to the registry (not using the docker daemon)
			args = append(args, "--push")
		} else {
			// Load the image into the local daemon so the dev loop and `docker push` (CLI push) can find it.
			// Under buildkit without a daemon, skip loading (the buildkit builder pushes/warns instead).
			_, err := b.daemon.ServerVersion(ctx)
			if err == nil {
				args = append(args, "--load")
			}
		}
	}

	if b.buildkit && b.buildKitBuilder != "" {
		args = append(args, "--builder", b.buildKitBuilder)
	}

	// temporary file for buildkit metadata containing the image digest:
	var metadata string
	if b.buildkit {
		metadata, err = getBuildKitMetadataFile()
		if err != nil {
			return "", fmt.Errorf("unable to create temp file: %w", err)
		}
		defer os.Remove(metadata)
		args = append(args, "--metadata-file", metadata)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(util.OSEnviron(), b.daemon.ExtraEnv()...)
	if len(platforms) > 1 && b.buildkit {
		// avoid "unknown/unknown" architecture/OS caused by buildkit default image attestation
		log.Entry(ctx).Warnf("setting BUILDX_NO_DEFAULT_ATTESTATIONS=1 for docker buildx for artifact %q since it targets platform %q to avoid unknown/unknown platform issue", name, platforms[0])
		cmd.Env = append(cmd.Env, "BUILDX_NO_DEFAULT_ATTESTATIONS=1")
	}
	cmd.Stdout = out

	var errBuffer bytes.Buffer
	stderr := io.MultiWriter(out, &errBuffer)
	cmd.Stderr = stderr

	if err := util.RunCmd(ctx, cmd); err != nil {
		if !b.buildkit {
			err = tryExecFormatErr(fmt.Errorf("running build: %w", err), errBuffer)
		} else {
			err = tryExecFormatErrBuildKit(fmt.Errorf("running build: %w", err), errBuffer)
		}
		return "", err
	}

	if !b.buildkit {
		return b.daemon.ImageID(ctx, opts.Tag)
	} else {
		return parseBuildKitMetadataFile(ctx, metadata)
	}
}

// osCreateTemp allows for replacing metadata for testing purposes
var osCreateTemp = os.CreateTemp

func getBuildKitMetadataFile() (string, error) {
	metadata, err := osCreateTemp("", "metadata*.json")
	if err != nil {
		return "", err
	}
	metadata.Close()
	return metadata.Name(), nil
}

func parseBuildKitMetadataFile(ctx context.Context, filename string) (string, error) {
	var metadata map[string]interface{}
	data, err := os.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(data, &metadata)
	}
	if err == nil {
		// avoid panic: interface conversion: interface {} is nil, not string (if keys don't exists)
		var digest string
		if value := metadata["containerimage.config.digest"]; value != nil {
			// image loaded to local docker daemon
			digest = value.(string)
		} else if value := metadata["containerimage.digest"]; value != nil {
			// image pushed to registry
			digest = value.(string)
		}
		var name string
		if value := metadata["image.name"]; value != nil {
			name = value.(string)
		}
		if digest != "" {
			log.Entry(ctx).Debugf("Image digest found in buildkit metadata: %s for %s", digest, name)
			return digest, nil
		}
	}
	log.Entry(ctx).Warnf("No digest found in buildkit metadata: %v", err)
	// if image is not pushed, it could not contain the digest log for debugging:
	log.Entry(ctx).Debugf("Full buildkit metadata: %s", data)
	return "", err
}
