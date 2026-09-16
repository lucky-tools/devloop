ARG BASE_PREFIX=mirror.gcr.io/library/
ARG BASE
FROM ${BASE_PREFIX}golang:1.18 as builder
WORKDIR /code
COPY main.go .
COPY go.mod .
# `devloop debug` sets DEVLOOP_GO_GCFLAGS to disable compiler optimizations
ARG DEVLOOP_GO_GCFLAGS
RUN go build -gcflags="${DEVLOOP_GO_GCFLAGS}" -trimpath -o /app .

FROM $BASE
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `devloop debug` (https://devloop.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
CMD ["./app"]
COPY --from=builder /app .
