---
title: "Deploy [UPDATED]"
linkTitle: "Deploy [UPDATED]"
weight: 42
featureId: deploy
aliases: [/docs/how-tos/deployers, /docs/pipeline-stages/deployers]
no_list: true
---

When Devloop deploys your application to Kubernetes, it goes through these steps:

In the default case (no manifest provided using the kubectl or kpt deployer ), devloop deploy will do the following:
* the Devloop renderer _renders_ the final Kubernetes manifests: Devloop replaces untagged image names in the Kubernetes manifests with the final tagged image names.
It also might go through the extra intermediate step of expanding templates (for helm) or calculating overlays (for kustomize).  Additionally some deployers (docker) do not render manifests as such don't use this phase.
* the Devloop deployer _deploys_ the final Kubernetes manifests to the cluster (or to local docker for the docker deployer)
* the Devloop deployer performs [status checks]({{< relref "/docs/status-check" >}}) and waits for the deployed resources to stabilize.
### Supported deployers

Devloop supports the following tools for deploying applications:

* [`kubectl`]({{< relref "./kubectl.md" >}})
* [`kpt`]({{< relref "./kpt.md" >}})
* [`docker`]({{< relref "./docker.md" >}}) (does not deploy to Kubernetes: see documentation for more details)

Devloop's deploy configuration is set through the `deploy` section
of the `devloop.yaml`. See each deployer's page for more information
on how to configure them for use in Devloop. It's also possible to use
a combination of multiple deployers in a single project.

For a detailed discussion on Devloop configuration, see
[Devloop Concepts]({{< relref "/docs/design/config.md" >}}) and
[devloop.yaml References]({{< relref "/docs/references/yaml" >}}).