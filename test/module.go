package test

import (
	"github.com/aholstenson/sprout-go/internal"
	"github.com/aholstenson/sprout-go/internal/health"
	"github.com/aholstenson/sprout-go/internal/logging"
	"go.opentelemetry.io/otel"
	logglobal "go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
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
func Module(t TB) fx.Option {
	logger := zaptest.NewLogger(t)
	return fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			logger := &fxevent.ZapLogger{Logger: logger.Named("fx")}
			logger.UseLogLevel(zap.DebugLevel)
			return logger
		}),
		fx.Supply(internal.ServiceInfo{
			Name:        "test",
			Version:     "dev",
			Development: internal.CheckIfDevelopment(),
			Testing:     true,
		}),
		logging.Module(logger),
		fx.Invoke(setupNoopOtel),
		fx.Provide(otel.GetTracerProvider),
		fx.Provide(otel.GetMeterProvider),
		fx.Provide(logglobal.GetLoggerProvider),
		health.Module,
	)
}

func setupNoopOtel() {
	otel.SetTracerProvider(tracenoop.NewTracerProvider())
	otel.SetMeterProvider(metricnoop.NewMeterProvider())
	logglobal.SetLoggerProvider(lognoop.NewLoggerProvider())
}
