package otel

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/levelfourab/sprout-go/internal/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"sprout:otel",
	fx.Provide(logging.LogrLogger("otel"), fx.Private),
	fx.Invoke(func(logger logr.Logger) {
		otel.SetLogger(logger)

		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
			logger.Error(err, "OpenTelemetry error")
		}))
	}),
	fx.Provide(createResource),
	fx.Provide(setupTracing),
	fx.Provide(setupMetrics),
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
