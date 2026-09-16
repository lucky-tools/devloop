---
title: "Quickstart"
linkTitle: "Quickstart"
weight: 20
---
{{% tabs %}}

{{% tab "STANDALONE" %}}

Follow this tutorial if you're using the Devloop [standalone binary]({{< relref "../install/#standalone-binary" >}}). It walks through running Devloop on a small Kubernetes app built with [Docker](https://www.docker.com/) inside [minikube](https://minikube.sigs.k8s.io)
and deployed with [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

{{< alert title="Note" >}}
If you are looking to generate a new project templated to use Devloop best-practices and features, see the [Google Cloud Solutions Template](https://github.com/GoogleCloudPlatform/solutions-template).
{{< /alert >}}

{{< alert title="Note">}}
Aside from `Docker` and `kubectl`, Devloop also supports a variety of other tools
and workflows; see [Tutorials]({{<relref "/docs/tutorials">}}) for
more information.
{{</alert>}}

In this quickstart, you will:

* Use **devloop init** to bootstrap your Devloop config.
* Use **devloop dev** to automatically build and deploy your application when your code changes.
* Use **devloop build** and **devloop test** to tag, push, and test your container images.
* Use **devloop render** and **devloop apply** to generate and deploy Kubernetes manifests as part of a GitOps workflow.

## Set up

### Install Devloop, minikube, and kubectl

This tutorial requires Devloop, minikube, and kubectl.

1. [Install Devloop]({{< relref "/docs/install" >}}).
1. [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).
1. [Install minikube](https://minikube.sigs.k8s.io/docs/start/).

This tutorial uses minikube because Devloop knows how to build the app using the Docker daemon hosted
inside minikube. This means we don't need a registry to host the app's container images.

### Clone the sample app

Let's get a sample application set up to use Devloop.

1. Clone the Devloop repository:

    ```bash
    git clone https://github.com/lucky-tools/devloop
    ```

1. Change to the `examples/buildpacks-node-tutorial` directory.

    ```bash
    cd devloop/examples/buildpacks-node-tutorial
    ```

## Initialize Devloop

Your working directory is the application directory, `devloop/examples/buildpacks-node-tutorial`. This will be our root Devloop directory.

This sample application is written in Node, but Devloop is language-agnostic and works with any containerized application.

### Bootstrap Devloop configuration

1. Run the following command to generate a `devloop.yaml` config file:


    ```bash
    devloop init
    ```

1. When prompted to choose the builder, press enter to accept the default selection.

1. When asked which builders you would like to create Kubernetes resources for, press enter to accept the default selection.

1. When asked if you want to write this configuration to devloop.yaml, type "y" for yes.

1. Open your new **devloop.yaml**, generated at `devloop/examples/buildpacks-node-tutorial/devloop.yaml`. All of your Devloop configuration lives in this file. We will go into more detail about how it works in later steps.

## Use Devloop for continuous development

Devloop speeds up your development loop by automatically building and deploying the application whenever your code changes.

### Start minikube

1. To see this in action, let's start up minikube so Devloop has a cluster to run your application.

    ```bash
    minikube start --profile custom
    devloop config set --global local-cluster true
    eval $(minikube -p custom docker-env)
    ```

This may take several minutes.

### Use `devloop dev`


1. Run the following command to begin using Devloop for continuous development:

    ```bash
    devloop dev
    ```

    Notice how Devloop automatically builds and deploys your application. You should see the following application output in your terminal:

    ```terminal
    Example app listening on port 3000!
    ```
    
    To browse to the web page, open a new terminal and run:
    ```terminal
    minikube tunnel -p custom
    ```
    
    Now open your browser at `http://localhost:3000`. This displays the content of `public/index.html` file. 

    Devloop is now watching for any file changes, and will rebuild your application automatically. Let's see this in action.


1. Open `devloop/examples/buildpacks-node-tutorial/src/index.js` and change line 10 to the following:

    ```
    app.listen(port, () => console.log(`Example app listening on port ${port}! This is version 2.`))
    ```

    Notice how Devloop automatically hot reloads your code changes to your application running in minikube, intelligently syncing only the file you changed. Your application is now automatically deployed with the changes you made, as it prints the following to your terminal:

    ```terminal
    Example app listening on port 3000! This is version 2.
    ```

### Exit dev mode

1. Let's stop continuous dev mode by pressing the following keys in your terminal:

    ```terminal
    Ctrl+C
    ```

    Devloop will clean up all deployed artifacts and end dev mode.

## Use Devloop for continuous integration

While Devloop shines for continuous development, it can also be used for continuous integration (CI). Let's use Devloop to build and test a container image.

### Build an image

Your CI pipelines can run `devloop build` to build, tag, and push your container images to a registry. 

1. Try this out by running the following command:

    ```bash
    export STATE=$(git rev-list -1 HEAD --abbrev-commit)
    devloop build --file-output build-$STATE.json
    ```

    Devloop writes the output of the build to a JSON file, which we'll pass to our continuous delivery (CD) process in the next step.

### Test an image

Devloop can also run tests against your images before deploying them.  Let's try this out by creating a simple custom test.

1. Open your<walkthrough-editor-open-file filePath="cloudshell_open/devloop/examples/buildpacks-node-tutorial/devloop.yaml">`devloop.yaml`</walkthrough-editor-open-file> and add the following test configuration to the bottom, without any additional indentation:

    ```
    test:
    - image: devloop-buildpacks-node
      custom:
        - command: echo This is a custom test commmand!
    ```

    Now you have a simple custom test set up that will run a bash command and await a successful response.

1. Run the following command to execute this test with Devloop:

    ```bash
    devloop test --build-artifacts build-$STATE.json
    ```

## Use Devloop for continuous delivery

Let's learn how Devloop can handle continuous delivery (CD).

### Deploy in a single step

1. For simple deployments, run `devloop deploy`:

    ```bash
    devloop deploy -a build-$STATE.json
    ```

    Devloop hydrates your Kubernetes manifest with the image you built and tagged in the previous step, and deploys the application.

### Render and apply in separate steps

For GitOps delivery workflows, you may want to decompose your deployments into separate render and apply phases. That way, you can commit your hydrated Kubernetes manifests to source control before they are applied.

1. Run the following command to render a hydrated manifest:

    ```bash
    devloop render -a build-$STATE.json --output render.yaml --digest-source local
    ```

    Open `devloop/examples/buildpacks-node-tutorial/render.yaml` to check out the hydrated manifest.


1. Next, run the following command to apply your hydrated manifest:

    ```bash
    devloop apply render.yaml
    ```

You have now successfully deployed your application in two ways.

## Congratulations, you successfully deployed with Devloop!

You have learned how to use Devloop for continuous development, integration, and delivery.

{{% /tab %}}

{{% tab "CLOUD CODE" %}}

Follow these quickstart guides if you're using Devloop with the [Cloud Code]({{< relref "../install/#managed-ide" >}}) IDE extensions:

### [Cloud Code for VSCode](https://cloud.google.com/code/docs/vscode/quickstart-k8s)

Create, locally develop, debug, and run a Kubernetes application with Cloud Code for VSCode.

<a href="https://cloud.google.com/code/docs/vscode/quickstart-k8s">![vscode](/images/cloud-code-quick-deploy.gif)</a>

<br />

### [Cloud Code for IntelliJ](https://cloud.google.com/code/docs/intellij/quickstart-k8s)

Create, locally develop, debug, and run a Kubernetes application with Cloud Code for IntelliJ.

<a href="https://cloud.google.com/code/docs/intellij/quickstart-k8s">![intellij](/images/intellij-quickstart-runthrough.gif)</a>

{{% /tab %}}
{{% tab "CLOUD SHELL" %}}

Skip any setup by using Google Cloud Platform's [_Cloud Shell_](http://cloud.google.com/shell),
which provides a [browser-based terminal/CLI and editor](https://cloud.google.com/shell#product-demo).
Cloud Shell comes with Devloop, Minikube, and Docker pre-installed, and is free
(requires a [Google Account](https://accounts.google.com/SignUp)).

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?show=ide%2Cterminal&cloudshell_git_repo=https://github.com/lucky-tools/devloop&walkthrough_id=devloop--devloop_onboarding&cloudshell_workspace=/examples/buildpacks-node-tutorial&cloudshell_open_in_editor=src/index.js)

{{% /tab %}}
{{% /tabs %}}

## What's next

For getting started with your project, see the [Getting Started With Your Project]({{<relref "/docs/workflows/getting-started-with-your-project" >}}) workflow.

For more in-depth topics of Devloop, explore [Configuration]({{< relref "/docs/design/config.md" >}}),
[Devloop Pipeline]({{<relref "/docs/pipeline-stages" >}}), and [Architecture and Design]({{< relref "/docs/design" >}}).

To learn more about how Devloop builds, tags, and deploys your app, see the How-to Guides on
using [Builders]({{<relref "/docs/builders" >}}), [Taggers]({{< relref "/docs/taggers">}}), and [Deployers]({{< relref "/docs/deployers" >}}).

[Devloop Tutorials]({{< relref "/docs/tutorials" >}}) details some of the common use cases of Devloop.

Questions?  See our [Community section]({{< relref "/docs/resources#Community" >}}) for ways to get in touch.

:mega: **Please fill out our [quick 5-question survey](https://forms.gle/BMTbGQXLWSdn7vEs6)** to tell us how satisfied you are with Devloop, and what improvements we should make. Thank you! :dancers:
