ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.15 as builder

COPY app.go .
ARG DEVLOOP_GO_GCFLAGS
RUN go build -gcflags="${DEVLOOP_GO_GCFLAGS}" -o /app .

FROM gcr.io/distroless/base
# `devloop debug` uses GOTRACEBACK as an indicator of the Go runtime
ENV GOTRACEBACK=all
WORKDIR /
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
