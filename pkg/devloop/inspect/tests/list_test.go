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
	"github.com/lucky-tools/devloop/testutil"
)

func TestPrintTestsList(t *testing.T) {
	tests := []struct {
		description string
		profiles    []string
		module      []string
		err         error
		expected    string
	}{
		{
			description: "print all tests",
			expected:    `{"tests":[{"testType":"structure-test","structureTest":"structure-test-default","structureTestArgs":null},{"testType":"custom-test","Command":"custom-test-default","TimeoutSeconds":0,"Dependencies":null}]}` + "\n",
		},
		{
			description: "print all tests for one module",
			expected:    `{"tests":[{"testType":"structure-test","structureTest":"structure-test-default","structureTestArgs":null}]}` + "\n",
			module:      []string{"cfg1"},
		},
		{
			description: "print all tests for two activated profiles",
			expected:    `{"tests":[{"testType":"custom-test","Command":"custom-test-profile","TimeoutSeconds":0,"Dependencies":null},{"testType":"structure-test","structureTest":"structure-test-profile","structureTestArgs":null}]}` + "\n",
			profiles:    []string{"custom-test", "structure-test"},
		},
		{
			description: "print all tests for one module and an activated profile",
			expected:    `{"tests":[{"testType":"custom-test","Command":"custom-test-profile","TimeoutSeconds":0,"Dependencies":null}]}` + "\n",
			module:      []string{"cfg1"},
			profiles:    []string{"custom-test"},
		},
		{
			description: "actionable error",
			err:         sErrors.MainConfigFileNotFoundErr("path/to/devloop.yaml", fmt.Errorf("failed to read file : %q", "devloop.yaml")),
			expected:    `{"errorCode":"CONFIG_FILE_NOT_FOUND_ERR","errorMessage":"unable to find configuration file \"path/to/devloop.yaml\": failed to read file : \"devloop.yaml\". Check that the specified configuration file exists at \"path/to/devloop.yaml\"."}` + "\n",
		},
		{
			description: "generic error",
			err:         errors.New("some error occurred"),
			expected:    `{"errorCode":"INSPECT_UNKNOWN_ERR","errorMessage":"some error occurred"}` + "\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configSet := parser.DevloopConfigSet{
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{
					Metadata: latest.Metadata{Name: "cfg1"},
					Pipeline: latest.Pipeline{Test: []*latest.TestCase{{StructureTests: []string{"structure-test-default"}}}},
					Profiles: []latest.Profile{
						{Name: "custom-test", Pipeline: latest.Pipeline{Test: []*latest.TestCase{{CustomTests: []latest.CustomTest{{Command: "custom-test-profile"}}}}}},
					},
				}, SourceFile: "path/to/cfg1"},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{
					Metadata: latest.Metadata{Name: "cfg2"},
					Pipeline: latest.Pipeline{Test: []*latest.TestCase{{CustomTests: []latest.CustomTest{{Command: "custom-test-default"}}}}},
					Profiles: []latest.Profile{
						{Name: "structure-test", Pipeline: latest.Pipeline{Test: []*latest.TestCase{{StructureTests: []string{"structure-test-profile"}}}}},
					},
				}, SourceFile: "path/to/cfg2"},
			}
			t.Override(&inspect.GetConfigSet, func(_ context.Context, opts config.DevloopOptions) (parser.DevloopConfigSet, error) {
				// mock profile activation
				var set parser.DevloopConfigSet
				for _, c := range configSet {
					if len(opts.ConfigurationFilter) > 0 && !stringslice.Contains(opts.ConfigurationFilter, c.Metadata.Name) {
						continue
					}
					for _, pName := range opts.Profiles {
						for _, profile := range c.Profiles {
							if profile.Name != pName {
								continue
							}
							c.Test = profile.Test
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			var buf bytes.Buffer
			err := PrintTestsList(context.Background(), &buf, inspect.Options{
				OutFormat: "json", Modules: test.module, Profiles: test.profiles,
			})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
