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

package runner

import (
	"context"
	"fmt"
	"io"
	"time"

	deployutil "github.com/lucky-tools/devloop/pkg/devloop/deploy/util"
	"github.com/lucky-tools/devloop/pkg/devloop/event"
	"github.com/lucky-tools/devloop/pkg/devloop/graph"
	"github.com/lucky-tools/devloop/pkg/devloop/instrumentation"
	"github.com/lucky-tools/devloop/pkg/devloop/kubernetes/manifest"
)

// Apply sends Kubernetes manifests to the cluster.
func (r *DevloopRunner) Apply(ctx context.Context, out io.Writer) error {
	var manifests manifest.ManifestList
	var err error
	manifests, err = deployutil.GetManifestsFromHydratedManifests(ctx, r.runCtx.HydratedManifests())
	manifestsByConfig := manifest.NewManifestListByConfig()
	manifestsByConfig.Add(r.deployer.ConfigName(), manifests)

	if err != nil {
		return fmt.Errorf("getting manifests from hydrated manifests: %w", err)
	}
	if err := r.applyResources(ctx, out, nil, nil, manifestsByConfig); err != nil {
		return err
	}

	statusCheckOut, postStatusCheckFn, err := deployutil.WithStatusCheckLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	postStatusCheckFn()
	if err != nil {
		return err
	}
	sErr := r.deployer.GetStatusMonitor().Check(ctx, statusCheckOut)
	return sErr
}

func (r *DevloopRunner) applyResources(ctx context.Context, out io.Writer, artifacts, _ []graph.Artifact, list manifest.ManifestListByConfig) error {
	deployOut, postDeployFn, err := deployutil.WithLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	if err != nil {
		return err
	}

	event.DeployInProgress()
	ctx, endTrace := instrumentation.StartTrace(ctx, "applyResources_Deploying")
	defer endTrace()
	err = r.deployer.Deploy(ctx, deployOut, artifacts, list)
	postDeployFn()
	if err != nil {
		event.DeployFailed(err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	r.deployManifests = list
	event.DeployComplete()
	return nil
}
