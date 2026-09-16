---
title: "Custom Test"
linkTitle: "Custom Test"
weight: 20
featureId: test.custom
aliases: [/docs/pipeline-stages/testers/custom]
---


Custom Test allows developers to run custom commands as part of their development pipeline. The command executes in the testing phase of the [Devloop pipeline](https://devloop.dev/docs/). It will run on the local machine where Devloop is being executed and works with all supported Devloop platforms. Users can opt out of running custom tests by using the `--skip-tests` flag.


Some example use cases for Custom Test are below:
- Run unit tests
- Run validation and security scans on images before deploying the image to a cluster for example by running [GCP Container Analysis](https://cloud.google.com/container-analysis/docs/on-demand-scanning-howto) or [Anchore Grype](https://github.com/anchore/grype#readme)


Custom tests are defined on a per image basis in the Devloop config. Every time an artifact is rebuilt, Devloop runs the associated custom tests as part of the Devloop dev loop. Multiple testers can be defined per test. The Devloop pipeline will be blocked on the custom test to complete or fail. Devloop will block deployment when the first test fails. For ongoing test failures in the dev loop, Devloop will stop the loop (not continue with the deploy) but will not exit the loop. Devloop would surface the errors to the user and will keep the dev loop running. Devloop will continue watching user specified test dependencies and re-trigger the loop whenever it detects another change. 

CustomTester has a configurable timeout option to wait for the command to return. If no timeout is specified, Devloop will wait indefinitely until the test command has completed execution.

### Contract between Devloop and Custom command

Devloop will pass in the environment variable `$IMAGE` to the custom command to access the image.

This variable can be set as a flag value input to the custom command `--flag=$IMAGE`.


### Configuration
To use a custom command, add a custom field to the corresponding test in the test section of the devloop.yaml. Supported schema for CustomTest includes:


{{< schema root="CustomTest" >}}



### Dependencies for a Custom Test

Users can specify `dependencies` for custom tests so that devloop knows when to retest during a dev loop. Dependencies can be specified per command. Users could list out directories and/or files (for example test scripts)  to watch per command. If no dependencies are specified, only the script file (if the command is a script file) will be watched as a dependency. Test dependencies cannot trigger rebuild of an image.

Supported schema for `dependencies` include:

{{< schema root="CustomTestDependencies" >}}


#### Paths and Ignore

`Paths` and `Ignore` are arrays used to list dependencies. This can be a glob. Any `paths` in `Ignore` will be ignored by the devloop file watcher, even if they are also specified in `Paths`. `Ignore` will only work in conjunction with `Paths`.

```yaml
    custom:
      - command: ./test.sh
        timeoutSeconds: 60
        dependencies:
          paths:
          -  "*_test.go"
          -  "test.sh"
```

#### Command for dependencies

Sometimes users might have a command or a script that can provide the dependencies for a given test. Custom tester can ask Devloop to execute a custom command, which Devloop can use to get the dependencies for the test for file watching.

The command *must* return dependencies as a JSON array, otherwise devloop will error out.

```yaml
    custom:
      - command: echo Hello world!!
        dependencies:
          command: echo [\"main_test.go\"] 
```

 
>*Note: Adding a file pattern to a test dependency doesn't automatically enable file sync on it.  Refer to the [`file sync`](https://devloop.dev/docs/filesync/) documentation, on how to set that up separately.*


### Logging

`STDOUT` and `STDERR` from the custom command script will be redirected and displayed within devloop logs.


## Usage

Custom tests will be automatically invoked as part of the run and dev commands, but can also be run independently by using the test subcommand.

- To execute the custom command as an independent test command run:
```devloop test```
- To execute custom command as part of the run command run:
```devloop run```
- To execute custom command as part of the dev loop run:
```devloop dev```
### Example
This following example shows the `customTest` section from a `devloop.yaml`.
It instructs Devloop to run unit tests (main_test.go) located in the local folder when the main application changes:
{{% readfile file="samples/testers/custom/customTest.yaml" %}}
A sample `test.sh` file, which runs unit tests when the test changes.
```
#!/bin/bash

set -e

echo "go custom test $@"

go test .
```



