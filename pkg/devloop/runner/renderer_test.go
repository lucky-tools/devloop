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
	"testing"

	"github.com/blang/semver"

	"github.com/lucky-tools/devloop/pkg/devloop/render/renderer"
	"github.com/lucky-tools/devloop/pkg/devloop/render/renderer/helm"
	"github.com/lucky-tools/devloop/pkg/devloop/runner/runcontext"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/testutil"
)

func TestGetRenderer(tOuter *testing.T) {
	rc := &runcontext.RunContext{
		Pipelines: runcontext.NewPipelines(
			map[string]latest.Pipeline{
				"default": {},
			}, []string{"default"}),
	}
	labels := map[string]string{}
	kubectlCfg := latest.RenderConfig{
		Generate: latest.Generate{
			RawK8s: []string{"k8s/*"},
		},
	}
	helmConfig := latest.RenderConfig{
		Generate: latest.Generate{
			Helm: &latest.Helm{
				Releases: []latest.HelmRelease{
					{Name: "test", ChartPath: "./test"},
					{Name: "test1", ChartPath: "./test1"},
				},
			},
		},
	}
	kptConfig := latest.RenderConfig{
		Generate: latest.Generate{
			Kpt: []string{"kptfile"},
		},
	}
	simulatedHelmVersion := semver.MustParse("4.0.0")
	testutil.Run(tOuter, "TestGetRenderer", func(t *testutil.T) {
		tests := []struct {
			description string
			cfg         latest.Pipeline

			// Using a function here to run this in the test scope: `t.Override(...)`
			// is applied so no real commands are executed.
			expected func() renderer.Renderer

			apply             bool
			shouldErr         bool
			deepCheckRenderer bool
		}{
			{
				description: "no renderer",
				expected: func() renderer.Renderer {
					return renderer.RenderMux{}
				},
			},
			{
				description: "helm renderer",
				cfg: latest.Pipeline{
					Render: helmConfig,
				},
				expected: func() renderer.Renderer {
					return renderer.NewRenderMux(
						renderer.GroupRenderer{
							Renderers: []renderer.Renderer{
								t.RequireNonNilResult(helm.New(t.Context(), rc, helmConfig, labels, "", nil)).(renderer.Renderer),
							},
						},
					)
				},
			},
			{
				description: "kubectl renderer",
				cfg: latest.Pipeline{
					Render: kubectlCfg,
				},
				expected: func() renderer.Renderer {
					return renderer.NewRenderMux(
						renderer.GroupRenderer{
							Renderers: []renderer.Renderer{
								t.RequireNonNilResult(helm.New(t.Context(), rc, kubectlCfg, labels, "", nil)).(renderer.Renderer),
							},
						},
					)
				},
			},
			{
				description: "kpt renderer",
				cfg: latest.Pipeline{
					Render: kptConfig,
				},
				expected: func() renderer.Renderer {
					return renderer.NewRenderMux(
						renderer.GroupRenderer{
							Renderers: []renderer.Renderer{
								t.RequireNonNilResult(helm.New(t.Context(), rc, kptConfig, labels, "", nil)).(renderer.Renderer),
							},
						},
					)
				},
			},
			{
				description: "kpt renderer when validate configured",
				cfg: latest.Pipeline{
					Render: latest.RenderConfig{
						Generate: latest.Generate{RawK8s: []string{"test"}},
						Validate: &[]latest.Validator{{Name: "kubeval"}},
					},
				},
				expected: func() renderer.Renderer {
					return renderer.NewRenderMux(
						renderer.GroupRenderer{
							Renderers: []renderer.Renderer{
								t.RequireNonNilResult(helm.New(t.Context(), rc, kptConfig, labels, "", nil)).(renderer.Renderer),
							},
						},
					)
				},
			},
		}
		for _, test := range tests {
			testutil.Run(t.T, test.description, func(t *testutil.T) {
				t.Override(&helm.RendererHelmVersionOverride, &simulatedHelmVersion)

				rs, err := GetRenderer(context.Background(), &runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": test.cfg,
						},
						[]string{"default"}),
				}, "", map[string]string{})

				t.CheckError(test.shouldErr, err)
				expected := test.expected()
				t.CheckTypeEquality(expected, rs)
			})
		}
	})
}
