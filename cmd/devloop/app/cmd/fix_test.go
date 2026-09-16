/*
Copyright 2019 The Skaffold Authors

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
	"bytes"
	"fmt"
	"testing"

	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/testutil"
)

func TestFix(t *testing.T) {
	tests := []struct {
		description   string
		inputYaml     string
		targetVersion string
		output        string
		shouldErr     bool
	}{
		{
			description:   "already target version",
			targetVersion: latest.Version,
			inputYaml: fmt.Sprintf(`apiVersion: %s
kind: Config
`, latest.Version),
			output: "config is already version " + latest.Version + "\n",
		},
		{
			description: "invalid input",
			inputYaml:   "invalid",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfgFile := t.TempFile("config", []byte(test.inputYaml))

			var b bytes.Buffer
			err := fix(&b, cfgFile, "", test.targetVersion, false)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.output, b.String())
		})
	}
}
