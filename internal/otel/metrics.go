package otel

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/fx"
)

func setupMetrics(service ServiceInfo, resource *resource.Resource, lifecycle fx.Lifecycle) (metric.MeterProvider, error) {
	if service.Development || service.Testing {
		// If running in development or testing mode, we don't want to send
		// any metrics to the collector
		provider := metric.NewNoopMeterProvider()
		global.SetMeterProvider(provider)
		return provider, nil
	}

	exporter, err := otlpmetricgrpc.New(context.Background())
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(resource),
	)
	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return provider.Shutdown(ctx)
		},
	})

	return provider, nil
}
