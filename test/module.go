package test

import (
	"github.com/aholstenson/sprout-go/internal"
	"github.com/aholstenson/sprout-go/internal/health"
	"github.com/aholstenson/sprout-go/internal/logging"
	"go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TB is the interface that testing.T and testing.B implement.
type TB interface {
	Logf(string, ...interface{})
	Errorf(string, ...interface{})
	Fail()
	Failed() bool
	Name() string
	FailNow()
}

// Option changes how Module sets up the test application.
type Option func(*options)

type options struct {
	serviceName    string
	serviceVersion string
	noopTelemetry  bool
}

// WithServiceName sets the service name of the test application. The default
// is "test".
func WithServiceName(name string) Option {
	return func(o *options) {
		o.serviceName = name
	}
}

// WithServiceVersion sets the service version of the test application. The
// default is "dev".
func WithServiceVersion(version string) Option {
	return func(o *options) {
		o.serviceVersion = version
	}
}

// WithNoopTelemetry makes Module use providers that do nothing for traces and
// metrics. Use it for benchmarks, as the span recorder keeps all spans in
// memory. With this option Module does not provide the
// *tracetest.SpanRecorder and the *sdkmetric.ManualReader.
func WithNoopTelemetry() Option {
	return func(o *options) {
		o.noopTelemetry = true
	}
}

// Module provides an Fx module that can be used to test Sprout applications.
// This will enable logging, tracing and metrics.
//
// Example:
//
//	app := fxtest.New(
//		t,
//		test.Module(testingT), // use testing.T or testing.B
//		otherModules,
//	)
//	app.RequireStart()
//
// Module records telemetry in memory, so that a test can examine it. Get the
// recorders with fx.Populate:
//
//   - *tracetest.SpanRecorder records all spans. Ended returns the spans that
//     have ended.
//   - *sdkmetric.ManualReader collects the metrics when the test calls
//     Collect.
//
// Each call to Module creates new providers and recorders. Module does not
// change the OpenTelemetry globals, so tests that run in parallel do not
// share telemetry. Code that gets a tracer or a meter from the globals, such
// as with otel.Tracer, does not send telemetry to the recorders. Use the
// trace.TracerProvider and metric.MeterProvider from Fx instead.
//
// Logs are not recorded. The log.LoggerProvider does nothing.
func Module(t TB, opts ...Option) fx.Option {
	o := options{
		serviceName:    "test",
		serviceVersion: "dev",
	}
	for _, opt := range opts {
		opt(&o)
	}

	logger := zaptest.NewLogger(t)
	return fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			logger := &fxevent.ZapLogger{Logger: logger.Named("fx")}
			logger.UseLogLevel(zap.DebugLevel)
			return logger
		}),
		fx.Supply(internal.ServiceInfo{
			Name:        o.serviceName,
			Version:     o.serviceVersion,
			Development: internal.CheckIfDevelopment(),
			Testing:     true,
		}),
		logging.Module(logger),
		telemetryOptions(o),
		fx.Provide(func() log.LoggerProvider {
			return lognoop.NewLoggerProvider()
		}),
		health.Module,
	)
}

// telemetryOptions returns the options that provide the tracer and meter
// providers.
func telemetryOptions(o options) fx.Option {
	if o.noopTelemetry {
		return fx.Provide(
			func() trace.TracerProvider {
				return tracenoop.NewTracerProvider()
			},
			func() metric.MeterProvider {
				return metricnoop.NewMeterProvider()
			},
		)
	}

	res := resource.NewSchemaless(
		semconv.ServiceNameKey.String(o.serviceName),
		semconv.ServiceVersionKey.String(o.serviceVersion),
	)

	// The providers are not shut down when the application stops. This lets a
	// test examine the telemetry after it has stopped the application. The
	// providers start no goroutines, so nothing leaks.
	return fx.Provide(
		tracetest.NewSpanRecorder,
		func(recorder *tracetest.SpanRecorder) trace.TracerProvider {
			return sdktrace.NewTracerProvider(
				sdktrace.WithResource(res),
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithSpanProcessor(recorder),
			)
		},
		func() *sdkmetric.ManualReader {
			return sdkmetric.NewManualReader()
		},
		func(reader *sdkmetric.ManualReader) metric.MeterProvider {
			return sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(res),
				sdkmetric.WithReader(reader),
			)
		},
	)
}
