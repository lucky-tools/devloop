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
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// PullOptions configures how an image is pulled from a registry.
type PullOptions struct {
	// Insecure forces the target registry to be treated as insecure
	// (plain HTTP and/or untrusted TLS).
	Insecure bool
	// InsecureRegistries is the set of registry hostnames to treat as insecure,
	// e.g. populated from the global config `insecure-registries`.
	InsecureRegistries map[string]bool
	// Platform, when set, selects a specific platform from a multi-platform image.
	Platform *v1.Platform
	// Username and Password, when set, are used for basic auth instead of the
	// default keychain (docker config / ~/.devloop config.json).
	Username string
	Password string
}

// Pull pulls an image from a registry and writes it to path as a docker-format
// tarball (docker save style).
func Pull(ctx context.Context, identifier, path string, opts PullOptions) error {
	ref, img, err := pullImage(ctx, identifier, opts)
	if err != nil {
		return err
	}

	if err := tarball.WriteToFile(path, ref, img); err != nil {
		return fmt.Errorf("writing image tarball %q: %w", path, err)
	}
	return nil
}

func pullImage(ctx context.Context, identifier string, opts PullOptions) (name.Reference, v1.Image, error) {
	ref, err := name.ParseReference(identifier)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing reference %q: %w", identifier, err)
	}

	insecure := opts.Insecure || IsInsecure(ref, opts.InsecureRegistries)
	if insecure {
		ref, err = name.ParseReference(identifier, name.Insecure)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing insecure reference %q: %w", identifier, err)
		}
	}

	options := []remote.Option{remote.WithContext(ctx)}
	if opts.Username != "" || opts.Password != "" {
		options = append(options, remote.WithAuth(&authn.Basic{Username: opts.Username, Password: opts.Password}))
	} else {
		options = append(options, remote.WithAuthFromKeychain(primaryKeychain))
	}
	if insecure {
		options = append(options, insecureTransportOption())
	}
	if opts.Platform != nil {
		options = append(options, remote.WithPlatform(*opts.Platform))
	}

	img, err := remoteImage(ref, options...)
	if err != nil {
		return nil, nil, fmt.Errorf("pulling image %q: %w", identifier, err)
	}
	return ref, img, nil
}

// PullOutputPath derives a default tarball path from an image reference, e.g.
// `localhost:5000/myorg/app:1.0` -> `app_1.0.tar`.
func PullOutputPath(identifier string) (string, error) {
	ref, err := name.ParseReference(identifier, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("parsing reference %q: %w", identifier, err)
	}

	repo := ref.Context().RepositoryStr()
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		repo = repo[i+1:]
	}
	if tag, ok := ref.(name.Tag); ok {
		return fmt.Sprintf("%s_%s.tar", repo, tag.Identifier()), nil
	}
	return repo + ".tar", nil
}
