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
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/blang/semver"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/docker"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/validation"
	"github.com/lucky-tools/devloop/pkg/devloop/update"
	"github.com/lucky-tools/devloop/testutil"
)

func TestCreateNewRunner(t *testing.T) {
	tests := []struct {
		description   string
		config        string
		options       config.DevloopOptions
		shouldErr     bool
		expectedError string
	}{
		{
			description: "valid config",
			config:      "",
			options: config.DevloopOptions{
				ConfigurationFile: "devloop.yaml",
				Trigger:           "polling",
			},
			shouldErr: false,
		},
		{
			description: "invalid config",
			config:      "invalid",
			options: config.DevloopOptions{
				ConfigurationFile: "devloop.yaml",
			},
			shouldErr: true,
		},
		{
			description: "missing config",
			config:      "",
			options: config.DevloopOptions{
				ConfigurationFile: "missing-devloop.yaml",
			},
			shouldErr: true,
		},
		{
			description: "unknown profile",
			config:      "",
			options: config.DevloopOptions{
				ConfigurationFile: "devloop.yaml",
				Profiles:          []string{"unknown-profile"},
			},
			shouldErr:     true,
			expectedError: `profile selection ["unknown-profile"] did not match those defined in any configurations`,
		},
		{
			description: "unsupported trigger",
			config:      "",
			options: config.DevloopOptions{
				ConfigurationFile: "devloop.yaml",
				Trigger:           "unknown trigger",
			},
			shouldErr:     true,
			expectedError: "unsupported trigger",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&validation.DefaultConfig, validation.Options{CheckDeploySource: false})
			t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return docker.NewLocalDaemon(&testutil.FakeAPIClient{
					ErrVersion: true,
				}, nil, false, nil), nil
			})

			t.Override(&update.GetLatestAndCurrentVersion, func() (semver.Version, semver.Version, error) {
				return semver.Version{}, semver.Version{}, nil
			})
			t.NewTempDir().
				Write("devloop.yaml", fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", latest.Version, test.config)).
				Chdir()

			_, _, _, err := createNewRunner(context.Background(), io.Discard, test.options)

			t.CheckError(test.shouldErr, err)
			if test.expectedError != "" {
				t.CheckErrorContains(test.expectedError, err)
			}
		})
	}
}
