### Example: deploy multiple releases with Helm

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/lucky-tools/devloop&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/helm-deployment)

You can deploy multiple releases with devloop, each will need a chartPath, a values file, and namespace.
Devloop can inject intermediate build tags in the the values map in the devloop.yaml.

Let's walk through the devloop yaml:

We'll be building an image called `devloop-helm`, and it's a dockerfile, so we'll add it to the artifacts.

```yaml
build:
  artifacts:
  - image: devloop-helm
```

Now, we want to deploy this image with helm.
We add a new release in the helm part of the manifests stanza.

```yaml
manifests:
  helm:
    releases:
    - name: devloop-helm
      chartPath: charts
      # namespace: devloop
      setValues:
        image: devloop-helm
      valuesFiles:
      - values.yaml
```

This part tells Devloop to set the `image` parameter of the values file to the built `devloop-helm` image and tag.

```yaml
      setValues:
        image: devloop-helm
```
