package logging

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RootLogger is the root logger of the application together with the things it
// writes to that must be released when the application stops.
type RootLogger struct {
	logger   *zap.Logger
	shutdown func(ctx context.Context) error
}

// NewRootLogger creates a RootLogger for an existing logger. The shutdown
// function releases what the logger writes to and may be nil.
func NewRootLogger(logger *zap.Logger, shutdown func(ctx context.Context) error) *RootLogger {
	return &RootLogger{
		logger:   logger,
		shutdown: shutdown,
	}
}

// Logger returns the zap.Logger to use as the root of the application.
func (r *RootLogger) Logger() *zap.Logger {
	return r.logger
}

// Module returns the Fx module that provides the logging bindings. When the
// application stops the module flushes the logger and then releases what it
// writes to.
func (r *RootLogger) Module() fx.Option {
	return fx.Module(
		"sprout:logging",
		fx.Provide(fx.Annotate(func(lifecycle fx.Lifecycle) *zap.Logger {
			lifecycle.Append(fx.Hook{
				OnStop: r.stop,
			})

			return r.logger
		}, fx.ResultTags(`name:"logging.zap"`))),
		fx.Provide(fx.Annotate(
			func(logger *zap.Logger) logr.Logger {
				return zapr.NewLogger(logger)
			},
			fx.ParamTags(`name:"logging.zap"`),
			fx.ResultTags(`name:"logging.logr"`),
		)),
	)
}

// stop flushes what the logger has buffered and then releases the things it
// writes to. The order matters, as records that are still buffered would
// otherwise be lost.
func (r *RootLogger) stop(ctx context.Context) error {
	_ = r.logger.Sync()

	if r.shutdown == nil {
		return nil
	}

	return r.shutdown(ctx)
}

// Module provides logging bindings based on a zap.Logger root logger. Use it
// when the logger writes to nothing that needs to be released, such as in
// tests.
func Module(logger *zap.Logger) fx.Option {
	return NewRootLogger(logger, nil).Module()
}
