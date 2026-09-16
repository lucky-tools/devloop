---
title: "Devloop 2.0 Documentation"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
no_list: true
---

{{% alert title="Announcement" color="primary" %}}
Devloop will be removed from gcloud CLI releases after January 15, 2027. Going forward, Devloop can be installed following the instructions in the [Installing Devloop documentation](https://devloop.dev/docs/install/).

On January 29, 2027 the Devloop GitHub repository will be archived. All existing releases will remain available to the users.{{% /alert %}}

Devloop is a command line tool that facilitates continuous development for container based &
Kubernetes applications. Devloop handles the workflow for building,
pushing, and deploying your application, and provides building blocks for
creating CI/CD pipelines. This enables you to focus on iterating on your
application locally while Devloop continuously deploys to your local or remote
Kubernetes cluster, local Docker environment or Cloud Run project.

## Features

* Fast local Kubernetes Development
  * **optimized "Source to Kubernetes"** - Devloop detects changes in your source code and handles the pipeline to
  **build**, **push**, **test** and **deploy** your application automatically with **policy-based image tagging** and **highly optimized, fast local workflows**
  * **continuous feedback** - Devloop automatically manages deployment logging and resource port-forwarding
* Devloop projects work everywhere
  * **share with other developers** - Devloop is the easiest way to **share your project** with the world: `git clone` and `devloop run`
  * **context aware** - use Devloop profiles, local user config, environment variables, and flags to easily incorporate differences across environments
  * **platform aware** - use cross-platform and multi-platform **build** support, with automatic platform detection, to easily handle operating system and architecture differences between the development machine and Kubernetes cluster nodes.
  * **CI/CD building blocks** - use `devloop build`, `devloop test` and `devloop deploy` as part of your CI/CD pipeline, or simply `devloop run` end-to-end
  * **GitOps integration** - use `devloop render` to build your images and render templated Kubernetes manifests for use in GitOps workflows
* devloop.yaml - a single pluggable, declarative configuration for your project
  * **devloop init** - Devloop can discover your build and deployment configuration and generate a Devloop config
  * **multi-component apps** - Devloop supports applications with many components, making it great for microservice-based applications
  * **bring your own tools** - Devloop has a pluggable architecture, allowing for different implementations of the build and deploy stages
* Lightweight
  * **client-side only** - Devloop has no cluster-side component, so there's no overhead or maintenance burden to
  your cluster
  * **minimal pipeline** - Devloop provides an opinionated, minimal pipeline to keep things simple

## Demo

![architecture](/images/intro.gif)

## Devloop Workflow and Architecture

Devloop simplifies your development workflow by organizing common development
stages into one simple command. Every time you run `devloop dev`, the system

1. Collects and watches your source code for changes
1. Syncs files directly to pods if user marks them as syncable
1. Builds artifacts from the source code
1. Tests the built artifacts using [container-structure-tests](https://github.com/GoogleContainerTools/container-structure-test) or custom scripts
1. Tags the artifacts
1. Pushes the artifacts
1. Deploys the artifacts
1. Monitors the deployed artifacts
1. Cleans up deployed artifacts on exit (Ctrl+C)

{{< alert title="Note" >}}
Any of these stages can be skipped.
{{< /alert >}}

The pluggable architecture is central to Devloop's design, allowing you to use
your preferred tool or technology in each stage. Also, Devloop's `profiles` feature
grants you the freedom to switch tools on the fly with a simple flag.

For example, if you are coding on a local machine, you can configure Devloop to build artifacts
with your local Docker daemon and deploy them to minikube using `kubectl`.
When you finalize your design, you can switch to your production profile and deploy with Helm.

Devloop supports the following tools:

{{% tabs %}}
{{% tab "IMAGE BUILDERS" %}}
* [Dockerfile](https://docs.docker.com/engine/reference/builder/)
  - locally with Docker
  - in-cluster with [Kaniko](https://github.com/GoogleContainerTools/kaniko)
* [Jib](https://github.com/GoogleContainerTools/jib) Maven and Gradle
  - locally
* [Cloud Native Buildpacks](https://buildpacks.io/)
  - locally with Docker
* Custom script
  - locally
  - in-cluster
{{% /tab %}}

{{% tab "TESTERS" %}}
* [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test)
* custom script
{{% /tab %}}

{{% tab "DEPLOYERS" %}}
* Kubernetes Command-Line Interface (`kubectl`)
* Helm
* kustomize
{{% /tab %}}

{{% tab "TAG POLICIES" %}}
* tag by git commit
* tag by current date & time
* tag by environment variables based template
* tag by digest of the Docker image
{{% /tab %}}

{{% tab "PUSH STRATEGIES" %}}
* don't push - keep the image on the local daemon
* push to registry
{{% /tab %}}
{{% /tabs %}}


![architecture](/images/architecture.png)


Besides the above steps, Devloop also automatically manages the following utilities for you:

* port-forwarding of deployed resources to your local machine using `kubectl port-forward`
* log aggregation from the deployed pods
