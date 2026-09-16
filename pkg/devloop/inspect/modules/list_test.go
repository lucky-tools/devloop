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

func TestPrintModulesList(t *testing.T) {
	tests := []struct {
		description string
		configSet   parser.DevloopConfigSet
		includeAll  bool
		err         error
		expected    string
	}{
		{
			description: "print modules",
			configSet: parser.DevloopConfigSet{
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: ""}}, SourceFile: "path/to/cfg1", SourceIndex: 0, IsRootConfig: true},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: "cfg1"}}, SourceFile: "path/to/cfg1", SourceIndex: 1, IsRootConfig: true},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: "cfg2"}}, SourceFile: "path/to/cfg2", SourceIndex: 0, IsRemote: true},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: ""}}, SourceFile: "path/to/cfg3", SourceIndex: 0},
			},
			expected: `{"modules":[{"name":"cfg1","path":"path/to/cfg1","isRoot":true},{"name":"cfg2","path":"path/to/cfg2","isRemote":true}]}` + "\n",
		},
		{
			description: "print modules; include all",
			configSet: parser.DevloopConfigSet{
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: ""}}, SourceFile: "path/to/cfg1", SourceIndex: 0, IsRootConfig: true},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: "cfg1"}}, SourceFile: "path/to/cfg1", SourceIndex: 1, IsRootConfig: true},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: "cfg2"}}, SourceFile: "path/to/cfg2", SourceIndex: 0, IsRemote: true},
				&parser.DevloopConfigEntry{DevloopConfig: &latest.DevloopConfig{Metadata: latest.Metadata{Name: ""}}, SourceFile: "path/to/cfg3", SourceIndex: 0},
			},
			includeAll: true,
			expected: `{"modules":[` +
				`{"name":"__config_0","path":"path/to/cfg1","isRoot":true},` +
				`{"name":"cfg1","path":"path/to/cfg1","isRoot":true},` +
				`{"name":"cfg2","path":"path/to/cfg2","isRemote":true},` +
				`{"name":"__config_0","path":"path/to/cfg3"}]}` + "\n",
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
			t.Override(&inspect.GetConfigSet, func(ctx context.Context, opts config.DevloopOptions) (parser.DevloopConfigSet, error) {
				if len(opts.ConfigurationFilter) == 0 {
					return test.configSet, test.err
				}
				var set parser.DevloopConfigSet
				if stringslice.Contains(opts.ConfigurationFilter, "cfg1") {
					set = append(set, test.configSet[0])
				}
				if stringslice.Contains(opts.ConfigurationFilter, "cfg2") {
					set = append(set, test.configSet[1])
				}
				return set, test.err
			})
			var buf bytes.Buffer
			err := PrintModulesList(context.Background(), &buf, inspect.Options{OutFormat: "json", ModulesOptions: inspect.ModulesOptions{IncludeAll: test.includeAll}})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
