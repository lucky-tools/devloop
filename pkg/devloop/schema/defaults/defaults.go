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

package defaults

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"

	"github.com/lucky-tools/devloop/pkg/devloop/build/kaniko"
	"github.com/lucky-tools/devloop/pkg/devloop/constants"
	kubectx "github.com/lucky-tools/devloop/pkg/devloop/kubernetes/context"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	schemautil "github.com/lucky-tools/devloop/pkg/devloop/schema/util"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
)

// Set makes sure default values are set on a DevloopConfig.
func Set(c *latest.DevloopConfig) error {
	defaultToLocalBuild(c)
	setDefaultTagger(c)
	setDefaultLogsConfig(c)

	for _, a := range c.Build.Artifacts {
		setDefaultWorkspace(a)
		setDefaultSync(a)

		if c.Build.Cluster != nil && a.CustomArtifact == nil && a.BuildpackArtifact == nil {
			defaultToKanikoArtifact(a)
		} else {
			defaultToDockerArtifact(a)
		}

		switch {
		case a.DockerArtifact != nil:
			setDockerArtifactDefaults(a.DockerArtifact)

		case a.KanikoArtifact != nil:
			setKanikoArtifactDefaults(a.KanikoArtifact)

		case a.CustomArtifact != nil:
			setCustomArtifactDefaults(a.CustomArtifact)

		case a.BuildpackArtifact != nil:
			setBuildpackArtifactDefaults(a.BuildpackArtifact)
		}

		for _, d := range a.Dependencies {
			setDefaultArtifactDependencyAlias(d)
		}
	}

	withLocalBuild(c, func(lb *latest.LocalBuild) {
		// don't set build concurrency if there are no artifacts in the current config
		if len(c.Build.Artifacts) > 0 {
			setDefaultConcurrency(lb)
		}
	})

	if err := withClusterConfig(c,
		setDefaultClusterNamespace,
		setDefaultClusterTimeout,
		setDefaultClusterPullSecret,
		setDefaultClusterDockerConfigSecret,
	); err != nil {
		return err
	}

	for i, pf := range c.PortForward {
		if pf == nil {
			return fmt.Errorf("portForward[%d] of config with name '%s' is empty, Please check if it has valid values", i, c.Metadata.Name)
		}
		setDefaultLocalPort(pf)
		setDefaultAddress(pf)
	}

	setDefaultTestWorkspace(c)
	return nil
}

// SetDefaultRenderer sets the default manifests to rawYaml.
func SetDefaultRenderer(c *latest.DevloopConfig) {
	if len(c.Render.Generate.Kpt) > 0 {
		return
	}
	if len(c.Render.Generate.RawK8s) > 0 {
		if c.Deploy.KubectlDeploy == nil && c.Deploy.CloudRunDeploy == nil {
			log.Entry(context.TODO()).Debug("Found raw k8s manifests without cloud run deploy, adding kubectl deployer")
			c.Deploy.KubectlDeploy = &latest.KubectlDeploy{}
		}
		return
	}
	if c.Render.Generate.Kustomize != nil {
		return
	}
	if c.Render.Generate.Helm != nil {
		return
	}
	if c.Deploy.DockerDeploy != nil {
		return
	}
	if c.Deploy.CloudRunDeploy != nil {
		return
	}
	// Set default manifests to "k8s/*.yaml", same as v1.
	c.Render.Generate.RawK8s = constants.DefaultKubectlManifests
}

// SetDefaultDeployer adds a default kubectl deploy configuration.
func SetDefaultDeployer(c *latest.DevloopConfig) {
	if c.Deploy.DeployType != (latest.DeployType{}) {
		return
	}

	log.Entry(context.TODO()).Debug("Defaulting deploy type to kubectl")
	c.Deploy.DeployType.KubectlDeploy = &latest.KubectlDeploy{}
}

func defaultToLocalBuild(c *latest.DevloopConfig) {
	if c.Build.BuildType != (latest.BuildType{}) {
		return
	}

	log.Entry(context.TODO()).Debug("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &latest.LocalBuild{}
}

func withLocalBuild(c *latest.DevloopConfig, operations ...func(*latest.LocalBuild)) {
	if local := c.Build.LocalBuild; local != nil {
		for _, operation := range operations {
			operation(local)
		}
	}
}

func setDefaultConcurrency(local *latest.LocalBuild) {
	if local.Concurrency == nil {
		local.Concurrency = &constants.DefaultLocalConcurrency
	}
}

func setDefaultTagger(c *latest.DevloopConfig) {
	if c.Build.TagPolicy != (latest.TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = latest.TagPolicy{GitTagger: &latest.GitTagger{}}
}

func setDefaultLogsConfig(c *latest.DevloopConfig) {
	if c.Deploy.Logs.Prefix == "" {
		c.Deploy.Logs.Prefix = "container"
	}
}

func defaultToDockerArtifact(a *latest.Artifact) {
	if a.ArtifactType == (latest.ArtifactType{}) {
		a.ArtifactType = latest.ArtifactType{
			DockerArtifact: &latest.DockerArtifact{},
		}
	}
}

func setCustomArtifactDefaults(a *latest.CustomArtifact) {
	if a.Dependencies == nil {
		a.Dependencies = &latest.CustomDependencies{
			Paths: []string{"."},
		}
	}
}

func setBuildpackArtifactDefaults(a *latest.BuildpackArtifact) {
	if a.ProjectDescriptor == "" {
		a.ProjectDescriptor = constants.DefaultProjectDescriptor
	}
	if a.Dependencies == nil {
		a.Dependencies = &latest.BuildpackDependencies{
			Paths: []string{"."},
		}
	}
	if a.Builder == "" && len(a.Buildpacks) == 0 {
		a.Builder = constants.DefaultBuildpacksBuilderImage
		a.TrustBuilder = true
	}
}

func setDockerArtifactDefaults(a *latest.DockerArtifact) {
	a.DockerfilePath = valueOrDefault(a.DockerfilePath, constants.DefaultDockerfilePath)
}

func setDefaultWorkspace(a *latest.Artifact) {
	a.Workspace = valueOrDefault(a.Workspace, ".")
}

func setDefaultSync(a *latest.Artifact) {
	if a.Sync != nil {
		if len(a.Sync.Manual) == 0 && len(a.Sync.Infer) == 0 && a.Sync.Auto == nil {
			switch {
			case a.JibArtifact != nil || a.BuildpackArtifact != nil:
				a.Sync.Auto = util.Ptr(true)
			default:
				a.Sync.Infer = []string{"**/*"}
			}
		}
	} else if a.BuildpackArtifact != nil {
		a.Sync = &latest.Sync{Auto: util.Ptr(true)}
	}
}

func withClusterConfig(c *latest.DevloopConfig, opts ...func(*latest.ClusterDetails) error) error {
	clusterDetails := c.Build.BuildType.Cluster
	if clusterDetails == nil {
		return nil
	}
	for _, o := range opts {
		if err := o(clusterDetails); err != nil {
			return err
		}
	}
	return nil
}

func setDefaultClusterNamespace(cluster *latest.ClusterDetails) error {
	if cluster.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			return fmt.Errorf("getting current namespace: %w", err)
		}
		cluster.Namespace = ns
	}
	return nil
}

func setDefaultClusterTimeout(cluster *latest.ClusterDetails) error {
	cluster.Timeout = valueOrDefault(cluster.Timeout, kaniko.DefaultTimeout)
	return nil
}

func setDefaultClusterPullSecret(cluster *latest.ClusterDetails) error {
	cluster.PullSecretMountPath = valueOrDefault(cluster.PullSecretMountPath, kaniko.DefaultSecretMountPath)
	if cluster.PullSecretPath != "" {
		absPath, err := homedir.Expand(cluster.PullSecretPath)
		if err != nil {
			return fmt.Errorf("unable to expand pullSecretPath %s", cluster.PullSecretPath)
		}
		cluster.PullSecretPath = absPath
		random := ""
		if cluster.RandomPullSecret {
			uid, _ := uuid.NewUUID()
			random = uid.String()
		}
		cluster.PullSecretName = valueOrDefault(cluster.PullSecretName, kaniko.DefaultSecretName+random)
		return nil
	}
	return nil
}

func setDefaultClusterDockerConfigSecret(cluster *latest.ClusterDetails) error {
	if cluster.DockerConfig == nil {
		return nil
	}

	random := ""
	if cluster.RandomDockerConfigSecret {
		uid, _ := uuid.NewUUID()
		random = uid.String()
	}

	cluster.DockerConfig.SecretName = valueOrDefault(cluster.DockerConfig.SecretName, kaniko.DefaultDockerConfigSecretName+random)

	if cluster.DockerConfig.Path == "" {
		return nil
	}

	absPath, err := homedir.Expand(cluster.DockerConfig.Path)
	if err != nil {
		return fmt.Errorf("unable to expand dockerConfig.path %s", cluster.DockerConfig.Path)
	}

	cluster.DockerConfig.Path = absPath
	return nil
}

func defaultToKanikoArtifact(artifact *latest.Artifact) {
	if artifact.KanikoArtifact == nil {
		artifact.KanikoArtifact = &latest.KanikoArtifact{}
	}
}

func setKanikoArtifactDefaults(a *latest.KanikoArtifact) {
	a.Image = valueOrDefault(a.Image, kaniko.DefaultImage)
	a.DockerfilePath = valueOrDefault(a.DockerfilePath, constants.DefaultDockerfilePath)
	a.InitImage = valueOrDefault(a.InitImage, constants.DefaultBusyboxImage)
	a.DigestFile = valueOrDefault(a.DigestFile, constants.DefaultKanikoDigestFile)
	if a.Cache != nil {
		a.Cache.CacheRunLayers = valueOrDefaultBool(a.Cache.CacheRunLayers, true)
	}
	a.CopyMaxRetries = valueOrDefaultInt(a.CopyMaxRetries, kaniko.DefaultCopyMaxRetries)
	a.CopyTimeout = valueOrDefault(a.CopyTimeout, kaniko.DefaultCopyTimeout)
	a.BuildContextCompressionLevel = valueOrDefaultInt(a.BuildContextCompressionLevel, kaniko.DefaultBuildContextCompressionLevel)
}

func valueOrDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

func valueOrDefaultInt(v *int, def int) *int {
	if v != nil {
		return v
	}
	return &def
}

func valueOrDefaultBool(v *bool, def bool) *bool {
	if v != nil {
		return v
	}
	return &def
}

func currentNamespace() (string, error) {
	cfg, err := kubectx.CurrentConfig()
	if err != nil {
		return "", err
	}

	current, present := cfg.Contexts[cfg.CurrentContext]
	if present {
		if current.Namespace != "" {
			return current.Namespace, nil
		}
	}

	return "default", nil
}

func setDefaultLocalPort(pf *latest.PortForwardResource) {
	if pf.LocalPort == 0 {
		if pf.Port.Type == schemautil.Int {
			pf.LocalPort = pf.Port.IntVal
		}
	}
}

func setDefaultAddress(pf *latest.PortForwardResource) {
	if pf.Address == "" {
		pf.Address = constants.DefaultPortForwardAddress
	}
}

func setDefaultArtifactDependencyAlias(d *latest.ArtifactDependency) {
	if d.Alias == "" {
		d.Alias = d.ImageName
	}
}

func setDefaultTestWorkspace(c *latest.DevloopConfig) {
	for _, tc := range c.Test {
		if tc == nil {
			continue
		}
		tc.Workspace = valueOrDefault(tc.Workspace, ".")
	}
}
