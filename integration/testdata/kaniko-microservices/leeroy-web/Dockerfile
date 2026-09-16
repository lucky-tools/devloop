ARG BASE_PREFIX=mirror.gcr.io/library/
ARG BASE
FROM ${BASE_PREFIX}golang:1.15 as builder
COPY web.go .
# `devloop debug` sets DEVLOOP_GO_GCFLAGS to disable compiler optimizations
ARG DEVLOOP_GO_GCFLAGS
RUN go build -gcflags="${DEVLOOP_GO_GCFLAGS}" -o /app .

FROM $BASE
COPY --from=builder /app .
