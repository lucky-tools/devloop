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

package docker

import (
	"context"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/lucky-tools/devloop/testutil"
)

func TestPullImage(t *testing.T) {
	tests := []struct {
		description    string
		image          string
		opts           PullOptions
		expectedScheme string
		shouldErr      bool
	}{
		{
			description:    "secure",
			image:          "gcr.io/secure/image:latest",
			expectedScheme: "https",
		},
		{
			description:    "insecure flag",
			image:          "my.insecure.registry/image:latest",
			opts:           PullOptions{Insecure: true},
			expectedScheme: "http",
		},
		{
			description: "insecure registries map",
			image:       "my.insecure.registry/image:latest",
			opts: PullOptions{
				InsecureRegistries: map[string]bool{"my.insecure.registry": true},
			},
			expectedScheme: "http",
		},
		{
			description: "invalid",
			image:       "invalid image",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&remoteImage, func(ref name.Reference, options ...remote.Option) (v1.Image, error) {
				return &fakeImage{Reference: ref}, nil
			})

			ref, img, err := pullImage(context.Background(), test.image, test.opts)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expectedScheme, ref.Context().Registry.Scheme())
				t.CheckNotNil(img)
			}
		})
	}
}

func TestPullOutputPath(t *testing.T) {
	tests := []struct {
		description string
		image       string
		expected    string
		shouldErr   bool
	}{
		{
			description: "repository and tag",
			image:       "localhost:5000/myorg/app:1.0",
			expected:    "app_1.0.tar",
		},
		{
			description: "default latest tag",
			image:       "localhost:5000/app",
			expected:    "app_latest.tar",
		},
		{
			description: "registry and image",
			image:       "registry.tld/app:latest",
			expected:    "app_latest.tar",
		},
		{
			description: "invalid",
			image:       "invalid image",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got, err := PullOutputPath(test.image)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, got)
		})
	}
}
