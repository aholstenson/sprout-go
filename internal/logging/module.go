package logging

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides logging bindings based on a zap.Logger root logger.
func Module(logger *zap.Logger) fx.Option {
	return fx.Module(
		"sprout:logging",
		fx.Provide(fx.Annotate(func() *zap.Logger {
			return logger
		}, fx.ResultTags(`name:"logging.zap"`))),
		fx.Provide(fx.Annotate(func(logger *zap.Logger) logr.Logger {
			return zapr.NewLogger(logger)
		}, fx.ParamTags(`name:"logging.zap"`), fx.ResultTags(`name:"logging.logr"`))),
	)
}
