ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.19 as builder
COPY main.go .
# `devloop debug` sets DEVLOOP_GO_GCFLAGS to disable compiler optimizations
ARG DEVLOOP_GO_GCFLAGS
RUN go build -gcflags="${DEVLOOP_GO_GCFLAGS}" -o /app main.go

FROM ${BASE_PREFIX}alpine:3
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `devloop debug` (https://devloop.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
CMD ["./app"]
COPY --from=builder /app .
