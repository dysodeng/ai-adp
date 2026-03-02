package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// ShutdownFunc is called on application shutdown to flush and stop the tracer.
type ShutdownFunc func()

// NewTracerProvider initialises OpenTelemetry tracing.
// When cfg.Enabled is false, a no-op tracer provider is returned.
// When cfg.Enabled is true, a real SDK provider is configured (OTLP exporter
// can be wired in later; for now it uses a stdout/noop exporter).
func NewTracerProvider(cfg config.TracingConfig) (trace.TracerProvider, ShutdownFunc, error) {
	if !cfg.Enabled {
		tp := noop.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, func() {}, nil
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "ai-adp"
	}

	sampleRate := cfg.SampleRate
	if sampleRate <= 0 || sampleRate > 1 {
		sampleRate = 1.0
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRate)),
	)

	otel.SetTracerProvider(tp)

	shutdown := func() {
		_ = tp.Shutdown(context.Background())
	}
	return tp, shutdown, nil
}
