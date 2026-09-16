---
title: "Kpt [UPDATED]"
linkTitle: "Kpt [UPDATED]"
weight: 30
featureId: deploy
aliases: [/docs/pipeline-stages/deployers/kpt]
---

## Rendering with kpt

[`kpt`](https://kpt.dev/) allows Kubernetes
developers to customize raw, template-free YAML files for multiple purposes.
Devloop can work with `kpt` by calling its command-line interface.

### Configuration

To use kpt with Devloop, add deploy type `kpt` to the `deploy`
section of `devloop.yaml`.

The `kpt` type offers the following options:

{{< schema root="KptDeploy" >}}

Each entry in `paths` should point to a folder with a kustomization file.

`flags` section offers the following options:

{{< schema root="KubectlFlags" >}}

### Example

The following `deploy` section instructs Devloop to deploy
artifacts using kpt:

{{% readfile file="samples/deployers/kpt.yaml" %}}

{{< alert title="Note" >}}
kpt CLI must be installed on your machine. Devloop will not
install it.
{{< /alert >}}
