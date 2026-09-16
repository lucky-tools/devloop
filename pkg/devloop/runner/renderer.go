/*
Copyright 2022 The Skaffold Authors

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

	"github.com/lucky-tools/devloop/pkg/devloop/hooks"
	"github.com/lucky-tools/devloop/pkg/devloop/render/renderer"
	"github.com/lucky-tools/devloop/pkg/devloop/runner/runcontext"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// GetRenderer creates a renderer from a given RunContext and pipeline definitions.
func GetRenderer(ctx context.Context, runCtx *runcontext.RunContext, hydrationDir string, labels map[string]string) (renderer.Renderer, error) {
	configNames := runCtx.Pipelines.AllOrderedConfigNames()

	var gr renderer.GroupRenderer
	var err error
	for _, configName := range configNames {
		p := runCtx.Pipelines.GetForConfigName(configName)
		mkvMap := map[string]string{}
		if runCtx.Opts.ManifestsValueFile != "" {
			mkvMap, err = util.ParseEnvVariablesFromFile(runCtx.Opts.ManifestsValueFile)
			if err != nil {
				return nil, err
			}
		}
		overridesMap := util.EnvSliceToMap(runCtx.Opts.ManifestsOverrides, "=")
		for k := range overridesMap {
			mkvMap[k] = overridesMap[k]
		}

		rs, err := renderer.New(ctx, runCtx, p.Render, hydrationDir, labels, configName, mkvMap)
		if err != nil {
			return nil, err
		}
		gr.Renderers = append(gr.Renderers, rs.Renderers...)
		gr.HookRunners = append(gr.HookRunners, hooks.NewRenderRunner(p.Render.LifecycleHooks, &[]string{runCtx.GetNamespace()},
			hooks.NewRenderEnvOpts(runCtx.KubeContext, []string{runCtx.GetNamespace()}), configName))
	}
	return renderer.NewRenderMux(gr), nil
}
