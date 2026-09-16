### Example: use image values in templated fields for build and deploy

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/lucky-tools/devloop&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/templated-fields)

This example shows how `IMAGE_REPO` and `IMAGE_TAG` keywords are available in templated fields for custom build and helm deploy

* **building** a single Go file app with ko
* **tagging** using the default tagPolicy (`gitCommit`)
* **deploying** two replica container pods using `helm`

#### Before you begin

For this tutorial to work, you will need to have Devloop, Helm and a Kubernetes cluster set up.
To learn more about how to set up, see the [getting started docs](https://devloop.dev/docs/getting-started).

#### Tutorial

This tutorial will demonstrate how Devloop can inject image repo and image tag values into the build and deploy stanzas.

First, clone the Devloop [repo](https://github.com/lucky-tools/devloop) and navigate to the [templated-fields example](https://github.com/lucky-tools/devloop/tree/main/examples/templated-fields) for sample code:

```sh
git clone https://github.com/lucky-tools/devloop
```
```sh
cd devloop/examples/templated-fields
```

`IMAGE_REPO` and `IMAGE_TAG` are available as templated fields in `build.sh` file:

[embedmd]:# (build.sh bash /^img=/ /$/)
```bash
img="${IMAGE_REPO}:${IMAGE_TAG}"
```

and also in the `helm` deploy section of the devloop config, which configures artifact `devloop-templated` to build with `build.sh`:

[embedmd]:# (devloop.yaml yaml /^.*setValueTemplates:/ /imageTag: .*$/)
```yaml
      setValueTemplates:
        imageRepo: "{{.IMAGE_REPO}}"
        imageTag: "{{.IMAGE_TAG}}"
```

These values are then being set as container environment variables `FOO_IMAGE_REPO` and `FOO_IMAGE_TAG` in the helm template `deployment.yaml` file, just as an example to show how they can be added to your helm templates.

[embedmd]:# (charts/templates/deployment.yaml yaml /^.*containers:/ $)
```yaml
      containers:
      - name: {{ .Chart.Name }}
        image: {{ .Values.image }}
        env:
          - name: FOO_IMAGE_REPO
            value: {{ .Values.imageRepo }}
          - name: FOO_IMAGE_TAG
            value: {{ .Values.imageTag }}
```

For more information about how this works, see the Devloop [custom builder](https://devloop.dev/docs/how-tos/builders/#custom-build-script-run-locally) and [helm](https://devloop.dev/docs/pipeline-stages/deployers/helm/) documentation.

Now, use Devloop to deploy this application to your Kubernetes cluster:

```sh
devloop run --tail --default-repo <your repo>
```

With this command, Devloop will build the `devloop-templated` artifact with ko and deploy the application to Kubernetes using helm.
You should be able to see something like:

```terminal
Running image devloop-templated:a866d5efd634062ea74662b20e172cd6e2d645f9f33f929bfaf8e856ec66bd94
```

 printed every second in the Devloop logs, since the code being executed is `main.go`.

[embedmd]:# (main.go go /fmt\.Printf/ /$/)
```go
fmt.Printf("Running image %v:%v\n", os.Getenv("FOO_IMAGE_REPO"), os.Getenv("FOO_IMAGE_TAG"))
```

#### Cleanup

To clean up your Kubernetes cluster, run:

```sh
devloop delete
```
