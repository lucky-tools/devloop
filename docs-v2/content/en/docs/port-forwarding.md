---
title: "Port Forwarding"
linkTitle: "Port Forwarding"
weight: 46
featureId: portforward
aliases: [/docs/how-tos/portforward, /docs/pipeline-stages/port-forwarding]
---

Devloop has built-in support for forwarding ports from exposed Kubernetes resources on your cluster
to your local machine when running in `dev`, `debug`, `deploy`, or `run` modes.

### Automatic Port Forwarding

Devloop supports automatic port forwarding the following classes of resources:

- `user`: explicit port-forwards defined in the `devloop.yaml` (called [_user-defined port forwards_](#UDPF))
- `services`: ports exposed on services deployed by Devloop.
- `debug`: debugging ports as enabled by `devloop debug` for Devloop-built images.
- `pods`: all `containerPort`s on deployed pods for Devloop-built images.

Devloop enables certain classes of forwards by default depending on the Devloop command used.
These defaults can be overridden with the `--port-forward` flag, and port-forwarding can be
disabled with `--port-forward=off`.

Command-line                          | Default modes
------------------------------------- | -------------------
`devloop dev`                        | `user`
`devloop dev --port-forward`         | `user`, `services`
`devloop dev --port-forward=off`     | _no ports forwarded_
`devloop debug`                      | `user`, `debug`
`devloop debug --port-forward`       | `user`, `services`, `debug` <small>(<em>see note below</em>)</small>
`devloop debug --port-forward=off`   | _no ports forwarded_
`devloop deploy`                     | `off`
`devloop deploy --port-forward`      | `user`, `services`
`devloop run`                        | `off`
`devloop run --port-forward`         | `user`, `services`

{{< alert title="Compatibility Note" >}}
Note that `devloop debug --port-forward` previously enabled the
equivalent of `pods` as Devloop did not have an equivalent of `debug`. 
We have replaced `pods` as it caused confusion.
{{< /alert >}}

### User-Defined Port Forwarding {#UDPF}

Users can define additional resources to port forward in the devloop config, to enable port forwarding for 

* additional resource types supported by `kubectl port-forward` e.g.`Deployment`or `ReplicaSet`.
* additional pods running containers which run images not built by Devloop.

For example:

```yaml
portForward:
- resourceType: deployment
  resourceName: myDep
  namespace: mynamespace
  port: 8080
  localPort: 9000 # *Optional*
```

For this example, Devloop will attempt to forward port 8080 to `localhost:9000`.
If port 9000 is unavailable, Devloop will forward to a random open port. 

{{< alert title="Note about forwarding System Ports" >}}
Devloop will request matching local ports only when the remote port is `> 1023`. So a service on port `8080` would still map to port `8080` (if available), but a service on port `80` will be mapped to some port `≥ 1024`.

User-defined port-forwards in the `devloop.yaml` are unaffected and can bind to system ports.
{{< /alert >}}

{{< alert title="Note about user-defined port-forwarding for Docker deployments" >}}
When [deploying to Docker]({{< relref "/docs/deployers/docker" >}}) with a user-defined port-forward in the `devloop.yaml`, the `resourceType` of `portForward` must be set to `container`. Otherwise, Devloop will not tell the Docker daemon to expose that port.
{{< /alert >}}

Devloop will run `kubectl port-forward` on each of these resources in addition to the automatic port forwarding described above.
Acceptable resource types include: `Service`, `Pod` and Controller resource type that has a pod spec: `ReplicaSet`, `ReplicationController`, `Deployment`, `StatefulSet`, `DaemonSet`, `Job`, `CronJob`. 


| Field        | Values           | Mandatory  |
| ------------- |-------------| -----|
| resourceType     | `pod`, `service`, `deployment`, `replicaset`, `statefulset`, `replicationcontroller`, `daemonset`, `job`, `cronjob`, `container` | Yes | 
| resourceName     | Name of the resource to forward.     | Yes | 
| namespace  | The namespace of the resource to port forward.     | No. Defaults to current namespace, or `default` if no current namespace is defined | 
| port | Port is the resource port that will be forwarded. | Yes |
| address | Address is the address on which the forward will be bound. | No. Defaults to `127.0.0.1` |
| localPort | LocalPort is the local port to forward too. | No. Defaults to value set for `port`. |


Devloop will run `kubectl port-forward` on all user defined resources.
`kubectl port-forward` will select one pod created by that resource to forward too.

For example, forwarding a deployment that creates 3 replicas could look like this:

```yaml
portForward:
- resourceType: deployment
  resourceName: myDep
  namespace: mynamespace
  port: 8080
  localPort: 9000
```

![portforward_deployment](/images/portforward.png)

If you want the port forward to to be available from other hosts and not from the local host only, you can bind
the port forward to the address `0.0.0.0`:

```yaml
portForward:
- resourceType: deployment
  resourceName: myDep
  namespace: mynamespace
  port: 8080
  address: 0.0.0.0
  localPort: 9000
```
