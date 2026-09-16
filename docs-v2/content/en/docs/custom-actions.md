---
title: "Custom Actions"
linkTitle: "Custom Actions"
weight: 48
featureId: customactions
---

With Devloop you can define generic actions in a declarative way using a Devloop config file (`devloop.yaml`), and execute them with the `devloop exec <action-name>` command.

A generic action (a.k.a. Custom Action), defines a list of containers that will be executed in parallel when the action is invoked. A Custom Action execution is considered successful if all its containers end without errors, and considered as failed if one or more of its containers report an error.

{{< alert title="Note">}}
Custom Actions are meant to run **specific, completable tasks**; are not meant to be use to execute your main application.
{{< /alert >}}

## Defining Custom Actions

A Devloop config file can define one or more Custom Actions using the [`customActions` stanza]({{< relref "/docs/references/yaml#customActions" >}}). Each action and container must be identified by an unique name across modules/configurations. **NOTE:** Two different [profiles]({{< relref "docs/environment/profiles" >}}) can have an action each, with the same name.

Therefore, a configuration for a Custom Action named `update-infra`, with two containers, `update-db-schema` and `setup-external-proxy`, will be defined like this in a `devloop.yaml` file:

{{% readfile file="samples/custom-actions/two-local-actions.yaml" %}}

Running `devloop exec update-infra` with the previous configuration will trigger the execution of the `update-infra` action, as a [local (Docker) action]({{< relref "#local-docker" >}}), creating and running a container for `update-db-schema` (using the `gcr.io/my-registry/db-updater:latest` image), and `setup-external-proxy` (using the `gcr.io/my-registry/proxy:latest` image). The output will look like this:

```console
$ devloop exec update-infra
Starting execution for update-infra
...
[setup-external-proxy] updating proxy version...
[setup-external-proxy] copying proxy rules...   
[setup-external-proxy] starting proxy...
[update-db-schema] starting db update...
[update-db-schema] db schema update completed
[setup-external-proxy] proxy configured
```

To check the list of available options to configure an action please refer to the [`customActions` stanza documentation]({{< relref "/docs/references/yaml#customActions" >}}).

## Executing Custom Actions

The `devloop exec <action-name>` command executes a defined Custom Action. During execution, Devloop streams the logs from the containers associated with the action. Upon completion, Devloop returns an exit code: `0` for success or `1` for failure. To see the available options for the `devloop exec` command, refer to the [CLI documentation]({{< relref "/docs/references/cli/#devloop-exec" >}}).

### Timeouts

Per default, a Custom Action does not have a timeout configured, which means, the action will run until it completes (success or fail). Using the[ `customActions[].timeout`]({{< relref "/docs/references/yaml/#customActions-timeout" >}}) property you can change the previous behaviour, adding a desired timeout in seconds:

{{% readfile file="samples/custom-actions/two-local-actions-timeout.yaml" %}}

Running `devloop exec update-infra` with the previous configuration will fail if the Custom Action takes more than 10 seconds to complete. If the timeout is triggered, Devloop will stop any running container and will return a status code `1`:

```console
$ devloop exec update-infra
tarting execution for update-infra
...
[setup-external-proxy] updating proxy version...
[setup-external-proxy] copying proxy rules...
[setup-external-proxy] starting proxy...
[update-db-schema] starting db update...
context deadline exceeded
```

Devloop will return status code `0` if all the containers associated with the given action finish their execution before the 10 seconds timeout.

### Fail strategy

A Custom Action will be run with a `fail-fast` strategy, which means, if one container associated with the action fails, Devloop will stop any running container, and will return a status code `1`:

The following `devloop.yaml` config:

{{% readfile file="samples/custom-actions/two-local-actions.yaml" %}}

With an error in the `update-db-schema` container, will produce the following output:

```console
$ devloop exec update-infra
Starting execution for update-infra
...
[setup-external-proxy] updating proxy version...
[setup-external-proxy] copying proxy rules...
[setup-external-proxy] starting proxy...
[update-db-schema] starting db update...
"update-db-schema" running container image "gcr.io/my-registry/db-updater:latest" errored during run with status code: 1
```

The previous default behaviour can be change with the [`customActions[].failFast` property]({{< relref "/docs/references/yaml/#customActions-failFast" >}}), changing its value to `false`:

{{% readfile file="samples/custom-actions/local-action-fail-safe.yaml" %}}

The previous configuration indicates Devloop to run the `update-infra` action with a `fail-safe` strategy, which means, Devloop will not interrupt any container if one or more of them fail; all the containers will run until they finish (success or fail):

```console
Starting execution for update-infra
...
[setup-external-proxy] updating proxy version...
[setup-external-proxy] copying proxy rules...
[setup-external-proxy] starting proxy...
[update-db-schema] starting db update...
[setup-external-proxy] proxy configured
1 error(s) occurred:
* "update-db-schema" running container image "gcr.io/my-registry/db-updater:latest" errored during run with status code: 1
```

### Execution modes

A Custom Action has an execution mode associated with it that indicates Devloop in which environment and how the containers of that action should be created and executed. This execution mode can be configured with the [`customActions[].executionMode` property]({{< relref "/docs/references/yaml/#customActions-executionMode" >}}). These are the available execution modes for a Custom Action:

#### Local (Docker) - default {#local-docker}

This is the default configuration when no [`customActions[].executionMode`]({{< relref "/docs/references/yaml/#customActions-executionMode" >}}) is specified. With this execution mode, Devloop will run every container associated to a given Custom Action with a Docker daemon.

#### Remote (K8s job)

With this execution mode, Devloop will create a K8s job for each container associated with the given action. For the following configuration:

{{% readfile file="samples/custom-actions/k8s-action.yaml" %}}

Devloop will create one K8s job for `update-db-schema` and another for `setup-external-proxy`. The jobs will use the following template per default:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: # <- Container name defined in devloop.yaml.
spec:
  template:
    spec:
      containers: # <- Only one container, the one defined in the devloop.yaml.
      # ...
      restartPolicy: Never
  backoffLimit: 0
```

The template can be extended using the [`customActions[].executionMode.kubernetesCluster.overrides`]({{< relref "/docs/references/yaml/#customActions-executionMode-kubernetesCluster-overrides" >}}) and [`customActions[].executionMode.kubernetesCluster.jobManifestPath`]({{< relref "/docs/references/yaml/#customActions-executionMode-kubernetesCluster-jobManifestPath" >}}) properties.

## Devloop build + exec

Custom Actions can be used together with [Devloop build]({{< relref "docs/builders/" >}}) so the Custom Actions can use images build by Devloop. 

Using the following `devloop.yaml` file:

{{% readfile file="samples/custom-actions/actions-local-build.yaml" %}}

We trigger an Devloop build using the `devloop build` command:

```console
$ devloop build --file-output=build.json
```

Devloop will create a new `build.json` file with the necessary info. Then, using the generated file, we can run `devloop exec`:

```console
$ devloop exec update-infra --build-artifacts=build.json
```

That way, Devloop will be able to run the `local-db-updater` image in the `update-infra` Custom Action.

