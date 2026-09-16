---
title: "Devloop Pipeline Stages"
linkTitle: "Devloop Pipeline Stages"
weight: 40
aliases: [/docs/concepts/pipeline]
no_list: true
---

Devloop features a multi-stage workflow:

![workflow](/images/workflow.png)

When you start Devloop, it collects source code in your project and builds
artifacts with the tool of your choice; the artifacts, once successfully built,
are tagged as you see fit and pushed to the repository you specify. In the
end of the workflow, Devloop also helps you deploy the artifacts to your
Kubernetes cluster, once again using the tools you prefer.

Devloop allows you to skip stages. If, for example, you run Kubernetes
locally with [Minikube](https://kubernetes.io/docs/setup/minikube/), Devloop
will not push artifacts to a remote repository.


| Devloop Pipeline stages|Description| 
|----------|-------|------|
| [Init]({{< relref "/docs/init" >}}) | generate a starting point for Devloop configuration | 
| [Build]({{< relref "/docs/builders" >}}) | build images with different builders | 
| [Render]({{< relref "/docs/renderers" >}}) | render manifests with different renderers | 
| [Tag]({{< relref "/docs/taggers" >}}) | tag images based on different policies |
| [Test]({{< relref "/docs/testers" >}}) | run tests with testers |
| [Deploy]({{< relref "/docs/deployers" >}}) |  deploy with kubectl, kustomize or helm |
| [Verify]({{< relref "/docs/verify" >}}) |  verify deployments with specified test containers |
| [File Sync]({{< relref "/docs/filesync" >}}) |  sync changed files directly to containers |
| [Log Tailing]({{< relref "/docs/log-tailing" >}}) |  tail logs from workloads |
| [Port Forwarding]({{< relref "/docs/port-forwarding" >}}) | forward ports from services and arbitrary resources to localhost  |
| [Deploy Status Checking]({{< relref "/docs/status-check" >}}) | wait for deployed resources to stabilize  |
| [Lifecycle Hooks]({{< relref "/docs/lifecycle-hooks" >}}) | run code triggered by different events during the devloop process lifecycle  |
| [Cleanup]({{< relref "/docs/cleanup" >}}) | cleanup manifests and images |
