package shared

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
)

// InitTracerProvider initialises the global OpenTelemetry TracerProvider and propagator.
//
// When cfg.OTELEndpoint is set, traces are exported via OTLP/gRPC to that endpoint
// (Jaeger, Honeycomb, Grafana Tempo, etc.). In development with no endpoint configured,
// traces are written to stdout so they're visible without any backend infrastructure.
//
// The returned shutdown function MUST be deferred in main() to flush buffered spans
// before the process exits. Failing to call it will drop the last batch of traces.
func InitTracerProvider(ctx context.Context, cfg *config.AppConfig, version string) (func(context.Context), error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("homegrown-academy"),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironmentName(string(cfg.Environment)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build OTel resource: %w", err)
	}

	var exporter sdktrace.SpanExporter
	if cfg.OTELEndpoint != nil {
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(*cfg.OTELEndpoint),
			otlptracegrpc.WithInsecure(), // TLS termination is handled by infra (sidecar/proxy)
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		slog.Info("OTel tracing: OTLP/gRPC exporter", "endpoint", *cfg.OTELEndpoint)
	} else if cfg.Environment == config.EnvironmentDevelopment {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout trace exporter: %w", err)
		}
		slog.Info("OTel tracing: stdout exporter (dev mode)")
	} else {
		// Non-development without an endpoint — install a noop provider so OTel
		// API calls are safe but no spans are exported.
		slog.Info("OTel tracing: disabled (set OTEL_EXPORTER_OTLP_ENDPOINT to enable)")
		tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		installGlobalOTel(tp)
		return func(c context.Context) { _ = tp.Shutdown(c) }, nil
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.OTELSampleRate)),
	)

	installGlobalOTel(tp)
	return func(c context.Context) { _ = tp.Shutdown(c) }, nil
}

// installGlobalOTel sets the global TracerProvider and the W3C TraceContext + Baggage
// composite propagator. Any OTel-aware library (including our own middleware) will pick
// these up automatically via otel.GetTracerProvider() and otel.GetTextMapPropagator().
func installGlobalOTel(tp *sdktrace.TracerProvider) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C traceparent / tracestate headers
		propagation.Baggage{},
	))
}
