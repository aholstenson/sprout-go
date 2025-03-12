package otel

import (
	"context"
	"os"

	"github.com/aholstenson/sprout-go/internal/config"
	"github.com/aholstenson/sprout-go/internal/logging"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"sprout:otel",
	fx.Provide(logging.LogrLogger("otel"), fx.Private),
	fx.Provide(logging.Logger("otel"), fx.Private),
	fx.Provide(config.Config("OTEL_TRACING", traceConfig{}), fx.Private),
	fx.Invoke(func(logger logr.Logger) {
		otel.SetLogger(logger)

		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
			logger.Error(err, "OpenTelemetry error")
		}))
	}),
	fx.Provide(createResource),
	fx.Provide(setupTracing),
	fx.Provide(setupMetrics),
	fx.Invoke(func(rp trace.TracerProvider, mp metric.MeterProvider) error {
		// This method depends on the tracer and meter providers which forces
		// them to be created.

		// Register runtime metric collection.
		err := runtime.Start(runtime.WithMeterProvider(mp))
		if err != nil {
			return err
		}

		return nil
	}),
)

type ServiceInfo struct {
	fx.In

	Name    string `name:"service:name"`
	Version string `name:"service:version"`

	Development bool `name:"env:development"`
	Testing     bool `name:"env:testing"`
}

func createResource(service ServiceInfo) (*resource.Resource, error) {
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
