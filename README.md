
---------------------

[![Code Coverage](https://codecov.io/gh/lucky-tools/devloop/branch/main/graph/badge.svg)](https://codecov.io/gh/lucky-tools/devloop)
[![LICENSE](https://img.shields.io/github/license/lucky-tools/devloop.svg)](https://github.com/lucky-tools/devloop/blob/main/LICENSE)
[![Releases](https://img.shields.io/github/release-pre/lucky-tools/devloop.svg)](https://github.com/lucky-tools/devloop/releases)

Devloop is a command line tool that facilitates continuous development for
Kubernetes applications. You can iterate on your application source code
locally then deploy to local or remote Kubernetes clusters. Devloop handles
the workflow for building, pushing and deploying your application. It also
provides building blocks and describe customizations for a CI/CD pipeline.

---------------------

## [Install Devloop](https://devloop.dev/docs/install/)

Or, check out our [Github Releases](https://github.com/lucky-tools/devloop/releases) page for release info or to install a specific version.

![Demo](docs/static/images/intro.gif)

## Features

* Blazing fast local development
  * **optimized source-to-deploy** - Devloop detects changes in your source code and handles the pipeline to
  **build**, **push**, and **deploy** your application automatically with **policy based image tagging**
  * **continuous feedback** - Devloop automatically aggregates logs from deployed resources and forwards container ports to your local machine
* Project portability
  * **share with other developers** - Devloop is the easiest way to **share your project** with the world: `git clone` and `devloop run`
  * **context aware** - use Devloop profiles, user level config, environment variables and flags to describe differences in environments
  * **CI/CD building blocks** - use `devloop run` end-to-end, or use individual Devloop phases to build up your CI/CD pipeline. `devloop render` outputs hydrated Kubernetes manifests that can be used in GitOps workflows.
* Pluggable, declarative configuration for your project
  * **devloop init** - Devloop discovers your files and generates its own config file
  * **multi-component apps** - Devloop supports applications consisting of multiple components
  * **bring your own tools** - Devloop has a pluggable architecture to integrate with any build or deploy tool
* Lightweight
  * **client-side only** - Devloop has no cluster-side component, so there is no overhead or maintenance burden
  * **minimal pipeline** - Devloop provides an opinionated, minimal pipeline to keep things simple

### Check out our [examples page](./examples) for more complex workflows!

## IDE integrations

For a managed experience of Devloop you can install the Google `Cloud Code` extensions:
- for [Visual Studio Code](https://cloud.google.com/code/docs/vscode/quickstart-k8s#installing)
- for [JetBrains IDEs](https://cloud.google.com/code/docs/intellij/quickstart-k8s#installing_the_plugin). 

It can manage and keep Devloop  up-to-date while providing a more guided startup experience, along with providing and managing other common dependencies, and works with any kubernetes cluster. 

## Contributing to Devloop

We welcome any contributions from the community with open arms - Devloop wouldn't be where it is today without contributions from the community! Have a look at our [contribution guide](./CONTRIBUTING.md) for more information on how to get started on sending your first PR.

## Community

* [#devloop on Kubernetes Slack](https://kubernetes.slack.com/messages/CABQMSZA6/)
* [devloop-users mailing list](https://groups.google.com/forum/#!forum/devloop-users)

## Support 

Devloop is generally available and considered production ready.
Detailed feature maturity information and how we deprecate features are described in our [Deprecation Policy](https://devloop.dev/docs/references/deprecation).

## Security Disclosures

Please see our [security disclosure process](SECURITY.md).  All [security advisories](https://github.com/lucky-tools/devloop/security/advisories) are managed on Github.
