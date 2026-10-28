---
title: "File Sync"
linkTitle: "File Sync"
weight: 44
featureId: sync
aliases: [/docs/how-tos/filesync, docs/pipeline-stages/filesync]
---

Devloop supports copying changed files to a deployed container so as to avoid the need to rebuild, redeploy, and restart the corresponding pod.
The file copying is enabled by adding a `sync` section with _sync rules_ to the `artifact` in the `devloop.yaml`.
Under the hood, Devloop creates a tar file with changed files that match the sync rules.
This tar file is sent to and extracted on the corresponding containers.

Multiple types of sync are supported by Devloop:

 + `manual`: The user must specify both the files in their local workspace and the destination in the running container.
   This sync mode is supported by every type of artifact.

 + `infer`: The destinations for each changed file is inferred from the builder.
   The docker and kaniko builders examine instructions in a Dockerfile.
   This inference is also supported for custom artifacts that **explicitly declare a dependency on a Dockerfile.**
   The ko builder can sync static content using this sync mode.

+ `auto`: Devloop automatically configures the sync.  This mode is only supported by Jib artifacts.

### Manual sync mode

A manual sync rule must specify the `src` and `dest` field.
The `src` field is a glob pattern to match files relative to the artifact _context_ directory, which may contain `**` to match nested files.
The `dest` field is the absolute or relative destination path in the container.
If the destination is a relative path, an absolute path will be inferred by prepending the path with the container's `WORKDIR`.
By default, matched files are transplanted with their whole directory hierarchy below the artifact context directory onto the destination.
The optional `strip` field can cut off some levels from the directory hierarchy.
The following example showcases manual filesync:

{{% readfile file="samples/filesync/filesync-manual.yaml" %}}

- The first rule synchronizes the file `.filebaserc` to the `/etc` folder in the container.
- The second rule synchronizes all `html` files in the `static-html` folder into the `<WORKDIR>/static` folder in the container.
  Note that this pattern does not match files in sub-folders below `static-html` (e.g. `static-html/a.html` but not `static-html/sub/a.html`).
- The third rule synchronizes all `png` files from any sub-folder into the `assets` folder on the container.
  For example, `img.png` ↷ `assets/img.png` or `sub/img.png` ↷ `assets/sub/img.png`.
- The last rule synchronizes all `md` files below the `content/en` directory into the `content` folder on the container.
  The `strip` directive ensures that only the directory hierarchy below `content/en` is re-created at the destination.
  For example, `content/en/index.md` ↷ `content/index.md` or `content/en/sub/index.md` ↷ `content/sub/index.md`.

### Inferred sync mode

For Docker artifacts, Devloop knows how to infer the desired destination from the artifact's `Dockerfile`
by examining the `ADD` and `COPY` instructions.

For Ko artifacts, Devloop infers the destination from the structure of your
codebase.

To enable syncing, you specify which files are eligible for syncing in the sync rules.
The sync rules for inferred sync mode is a list of glob patterns.

The following example showcases this filesync mode for Docker artifacts:

Given this Dockerfile:

```Dockerfile
FROM hugo

ADD .filebaserc /etc/
ADD assets assets/
COPY static-html static/
COPY content/en content/
```

And a `devloop.yaml` with the following sync configuration:

{{% readfile file="samples/filesync/filesync-infer.yaml" %}}

- The first rule synchronizes the file `.filebaserc` to `/etc/.filebaserc` in the container.
- The second rule synchronizes all `html` files in the `static-html` folder into the `<WORKDIR>/static` folder in the container.
  Note that this pattern does not match files in sub-folders below `static-html` (e.g. `static-html/a.html` but not `static-html/sub/a.html`).
- The third rule synchronizes any `png`. For example if `assest/img.png` ↷ `assets/img.png` or `static-html/imgs/demo.png` ↷ `static/imgs/demo.png`.
- The last rule enables synchronization for all `md` files below the `content/en`.
  For example, `content/en/sub/index.md` ↷ `content/sub/index.md` but _not_ `content/en_GB/index.md`.
  
For Docker artifacts, inferred sync mode only applies to modified and added
files; file deletion will cause a complete rebuild.

For multi-stage Dockerfiles, Devloop only examines the last stage.
Use manual sync rules to sync file copies from other stages.

[Ko artifacts supports syncing static content]({{<relref "/docs/builders/builder-types/ko#file-sync">}}),
and the sync rules apply to added, modified, and deleted files.

### Auto sync mode

In auto sync mode, Devloop automatically generates sync rules for known file types. 
Changes to other file types will result in a complete rebuild.

#### Jib

Jib integration with Devloop allows for zero-config `auto` sync. In this mode, Jib will sync your class files, resource files, and Jib's "extra directories" files to a remote container as changes are made. It can only be used with Jib in the default build mode (exploded) for non-WAR applications. It was primarily designed around [Spring Boot Developer Tools](https://docs.spring.io/spring-boot/docs/current/reference/html/using-spring-boot.html#using-boot-devtools), but can work with any embedded server that can reload/restart.

Check out the [Jib Sync example](https://github.com/lucky-tools/devloop/tree/main/examples/jib-sync) for more details.

## Limitations

File sync has some limitations:

  - File sync can only update files that can be modified by the container's configured User ID.
  - File sync requires the `tar` command to be available in the container.
  - Only local source files can be synchronized: files created by the builder will not be copied.
  - It is currently not allowed to mix `manual`, `infer` and `auto` sync modes.
    If you have a use-case for this, please let us know!
