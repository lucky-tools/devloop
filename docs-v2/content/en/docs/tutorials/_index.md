---
title: "Tutorials"
linkTitle: "Tutorials"
weight: 90
simple_list: true
---

See the [Github Examples page](https://github.com/lucky-tools/devloop/tree/main/examples) for more examples.

{{< alert title="Deploying examples to a remote cluster" >}}
When deploying to a remote cluster you have to point Devloop to your default image repository in one of the four ways:

 1. flag: `devloop dev --default-repo <myrepo>`
 1. env var: `DEVLOOP_DEFAULT_REPO=<myrepo> devloop dev`
 1. global devloop config (one time): `devloop config set --global default-repo <myrepo>`
 1. devloop config for current kubectl context: `devloop config set default-repo <myrepo>`
{{< /alert >}}
