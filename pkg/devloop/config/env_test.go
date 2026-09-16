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

package config

import (
	"os"
	"testing"
)

func TestSetEnvFromConfig(t *testing.T) {
	tests := []struct {
		name      string
		env       []string
		preSet    map[string]string
		want      map[string]string
		shouldErr bool
	}{
		{
			name: "sets new variables",
			env:  []string{"DEVLOOP_TEST_ENV_A=1", "DEVLOOP_TEST_ENV_B=2"},
			want: map[string]string{"DEVLOOP_TEST_ENV_A": "1", "DEVLOOP_TEST_ENV_B": "2"},
		},
		{
			name:   "keeps existing variables",
			env:    []string{"DEVLOOP_TEST_ENV_A=from_config"},
			preSet: map[string]string{"DEVLOOP_TEST_ENV_A": "from_shell"},
			want:   map[string]string{"DEVLOOP_TEST_ENV_A": "from_shell"},
		},
		{
			name: "empty list is a no-op",
			want: map[string]string{},
		},
		{
			name:      "rejects malformed entry",
			env:       []string{"NO_EQUALS_SIGN"},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset any state left behind by a prior subtest.
			for k := range tt.preSet {
				os.Unsetenv(k)
			}
			for k := range tt.want {
				os.Unsetenv(k)
			}
			for k, v := range tt.preSet {
				t.Setenv(k, v)
			}

			err := SetEnvFromConfig(tt.env)
			if tt.shouldErr {
				if err == nil {
					t.Fatalf("SetEnvFromConfig() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("SetEnvFromConfig() unexpected error: %v", err)
			}
			for k, want := range tt.want {
				if got, ok := os.LookupEnv(k); !ok || got != want {
					t.Errorf("env[%q] = %q (present=%v), want %q", k, got, ok, want)
				}
			}
		})
	}
}
