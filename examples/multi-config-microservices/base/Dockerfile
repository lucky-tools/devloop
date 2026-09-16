ARG BASE_PREFIX=mirror.gcr.io/library/
FROM gcr.io/distroless/base
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `devloop debug` (https://devloop.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
CMD ["./app"]
