### Example: Jib (Gradle)

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/lucky-tools/devloop&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/jib-gradle)

[Jib](https://github.com/GoogleContainerTools/jib) is one of the supported builders in Devloop.
It builds Docker and OCI images
for your Java applications and is available as plugins for Maven and Gradle.

The way you configure it in `devloop.yaml` is the following build stanza:

```yaml
build:
     artifacts:
     - image: devloop-example
       context: .
       jib: {}
```
