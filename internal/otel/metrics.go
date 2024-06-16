package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func setupMetrics(
	service ServiceInfo,
	resource *resource.Resource,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) (metric.MeterProvider, error) {
	if !hasExporterEndpoint(false) {
		// If no endpoint is set, we don't want to send any metrics to the
		// collector
		logger.Warn("No metrics exporter endpoint set, disabling metrics")
		return noopMetrics()
	}

	exporter, err := otlpmetricgrpc.New(context.Background())
	if err != nil {
		return nil, err
	}

	logger.Info("Metrics enabled")
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(resource),
	)
	otel.SetMeterProvider(provider)

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return provider.Shutdown(ctx)
		},
	})

	return provider, nil
}

func noopMetrics() (metric.MeterProvider, error) {
	provider := noop.NewMeterProvider()
	otel.SetMeterProvider(provider)
	return provider, nil
}
