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
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/lucky-tools/devloop/pkg/devloop/access"
	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/debug"
	"github.com/lucky-tools/devloop/pkg/devloop/deploy"
	"github.com/lucky-tools/devloop/pkg/devloop/deploy/cloudrun"
	component "github.com/lucky-tools/devloop/pkg/devloop/deploy/component/kubernetes"
	kptV2 "github.com/lucky-tools/devloop/pkg/devloop/deploy/kpt"
	"github.com/lucky-tools/devloop/pkg/devloop/deploy/kubectl"
	"github.com/lucky-tools/devloop/pkg/devloop/deploy/label"
	pkgkubectl "github.com/lucky-tools/devloop/pkg/devloop/kubectl"
	"github.com/lucky-tools/devloop/pkg/devloop/kubernetes"
	k8sloader "github.com/lucky-tools/devloop/pkg/devloop/kubernetes/loader"
	k8slogger "github.com/lucky-tools/devloop/pkg/devloop/kubernetes/logger"
	"github.com/lucky-tools/devloop/pkg/devloop/kubernetes/manifest"
	"github.com/lucky-tools/devloop/pkg/devloop/kubernetes/portforward"
	k8sstatus "github.com/lucky-tools/devloop/pkg/devloop/kubernetes/status"
	"github.com/lucky-tools/devloop/pkg/devloop/loader"
	"github.com/lucky-tools/devloop/pkg/devloop/runner/runcontext"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/sync"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/testutil"
)

func TestGetDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		tests := []struct {
			description       string
			cfg               latest.Pipeline
			expected          deploy.Deployer
			apply             bool
			shouldErr         bool
			deepCheckDeployer bool
		}{
			{
				description: "no deployer",
				expected:    deploy.DeployerMux{},
			},
			{
				description: "kubectl deployer",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{KubectlDeploy: &latest.KubectlDeploy{}},
					},
				},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
						Pipelines: runcontext.NewPipelines(
							map[string]latest.Pipeline{
								"default": {},
							},
							[]string{"default"}),
					}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
						Flags: latest.KubectlFlags{},
					}, nil, "default", nil)).(deploy.Deployer),
				}, false),
			},
			{
				description: "kpt deployer",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{KptDeploy: &latest.KptDeploy{}},
					},
				},
				expected: deploy.NewDeployerMux([]deploy.Deployer{
					&kptV2.Deployer{},
				}, false),
			},
			{
				description: "cloud run deployer",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							10*time.Minute,
							false,
							util.Ptr(true)),
						).(deploy.Deployer),
					},
					false),
			},
			{
				description: "cloud run deployer with StatusCheckDeadlineSeconds specified",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						StatusCheckDeadlineSeconds: 300,
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							5*time.Minute,
							false,
							util.Ptr(true)),
						).(deploy.Deployer),
					},
					false),
			},
			{
				description: "cloud run deployer with tolerateFailures set to true",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						StatusCheckDeadlineSeconds:    900,
						TolerateFailuresUntilDeadline: true,
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							15*time.Minute,
							true,
							util.Ptr(true)),
						).(deploy.Deployer),
					},
					false),
			},
			{
				description: "cloud run deployer with statusCheck disabled",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						StatusCheck: util.Ptr(false),
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
						},
					},
				},
				expected: deploy.NewDeployerMux(
					[]deploy.Deployer{
						t.RequireNonNilResult(cloudrun.NewDeployer(
							&runcontext.RunContext{},
							&label.DefaultLabeller{},
							&latest.CloudRunDeploy{},
							"default",
							10*time.Minute,
							false,
							util.Ptr(false)),
						).(deploy.Deployer),
					},
					false),
			},
			{
				description: "apply forces creation of kubectl deployer with kpt config",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{KptDeploy: &latest.KptDeploy{}},
					},
				},
				apply: true,
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				}, nil, "default", nil)).(deploy.Deployer),
			},
			{
				description: "apply works with Cloud Run",
				apply:       true,
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "TestProject", Region: "us-central1"},
						},
					},
				},
				expected: t.RequireNonNilResult(cloudrun.NewDeployer(
					&runcontext.RunContext{},
					&label.DefaultLabeller{},
					&latest.CloudRunDeploy{
						ProjectID: "TestProject",
						Region:    "us-central1",
					},
					"default",
					10*time.Minute,
					false,
					util.Ptr(true)),
				).(deploy.Deployer),
				deepCheckDeployer: true,
			},
			{
				description: "apply does not allow multiple deployers when Cloud Run is used",
				apply:       true,
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							CloudRunDeploy: &latest.CloudRunDeploy{},
							KubectlDeploy:  &latest.KubectlDeploy{},
						},
					},
				},
				shouldErr: true,
			},
		}
		for _, test := range tests {
			testutil.Run(tOuter, test.description, func(t *testutil.T) {
				deployer, err := GetDeployer(context.Background(), &runcontext.RunContext{
					Opts: config.DevloopOptions{
						Apply: test.apply,
					},
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": test.cfg,
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, "")

				t.CheckError(test.shouldErr, err)
				t.CheckTypeEquality(test.expected, deployer)

				if reflect.TypeOf(test.expected) == reflect.TypeOf(deploy.DeployerMux{}) {
					expected := test.expected.(deploy.DeployerMux).GetDeployers()
					deployers := deployer.(deploy.DeployerMux).GetDeployers()
					t.CheckDeepEqual(len(expected), len(deployers))
					for i, v := range expected {
						t.CheckTypeEquality(v, deployers[i])
						if test.deepCheckDeployer {
							t.CheckDeepEqual(v, deployers[i], cmpopts.IgnoreUnexported(cloudrun.Deployer{}, kubectl.Deployer{}, kptV2.Deployer{}))
						}
					}
				} else if test.deepCheckDeployer {
					// it's not a mux so if we want to deep check, do so directly.
					t.CheckDeepEqual(test.expected, deployer, cmpopts.IgnoreUnexported(cloudrun.Deployer{}, kubectl.Deployer{}, kptV2.Deployer{}))
				}
			})
		}
	})
}

func TestGetDefaultDeployer(tOuter *testing.T) {
	testutil.Run(tOuter, "TestGetDeployer", func(t *testutil.T) {
		t.Override(&component.NewAccessor, func(portforward.Config, string, *pkgkubectl.CLI, kubernetes.PodSelector, label.Config, *[]string) access.Accessor {
			return &access.NoopAccessor{}
		})
		t.Override(&component.NewDebugger, func(config.RunMode, kubernetes.PodSelector, *[]string, string) debug.Debugger {
			return &debug.NoopDebugger{}
		})
		t.Override(&component.NewMonitor, func(k8sstatus.Config, string, *label.DefaultLabeller, *[]string, []manifest.GroupKindSelector) k8sstatus.Monitor {
			return &k8sstatus.NoopMonitor{}
		})
		t.Override(&component.NewImageLoader, func(k8sloader.Config, *pkgkubectl.CLI) loader.ImageLoader {
			return &loader.NoopImageLoader{}
		})
		t.Override(&component.NewSyncer, func(*pkgkubectl.CLI, *[]string, k8slogger.Formatter) sync.Syncer {
			return &sync.NoopSyncer{}
		})
		t.Override(&component.NewLogger, func(k8slogger.Config, *pkgkubectl.CLI, kubernetes.PodSelector, *[]string) k8slogger.Logger {
			return &k8slogger.NoopLogger{}
		})
		const configNameForDefaultDeployer = ""
		tests := []struct {
			name      string
			cfgs      map[string]latest.DeployType
			expected  *kubectl.Deployer
			shouldErr bool
		}{
			{
				name: "one config with kubectl deploy",
				cfgs: map[string]latest.DeployType{
					configNameForDefaultDeployer: {KubectlDeploy: &latest.KubectlDeploy{}},
				},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							configNameForDefaultDeployer: {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{},
				}, nil, configNameForDefaultDeployer, nil)).(*kubectl.Deployer),
			},
			{
				name: "one config with kubectl deploy, with flags",
				cfgs: map[string]latest.DeployType{
					configNameForDefaultDeployer: {
						KubectlDeploy: &latest.KubectlDeploy{
							Flags: latest.KubectlFlags{
								Apply:  []string{"--foo"},
								Global: []string{"--bar"},
							},
						},
					},
				},
				expected: t.RequireNonNilResult(kubectl.NewDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							configNameForDefaultDeployer: {},
						},
						[]string{"default"}),
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{
					Flags: latest.KubectlFlags{
						Apply:  []string{"--foo"},
						Global: []string{"--bar"},
					},
				}, nil, configNameForDefaultDeployer, nil)).(*kubectl.Deployer),
			},
			{
				name: "two kubectl configs with mismatched flags should fail",
				cfgs: map[string]latest.DeployType{
					"config1": {
						KubectlDeploy: &latest.KubectlDeploy{
							Flags: latest.KubectlFlags{
								Apply: []string{"--foo"},
							},
						},
					},
					"config2": {
						KubectlDeploy: &latest.KubectlDeploy{
							Flags: latest.KubectlFlags{
								Apply: []string{"--bar"},
							},
						},
					},
				},
				shouldErr: true,
			},
		}

		for _, test := range tests {
			testutil.Run(tOuter, test.name, func(t *testutil.T) {
				pipelines := map[string]latest.Pipeline{}
				var orderedConfigNames []string
				for configName, cfg := range test.cfgs {
					pipeline := latest.Pipeline{
						Deploy: latest.DeployConfig{
							DeployType: cfg,
						},
					}
					pipelines[configName] = pipeline
					orderedConfigNames = append(orderedConfigNames, configName)
				}
				deployer, err := getDefaultDeployer(&runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(pipelines, orderedConfigNames),
				}, &label.DefaultLabeller{}, nil)

				t.CheckErrorAndFailNow(test.shouldErr, err)

				// if we were expecting an error, this implies that the returned deployer is nil
				// this error was checked in the previous call, so if we didn't fail there (i.e. the encountered error was correct),
				// then the test is finished and we can continue.
				if !test.shouldErr {
					t.CheckTypeEquality(&kubectl.Deployer{}, deployer)

					kDeployer := deployer.(*kubectl.Deployer)
					if !reflect.DeepEqual(kDeployer, test.expected) {
						t.Fail()
					}
				}
			})
		}
	})
}

func TestGetCloudRunDeployer(tOuter *testing.T) {
	tests := []struct {
		name     string
		opts     config.DevloopOptions
		cfgs     map[string]latest.DeployType
		expected *cloudrun.Deployer
		haveErr  bool
	}{
		{
			name: "deploy one config with no flags set",
			cfgs: map[string]latest.DeployType{"": {
				CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"},
			}},
			expected: &cloudrun.Deployer{Project: "test-project", Region: "test-region", CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"}},
		},
		{
			name: "deploy with two configs and conflicting processes",
			cfgs: map[string]latest.DeployType{
				"": {
					CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"},
				},
				"second": {
					CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project2", Region: "test-region"},
				},
			},
			haveErr: true,
		},
		{
			name: "deploy with flags set overrides config",
			opts: config.DevloopOptions{CloudRunProject: "overridden-project"},
			cfgs: map[string]latest.DeployType{"": {
				CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "test-project", Region: "test-region"},
			}},
			expected: &cloudrun.Deployer{Project: "overridden-project", Region: "test-region", CloudRunDeploy: &latest.CloudRunDeploy{ProjectID: "overridden-project", Region: "test-region"}},
		},
	}
	for _, test := range tests {
		testutil.Run(tOuter, test.name, func(t *testutil.T) {
			pipelines := make(map[string]latest.Pipeline)
			var orderedConfigNames []string
			for name, config := range test.cfgs {
				pipelines[name] = latest.Pipeline{Deploy: latest.DeployConfig{DeployType: config}}
				orderedConfigNames = append(orderedConfigNames, name)
			}
			rctx := &runcontext.RunContext{
				Opts:      test.opts,
				Pipelines: runcontext.NewPipelines(pipelines, orderedConfigNames),
			}
			crDeployer, err := getCloudRunDeployer(rctx, &label.DefaultLabeller{}, rctx.DeployConfigs(), "")
			t.CheckErrorAndFailNow(test.haveErr, err)
			t.CheckDeepEqual(crDeployer, test.expected, cmpopts.IgnoreUnexported(cloudrun.Deployer{}))
		})
	}
}
