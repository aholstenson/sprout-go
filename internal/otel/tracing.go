package otel

import (
	"context"

	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type traceConfig struct {
	// Development is a flag that enables logging of traces in development mode.
	Development bool `env:"DEVELOPMENT"`
}

// setupTracing configures OpenTelemetry tracing.
func setupTracing(
	service ServiceInfo,
	resource *resource.Resource,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	config traceConfig,
) (trace.TracerProvider, error) {
	// Set a better default for propagators, since the default is a no-op
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	var exportingOption sdktrace.TracerProviderOption

	if service.Development || service.Testing {
		// If running in development or testing mode, we don't want to send
		// any traces to the collector
		if config.Development {
			// If tracing development mode is enabled, we want to log the
			// traces
			logger.Info("Traces enabled for development mode, logging traces")
			exportingOption = sdktrace.WithSyncer(&loggingTraceExporter{
				logger: logger.Named("trace"),
			})
		} else {
			// In other cases we use a no-op tracer provider
			logger.Info("Disabling tracing due to development mode")
			return noopTracing()
		}
	} else {
		if !hasExporterEndpoint(true) {
			logger.Warn("No tracing exporter endpoint set, disabling tracing")
			return noopTracing()
		}

		// TODO: Load config and error if not set correctly
		exporter := otlptracegrpc.NewUnstarted()
		exportingOption = sdktrace.WithBatcher(exporter)

		lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return exporter.Start(ctx)
			},
		})
	}

	// TODO: Support for sampling
	tp := sdktrace.NewTracerProvider(
		exportingOption,
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(tp)

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
	})

	return tp, nil
}

func noopTracing() (trace.TracerProvider, error) {
	provider := trace.NewNoopTracerProvider()
	otel.SetTracerProvider(provider)
	return provider, nil
}

type loggingTraceExporter struct {
	logger *zap.Logger
}

func (e *loggingTraceExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error { //nolint:revive
	for _, span := range spans {
		// Collect the the context as fields
		fields := []zap.Field{
			zap.String("traceID", span.SpanContext().TraceID().String()),
			zap.String("spanID", span.SpanContext().SpanID().String()),
			zap.String("parentSpanID", span.Parent().SpanID().String()),
			zap.String("status", span.Status().Code.String()),
			zap.Time("startTime", span.StartTime()),
			zap.Duration("duration", span.EndTime().Sub(span.StartTime())),
			zap.Namespace("attributes"),
		}

		// Add the attributes as fields
		for _, attr := range span.Attributes() {
			var value any

			// Attributes come in some different types, so handle them for
			// prettier logging
			switch attr.Value.Type() {
			case attribute.BOOL:
				value = attr.Value.AsBool()
			case attribute.INT64:
				value = attr.Value.AsInt64()
			case attribute.FLOAT64:
				value = attr.Value.AsFloat64()
			case attribute.STRING:
				value = attr.Value.AsString()
			case attribute.BOOLSLICE:
				value = attr.Value.AsBoolSlice()
			case attribute.STRINGSLICE:
				value = attr.Value.AsStringSlice()
			case attribute.INT64SLICE:
				value = attr.Value.AsInt64Slice()
			case attribute.FLOAT64SLICE:
				value = attr.Value.AsFloat64Slice()
			case attribute.INVALID:
				// Fallback to using attr.Value directly
				value = attr.Value
			}

			fields = append(fields, zap.Any(string(attr.Key), value))
		}

		// Log as debug value
		e.logger.Debug(span.Name(), fields...)
	}

	return nil
}

func (e *loggingTraceExporter) Shutdown(ctx context.Context) error { //nolint:revive
	return nil
}
