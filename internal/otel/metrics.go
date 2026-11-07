package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// SetupMetrics configures OpenTelemetry metrics.
func SetupMetrics(
	resource *resource.Resource,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) (metric.MeterProvider, error) {
	config, err := readSignalConfig(moduleMetrics, false)
	if err != nil {
		return nil, err
	}

	if !config.enabled() {
		logger.Warn("Metrics disabled", zap.String("reason", config.DisabledReason))
		return noopMetrics()
	}

	options := []sdkmetric.Option{
		sdkmetric.WithResource(resource),
	}

	if config.OTLP {
		exporter, err := newMetricExporter(context.Background(), config.Protocol)
		if err != nil {
			return nil, err
		}

		options = append(options, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)))
	}

	if config.Console {
		// The console exporter writes each collection as one line of JSON to
		// stdout.
		exporter, err := stdoutmetric.New()
		if err != nil {
			return nil, err
		}

		options = append(options, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)))
	}

	logger.Info(
		"Metrics enabled",
		zap.Bool("otlp", config.OTLP),
		zap.String("protocol", string(config.Protocol)),
		zap.Bool("console", config.Console),
	)
	provider := sdkmetric.NewMeterProvider(options...)
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
