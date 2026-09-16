# Examples

Each of those examples can be tried with `devloop dev`. For example:

```
cd getting-started
devloop dev
```

Read the [Quickstart](https://devloop.dev/docs/quickstart/) for more detailed instructions.

These examples are made to work with the latest release of Devloop.

If you are running Devloop at HEAD or have built it from source, please use the examples at `integration/examples`.

*Note for contributors*: If you wish to make changes to these examples, please edit the ones at `integration/examples`,
as those will be synced on release.

## Deploying to a local cluster

When deploying to a [local cluster](https://devloop.dev/docs/environment/local-cluster/) such as minikube or Docker Desktop, no additional configuration step is required.

## Deploying to a remote cluster

When deploying to a remote cluster you have to point Devloop to your default image repository in one of the five ways:

* flag: `devloop dev --default-repo <myrepo>`
* env var: `DEVLOOP_DEFAULT_REPO=<myrepo> devloop dev`
* env var file: create a file `devloop.env` in the project root, with the line `DEVLOOP_DEFAULT_REPO=<myrepo>`
* global devloop config (one time): `devloop config set --global default-repo <myrepo>`
* devloop config for current kubectl context: `devloop config set default-repo <myrepo>`

