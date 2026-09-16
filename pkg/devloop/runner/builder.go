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

package runner

import (
	"context"
	"fmt"
	"io"

	"github.com/lucky-tools/devloop/pkg/devloop/build"
	"github.com/lucky-tools/devloop/pkg/devloop/build/cluster"
	"github.com/lucky-tools/devloop/pkg/devloop/build/local"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/hooks"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/runner/runcontext"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

// builderCtx encapsulates a given devloop run context along with additional builder constructs.
type builderCtx struct {
	*runcontext.RunContext
	artifactStore           build.ArtifactStore
	sourceDependenciesCache graph.SourceDependenciesCache
}

func (b *builderCtx) ArtifactStore() build.ArtifactStore {
	return b.artifactStore
}

func (b *builderCtx) SourceDependenciesResolver() graph.SourceDependenciesCache {
	return b.sourceDependenciesCache
}

// GetBuilder creates a builder from a given RunContext and build pipeline type.
func GetBuilder(ctx context.Context, r *runcontext.RunContext, s build.ArtifactStore, d graph.SourceDependenciesCache, p latest.Pipeline) (build.PipelineBuilder, error) {
	bCtx := &builderCtx{artifactStore: s, sourceDependenciesCache: d, RunContext: r}
	switch {
	case p.Build.LocalBuild != nil:
		log.Entry(context.TODO()).Debug("Using builder: local")
		builder, err := local.NewBuilder(ctx, bCtx, p.Build.LocalBuild)
		if err != nil {
			return nil, err
		}
		return builder, nil

	case p.Build.Cluster != nil:
		log.Entry(context.TODO()).Debug("Using builder: cluster")
		builder, err := cluster.NewBuilder(bCtx, p.Build.Cluster)
		if err != nil {
			return nil, err
		}
		return builder, err

	default:
		return nil, fmt.Errorf("unknown builder for config %+v", p.Build)
	}
}

type pipelineBuilderWithHooks struct {
	build.PipelineBuilder
	hooksRunner hooks.Runner
}

// PreBuild executes any one-time setup required prior to starting any build on this builder,
// followed by any Build pre-hooks set for the pipeline.
func (b *pipelineBuilderWithHooks) PreBuild(ctx context.Context, out io.Writer) error {
	if err := b.PipelineBuilder.PreBuild(ctx, out); err != nil {
		return err
	}

	if err := b.hooksRunner.RunPreHooks(ctx, out); err != nil {
		return fmt.Errorf("running builder pre-hooks: %w", err)
	}

	return nil
}

// PostBuild executes any one-time teardown required after all builds on this builder are complete,
// followed by any Build post-hooks set for the pipeline.
func (b *pipelineBuilderWithHooks) PostBuild(ctx context.Context, out io.Writer) error {
	if err := b.PipelineBuilder.PostBuild(ctx, out); err != nil {
		return err
	}

	if err := b.hooksRunner.RunPostHooks(ctx, out); err != nil {
		return fmt.Errorf("running builder post-hooks: %w", err)
	}

	return nil
}

func withPipelineBuildHooks(pb build.PipelineBuilder, buildHooks latest.BuildHooks) build.PipelineBuilder {
	return &pipelineBuilderWithHooks{
		PipelineBuilder: pb,
		hooksRunner:     hooks.BuildRunner(buildHooks, hooks.BuildEnvOpts{}),
	}
}
