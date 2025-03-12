package main

import (
	"context"

	"github.com/aholstenson/sprout-go"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Conf struct {
	Name string `env:"NAME"`
}

func main() {
	sprout.New("example", "v0.0.0").With(
		Module,
	).Run()
}

var Module = fx.Module(
	"example",
	fx.Provide(sprout.Config("", Conf{}), fx.Private),
	fx.Provide(sprout.Logger("example"), fx.Private),

	fx.Invoke(func(checks sprout.Health, logger *zap.Logger) {
		checks.AddLivenessCheck(sprout.HealthCheck{
			Name: "example",
			Check: func(ctx context.Context) error {
				logger.Info("Checked health")
				return nil
			},
		})
	}),

	fx.Provide(sprout.Tracer("example")),
	fx.Invoke(func(logger *zap.Logger, conf Conf, tracer trace.Tracer) {
		_, span := tracer.Start(context.Background(), "example")
		defer span.End()
		logger.Info("Hello world", zap.String("name", conf.Name))
	}),
)
