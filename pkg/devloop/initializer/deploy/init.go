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

package deploy

import (
	"github.com/lucky-tools/devloop/pkg/devloop/initializer/analyze"
	"github.com/lucky-tools/devloop/pkg/devloop/initializer/config"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

// Initializer detects a deployment type and is able to extract image names from it
type Initializer interface {
	// DeployConfig generates Deploy Config for devloop configuration.
	DeployConfig() latest.DeployConfig
}

type emptyDeployInit struct{}

func (e *emptyDeployInit) DeployConfig() latest.DeployConfig {
	return latest.DeployConfig{}
}

// NewInitializer returns an empty deploy initializer; deployment defaults to kubectl.
func NewInitializer(h analyze.HelmChartInfo, c config.Config) Initializer {
	return &emptyDeployInit{}
}
