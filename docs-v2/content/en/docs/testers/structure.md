---
title: "Container Structure Test"
linkTitle: "Container Structure Test"
weight: 20
featureId: test.structure
aliases: [/docs/pipeline-stages/testers/structure]
---

It's common practice to validate built container images before deploying them to our cluster.
To do this, Devloop has an integrated testing phase between the build and deploy phases of the pipeline.
Natively, Devloop has support for running [container-structure-tests](https://github.com/GoogleContainerTools/container-structure-test)
on built images, which validate the structural integrity of container images.
The container-structure-test [binary](https://github.com/GoogleContainerTools/container-structure-test/releases)
must be installed to run these tests.

Structure tests are defined per image in the Devloop config.
Every time an artifact is rebuilt, Devloop runs the associated structure tests on that image.
If the tests fail, Devloop will not continue on to the deploy stage.
If frequent tests are prohibitive, long-running tests should be moved to a dedicated Devloop profile.
Users can opt out of running container structure tests by using the `--skip-tests` flag.

### Example
This following example shows the `test` section from a `devloop.yaml`.
It instructs Devloop to run all container structure tests in the `structure-test` folder relative to the Devloop root directory:

{{% readfile file="samples/testers/structure/test.yaml" %}}

The files matched by the `structureTests` key are `container-structure-test` test configurations, such as:

{{% readfile file="samples/testers/structure/structureTest.yaml" %}}

For a reference how to write container structure tests, see its [documentation](https://github.com/GoogleContainerTools/container-structure-test#command-tests).

In order to restrict the executed structure tests, a `profile` section can override the file pattern:

{{% readfile file="samples/testers/structure/testProfile.yaml" %}}

User can customize `container-structure-test` behavior by passing a list of configuration flags as a value of `structureTestsArgs` yaml property in `devloop.yaml`, e.g.:

{{% readfile file="samples/testers/structure/structureTestArgs.yaml" %}}

To execute the tests once, run `devloop test --profile quickcheck`.
