### Example: inferred file sync using the ko builder

This example uses
[inferred file sync](https://devloop.dev/docs/pipeline-stages/filesync/#inferred-sync-mode)
for static assets with the
[`ko` builder](https://devloop.dev/docs/pipeline-stages/builders/ko/)
for a Go web app.

To observe the behavior of file sync, run this command:

```shell
devloop dev
```

Try changing the HTML file in the `kodata` directory to see how Devloop
syncs the file.

If change the the `main.go` file, Devloop will rebuild and redeploy the image.
