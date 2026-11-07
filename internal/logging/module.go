package logging

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// releaseTimeout limits how long the release of the outputs can take when the
// application has ended.
const releaseTimeout = 5 * time.Second

// Outputs are the operations on the things a root logger writes to that need
// more than a Sync of the logger, such as an exporter that buffers records.
type Outputs struct {
	// Flush writes what the outputs have buffered. It may be nil.
	Flush func(ctx context.Context) error

	// Release writes what the outputs have buffered and then releases them.
	// After it, the outputs drop what is written to them. It may be nil.
	Release func(ctx context.Context) error
}

// RootLogger is the root logger of the application together with the things it
// writes to.
//
// The outputs are released when the application has ended, which is after
// Fx has logged its last event. The Fx logger from FxLogger does this, as an
// Fx lifecycle hook runs too early: when a start fails, Fx stops the started
// hooks before it logs the failure, so the failure would not be exported.
type RootLogger struct {
	logger  *zap.Logger
	outputs Outputs

	releaseOnce sync.Once
	releaseErr  error
}

// NewRootLogger creates a RootLogger for an existing logger.
func NewRootLogger(logger *zap.Logger, outputs Outputs) *RootLogger {
	return &RootLogger{
		logger:  logger,
		outputs: outputs,
	}
}

// Logger returns the zap.Logger to use as the root of the application.
func (r *RootLogger) Logger() *zap.Logger {
	return r.logger
}

// Module returns the Fx module that provides the logging bindings. When the
// application stops the module flushes the logger. It does not release the
// outputs, see FxLogger.
func (r *RootLogger) Module() fx.Option {
	return fx.Module(
		"sprout:logging",
		fx.Provide(fx.Annotate(func(lifecycle fx.Lifecycle) *zap.Logger {
			lifecycle.Append(fx.Hook{
				OnStop: r.Flush,
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

// FxLogger returns the logger for Fx events. After it has logged the last
// event of the application, which is a failed start or a stop, it releases the
// outputs of the root logger.
//
// When the application uses a different Fx logger, the outputs are not
// released. The flush when the application stops then still writes most
// records.
func (r *RootLogger) FxLogger(detailedLogging bool) fxevent.Logger {
	logger := newFxLogger(r.logger, detailedLogging)
	logger.onEnd = r.releaseAtEnd
	return logger
}

// Flush flushes what the logger has buffered and then what the outputs have
// buffered. The order matters, as records that are still buffered in the
// logger would otherwise not reach the outputs.
func (r *RootLogger) Flush(ctx context.Context) error {
	_ = r.logger.Sync()

	if r.outputs.Flush == nil {
		return nil
	}

	return r.outputs.Flush(ctx)
}

// Release flushes the logger and then releases the outputs. Only the first
// call releases them, later calls return the result of the first.
func (r *RootLogger) Release(ctx context.Context) error {
	r.releaseOnce.Do(func() {
		_ = r.logger.Sync()

		if r.outputs.Release != nil {
			r.releaseErr = r.outputs.Release(ctx)
		}
	})

	return r.releaseErr
}

// releaseAtEnd releases the outputs when the application has ended. The
// failure goes to stderr, as the outputs of the logger can be released.
func (r *RootLogger) releaseAtEnd() {
	ctx, cancel := context.WithTimeout(context.Background(), releaseTimeout)
	defer cancel()

	err := r.Release(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to release log outputs: %v\n", err)
	}
}

// Module provides logging bindings based on a zap.Logger root logger. Use it
// when the logger writes to nothing that needs to be released, such as in
// tests.
func Module(logger *zap.Logger) fx.Option {
	return NewRootLogger(logger, Outputs{}).Module()
}
