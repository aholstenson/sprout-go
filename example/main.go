package main

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/levelfourab/sprout-go"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
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
	fx.Provide(sprout.Config("", Conf{})),
	fx.Decorate(sprout.Logger("example")),

	fx.Provide(sprout.AsLivenessCheck(func(logger logr.Logger) sprout.HealthCheck {
		return sprout.HealthCheck{
			Name: "example",
			Check: func(ctx context.Context) error {
				logger.V(1).Info("Checked health")
				return nil
			},
		}
	})),

	fx.Provide(sprout.Tracer("example")),
	fx.Invoke(func(logger logr.Logger, conf Conf, tracer trace.Tracer) {
		_, span := tracer.Start(context.Background(), "example")
		defer span.End()
		logger.Info("Hello world", "name", conf.Name)
	}),
)
