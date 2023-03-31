package sprout

import "github.com/levelfourab/sprout-go/internal/logging"

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
//		fx.Invoke(func(logger logr.Logger) {
//			// ...
//		}),
//	)
func Logger(name ...string) any {
	return logging.Logger(name...)
}
