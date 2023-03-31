package test

import (
	"github.com/levelfourab/sprout-go/internal"
	"github.com/levelfourab/sprout-go/internal/logging"
	"github.com/levelfourab/sprout-go/internal/otel"
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
	return fx.Options(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log.Named("fx")}
		}),
		fx.Supply(zaptest.NewLogger(t)),
		fx.Supply(internal.ServiceInfo{
			Name:        "test",
			Version:     "dev",
			Development: internal.CheckIfDevelopment(),
			Testing:     true,
		}),
		logging.Module,
		otel.Module,
	)
}
