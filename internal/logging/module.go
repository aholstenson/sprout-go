package logging

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides logging bindings based on a zap.Logger root logger.
func Module(logger *zap.Logger) fx.Option {
	return fx.Module(
		"sprout:logging",
		fx.Provide(fx.Annotate(func(lifecycle fx.Lifecycle) *zap.Logger {
			lifecycle.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					_ = logger.Sync()
					return nil
				},
			})

			return logger
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
