/*
Copyright 2026 The Skaffold Authors

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

package docker

import (
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
)

// BuildKitAuthProvider returns a session.Attachable that supplies registry
// credentials to a buildkit solve, using the Devloop docker config (including
// the GCR credential-helper auto-configuration done in loadDockerConfig).
// If the docker config can't be loaded, it falls back to anonymous auth.
//
// Exposed as a variable so tests can override it.
var BuildKitAuthProvider = buildKitAuthProvider

func buildKitAuthProvider() (session.Attachable, error) {
	cf, err := loadDockerConfig()
	if err != nil {
		return authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{}), nil
	}
	return authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
		AuthConfigProvider: authprovider.LoadAuthConfig(cf),
	}), nil
}
