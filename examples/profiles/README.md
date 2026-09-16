### Example: Getting started with a simple go app

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/lucky-tools/devloop&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/profiles)

This is a simple show-case of Devloop profiles

#### Init

Use the `--profile` option to enable a particular profile `devloop dev --profile staging-profile`

#### Workflow

* Build only the `world-service` when using the main profile
* Activate `minikube-profile` automatically when the current context is `minikube`. Only build the `hello-service` in that case.
* Build both services when the `staging-profile` is used. Override the kube-context to `staging` in that case.
