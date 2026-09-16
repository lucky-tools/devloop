---
title: "Devloop Pipeline"
linkTitle: "Devloop Pipeline"
weight: 40
aliases: [/docs/concepts/config]
---

You can configure Devloop with the Devloop configuration file,
`devloop.yaml`.  A single configuration consists of several different components:

| Component  | Description |
| ---------- | ------------|
| `apiVersion` | The Devloop API version you would like to use. The current API version is {{< devloop-version >}}. |
| `kind`  |  The Devloop configuration file has the kind `Config`.  |
| `metadata`  |  Holds additional properties like the `name` of this configuration.  |
| `build`  |  Specifies how Devloop builds artifacts. You have control over what tool Devloop can use, how Devloop tags artifacts and how Devloop pushes artifacts. Devloop supports using the local Docker daemon or Kaniko to build artifacts. See [Builders](/docs/builders) and [Taggers]({{< relref "/docs/taggers" >}}) for more information. |
| `test` |  Specifies how Devloop tests artifacts. Devloop supports [container-structure-tests](https://github.com/GoogleContainerTools/container-structure-test) to test built artifacts and custom tests to run custom commands as part of the development pipeline. See [Testers]({{< relref "/docs/testers" >}}) for more information. |
| `deploy` |  Specifies how Devloop deploys artifacts. Devloop supports using `kubectl`, `helm`, or `kustomize` to deploy artifacts. See [Deployers]({{< relref "/docs/deployers" >}}) for more information. |
| `profiles`|  Profile is a set of settings that, when activated, overrides the current configuration. You can use Profile to override the `build`, `test` and `deploy` sections. |
| `requires`|  Specifies a list of other devloop configurations to import into the current config |

You can [learn more]({{< relref "/docs/references/yaml" >}}) about the syntax of `devloop.yaml`.

Devloop normally expects to find the configuration file as
`devloop.yaml` in the current directory, but the location can be
overridden with the `--filename` flag.

### File resolution

The Devloop configuration file often references other files and
directories.  These files and directories are resolved
relative to the current directory _and not to the location of
the Devloop configuration file_.  There are two important exceptions:
1. Files referenced from a build artifact definition are resolved relative to the build artifact's _context_ directory.
   When omitted, the context directory defaults to the current directory.
2. For [configurations resolved as dependencies](#configuration-dependencies"), paths are always resolved relative to the directory containing the imported configuration file.

For example, consider a project with the following layout:
```
.
├── frontend
│   └── Dockerfile
├── helm
│   └── project
│       └── dev-values.yaml
└── devloop.yaml
```

The config file might look like:
```yaml
apiVersion: devloop/v1
kind: Config
build:
  artifacts:
  - image: app
    context: frontend
    docker:
      dockerfile: "Dockerfile"
deploy:
  helm:
    releases:
    - name: project
      chartPath: helm/project
      valuesFiles:
      - "helm/project/dev-values.yaml"
```

In this example, the `Dockerfile` for building `app`
is resolved relative to `app`'s context directory,
whereas the the Helm chart's location and its values-files are
relative to the current directory in `helm/project`.

We generally recommend placing the configuration file in the root directory of the Devloop project.

## Multiple configuration support

A single `devloop.yaml` file can define multiple devloop configurations in the schema described above using the separator `---`. If these configuration objects define the `metadata.name` property then we consider them as `modules`, that can then be activated by name.

Consider a `devloop.yaml` defined as:

```yaml
apiVersion: devloop/v1
kind: Config
metadata:
  name: cfg1
build:
  # build definition
deploy:
  # deploy definition

---

apiVersion: devloop/v1
kind: Config
metadata:
  name: cfg2
build:
  # build definition
deploy:
  # deploy definition
```

Here `cfg1` and `cfg2` are independent devloop modules. Running `devloop dev` for instance will execute actions from both these modules. You could also run `devloop dev --module cfg1` to only activate the `cfg1` module and skip `cfg2`.

## Configuration dependencies

In addition to authoring configurations in a `devloop.yaml` file, we can also import other existing configurations as dependencies. Devloop manages all imported and defined configurations in the same session. It also ensures all artifacts in a required config are built prior to those in current config (provided the artifacts have dependencies defined); and all deploys in required configs are applied prior to those in current config.

{{< alert title="Note:" >}}
Running `devloop <command> --module <config-name>` will filter to the specified target module, but also include the transitive closure of all other configurations in its dependency graph. For instance, if a module `cfg1` imported another module `cfg2` as a dependency while `cfg2` imported `cfg3` and `cfg4`, then running `devloop dev --module cfg1` would activate all of `cfg1`, `cfg2`, `cfg3` and `cfg4` and execute them in dependency order.
{{< /alert >}}

### Local config dependency

Consider the same `devloop.yaml` defined above. Modules `cfg1` and `cfg2` from the above file can be imported as dependencies in your current config definition, via:

```yaml
apiVersion: devloop/v1
kind: Config
requires:
  - configs: ["cfg1", "cfg2"]
    path: path/to/other/devloop.yaml 
build:
  # build definition
deploy:
  # deploy definition
```

If the `configs` list isn't defined then it imports all the configs defined in the file pointed by `path`. Additionally, if the `path` to the configuration isn't defined it assumes that all the required configs are defined in the same file as the current config.

{{< alert title="Note:" >}}
In imported configurations, files are resolved relative to the location of imported Devloop configuration file.
{{< /alert >}}

### Remote config dependency

The required devloop config can live in a remote git repository or in Google Cloud Storage:

```yaml
apiVersion: devloop/v1
kind: Config
requires:
  - configs: ["cfg1", "cfg2"]
    git:
      repo: http://github.com/lucky-tools/devloop.git
      path: getting-started/devloop.yaml
      ref: main
  - configs: ["cfg3"]
    googleCloudStorage:
      source: gs://my-bucket/dir1/*
      path: config/devloop.yaml
```

The environment variable `DEVLOOP_REMOTE_CACHE_DIR` or flag `--remote-cache-dir` specifies the download location for all remote dependency contents. If undefined then it defaults to `~/.devloop/remote-cache`. The remote cache directory consists of subdirectories with the contents retrieved from the remote dependency. For git dependencies the subdirectory name is a hash of the repo `uri` and the `branch/ref`. For Google Cloud Storage dependencies the subdirectory name is a hash of the `source`.

The remote config gets treated like a local config after substituting the path with the actual path in the cache directory.

### Profile Activation in required configs

Profiles specified by the `--profile` flag are also propagated to all  configurations imported as dependencies, if they define them. This behavior can be disabled by setting the `--propagate-profiles` flag to `false`.

You can additionally set up more granular and conditional profile activations across dependencies through the `activeProfiles` stanza:

```yaml
apiVersion: devloop/v1
kind: Config
metadata:
    name: cfg
requires:
  - path: ./path/to/required/devloop.yaml
    configs: [cfg1, cfg2]                 
    activeProfiles:                                     
     - name: profile1                               
       activatedBy: [profile2, profile3] 
```

Here, `profile1` is a profile that needs to exist in both configs `cfg1` and `cfg2`; while `profile2` and `profile3` are profiles defined in the current config `cfg`. If the current config is activated with either `profile2` or `profile3` then the required configs `cfg1` and `cfg2` are imported with `profile1` applied. If the `activatedBy` clause is omitted then that `profile1` always gets applied for the imported configs.



{{< alert title="Follow up" >}}
Take a look at the [tutorial]({{< relref "/docs/tutorials/config-dependencies" >}}) section to see this in action.
{{< /alert >}}
