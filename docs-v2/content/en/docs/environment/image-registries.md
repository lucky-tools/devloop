---
title: "Image Repository Handling"
linkTitle: "Image Repository Handling"
weight: 70
featureId: default_repo
aliases: [/docs/concepts/image_repositories]
---

Often, a Kubernetes manifest (or `devloop.yaml`) makes references to images that push to
registries that we might not have access to. Modifying these individual image names manually
is tedious, so Devloop supports automatically prefixing these image names with a registry
specified by the user. Using this, any project configured with Devloop can be run by any user
with minimal configuration, and no manual YAML editing!

This is accomplished through the `default-repo` functionality, and can be used one of three ways:

1. `--default-repo` flag

    ```bash
    devloop dev --default-repo <myrepo>
    ```

1. `DEVLOOP_DEFAULT_REPO` environment variable

    ```bash
    DEVLOOP_DEFAULT_REPO=<myrepo> devloop dev
    ```

1. Devloop's global config

    ```bash
    devloop config set default-repo <myrepo>
    ```

If no `default-repo` is provided by the user, there is no automated image name rewriting, and Devloop will
try to push the image as provided in the yaml.

The image name rewriting strategies are designed to be *conflict-free*:
the full image name is rewritten on top of the default-repo so similar image names don't collide in the base namespace (e.g.: repo1/example and repo2/example would collide in the target_namespace/example without this)

Automated image name rewriting strategies are determined based on the default-repo and the original image repository:

* default-repo domain does not contain `gcr.io` or `-docker.pkg.dev`
  * **strategy**: 		escape & concat & truncate to 256

    ```
     original image: 	gcr.io/k8s-devloop/devloop-example1
     default-repo:      aws_account_id.dkr.ecr.region.amazonaws.com
     rewritten image:   aws_account_id.dkr.ecr.region.amazonaws.com/gcr_io_k8s-devloop_devloop-example1
    ```

* default-repo contain `gcr.io` or `-docker.pkg.dev` (special cases - as GCR and AR allow for arbitrarily deep directory structure in image repo names)
  * **strategy**: concat unless prefix matches
  * **example1**: prefix doesn't match:

    ```
      original image: 	gcr.io/k8s-devloop/devloop-example1
      default-repo: 	gcr.io/myproject/myimage
      rewritten image:  gcr.io/myproject/myimage/gcr.io/k8s-devloop/devloop-example1
    ```
  * **example2**: prefix matches:

    ```
      original image: 	gcr.io/k8s-devloop/devloop-example1
      default-repo: 	gcr.io/k8s-devloop
      rewritten image:  gcr.io/k8s-devloop/devloop-example1
    ```
  * **example3**: shared prefix:

    ```
      original image: 	gcr.io/k8s-devloop/devloop-example1
      default-repo: 	gcr.io/k8s-devloop/myimage
      rewritten image:  gcr.io/k8s-devloop/myimage/devloop-example1
    ```

## Insecure image registries

During development you may be forced to push images to a registry that does not support HTTPS.
By itself, Devloop will never try to downgrade a connection to a registry to plain HTTP.
In order to access insecure registries, this has to be explicitly configured per registry name.

There are several levels of granularity to allow insecure communication with some registry:

1. Per Devloop run via the repeatable `--insecure-registry` flag

    ```bash
    devloop dev --insecure-registry insecure1.io --insecure-registry insecure2.io
    ```

1. Per Devloop run via `DEVLOOP_INSECURE_REGISTRY` environment variable

    ```bash
    DEVLOOP_INSECURE_REGISTRY='insecure1.io,insecure2.io' devloop dev
    ```

1. Per project via the Devloop pipeline config `devloop.yaml`

    ```yaml
    build:
        insecureRegistries:
        - insecure1.io
        - insecure2.io
    ```

1. Per user via Devloop's global config

    ```bash
    devloop config set insecure-registries insecure1.io           # for the current kube-context
    devloop config set --global insecure-registries insecure2.io  # for any kube-context
    ```

    Note that multiple set commands _add_ to the existing list of insecure registries.
    To clear the list, run `devloop config unset insecure-registries`.

Devloop will join the lists of insecure registries, if configured via multiple sources.
