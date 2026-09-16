ARG BASE_PREFIX=mirror.gcr.io/library/
# As Go supports cross-compilation, `--platform=$BUILDPLATFORM`
# results in a dramatic speed-up as the build runs natively
# instead of using emulation.
# BUILD{PLATFORM,OS,ARCH} are set to your docker engine's platform
# TARGET{PLATFORM,OS,ARCH} are set to the desired platform
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

WORKDIR /code
COPY main.go .
COPY go.mod .

# `devloop debug` sets DEVLOOP_GO_GCFLAGS to disable compiler optimizations
ARG DEVLOOP_GO_GCFLAGS
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -gcflags="${DEVLOOP_GO_GCFLAGS}" -trimpath -o /app main.go

FROM ${BASE_PREFIX}alpine:3
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `devloop debug` (https://devloop.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
CMD ["./app"]
COPY --from=builder /app .
