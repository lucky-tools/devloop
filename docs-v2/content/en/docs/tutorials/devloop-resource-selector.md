---
title: "Manage CRDs w/ Devloop - Configuring Which K8s Resources & Fields Devloop Manages"
linkTitle: "Manage CRDs w/ Devloop - Configuring Which K8s Resources & Fields Devloop Manages"
weight: 90
featureId: devloop-resource-selector
aliases: [/docs/how-tos/devloop-resource-selector]
---

Common Use Cases This Page Helps Resolve:
* Users who want devloop to properly manage the rendering and deployment of their custom CRDs (as devloop does with K8s objects like Pod, Deployment.apps, etc.)
  * Additionally users w/ a CRD that uses a different field name for `image:` (eg: `foo:`) and want devloop to properly modify the value to instead have the image label for the image devloop recently built
* Users who are seeing issues with devloop's default resource field overwriting for a given resource - eg: devloop errors as it tries to mutate immutable config on re-deployment

Currently devloop modifies the manifests it renders and deploys for the following functionality:
- status checking - done by mutating the manifest/K8s-Object by adding a label - devloop/dev/run-id.  Devloop uses this run-id to identify the deployments Devloop manages with it's status checking.
- image label overwriting - done by mutating the manifest/K8s-Object by substituting the `image:$ORIGINAL_IMAGE_TAG` value(s) in a manifest with `image:$RECENT_DEVLOOP_BUILT_IMAGE`


Devloop has by default the following resources set for management via field "labels:" and "image:" overwriting:

_The below list is derived from the values defined [here](https://github.com/lucky-tools/devloop/blob/main/pkg/devloop/kubernetes/manifest/visitor.go)_


* Pod
* DaemonSet.apps
* Deployment.apps
* ReplicaSet.apps
* StatefulSet.apps (with the exception of `.spec.volumeClaimTemplates.*.metadata.labels` field(s))
* CronJob.batch
* Job.batch
* DaemonSet.extensions
* Deployment.extensions
* ReplicaSet.extension
* Service.serving.knative.dev
* Fleet.agones.dev
* GameServer.agones.dev
* Rollout.argoproj.io
* Workflow.argoproj.io
* CronWorkflow.argoproj.io
* WorkflowTemplate.argoproj.io
* ClusterWorkflowTemplate.argoproj.io
* *.cnrm.cloud.google.com

_This default overwriting modifies all JSON Paths for those GroupKinds of the form:_
* _*.metadata.labels (devloop appends a `run-id` label to existing labels or adds a `labels `field with a `run-id` entry if it didn't exist prior)_
* _*.image (changes `image:` value to be the devloop built image ONLY IF devloop manages the original `image:` value)_


The GroupKind's that Devloop manages (via resource field overwriting) are user configurable via the `resourceSelector:` top level configuration.  
The `resourceSelector` configuration allows users to modify and extend which resources and what fields of those resources devloop modifies.  
Currently devloop only supports `label:` and `.metadata.labels` related modifications.

`resourceSelector` spec (from `pkg/devloop/schema/latest/config.go`)
```go
// ResourceSelector describes user defined filters describing how devloop should treat objects/fields during rendering.
ResourceSelector ResourceSelectorConfig `yaml:"resourceSelector,omitempty"`
```
```go
// ResourceSelectorConfig contains all the configuration needed by the deploy steps.
type ResourceSelectorConfig struct {
	// Allow configures an allowlist for transforming manifests.
	Allow []ResourceFilter `yaml:"allow,omitempty"`
	// Deny configures an allowlist for transforming manifests.
	Deny []ResourceFilter `yaml:"deny,omitempty"`
}
```
```go
// ResourceFilter contains definition to filter which resource to transform.
type ResourceFilter struct {
	// GroupKind is the compact format of a resource type.
	GroupKind string `yaml:"groupKind" yamltags:"required"`
	// Image is an optional slice of JSON-path-like paths of where to rewrite images.
	Image []string `yaml:"image,omitempty"`
	// Labels is an optional slice of JSON-path-like paths of where to add a labels block if missing.
	Labels []string `yaml:"labels,omitempty"`
	// PodSpec is an optional slice of JSON-path-like paths of where pod spec properties can be overwritten.
	PodSpec []string `yaml:"podSpec,omitempty"`
}
```

The values for `Image` and `Labels` support a JSON Path style string which designates a path to a field in the speified GroupKind.  
Additionally there is a special `.*` value that can be used which means that devloop will attempt to overwrite all relevant labels following the below rules:
- image: [".*"] -> replace all fields which follow `*.image:` where the value is an image that devloop manages/builds
- labels: [".*"] -> append-to or create a field named `*.metadata.labels` if a field `*.metadata` is found

Some example use cases and motivations for the `resourceSelector` are shown below:
* Devloop Management of Custom CRD - The below snippet using `resourceSelector` allows a user to configure devloop to manage a custom CRD (eg: CustomDeployment.devloop.dev) they've created for their application in a devloop.  
_Without this snippet, devloop would apply the yaml but would not properly wait for child resources or replace the `image:` values with devloop built images_
{{% readfile file="samples/resource-selector/resource-selector-crd-example.yaml" %}}

Using the above configuation, devloop will properly update any `*.image` field and `*.metadata.labels` field allowing it to work as expected (similar to Pod, Deployment.apps, etc.)

* Fix Issue With Devloop Overwriting Immutable Field - The below snippet using `resourceSelector` shows a user configuring devloop to change it's behaviour to NOT overwrite a resource's field to prevent K8s errors related to overwriting an immutable field:  
_The below configuration is actually a part of devloop's default configuration, just made into a snippet to use as an example_
{{% readfile file="samples/resource-selector/resource-selector-deny-example.yaml" %}}

* Allow `image:` Overwriting For Differently Named Image Field(s) - The below snippet using `resourceSelector` shows a user configuring devloop to change it's behaviour to overwrite a resource's `foo:`field with devloop built images.  
This allows devloop to properly support the images devloop builds this resources which uses `foo:` instead of `image:` for an image value:

{{% readfile file="samples/resource-selector/resource-selector-allow-example.yaml" %}}
