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

package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/lucky-tools/devloop/pkg/devloop/update"
	"github.com/lucky-tools/devloop/testutil"
)

func TestPreReleaseVersion(t *testing.T) {
	tests := []struct {
		description string
		versionStr  string
		expected    bool
	}{
		{
			description: "pre release version",
			versionStr:  "v1.20.0-42-g92900f245-dirty",
			expected:    true,
		},
		{
			description: "release version",
			versionStr:  "v1.20.0",
		},
		{
			description: "incorrect version indicates pre release or dev version",
			versionStr:  "blah",
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := preReleaseVersion(test.versionStr)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestSetFlagsFromEnvVariables(t *testing.T) {
	buildCmd := func() (*cobra.Command, *string) {
		var val string
		root := &cobra.Command{Use: "root"}
		sub := &cobra.Command{Use: "sub"}
		sub.Flags().StringVar(&val, "images", "", "images to deploy")
		root.AddCommand(sub)
		return root, &val
	}

	tests := []struct {
		name        string
		skipChanged bool
		preSet      string
		envVal      string
		want        string
	}{
		{
			name:   "applies env var at startup",
			envVal: "fund-download:latest",
			want:   "fund-download:latest",
		},
		{
			name:        "applies env var when flag unchanged after config",
			skipChanged: true,
			envVal:      "fund-download:latest",
			want:        "fund-download:latest",
		},
		{
			name:        "skips already-changed flag",
			skipChanged: true,
			preSet:      "cli-value",
			envVal:      "fund-download:latest",
			want:        "cli-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEVLOOP_IMAGES", tt.envVal)
			root, val := buildCmd()
			if tt.preSet != "" {
				if err := root.Commands()[0].Flags().Set("images", tt.preSet); err != nil {
					t.Fatalf("pre-setting flag: %v", err)
				}
			}
			setFlagsFromEnvVariables(root, tt.skipChanged)
			if *val != tt.want {
				t.Errorf("flag value = %q, want %q", *val, tt.want)
			}
		})
	}
}

func TestUpdateCheck(t *testing.T) {
	tests := []struct {
		description string
		versionStr  string
		expected    string
		versionMsg  string
		updateCheck bool
	}{
		{
			description: "pre release version, update check is disabled",
			versionStr:  "v1.20.0-42-g92900f245-dirty",
		},
		{
			description: "release version but update check disabled",
			versionStr:  "v1.20.0",
		},
		{
			description: "release version and update check enabled",
			versionStr:  "v1.20.0",
			updateCheck: true,
			versionMsg:  "newer version is available",
			expected:    "newer version is available",
		},
		{
			description: "update check enabled but version already up to date",
			versionStr:  "v1.20.0",
			updateCheck: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&updateCheck, func(string) (string, error) {
				return test.versionMsg, nil
			})
			t.Override(&update.EnableCheck, test.updateCheck)
			actual := updateCheckForReleasedVersionsIfNotDisabled(test.versionStr)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
