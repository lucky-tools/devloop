---
title: "Load environment variables from a file"
linkTitle: "Load ENV from a file"
weight: 50
aliases: [/docs/concepts/env_file]
---

In Devloop, a `devloop.env` file can be defined in the project root directory to specify environment variables that Devloop will load into the process. This provides a more organized and manageable way of setting environment variables, rather than passing them as command line arguments.

The `devloop.env` file should be in the format of `KEY=value` pairs, with one pair per line. Devloop will automatically load these variables into the environment before running any commands.

Here is an example `devloop.env` file:

```txt
ENV_VAR_1=value1
ENV_VAR_2=value2
```

{{< alert title="Note" >}}
Values set in a `devloop.env` file will not overwrite existing environment variables in the process.
{{< /alert >}}

### Using a different Docker host for `devloop sync`

Some workflows build images on one machine and sync files into containers running on another. Set `DOCKER_HOST_SYNC` — in the process environment, the `devloop.env` file, or the config's `env` list — and `devloop sync` uses it in place of `DOCKER_HOST`. Other commands ignore it.

```yaml
env:
  - DOCKER_HOST=ssh://root@192.168.1.51        # build/deploy host
  - DOCKER_HOST_SYNC=ssh://root@192.168.1.53   # sync target host
```

With this configuration, `devloop build` uses `192.168.1.51` while `devloop sync` uses `192.168.1.53`, without editing the config between the two commands.

{{< alert title="Note" >}}
`DOCKER_HOST_SYNC` must be a Docker host URL (`ssh://`, `tcp://` or `unix://`), not a Docker context name. When it is applied, `DOCKER_CONTEXT` is cleared for the `sync` process so the Docker CLI and the Docker API client target the same daemon.
{{< /alert >}}

### Setting Devloop Flags with Environment Variables

In addition to loading environment variables from the `devloop.env` file, Devloop also allows users to set flags using environment variables. To set a flag using an environment variable, use the `DEVLOOP_` prefix and convert the flag name to uppercase.

For example, to set the `--cache-artifacts` flag to `true`, the equivalent environment variable would be `DEVLOOP_CACHE_ARTIFACTS=true`.

Here is an example usage in the `devloop.env` file:

```txt
DEVLOOP_CACHE_ARTIFACTS=true
DEVLOOP_NAMESPACE=mynamespace
```

{{< alert title="Note" >}}
If a flag is set both in the `devloop.env` file and as a command line argument, the value specified in the command line argument will take precedence.
{{< /alert >}}
