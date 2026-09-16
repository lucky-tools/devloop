# Transparent Init

* Author(s): Marlon Gamez
* Design Shepherd: \<devloop-core-team-member\>
* Date: 10/13/2020
* Status: [Reviewed/Cancelled/Under implementation/Complete]

## Background

This proposal is brought about by an older issue, [#1273](https://github.com/lucky-tools/devloop/issues/1273)

With the recent focus on devloop UX, we've decided to see how we can improve the onboarding experience when using devloop.
One way we'd like to do this is by allowing the user to simply install devloop and run devloop commands like `run`, `dev`, or `debug` in their project, without having to first run `devloop init` and go through the interactive portion of creating a config for their project.
To do this, we'd like to automatically create a config upon invocation of devloop commands, so that the user doesn't have to see something like this
```
❯ devloop dev
devloop config file devloop.yaml not found - check your current working directory, or try running `devloop init`
```
when trying to run devloop.

## Scope

This proposal is meant to design the flow of a user running a command without a devloop config file already on disk. It does not cover the actual functionality of `devloop init --force`, and doesn't address the further changes that will be made to that functionality.

This functionality is planned to initially work with the `run`, `dev`, and `debug` commands, as we want to cater to the commands that are meant to provide the best first time UX.

## Design

With transparent init, a user's first run of `devloop dev` could look something like this:

Given a simple project structure:
```
project-dir
├── Dockerfile
├── README.md
├── k8s-pod.yaml
└── main.go
```
Running devloop dev:
```
❯ devloop dev
This seems to be your first time running devloop in this project. If you choose to continue, devloop will:
- Create a devloop config file for you
- Build your application using docker
- Deploy your application to your current kubernetes context using kubectl

Please double check the above steps. Deploying to production kubernetes clusters can be destructive.

Would you like to continue?
> yes
  no

Listing files to watch...
 - devloop-example
Generating tags...
 - devloop-example -> devloop-example:v1.14.0-59-g7e5ae4cbc-dirty
Checking cache...
 - devloop-example: Not found. Building
Found [minikube] context, using local docker daemon.
Building [devloop-example]...
Sending build context to Docker daemon  3.072kB
...
```

There are two approaches to implementation that I've considered:

**Creating the config in memory using init functions (Preferred)**

Under the hood view:
- `devloop dev` is run
- devloop searches for a `devloop.yaml` file in the directory, finds nothing
- `devloop init` is run to generate a config
- user is prompted as shown above
- if user selects "yes", `devloop dev` continues like normal

**Creating a temporary `devloop.yaml` by running `devloop init --force`**

Under the hood view:
- `devloop dev` is run
- devloop searches for a `devloop.yaml` file in the directory, finds nothing
- devloop creates a new config using `devloop init --force`
- the new config is parsed and brought into memory
- the config file is deleted from disk
- `devloop dev` continues like normal

## Open Issues/Questions

**Should generation of the config all happen in memory? By refactoring `DoInit()` we could probably generate the config and start using it without having to write/read the `devloop.yaml` from disk**

I think this would be preferred. We could eliminate an unecessary read from disk by doing so.

**Should we prompt the user to ask if they'd like to save this config for future use?**

I believe that it would be fine to write to disk automatically. The logging will make it clear that this is being done and after the devloop session is finished they can decide to remove the file or not. 

**Should the config being used be printed to the terminal? Tradeoff of information vs clutter**

I would say that we don't need to. Printing the config would add a large clutter, and if the user needs to view the config used, they can view the devloop config that is written to disk.

## Implementation plan

If we decide to have the config written to disk and kept in memory, we may want to split this into a couple PRs. Maybe something like this:
1. Refactor `DoInit()` so that it is broken into parts that we can use later
2. Use the new parts to implement the automatic generation of the config

If we decide upon the approach in which the `devloop.yaml` is written to/read from disk, this could be done in 1 PR, as a refactor wouldn't be necessary.

## Integration test plan

New integration tests can be added to check that invocations of devloop commands in folders without a devloop config still run properly. With the other plans to change devloop init, these will have to be updated as well.
