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
	"fmt"
	"os"
	"strings"
)

// SetEnvFromConfig sets the environment variables declared in the `env` field of a
// devloop config. Each entry must be in the `KEY=value` form. Variables that are
// already present in the process environment are left untouched, matching the
// behavior of the `devloop.env` file.
func SetEnvFromConfig(env []string) error {
	for _, kv := range env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) != 2 || kvp[0] == "" {
			return fmt.Errorf("invalid env variable %q: should be in `KEY=value` form", kv)
		}

		key, value := kvp[0], kvp[1]
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("setting env variable %q: %w", key, err)
		}
	}
	return nil
}
