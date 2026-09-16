### Example: Remote config µSvcs with Devloop

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/lucky-tools/devloop&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/remote-multi-config-microservices)

In this example:

* Deploy microservice applications from a remote git repository using Devloop.

Devloop can build and deploy from configurations defined in remote git repositories. In this example, we'll walk through using devloop to deploy two applications, an exposed "web" frontend which calls an unexposed "app" backend from the [examples/multi-config-microservices](../multi-config-microservices) project as a remote dependency.

**WARNING: If you're running this on a cloud cluster, this example will create a service and expose a webserver.
It's highly suggested that you only run this example on a local, private cluster like minikube or Kubernetes in Docker for Desktop.**

#### Running the example on minikube

From this directory, run:

```bash
devloop dev
```

Now, in a different terminal, hit the `leeroy-web` endpoint

```bash
$ curl localhost:9000
leeroooooy app!!
```
Hitting `Ctrl + C` on the first terminal should kill the process and clean up the deployments.

#### Configuration walkthrough

The [`devloop.yaml`](./devloop.yaml) looks like:

```yaml
apiVersion: devloop/v1
kind: Config
requires:
- git:
    repo: https://github.com/lucky-tools/devloop
    path: examples/multi-config-microservices/leeroy-app
    ref: main

- git:
    repo: https://github.com/lucky-tools/devloop
    path: examples/multi-config-microservices/leeroy-web
    ref: main

```

There are two `git` dependencies from the same repository `lucky-tools/devloop`. You can add as many dependencies as you want across the same or different repositories; even between different branches of the same repository. Devloop downloads each referenced repository (one copy per referenced branch) to its cache folder (`~/.devloop/remote-cache` by default).

The remote dependency caches should not be modified directly by the user. Devloop will reset the cache to the latest from the remote on each run.
