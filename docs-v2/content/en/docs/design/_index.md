---
title: "Architecture and Design"
linkTitle: "Architecture and Design"
weight: 50
aliases: [/docs/concepts,/docs/concepts/architecture]
no_list: true
---

Devloop is designed with pluggability in mind:

![architecture](/images/architecture.png)

The architecture allows you to use Devloop with the tool you prefer. Devloop
provides built-in support for the following tools:

* **Build**
  * Dockerfile locally, in-cluster with kaniko
  * Jib Maven and Jib Gradle locally
  * Cloud Native Buildpacks locally
  * Custom script locally or in-cluster
* **Test**
  * [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test)
* **Tag**
  * Git tagger
  * Sha256 tagger
  * Input Digest tagger
  * Env Template tagger
  * DateTime tagger
* **Deploy**
  * Kubernetes Command-Line Interface (`kubectl`)
  * [Helm](https://helm.sh/)
  * [kustomize](https://github.com/kubernetes-sigs/kustomize)

You can combine the tools as you see fit in Devloop. For experimental
projects, you may want to use local Docker daemon for building artifacts, and
deploy them to a Minikube local Kubernetes cluster with `kubectl`:

![workflow_local](/images/workflow_local.png)


Devloop also supports development profiles. You can specify multiple different
profiles in your configuration and use the one that best serves your needs
without having to modify the configuration file. You can learn more about
profiles [here]({{< relref "../environment/profiles.md" >}}).
