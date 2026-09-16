---
title: "Log Tailing"
linkTitle: "Log Tailing"
weight: 44
featureId: logging
aliases: [docs/pipeline-stages/log-tailing]
---

Devloop has built-in support for tailing logs for containers **built and deployed by Devloop** on your cluster
to your local machine when running in either `dev`, `debug` or `run` mode.

{{< alert title="Note" >}}
Log Tailing is **enabled by default** for [`dev`]({{<relref "/docs/workflows/dev" >}}) and [`debug`]({{<relref "/docs/workflows/debug" >}}).<br>
Log Tailing is **disabled by default** for `run` mode; it can be enabled with the `--tail` flag.
{{< /alert >}}


## Log Structure
To view log structure, run `devloop run --tail` in [`examples/microservices`](https://github.com/lucky-tools/devloop/tree/main/examples/microservices)

```bash
devloop run --tail
```

will produce an output like this

![logging-output](/images/logging-output.png)


For every log line, devloop will prefix the pod name and container name if they're not the same.

![logging-output](/images/log-line-single.png)

In the above example, `leeroy-web-75ff54dc77-9shwm` is the pod name and `leeroy-web` is container name
defined in the spec for this deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: leeroy-web
  labels:
    app: leeroy-web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-web
  template:
    metadata:
      labels:
        app: leeroy-web
    spec:
      containers:
        - name: leeroy-web
          image: gcr.io/k8s-devloop/leeroy-web
          ports:
            - containerPort: 8080 
```

Devloop will choose a unique color for each container to make it easy for users to read the logs.

## JSON Parsing
In some cases, logs may simply be JSON objects.
If you know this ahead of time and know that you'd like to only get specific fields from these objects,
you can add a `deploy.logs.jsonParse` stanza to your `devloop.yaml` file to configure which fields you'd like to see.

```yaml
apiVersion: devloop/v1
kind: Config
build:
  artifacts:
  - image: devloop-example
deploy:
  logs:
    jsonParse:
      fields: ["message", "severity"]
  kubectl:
    manifests:
      - k8s-*
```
In the above example, only the fields `message` and `severity` will be gathered from the incoming JSON logs.
So, if the logs coming through were structured like so:
```
[getting-started] {"timestampSeconds":1643740871,"timestampNanos":446000000,"severity":"INFO","thread":"main","message":"Hello World!","context":"default"}
```
with the `deploy.logs.jsonParse` config added, they would look like this:
```
[getting-started] message: Hello World!, severity: INFO
```
