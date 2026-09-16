---
title: "Render [NEW]"
linkTitle: "Render [NEW]"
weight: 42
featureId: render
aliases: [/docs/how-tos/renderers, /docs/pipeline-stages/renderers/]
no_list: true
---

When Devloop renders your application to Kubernetes, it goes through the following process:

* the Devloop renderer _renders_ the final Kubernetes manifests: Devloop replaces untagged image names in the Kubernetes manifests with the final tagged image names.
It also might go through the extra intermediate step of expanding templates (for helm) or calculating overlays (for kustomize).

### Supported renderers

Devloop supports the following tools for rendering applications:

* [`rawYaml`]({{< relref "./rawYaml.md" >}}) - use this if you don't currently use a rendering tool
* [`helm`]({{< relref "./helm.md" >}})
* [`kpt`]({{< relref "./kpt.md" >}})
* [`kustomize`]({{< relref "./kustomize.md" >}})

Devloop's render configuration is set through the `manifests` section
of the `devloop.yaml`. See each renderer's page for more information
on how to configure them for use in Devloop. It's also possible to use
a combination of multiple renderers in a single project.

For a detailed discussion on Devloop configuration, see
[Devloop Concepts]({{< relref "/docs/design/config.md" >}}) and
[devloop.yaml References]({{< relref "/docs/references/yaml" >}}).
