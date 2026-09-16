/*
Copyright 2020 The Skaffold Authors

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

package initializer

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lucky-tools/devloop/pkg/devloop/initializer/build"
	"github.com/lucky-tools/devloop/pkg/devloop/initializer/deploy"
	"github.com/lucky-tools/devloop/pkg/devloop/initializer/render"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/warnings"
)

// for testing
var getWd = os.Getwd

func generateDevloopConfig(b build.Initializer, r render.Initializer, d deploy.Initializer) *latest.DevloopConfig {
	// if we're here, the user has no devloop yaml so we need to generate one
	// if the user doesn't have any k8s yamls, generate one for each dockerfile
	log.Entry(context.TODO()).Info("generating devloop config")

	name, err := suggestConfigName()
	if err != nil {
		warnings.Printf("Couldn't generate default config name: %s", err.Error())
	}

	renderConfig, profiles := r.RenderConfig()
	deployConfig := d.DeployConfig()
	buildConfig, portForward := b.BuildConfig()

	return &latest.DevloopConfig{
		APIVersion: latest.Version,
		Kind:       "Config",
		Metadata: latest.Metadata{
			Name: name,
		},
		Pipeline: latest.Pipeline{
			Build:       buildConfig,
			Render:      renderConfig,
			Deploy:      deployConfig,
			PortForward: portForward,
		},
		Profiles: profiles,
	}
}

func suggestConfigName() (string, error) {
	cwd, err := getWd()
	if err != nil {
		return "", err
	}

	base := filepath.Base(cwd)

	// give up for edge cases
	if base == "." || base == string(filepath.Separator) {
		return "", nil
	}

	return canonicalizeName(base), nil
}

// canonicalizeName converts a given string to a valid k8s name string.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names for details
func canonicalizeName(name string) string {
	forbidden := regexp.MustCompile(`[^-.a-z]+`)
	canonicalized := forbidden.ReplaceAllString(strings.ToLower(name), "-")
	if len(canonicalized) <= 253 {
		return canonicalized
	}
	return canonicalized[:253]
}
