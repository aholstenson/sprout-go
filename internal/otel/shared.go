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

func hasExporterEndpoint(tracing bool) bool {
	// OTEL_EXPORTER_OTLP_ENDPOINT is the default environment variable
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		return true
	}

	if tracing {
		// Tracing also uses OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
		return os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
	}

	// Metrics uses OTEL_EXPORTER_OTLP_METRICS_ENDPOINT
	return os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != ""
}
