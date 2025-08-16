package otel

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/fx"
)

type ServiceInfo struct {
	fx.In

	Name    string `name:"service:name"`
	Version string `name:"service:version"`

	Development bool `name:"env:development"`
	Testing     bool `name:"env:testing"`
}

func CreateResource(service ServiceInfo) (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(service.Name),
			semconv.ServiceVersionKey.String(service.Version),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
}

type module string

const (
	moduleTracing module = "tracing"
	moduleMetrics module = "metrics"
	moduleLogging module = "logging"
)

func hasExporterEndpoint(module module) bool {
	// OTEL_EXPORTER_OTLP_ENDPOINT is the default environment variable
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		return true
	}

	switch module {
	case moduleTracing:
		// Tracing also uses OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
		return os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
	case moduleMetrics:
		// Metrics uses OTEL_EXPORTER_OTLP_METRICS_ENDPOINT
		return os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != ""
	case moduleLogging:
		// Logging uses OTEL_EXPORTER_OTLP_LOGS_ENDPOINT
		return os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT") != ""
	}

	return false
}
