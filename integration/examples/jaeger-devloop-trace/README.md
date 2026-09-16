### Example: Devloop Command Tracing with Jaeger

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/lucky-tools/devloop&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/jaegar-devloop-trace)


_**WARNING: Devloop's trace functionality is experimental and may change without notice.**_

In this example:

* Use Devloop to deploy a local/remote Jaeger instance
* Enable Devloop tracing functionality to get trace information about devloop `dev`, `build`, `deploy`, etc. timings
* Send Devloop trace information to Jaeger and have that information be visible in the Jaeger UI,

In this example, we'll walk through enabling Devloop trace information that can be used to explore performance bottlenecks and to get a more in depth view of user's local dev loop.

_**WARNING: If you're running this on a cloud cluster, this example will create a service and expose a webserver.
It's highly recommended that you only run this example on a local, private cluster like minikube or Kubernetes in Docker for Desktop.**_

#### Setting up Jaeger locally 

Use docker to start a local jaeger instance using the Jaeger project's [all-in-one docker setup](https://www.jaegertracing.io/docs/getting-started/#all-in-one):
```bash
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:1.22
```

Now, in a different terminal, go to another Devloop example (eg: microservices), enable DEVLOOP_TRACE with Jaeger and start dev session there:
```bash
cd ../microservices
export DEVLOOP_TRACE=jaeger
devloop dev
```

Now go to the Jaeger UI that Devloop will port-forward to localhost at http://127.0.0.1:16686/

Select service:`devloop-trace` in the left bar and then click `Find Traces` on the bottom of the left bar.  

From here you should be able to view all of the relevant traces

#### Cleaning up
To cleanup Jaeger all-in-one setup, run the following:
```
docker kill jaeger # stops the running jaeger container
docker rm jaeger #removes the container image
```