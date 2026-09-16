---
title: "Installing Devloop"
linkTitle: "Installing Devloop"
weight: 10
aliases: [/docs/getting-started]
---

{{< alert title="Note" >}}

To keep Devloop up to date, update checks are made to Google servers to see if a new version of
Devloop is available.

You can turn this update check off by following [these instructions]({{<relref "/docs/references/privacy#update-check">}}).

To help prioritize features and work on improving Devloop, we collect anonymized Devloop usage data.
You can opt out of data collection by following [these instructions]({{<relref "/docs/resources/telemetry">}}).

Your use of this software is subject to the [Google Privacy Policy](https://policies.google.com/privacy)

{{< /alert >}}

### Managed IDE

{{% tabs %}}

{{% tab "CLOUD CODE" %}}

[Cloud Code](https://cloud.google.com/code) provides a managed experience of using Devloop in supported IDEs. You can install the `Cloud Code` extension for [Visual Studio Code](https://cloud.google.com/code/docs/vscode/install) or the plugin for [JetBrains IDEs](https://cloud.google.com/code/docs/intellij/quickstart-k8s#installing_the_plugin). It manages and keeps Devloop  up-to-date, along with other common dependencies, and works with any kubernetes cluster.

{{% /tab %}}

{{% tab "GOOGLE CLOUD SHELL" %}}

Google Cloud Platform's [_Cloud Shell_](http://cloud.google.com/shell)
provides a free [browser-based terminal/CLI and editor](https://cloud.google.com/shell#product-demo)
with Devloop, Minikube, and Docker pre-installed.
(Requires a [Google Account](https://accounts.google.com/SignUp).)

Cloud Shell is a great way to try Devloop out.

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?shellonly=true&cloudshell_git_repo=https%3A%2F%2Fgithub.com%2FGoogleContainerTools%2Fdevloop&cloudshell_working_dir=examples%2Fgetting-started)

{{% /tab %}}

{{% /tabs %}}

### Standalone binary

{{% tabs %}}

{{% tab "LINUX" %}}
The latest **stable** binaries can be found here:

- Linux x86_64 (amd64): https://storage.googleapis.com/devloop/releases/latest/devloop-linux-amd64
- Linux ARMv8 (arm64): https://storage.googleapis.com/devloop/releases/latest/devloop-linux-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For Linux x86_64 (amd64)
curl -Lo devloop https://storage.googleapis.com/devloop/releases/latest/devloop-linux-amd64 && \
sudo install devloop /usr/local/bin/
```

```bash
# For Linux ARMv8 (arm64)
curl -Lo devloop https://storage.googleapis.com/devloop/releases/latest/devloop-linux-arm64 && \
sudo install devloop /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

- Linux x86_64 (amd64): https://storage.googleapis.com/devloop/builds/latest/devloop-linux-amd64
- Linux ARMv8 (arm64): https://storage.googleapis.com/devloop/builds/latest/devloop-linux-arm64

```bash
# For Linux on x86_64 (amd64)
curl -Lo devloop https://storage.googleapis.com/devloop/builds/latest/devloop-linux-amd64 && \
sudo install devloop /usr/local/bin/
```

```bash
# For Linux on ARMv8 (arm64)
curl -Lo devloop https://storage.googleapis.com/devloop/builds/latest/devloop-linux-arm64 && \
sudo install devloop /usr/local/bin/
```

{{% /tab %}}

{{% tab "MACOS" %}}

The latest **stable** binaries can be found here:

- Darwin x86_64 (amd64): https://storage.googleapis.com/devloop/releases/latest/devloop-darwin-amd64
- Darwin ARMv8 (arm64): https://storage.googleapis.com/devloop/releases/latest/devloop-darwin-arm64

Simply download the appropriate binary and add it to your `PATH`. Or, copy+paste one of the following commands in your terminal:

```bash
# For macOS on x86_64 (amd64)
curl -Lo devloop https://storage.googleapis.com/devloop/releases/latest/devloop-darwin-amd64 && \
sudo install devloop /usr/local/bin/
```

```bash
# For macOS on ARMv8 (arm64)
curl -Lo devloop https://storage.googleapis.com/devloop/releases/latest/devloop-darwin-arm64 && \
sudo install devloop /usr/local/bin/
```

We also release a **bleeding edge** build, built from the latest commit:

- Darwin x86_64 (amd64): https://storage.googleapis.com/devloop/builds/latest/devloop-darwin-amd64
- Darwin ARMv8 (arm64): https://storage.googleapis.com/devloop/builds/latest/devloop-darwin-arm64

```bash
# For macOS on x86_64 (amd64)
curl -Lo devloop https://storage.googleapis.com/devloop/builds/latest/devloop-darwin-amd64 && \
sudo install devloop /usr/local/bin/
```

```bash
# For macOS on ARMv8 (arm64)
curl -Lo devloop https://storage.googleapis.com/devloop/builds/latest/devloop-darwin-arm64 && \
sudo install devloop /usr/local/bin/
```

Devloop is also kept up to date on a few central package managers:

### Homebrew

```bash
brew install devloop
```

### MacPorts

```bash
sudo port install devloop
```

{{% /tab %}}

{{% tab "WINDOWS" %}}

The latest **stable** release binary can be found here:

https://storage.googleapis.com/devloop/releases/latest/devloop-windows-amd64.exe

Simply download it and place it in your `PATH` as `devloop.exe`.

We also release a **bleeding edge** build, built from the latest commit:

https://storage.googleapis.com/devloop/builds/latest/devloop-windows-amd64.exe

---

### Scoop

Devloop can be installed using the [Scoop package manager](https://scoop.sh/)
from the [extras bucket](https://github.com/lukesampson/scoop-extras#readme).
This package is not maintained by the Devloop team.

```powershell
scoop bucket add extras
scoop install devloop
```

### Chocolatey

Devloop can be installed using the [Chocolatey package manager](https://chocolatey.org/packages/devloop).
This package is not maintained by the Devloop team.

{{< alert title="Caution" >}}

Chocolatey's installation mechanism interferes with <kbd>Ctrl</kbd>+<kbd>C</kbd> handling
and [prevents Devloop from cleaning up deployments](https://github.com/lucky-tools/devloop/issues/4815).
This cannot be fixed by Devloop.
For more information about this defect see
[chocolatey/shimgen#32](https://github.com/chocolatey/shimgen/issues/32).

{{< /alert >}}

```bash
choco install -y devloop
```
{{% /tab %}}

{{% tab "GCLOUD" %}}

If you have the Google Cloud SDK installed on your machine, you can quickly install Devloop as a bundled component.

Make sure your gcloud installation and the components are up to date:

`gcloud components update`

Then, install Devloop:

`gcloud components install devloop`

{{% /tab %}}

{{% tab "DOCKER" %}}

### Stable binary

For the latest **stable** release, you can use:

`docker run gcr.io/k8s-devloop/devloop:latest devloop <command>`

### Bleeding edge binary

For the latest **bleeding edge** build:

`docker run gcr.io/k8s-devloop/devloop:edge devloop <command>`

{{% /tab %}}

{{% /tabs %}}
