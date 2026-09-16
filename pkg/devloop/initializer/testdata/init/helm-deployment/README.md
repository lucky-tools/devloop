### Example: deploy multiple releases with Helm

You can deploy multiple releases with Devloop, each will need a chartPath, a values file, and an optional namespace.
Devloop can inject intermediate build tags in the the values map in the `devloop.yaml`.

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
      valuesFiles:
      - values.yaml
```
