package otel

import (
	"context"

	"github.com/aholstenson/sprout-go/internal"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// InitLogging initializes OpenTelemetry log exporting if configured.
//
// The returned function flushes the records that are still buffered and then
// releases the exporter. It is nil when no exporter endpoint is configured,
// in which case the returned provider discards all records.
func InitLogging(
	serviceInfo internal.ServiceInfo,
) (log.LoggerProvider, func(ctx context.Context) error, error) {
	if !hasExporterEndpoint(moduleLogging) {
		return noop.NewLoggerProvider(), nil, nil
	}

	resource, err := CreateResource(ServiceInfo{
		Name:        serviceInfo.Name,
		Version:     serviceInfo.Version,
		Development: serviceInfo.Development,
		Testing:     serviceInfo.Testing,
	})
	if err != nil {
		return nil, nil, err
	}

	exporter, err := otlploggrpc.New(context.Background())
	if err != nil {
		return nil, nil, err
	}

	options := []sdklog.LoggerProviderOption{
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	}

	provider := sdklog.NewLoggerProvider(options...)
	global.SetLoggerProvider(provider)

	return provider, provider.Shutdown, nil
}
