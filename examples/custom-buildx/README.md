### Example: use the custom builder with `docker buildx`

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/lucky-tools/devloop&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/custom-buildx)

[Docker Buildx](https://github.com/docker/buildx#buildx) is an
experimental feature for building container images for multiple
platforms.  This example shows how `docker buildx` can be used as
a Devloop _custom builder_ to create container images for the
`linux/arm64` and `linux/amd64` platforms.


#### Before you begin

For this tutorial to work you need to ensure Devloop and a Kubernetes
cluster are set up.  To learn more about how to set up Devloop and
a Kubernetes cluster, see the [getting started docs](https://devloop.dev/docs/getting-started).

Note that this example builds for two different platforms and
requires pushing images to a container registry such as
[Google Artifact Registry](https://cloud.google.com/artifact-registry).

#### Tutorial

This tutorial demonstrates how to use Devloop's _custom builders_
to build a simple _Hello World_ Go application for `linux/amd64`
and `linux/arm64` using `docker buildx` and deploy it to a Kubernetes
cluster.

##### Step 1: Obtain the example

First, clone the Devloop [repo](https://github.com/lucky-tools/devloop)
and navigate to the [`custom-buildx` example](https://github.com/lucky-tools/devloop/tree/main/examples/custom) for sample code:

```shell
$ git clone https://github.com/lucky-tools/devloop
$ cd devloop/examples/custom-buildx
```

##### Step 2: Configure the custom builder

Take a look at the [`devloop.yaml`](devloop.yaml), which uses a
_custom builder_
```yaml
  - image: devloop-examples-buildx
    custom:
      buildCommand: sh buildx.sh
      dependencies:
        paths: ["go.mod", "**.go", "buildx.sh"]
```

Simple build commands can be inlined into the `devloop.yaml`, but
in this example we have created a separate [`build.sh`](build.sh)
script.  This script uses `docker buildx` to containerize
source code for two platforms, `linux/amd64` and `linux/arm64`.

For more information on configuring a custom builder, see the Devloop custom
builder [documentation](https://devloop.dev/docs/how-tos/builders/#custom-build-script-run-locally).
Note that Devloop builders are different from `docker buildx` builders.

Note that Buildx does not support loading images for multiple platforms
o the Docker Daemon.  So this [`build.sh`](build.sh) only uses Buildx
when pushing an image to a registry.  See the _Cautions_ section below.


##### Step 3: Configure node affinities

Next look at the Kubernetes [pod descriptor](k8s/pod.yaml) and notice
the use of _node affinities_ to instruct Kubernetes to schedule the workload
on nodes running either `linux/arm64` or `linux/amd64`.  It is important
to realize that Kubernetes does not examine the referenced container images
to determine the possible platforms.


##### Step 4: Build and deploy the example

Now, use Devloop to deploy this application to your Kubernetes cluster:

```shell
$ devloop run --tail --default-repo <your repo>
```

With this command, Devloop will build the artifact with `docker buildx`
and deploy the application to Kubernetes.  You should be able to
see *Hello, World from <OS><ARCH>!* printed every second in the Devloop logs.

We need to use `--default-repo` to push to a repository as the
Docker Daemon does not support loading multi-platform images with
the same name.


#### Cleanup

To clean up your Kubernetes cluster, run:

```shell
$ devloop delete
```


##### &#x26A0; Caution &#x26A0;

Using `buildx` to build for multiple platforms has some subtle
interactions with Devloop's artifact caching.

Devloop normally caches the artifact after a successful container
image build.  Using the principle that given the same source inputs,
a build should produce the same container image, Devloop records
an association of the resulting image digest with a hash of the
artifact source.  This is the same principle used followed Docker
and other builders to cache image layers to speed up builds.

In this example, the build script configures Buildx differently
depending on the `$PUSH_IMAGE` flag. 

  - When `true`, the result is to be pushed to a registry, and all
    registries support multi-platform images. 
  - When `false`, the result is to be loaded to the Docker Daemon. 
    Buildx does not support loading multi-platform images to
    the Docker Daemon, and so this example builds a single container
    image for the local platform.

But Devloop is unaware that the build result differs based on `$PUSH_IMAGE`.
So on a local build (`$PUSH_IMAGE=false`), Devloop will cache the single-platform image,
and that single-platform image will be used for subsequent deployments _even when pushing
to a remote registry_ providing the source is unchanged.  To avoid
this scenario, disable Devloop's artifact caching when the result
is to be pushed to a remote registry:

```
devloop build --cache-artifacts=false
```

##### Considerations with `devloop dev`

When using `devloop dev` with a remote cluster, this example causes unnecessary
work as it builds and pushes for multiple platforms on each change.  You could
optimize for this case by using a Docker build through a [command-activated
profile](https://devloop.dev/docs/environment/profiles/#activation).
