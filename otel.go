package sprout

import (
	"github.com/aholstenson/sprout-go/internal/logging"
	sproutotel "github.com/aholstenson/sprout-go/internal/otel"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// otelModule provides OpenTelemetry integration for Fx.
var otelModule = fx.Module(
	"sprout:otel",
	fx.Provide(logging.LogrLogger("otel"), fx.Private),
	fx.Provide(logging.Logger("otel"), fx.Private),
	fx.Invoke(func(logger logr.Logger) {
		otel.SetLogger(logger)

		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
			logger.Error(err, "OpenTelemetry error")
		}))
	}),
	fx.Provide(sproutotel.CreateResource),
	fx.Provide(sproutotel.SetupTracing),
	fx.Provide(sproutotel.SetupMetrics),
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
