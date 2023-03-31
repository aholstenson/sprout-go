package logging

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides logging. It will initialize a logger based on the
// environment, such as using a pretty console logger in development mode.
var Module = fx.Module(
	"sprout:logging",
	fx.Provide(func(logger *zap.Logger) logr.Logger {
		return zapr.NewLogger(logger)
	}),
)
