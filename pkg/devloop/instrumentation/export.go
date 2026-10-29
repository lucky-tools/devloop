/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instrumentation

import (
	"context"
	"fmt"
	"io"
	"os"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
)

type TraceExporterConfig struct {
	writer io.Writer
}

type TraceExporterOption func(te *TraceExporterConfig)

func WithWriter(w io.Writer) TraceExporterOption {
	return func(teconf *TraceExporterConfig) {
		teconf.writer = w
	}
}

func initTraceExporter(opts ...TraceExporterOption) (trace.TracerProvider, func(context.Context) error, error) {
	teconf := TraceExporterConfig{
		writer: os.Stdout,
	}

	for _, opt := range opts {
		opt(&teconf)
	}

	switch os.Getenv("DEVLOOP_TRACE") {
	case "stdout":
		log.Entry(context.TODO()).Debug("using stdout trace exporter")
		return initIOWriterTracer(teconf.writer)
	case "gcp-adc":
		log.Entry(context.TODO()).Debug("using cloud trace exporter w/ application default creds")
		tp, shutdown, err := initCloudTraceExporterApplicationDefaultCreds()
		return tp, func(context.Context) error { shutdown(); return nil }, err
	case "jaeger":
		log.Entry(context.TODO()).Debug("using jaeger trace exporter")
		tp, shutdown, err := initJaegerTraceExporter()
		return tp, func(context.Context) error { shutdown(); return nil }, err
	}

	if otelTraceExporterVal, ok := os.LookupEnv("OTEL_TRACES_EXPORTER"); ok {
		log.Entry(context.TODO()).Debugf("using otel default exporter - OTEL_TRACES_EXPORTER=%s", otelTraceExporterVal)
		return nil, func(context.Context) error { return nil }, nil
	}

	return nil, func(context.Context) error { return nil }, fmt.Errorf("error initializing trace exporter")
}

// initIOWriterTracer creates and registers trace provider instance that writes to an io.Writer interface
func initIOWriterTracer(w io.Writer) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, nil, err
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	return tp, tp.Shutdown, nil
}

func initCloudTraceExporterApplicationDefaultCreds() (trace.TracerProvider, func(), error) {
	otel.SetErrorHandler(errHandler{})
	exporter, err := texporter.New()
	if err != nil {
		return nil, nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))

	return tp, func() {
		tp.Shutdown(context.Background())
	}, nil
}

// initJaegerTraceExporter returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also be configured to report spans as soon as possible.
func initJaegerTraceExporter() (trace.TracerProvider, func(), error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		return nil, nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("devloop-trace"),
			attribute.Int64("ID", 1), // TODO(aaron-prindle) verify this value makes sense
		)),
	)
	return tp, func() {}, nil
}
