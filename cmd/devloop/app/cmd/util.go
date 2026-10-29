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

package cmd

import (
	"context"
	"fmt"

	"github.com/lucky-tools/devloop/pkg/devloop/build"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	tag "github.com/lucky-tools/devloop/pkg/devloop/tag/util"
)

// DefaultRepoFn takes an image tag and returns either a new tag with the default repo prefixed, or the original tag if
// no default repo is specified.
type DefaultRepoFn func(string) (string, error)

// TagFn generates a fully-qualified image name (image:tag) for a config artifact using
// the configured tag policy. It is used as a fallback for artifacts whose tag was not
// provided via --build-artifacts, --images, or --tag.
type TagFn func(context.Context, latest.Artifact) (string, error)

func getBuildArtifactsAndSetTags(artifacts []*latest.Artifact, defaulterFn DefaultRepoFn) ([]graph.Artifact, error) {
	buildArtifacts, err := mergeBuildArtifacts(fromBuildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts(), artifacts)
	if err != nil {
		return nil, err
	}

	return applyDefaultRepoToArtifacts(buildArtifacts, defaulterFn)
}

// getBuildArtifactsAndSetTagsWithTagPolicy resolves tags for non-building commands
// (sync, restart). It first merges tags from --build-artifacts, --images, and --tag,
// then falls back to the tag policy for any artifact that still has no tag.
func getBuildArtifactsAndSetTagsWithTagPolicy(ctx context.Context, artifacts []*latest.Artifact, defaulterFn DefaultRepoFn, tagFn TagFn) ([]graph.Artifact, error) {
	buildArtifacts, err := mergeBuildArtifactsWithoutValidation(fromBuildOutputFile.BuildArtifacts(), preBuiltImages.Artifacts(), artifacts)
	if err != nil {
		return nil, err
	}

	buildArtifacts, err = fillTagsFromTagPolicy(ctx, artifacts, buildArtifacts, tagFn)
	if err != nil {
		return nil, err
	}

	if err := validateArtifactTags(buildArtifacts); err != nil {
		return nil, err
	}

	return applyDefaultRepoToArtifacts(buildArtifacts, defaulterFn)
}

// fillTagsFromTagPolicy fills the empty tags of buildArtifacts using tagFn, leaving
// tags that were already provided via --build-artifacts, --images, or --tag untouched.
func fillTagsFromTagPolicy(ctx context.Context, artifacts []*latest.Artifact, buildArtifacts []graph.Artifact, tagFn TagFn) ([]graph.Artifact, error) {
	if tagFn == nil {
		return buildArtifacts, nil
	}
	for i := range buildArtifacts {
		if buildArtifacts[i].Tag != "" {
			continue
		}
		a := findArtifactByName(artifacts, buildArtifacts[i].ImageName)
		if a == nil {
			continue
		}
		tag, err := tagFn(ctx, *a)
		if err != nil {
			return nil, fmt.Errorf("generating tag for image %q: %w", buildArtifacts[i].ImageName, err)
		}
		buildArtifacts[i].Tag = tag
	}
	return buildArtifacts, nil
}

func findArtifactByName(artifacts []*latest.Artifact, name string) *latest.Artifact {
	for _, a := range artifacts {
		if a.ImageName == name {
			return a
		}
	}
	return nil
}

func applyDefaultRepoToArtifacts(artifacts []graph.Artifact, defaulterFn DefaultRepoFn) ([]graph.Artifact, error) {
	for i := range artifacts {
		updatedTag, err := defaulterFn(artifacts[i].Tag)
		if err != nil {
			return nil, err
		}
		artifacts[i].Tag = updatedTag
	}

	return artifacts, nil
}

func mergeBuildArtifacts(fromFile, fromCLI []graph.Artifact, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	buildArtifacts, err := mergeBuildArtifactsWithoutValidation(fromFile, fromCLI, artifacts)
	if err != nil {
		return nil, err
	}

	// Check that every image has a non empty tag
	if err := validateArtifactTags(buildArtifacts); err != nil {
		return nil, err
	}

	return buildArtifacts, nil
}

// mergeBuildArtifactsWithoutValidation merges config artifacts with tags provided on
// the command line and in the build output file, and applies the --tag override. It
// does not require every artifact to have a non-empty tag, so callers can fill the
// remaining tags from another source (e.g. the tag policy).
func mergeBuildArtifactsWithoutValidation(fromFile, fromCLI []graph.Artifact, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	var buildArtifacts []graph.Artifact
	for _, artifact := range artifacts {
		buildArtifacts = append(buildArtifacts, graph.Artifact{
			ImageName:   artifact.ImageName,
			RuntimeType: artifact.RuntimeType,
		})
	}

	// Tags provided by file take precedence over those provided on the command line
	buildArtifacts = build.MergeWithPreviousBuilds(fromCLI, buildArtifacts)
	buildArtifacts = build.MergeWithPreviousBuilds(fromFile, buildArtifacts)

	return applyCustomTag(buildArtifacts)
}

func applyCustomTag(artifacts []graph.Artifact) ([]graph.Artifact, error) {
	if opts.CustomTag != "" {
		var result []graph.Artifact
		for _, artifact := range artifacts {
			if artifact.Tag == "" {
				artifact.Tag = artifact.ImageName + ":" + opts.CustomTag
			} else {
				newTag, err := tag.SetImageTag(artifact.Tag, opts.CustomTag)
				if err != nil {
					return nil, err
				}
				artifact.Tag = newTag
			}
			result = append(result, artifact)
		}
		return result, nil
	}
	return artifacts, nil
}

func validateArtifactTags(artifacts []graph.Artifact) error {
	for _, artifact := range artifacts {
		if artifact.Tag == "" {
			return fmt.Errorf("no tag provided for image [%s]", artifact.ImageName)
		}
	}
	return nil
}
