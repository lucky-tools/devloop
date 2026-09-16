---
title: "Building Artifacts with a Custom Build Script"
linkTitle: "Custom Build Script"
weight: 100
---

This page describes building Devloop artifacts using a custom build script, which builds images using [ko](https://github.com/google/ko).
ko builds containers from Go source code, without the need for a Dockerfile or
even installing Docker.

## Before you begin

First, you will need to have Devloop and a Kubernetes cluster set up.
To learn more about how to set up Devloop and a Kubernetes cluster, see the [quickstart docs]({{< relref "/docs/quickstart" >}}).

## Tutorial - Hello World in Go

This tutorial will be based on the [custom example](https://github.com/lucky-tools/devloop/tree/main/examples/custom) in our repository.


## Adding a Custom Builder to Your Devloop Project

We'll need to configure your Devloop config to build artifacts with [ko](https://github.com/google/ko).
To do this, we will take advantage of the [custom builder]({{<relref "/docs/builders/builder-types/custom" >}}) in Devloop.

First, add a `build.sh` file which Devloop will call to build artifacts:

{{% readfile file="samples/builders/custom-buildpacks/build.sh" %}}

Then, configure artifacts in your `devloop.yaml` to build with `build.sh`: 

{{% readfile file="samples/builders/custom-buildpacks/devloop.yaml" %}}

List the file dependencies for each artifact; in the example above, Devloop watches all files in the build context.
For more information about listing dependencies for custom artifacts, see the documentation [here]({{<relref "/docs/builders/builder-types/custom#dependencies-from-a-command" >}}).

You can check custom builder is properly configured by running `devloop build`.
This command should build the artifacts and exit successfully.
