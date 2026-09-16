---
title: "Getting Started With Your Project"
linkTitle: "Getting Started With Your Project"
weight: 10
no_list: true
---

Devloop requires a `devloop.yaml`, but - for supported projects - Devloop can
generate a simple config for you that you can get started with. To configure
Devloop for your application you can run [`devloop init`][init].

Running [`devloop init`][init] at the root of your project directory will walk you
through a wizard and create a `devloop.yaml` that defines how your project is
built and deployed.

[init]: {{<relref "docs/init" >}}

```bash
devloop init
```

![init-flow](/images/init-flow.png)

## What's next
You can further set up [File Sync]({{<relref "/docs/filesync" >}}) for source files 
that do not need a rebuild in [dev mode]({{<relref "/docs/workflows/dev">}}). 

Devloop automatically forwards Kubernetes Services in [dev mode]({{<relref "/docs/workflows/dev">}}) if you run it with `--port-forward`. If your project contains resources other than services, you can set-up [port-forwarding]({{<relref "/docs/port-forwarding" >}})
to port-forward these resources in [`dev`]({{<relref "docs/workflows/dev" >}}) or [`debug`]({{<relref "/docs/workflows/debug" >}}) mode.


For more understanding on how init works, see [`devloop init`]({{<relref "/docs/init" >}}).
