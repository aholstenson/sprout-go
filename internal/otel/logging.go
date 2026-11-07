package otel

import (
	"context"

	"github.com/aholstenson/sprout-go/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// InitLogging initializes OpenTelemetry log exporting if configured.
//
// It returns nil when no exporter endpoint is configured. Otherwise the caller
// owns the returned provider: ForceFlush exports the records that are still
// buffered, and Shutdown exports them and then releases the exporter.
func InitLogging(
	serviceInfo internal.ServiceInfo,
) (*sdklog.LoggerProvider, error) {
	if !hasExporterEndpoint(moduleLogging) {
		return nil, nil //nolint:nilnil // A nil provider means that exporting is not configured.
	}

	resource, err := CreateResource(ServiceInfo{
		Name:        serviceInfo.Name,
		Version:     serviceInfo.Version,
		Development: serviceInfo.Development,
		Testing:     serviceInfo.Testing,
	})
	if err != nil {
		return nil, err
	}

	exporter, err := otlploggrpc.New(context.Background())
	if err != nil {
		return nil, err
	}

	options := []sdklog.LoggerProviderOption{
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	}

	provider := sdklog.NewLoggerProvider(options...)
	otel.SetLoggerProvider(provider)

	return provider, nil
}
