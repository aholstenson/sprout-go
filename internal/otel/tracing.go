package otel

import (
	"context"

	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// setupTracing configures OpenTelemetry tracing.
func setupTracing(service ServiceInfo, resource *resource.Resource, lifecycle fx.Lifecycle) (trace.TracerProvider, error) {
	// Set a better default for propagators, since the default is a no-op
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	if service.Development || service.Testing {
		// If running in development or testing mode, we don't want to send
		// any traces to the collector. Instead, we use a no-op tracer provider.
		provider := trace.NewNoopTracerProvider()
		otel.SetTracerProvider(provider)
		return provider, nil
	}

	// TODO: Load config and error if not set correctly
	exporter := otlptracegrpc.NewUnstarted()
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return exporter.Start(ctx)
		},
	})

	// TODO: Support for sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
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
