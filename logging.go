package sprout

import (
	"github.com/levelfourab/sprout-go/internal/logging"
)

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
//		fx.Decorate(sprout.Logger("name", "of", "logger")),
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
//		fx.Decorate(sprout.SugaredLogger("name", "of", "logger")),
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
//		fx.Decorate(sprout.LogrLogger("name", "of", "logger")),
//		fx.Invoke(func(logger logr.Logger) {
//			// ...
//		}),
//	)
func LogrLogger(name ...string) any {
	return logging.LogrLogger(name...)
}
