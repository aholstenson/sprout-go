package otel

import (
	"context"

	"github.com/aholstenson/sprout-go/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// InitLogging initializes OpenTelemetry log exporting if configured.
//
// It returns nil when no log exporter is configured. Otherwise the caller owns
// the returned provider: ForceFlush exports the records that are still
// buffered, and Shutdown exports them and then releases the exporters.
func InitLogging(
	serviceInfo internal.ServiceInfo,
) (*sdklog.LoggerProvider, error) {
	config, err := readSignalConfig(moduleLogging, false)
	if err != nil {
		return nil, err
	}

	if !config.enabled() {
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

	options := []sdklog.LoggerProviderOption{
		sdklog.WithResource(resource),
	}

	if config.OTLP {
		exporter, err := newLogExporter(context.Background(), config.Protocol)
		if err != nil {
			return nil, err
		}

		options = append(options, sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)))
	}

	if config.Console {
		// The console exporter writes each record as one line of JSON to
		// stdout. The simple processor writes the record immediately.
		exporter, err := stdoutlog.New()
		if err != nil {
			return nil, err
		}

		options = append(options, sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)))
	}

	provider := sdklog.NewLoggerProvider(options...)
	otel.SetLoggerProvider(provider)

	return provider, nil
}
