package sprout

import (
	"github.com/levelfourab/sprout-go/internal/logging"
	"go.uber.org/zap"
)

// CreateLogger creates a logger with a name. This is provided as an
// alternative to defining loggers in your module with fx.Provide via
// sprout.Logger.
func CreateLogger(name ...string) *zap.Logger {
	return logging.CreateLogger(zap.L(), name)
}

// Logger creates a function that can be used to create a logger with a name.
// Can be used with fx.Decorate or fx.Provide.
//
// It is mostly used when creating a module, to supply a logger with a name
// that is specific to the module.
//
// Example:
//
//	var Module = fx.Module(
//		"example",
//		fx.Provide(sprout.Logger("name", "of", "logger"), fx.PRivate),
//		fx.Invoke(func(logger *zap.Logger) {
//			// ...
//		}),
//	)
func Logger(name ...string) any {
	return logging.Logger(name...)
}

// SugaredLogger creates a function that can be used to create a sugared logger
// with a name. Can be used with fx.Decorate or fx.Provide.
//
// It is mostly used when creating a module, to supply a sugared logger with a
// name that is specific to the module.
//
// Example:
//
//	var Module = fx.Module(
//		"example",
//		fx.Provide(sprout.SugaredLogger("name", "of", "logger"), fx.Private),
//		fx.Invoke(func(logger *zap.SugaredLogger) {
//			// ...
//		}),
//	)
func SugaredLogger(name ...string) any {
	return logging.SugaredLogger(name...)
}

// LogrLogger creates a function that can be used to create a logr logger with
// a name. Can be used with fx.Decorate or fx.Provide.
//
// It is mostly used when creating a module, to supply a logr logger with a
// name that is specific to the module.
//
// Example:
//
//	var Module = fx.Module(
//		"example",
//		fx.Provide(sprout.LogrLogger("name", "of", "logger"), fx.Private),
//		fx.Invoke(func(logger logr.Logger) {
//			// ...
//		}),
//	)
func LogrLogger(name ...string) any {
	return logging.LogrLogger(name...)
}

// SlogLogger creates a function that can be used to create a log/slog logger
// with a name. Can be used with fx.Decorate or fx.Provide.
//
// It is mostly used when creating a module, to supply a log/slog logger to
// libraries that require it.
//
// Example:
//
//	var Module = fx.Module(
//		"example",
//		fx.Provide(sprout.SlogLogger("name", "of", "logger"), fx.Private),
//		fx.Invoke(func(logger *slog.Logger) {
//			// ...
//		}),
//	)
func SlogLogger(name ...string) any {
	return logging.SlogLogger(name...)
}
