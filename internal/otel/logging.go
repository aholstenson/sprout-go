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
func InitLogging(
	serviceInfo internal.ServiceInfo,
) (log.LoggerProvider, bool, error) {
	if !hasExporterEndpoint(moduleLogging) {
		return noop.NewLoggerProvider(), false, nil
	}

	resource, err := CreateResource(ServiceInfo{
		Name:        serviceInfo.Name,
		Version:     serviceInfo.Version,
		Development: serviceInfo.Development,
		Testing:     serviceInfo.Testing,
	})
	if err != nil {
		return nil, false, err
	}

	processor, err := otlploggrpc.New(context.Background())
	if err != nil {
		return nil, false, err
	}

	options := []sdklog.LoggerProviderOption{
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(processor)),
	}

	provider := sdklog.NewLoggerProvider(options...)
	global.SetLoggerProvider(provider)

	return provider, true, nil
}
