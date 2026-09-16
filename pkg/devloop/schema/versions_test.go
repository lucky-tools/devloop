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

package schema

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/lucky-tools/devloop/pkg/devloop/build/kaniko"
	"github.com/lucky-tools/devloop/pkg/devloop/constants"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/defaults"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/util"
	"github.com/lucky-tools/devloop/testutil"
)

const (
	minimalConfig = ``

	simpleConfig = `
build:
  tagPolicy:
    gitCommit: {}
  artifacts:
  - image: example
`
	// This config has two tag policies set.
	invalidConfig = `
build:
  tagPolicy:
    sha256: {}
    gitCommit: {}
  artifacts:
  - image: example
deploy:
  name: example
`

	completeConfig = `
build:
  tagPolicy:
    sha256: {}
  artifacts:
  - image: image1
    context: ./examples/app1
    docker:
      dockerfile: Dockerfile.dev
manifests:
  rawYaml:
   - dep.yaml
   - svc.yaml
`
	completeV1Config = `
build:
  tagPolicy:
    sha256: {}
  artifacts:
  - image: image1
    context: ./examples/app1
    docker:
      dockerfile: Dockerfile.dev
deploy:
  kubectl:
    manifests:
    - dep.yaml
    - svc.yaml
`
	minimalClusterConfig = `
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko: {}
  cluster: {}
`
	kanikoConfigMap = `
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko:
      volumeMounts:
      - name: docker-config
        mountPath: /kaniko/.docker
  cluster:
    pullSecretName: "some-secret"
    volumes:
    - name: docker-config
      configMap:
        name: docker-config
`

	completeClusterConfig = `
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko: {}
  cluster:
    pullSecretPath: /secret.json
    pullSecretName: secret-name
    namespace: nskaniko
    timeout: 120m
    dockerConfig:
      secretName: config-name
      path: /kaniko/.docker
`

	badConfig = "bad config"

	invalidStatusCheckConfig = `
deploy:
  statusCheckDeadlineSeconds: s
`
	validStatusCheckConfig = `
deploy:
  statusCheckDeadlineSeconds: 10
`

	customLogPrefix = `
deploy:
  logs:
    prefix: none
`
)

func TestIsDevloopConfig(t *testing.T) {
	tests := []struct {
		description string
		contents    string
		isValid     bool
	}{
		{
			description: "valid devloop config",
			contents: `apiVersion: devloop/v1
kind: Config`,
			isValid: true,
		},
		{
			description: "not a valid format",
			contents:    "test",
			isValid:     false,
		},
		{
			description: "invalid devloop config version",
			contents: `apiVersion: devloop/v1
kind: Config
deploy:
  kustomize: {}`,
			isValid: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("devloop.yaml", test.contents)

			isValid := IsDevloopConfig(tmpDir.Path("devloop.yaml"))

			t.CheckDeepEqual(test.isValid, isValid)
		})
	}
}

func TestParseConfigAndUpgrade(t *testing.T) {
	tests := []struct {
		apiVersion  []string
		description string
		config      []string
		expected    []util.VersionedConfig
		shouldErr   bool
	}{
		{
			apiVersion:  []string{latest.Version},
			description: "Kaniko Volume Mount - ConfigMap",
			config:      []string{kanikoConfigMap},
			expected: []util.VersionedConfig{config(
				withClusterBuild("some-secret", "/secret", "default", "", "20m",
					withGitTagger(),
					withKanikoArtifact(),
					withKanikoVolumeMount("docker-config", "/kaniko/.docker"),
					withVolume(v1.Volume{
						Name: "docker-config",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "docker-config",
								},
							},
						},
					})),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version},
			description: "Minimal config",
			config:      []string{minimalConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
				),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version},
			description: "Simple config",
			config:      []string{simpleConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withLocalBuildConcurrency(&constants.DefaultLocalConcurrency),
					withGitTagger(),
					withDockerArtifact("example", ".", "Dockerfile"),
				),
				withRawK8s(),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version},
			description: "Complete config v1",
			config:      []string{completeConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withLocalBuildConcurrency(&constants.DefaultLocalConcurrency),
					withShaTagger(),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
				),
				withRawK8s("dep.yaml", "svc.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version, latest.Version},
			description: "Multiple complete config with same API versions",
			config:      []string{completeConfig, completeClusterConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withLocalBuildConcurrency(&constants.DefaultLocalConcurrency),
					withShaTagger(),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
				),
				withRawK8s("dep.yaml", "svc.yaml"),
				withLogsPrefix("container"),
			), config(
				withClusterBuild("secret-name", "/secret", "nskaniko", "/secret.json", "120m",
					withGitTagger(),
					withDockerConfig("config-name", "/kaniko/.docker"),
					withKanikoArtifact(),
				),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version},
			description: "Minimal Cluster config",
			config:      []string{minimalClusterConfig},
			expected: []util.VersionedConfig{config(
				withClusterBuild("", "/secret", "default", "", "20m",
					withGitTagger(),
					withKanikoArtifact(),
				),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version},
			description: "Complete Cluster config",
			config:      []string{completeClusterConfig},
			expected: []util.VersionedConfig{config(
				withClusterBuild("secret-name", "/secret", "nskaniko", "/secret.json", "120m",
					withGitTagger(),
					withDockerConfig("config-name", "/kaniko/.docker"),
					withKanikoArtifact(),
				),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version},
			description: "Bad config",
			config:      []string{badConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{latest.Version},
			description: "two taggers defined",
			config:      []string{invalidConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{""},
			description: "ApiVersion not specified",
			config:      []string{minimalConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{latest.Version},
			description: "invalid statusCheckDeadline",
			config:      []string{invalidStatusCheckConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{latest.Version},
			description: "valid statusCheckDeadline",
			config:      []string{validStatusCheckConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
				),
				withStatusCheckDeadline(10),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latest.Version},
			description: "custom log prefix",
			config:      []string{customLogPrefix},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
				),
				withLogsPrefix("none"),
			)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			tmpDir := t.NewTempDir().
				Write("devloop.yaml", format(t, test.config, test.apiVersion))

			cfgs, err := ParseConfigAndUpgrade(tmpDir.Path("devloop.yaml"))
			for _, cfg := range cfgs {
				err := defaults.Set(cfg.(*latest.DevloopConfig))
				t.CheckNoError(err)
			}

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, cfgs)
		})
	}
}

func TestMarshalConfig(t *testing.T) {
	tests := []struct {
		description string
		config      *latest.DevloopConfig
		shouldErr   bool
	}{
		{
			description: "Kaniko Volume Mount - ConfigMap",
			config: config(
				withClusterBuild("some-secret", "/some/secret", "default", "", "20m",
					withGitTagger(),
					withKanikoArtifact(),
					withKanikoVolumeMount("docker-config", "/kaniko/.docker"),
					withVolume(v1.Volume{
						Name: "docker-config",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "docker-config",
								},
							},
						},
					})),
				withLogsPrefix("container"),
			),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			actual, err := yaml.Marshal(test.config)
			t.CheckNoError(err)

			// Unmarshal the YAML and make sure it equals the original.
			// We can't compare the strings because the YAML serializer isn't deterministic.
			// TestParseConfigAndUpgrade verifies that YAML -> Go works correctly.
			// This test verifies Go -> YAML -> Go returns the original structure. Since we know
			// YAML -> Go is working this ensures Go -> YAML is correct.
			recovered := &latest.DevloopConfig{}

			err = yaml.Unmarshal(actual, recovered)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.config, recovered)
		})
	}
}

func config(ops ...func(*latest.DevloopConfig)) *latest.DevloopConfig {
	cfg := &latest.DevloopConfig{APIVersion: latest.Version, Kind: "Config"}
	for _, op := range ops {
		op(cfg)
	}
	return cfg
}

func format(t *testutil.T, configs []string, apiVersions []string) string {
	var str []string
	t.CheckDeepEqual(len(configs), len(apiVersions))
	for i := range configs {
		str = append(str, fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", apiVersions[i], configs[i]))
	}
	return strings.Join(str, "\n---\n")
}

func withLocalBuild(ops ...func(*latest.BuildConfig)) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		b := latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withClusterBuild(secretName, mountPath, namespace, secret string, timeout string, ops ...func(*latest.BuildConfig)) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		b := latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{
			PullSecretName:      secretName,
			Namespace:           namespace,
			PullSecretPath:      secret,
			PullSecretMountPath: mountPath,
			Timeout:             timeout,
		}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withDockerConfig(secretName string, path string) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		cfg.Cluster.DockerConfig = &latest.DockerConfig{
			SecretName: secretName,
			Path:       path,
		}
	}
}

func withRawK8s(manifests ...string) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.Render.RawK8s = manifests
	}
}

func withKubectlDeployer() func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.Deploy.KubectlDeploy = &latest.KubectlDeploy{}
	}
}

func withKubeContext(kubeContext string) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.Deploy = latest.DeployConfig{
			KubeContext: kubeContext,
		}
	}
}

func withDockerArtifact(image, workspace, dockerfile string) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &latest.Artifact{
			ImageName: image,
			Workspace: workspace,
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: dockerfile,
				},
			},
		})
	}
}

func withKanikoArtifact() func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		copyMaxRetries := 3
		compressionLevel := 1
		cfg.Artifacts = append(cfg.Artifacts, &latest.Artifact{
			ImageName: "image1",
			Workspace: "./examples/app1",
			ArtifactType: latest.ArtifactType{
				KanikoArtifact: &latest.KanikoArtifact{
					DockerfilePath:               "Dockerfile",
					InitImage:                    constants.DefaultBusyboxImage,
					Image:                        kaniko.DefaultImage,
					DigestFile:                   "/dev/termination-log",
					CopyMaxRetries:               &copyMaxRetries,
					CopyTimeout:                  "5m",
					BuildContextCompressionLevel: &compressionLevel,
				},
			},
		})
	}
}

// withKanikoVolumeMount appends a volume mount to the latest Kaniko artifact
func withKanikoVolumeMount(name, mountPath string) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		if cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts == nil {
			cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts = []v1.VolumeMount{}
		}

		cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts = append(
			cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts,
			v1.VolumeMount{
				Name:      name,
				MountPath: mountPath,
			},
		)
	}
}

// withVolume appends a volume to the cluster
func withVolume(v v1.Volume) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		cfg.Cluster.Volumes = append(cfg.Cluster.Volumes, v)
	}
}

func withLocalBuildConcurrency(concurrency *int) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) { cfg.LocalBuild.Concurrency = concurrency }
}

func withTagPolicy(tagPolicy latest.TagPolicy) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) { cfg.TagPolicy = tagPolicy }
}

func withGitTagger() func(*latest.BuildConfig) {
	return withTagPolicy(latest.TagPolicy{GitTagger: &latest.GitTagger{}})
}

func withShaTagger() func(*latest.BuildConfig) {
	return withTagPolicy(latest.TagPolicy{ShaTagger: &latest.ShaTagger{}})
}

func withProfiles(profiles ...latest.Profile) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.Profiles = profiles
	}
}

func withTests(testCases ...*latest.TestCase) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.Test = testCases
	}
}

func withPortForward(portForward ...*latest.PortForwardResource) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.PortForward = portForward
	}
}

func withStatusCheckDeadline(deadline int) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.Deploy.StatusCheckDeadlineSeconds = deadline
	}
}

func withLogsPrefix(prefix string) func(*latest.DevloopConfig) {
	return func(cfg *latest.DevloopConfig) {
		cfg.Deploy.Logs.Prefix = prefix
	}
}
