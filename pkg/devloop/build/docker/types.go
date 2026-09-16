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

	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

// APIBuilder builds Docker artifacts against the Docker Engine HTTP API.
type APIBuilder struct {
	daemon             docker.LocalDaemon
	cfg                docker.Config
	pushImages         bool
	artifacts          ArtifactResolver
	sourceDependencies TransitiveSourceDependenciesResolver
}

// NewAPIBuilder returns a new instance of an API-backed docker artifact builder.
func NewAPIBuilder(daemon docker.LocalDaemon, cfg docker.Config, pushImages bool, ar ArtifactResolver, dr TransitiveSourceDependenciesResolver) *APIBuilder {
	return &APIBuilder{
		daemon:             daemon,
		cfg:                cfg,
		pushImages:         pushImages,
		artifacts:          ar,
		sourceDependencies: dr,
	}
}

// CLIBuilder builds Docker artifacts via the docker command-line interface.
type CLIBuilder struct {
	daemon             docker.LocalDaemon
	cfg                docker.Config
	pushImages         bool
	buildkit           bool
	buildKitBuilder    string
	cacheTag           string
	artifacts          ArtifactResolver
	sourceDependencies TransitiveSourceDependenciesResolver
}

// NewCLIBuilder returns a new instance of a CLI-backed docker artifact builder.
func NewCLIBuilder(daemon docker.LocalDaemon, cfg docker.Config, pushImages bool, buildkit bool, buildKitBuilder string, cacheTag string, ar ArtifactResolver, dr TransitiveSourceDependenciesResolver) *CLIBuilder {
	return &CLIBuilder{
		daemon:             daemon,
		cfg:                cfg,
		pushImages:         pushImages,
		buildkit:           buildkit,
		buildKitBuilder:    buildKitBuilder,
		cacheTag:           cacheTag,
		artifacts:          ar,
		sourceDependencies: dr,
	}
}

// ArtifactResolver provides an interface to resolve built artifact tags by image name.
type ArtifactResolver interface {
	GetImageTag(imageName string) (string, bool)
}

// TransitiveSourceDependenciesResolver provides an interface to to evaluate the source dependencies for artifacts.
type TransitiveSourceDependenciesResolver interface {
	TransitiveArtifactDependencies(ctx context.Context, a *latest.Artifact) ([]string, error)
}
