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

package inspect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/inspect"
	"github.com/lucky-tools/devloop/pkg/devloop/parser"
	sErrors "github.com/lucky-tools/devloop/pkg/devloop/schema/errors"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/util/stringslice"
	"github.com/lucky-tools/devloop/pkg/devloop/yaml"
	"github.com/lucky-tools/devloop/testutil"
)

const (
	pathToCfg1 = "path/to/cfg1"
	pathToCfg2 = "path/to/cfg2"
)

func TestAddClusterBuildEnv(t *testing.T) {
	tests := []struct {
		description     string
		profile         string
		modules         []string
		buildEnvOpts    inspect.BuildEnvOptions
		expectedConfigs []string
		err             error
		expectedErrMsg  string
	}{
		{
			description:  "add to default pipeline",
			buildEnvOpts: inspect.BuildEnvOptions{RunAsUser: -1, PullSecretPath: "path/to/secret", DockerConfigPath: "path/to/docker/config", Timeout: "128", Concurrency: 2},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  cluster:
    concurrency: 2
    dockerConfig:
      path: path/to/docker/config
    pullSecretPath: path/to/secret
    timeout: "128"
profiles:
- name: p1
  build:
    cluster: {}
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
build:
  cluster:
    concurrency: 2
    dockerConfig:
      path: path/to/docker/config
    pullSecretPath: path/to/secret
    timeout: "128"
profiles:
- name: p1
  build:
    cluster: {}
`, ``,
			},
		},
		{
			description:  "add to existing profile",
			profile:      "p1",
			buildEnvOpts: inspect.BuildEnvOptions{RunAsUser: -1, PullSecretPath: "path/to/secret", DockerConfigPath: "path/to/docker/config", Timeout: "128", Concurrency: 2},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  local: {}
profiles:
- name: p1
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
  activeProfiles:
  - name: p1
    activatedBy:
    - p1
build:
  local: {}
profiles:
- name: p1
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  local: {}
profiles:
- name: p1
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
`,
			},
		},
		{
			description:  "add to new profile",
			profile:      "p2",
			buildEnvOpts: inspect.BuildEnvOptions{RunAsUser: -1, PullSecretPath: "path/to/secret", DockerConfigPath: "path/to/docker/config", Timeout: "128", Concurrency: 2},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
- name: p2
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
  activeProfiles:
  - name: p2
    activatedBy:
    - p2
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
- name: p2
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  local: {}
profiles:
- name: p1
  build:
    local: {}
- name: p2
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
`,
			},
		},
		{
			description:  "add to new profile in selected modules",
			modules:      []string{"cfg1_1"},
			profile:      "p2",
			buildEnvOpts: inspect.BuildEnvOptions{RunAsUser: -1, PullSecretPath: "path/to/secret", DockerConfigPath: "path/to/docker/config", Timeout: "128", Concurrency: 2},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1_0
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
---
apiVersion: ""
kind: ""
metadata:
  name: cfg1_1
requires:
- path: path/to/cfg2
  activeProfiles:
  - name: p2
    activatedBy:
    - p2
build:
  local: {}
profiles:
- name: p1
  build:
    cluster: {}
- name: p2
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  local: {}
profiles:
- name: p1
  build:
    local: {}
- name: p2
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
`, "",
			},
		},
		{
			description:  "add to new profile in nested module",
			modules:      []string{"cfg2"},
			profile:      "p2",
			buildEnvOpts: inspect.BuildEnvOptions{RunAsUser: -1, PullSecretPath: "path/to/secret", DockerConfigPath: "path/to/docker/config", Timeout: "128", Concurrency: 2},
			expectedConfigs: []string{
				"",
				`apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  local: {}
profiles:
- name: p1
  build:
    local: {}
- name: p2
  build:
    cluster:
      concurrency: 2
      dockerConfig:
        path: path/to/docker/config
      pullSecretPath: path/to/secret
      timeout: "128"
`,
			},
		},
		{
			description:    "actionable error",
			err:            sErrors.MainConfigFileNotFoundErr("path/to/devloop.yaml", fmt.Errorf("failed to read file : %q", "devloop.yaml")),
			expectedErrMsg: `{"errorCode":"CONFIG_FILE_NOT_FOUND_ERR","errorMessage":"unable to find configuration file \"path/to/devloop.yaml\": failed to read file : \"devloop.yaml\". Check that the specified configuration file exists at \"path/to/devloop.yaml\"."}` + "\n",
		},
		{
			description:    "generic error",
			err:            errors.New("some error occurred"),
			expectedErrMsg: `{"errorCode":"INSPECT_UNKNOWN_ERR","errorMessage":"some error occurred"}` + "\n",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configSet := parser.DevloopConfigSet{
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{
					Metadata: latest.Metadata{Name: "cfg1_0"},
					Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}},
					Profiles: []latest.Profile{
						{Name: "p1", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}}},
					},
				}, SourceFile: pathToCfg1, IsRootConfig: true, SourceIndex: 0},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{
					Metadata:     latest.Metadata{Name: "cfg1_1"},
					Dependencies: []latest.ConfigDependency{{Path: pathToCfg2}},
					Pipeline:     latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}},
					Profiles: []latest.Profile{
						{Name: "p1", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}}},
					},
				}, SourceFile: pathToCfg1, IsRootConfig: true, SourceIndex: 1},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{
					Metadata: latest.Metadata{Name: "cfg2"},
					Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}},
					Profiles: []latest.Profile{
						{Name: "p1", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					},
				}, SourceFile: pathToCfg2, SourceIndex: 0},
			}
			t.Override(&inspect.GetConfigSet, func(ctx context.Context, opts config.DevloopOptions) (parser.DevloopConfigSet, error) {
				if test.err != nil {
					return nil, test.err
				}
				var sets parser.DevloopConfigSet
				if len(opts.ConfigurationFilter) == 0 || stringslice.Contains(opts.ConfigurationFilter, "cfg2") || stringslice.Contains(opts.ConfigurationFilter, "cfg1_1") {
					sets = append(sets, configSet[2])
				}
				if len(opts.ConfigurationFilter) == 0 || stringslice.Contains(opts.ConfigurationFilter, "cfg1_0") {
					sets = append(sets, configSet[0])
				}
				if len(opts.ConfigurationFilter) == 0 || stringslice.Contains(opts.ConfigurationFilter, "cfg1_1") {
					sets = append(sets, configSet[1])
				}
				return sets, nil
			})
			t.Override(&inspect.ReadFileFunc, func(filename string) ([]byte, error) {
				if filename == pathToCfg1 {
					return yaml.MarshalWithSeparator([]*latest.DevloopConfig{configSet[0].DevloopConfig, configSet[1].DevloopConfig})
				} else if filename == pathToCfg2 {
					return yaml.MarshalWithSeparator([]*latest.DevloopConfig{configSet[2].DevloopConfig})
				}
				t.FailNow()
				return nil, nil
			})
			var actualCfg1, actualCfg2 string
			t.Override(&inspect.WriteFileFunc, func(filename string, data []byte) error {
				switch filename {
				case pathToCfg1:
					actualCfg1 = string(data)
				case pathToCfg2:
					actualCfg2 = string(data)
				default:
					t.FailNow()
				}
				return nil
			})

			var buf bytes.Buffer
			err := AddClusterBuildEnv(context.Background(), &buf, inspect.Options{OutFormat: "json", Modules: test.modules, Profile: test.profile, BuildEnvOptions: test.buildEnvOpts})
			t.CheckError(test.err != nil, err)
			if test.err == nil {
				t.CheckDeepEqual(test.expectedConfigs[0], actualCfg1, testutil.YamlObj(t.T))
				t.CheckDeepEqual(test.expectedConfigs[1], actualCfg2, testutil.YamlObj(t.T))
			} else {
				t.CheckDeepEqual(test.expectedErrMsg, buf.String())
			}
		})
	}
}
