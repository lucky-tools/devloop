---
title: "Build"
linkTitle: "Build"
weight: 42
featureId: build
aliases: [/docs/how-tos/builders, docs/pipeline-stages/builders]
no_list: true
---

Devloop supports different [tools]({{< relref "/docs/builders/builder-types" >}}) for building images across different [build environments]({{< relref "/docs/builders/build-environments" >}}).

|    | [Local Build]({{< relref "/docs/builders/build-environments/local" >}}) | [In Cluster Build]({{< relref "/docs/builders/build-environments/in-cluster" >}}) |
|----|:-----------:|:----------------:|
| **Dockerfile** | [Yes]({{< relref "/docs/builders/builder-types/docker#dockerfile-locally" >}}) | [Yes]({{< relref "/docs/builders/builder-types/docker#dockerfile-in-cluster-with-kaniko" >}}) |
| **Jib Maven and Gradle** | [Yes]({{< relref "/docs/builders/builder-types/jib#jib-maven-and-gradle-locally" >}}) | - |
| **Cloud Native Buildpacks** | [Yes]({{< relref "/docs/builders/builder-types/buildpacks" >}}) | - |
| **ko** | [Yes]({{< relref "/docs/builders/builder-types/ko" >}}) | - |
| **Custom Script** | [Yes]({{<relref "/docs/builders/builder-types/custom#custom-build-script-locally" >}}) | [Yes]({{<relref "/docs/builders/builder-types/custom#custom-build-script-in-cluster" >}}) |

## Configuration

The `build` section in the Devloop configuration file, `devloop.yaml`,
controls how artifacts are built. To use a specific tool for building
artifacts, add the value representing the tool and options for using that tool
to the `build` section.

For detailed per-builder [Devloop Configuration]({{< relref "/docs/design/config.md" >}}) options,
see [devloop.yaml References]({{< relref "/docs/references/yaml" >}}).
