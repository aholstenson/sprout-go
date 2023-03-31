package health

import (
	"github.com/alexliesenfeld/health"
	"github.com/go-logr/logr"
	"github.com/levelfourab/sprout-go/internal/config"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"sprout:health",
	fx.Provide(config.Config("HEALTH_SERVER", Config{})),
	fx.Decorate(func(logger logr.Logger) logr.Logger {
		return logger.WithName("health")
	}),
	fx.Invoke(server),
)

type Check = health.Check

func AsLivenessCheck(f any) any {
	return fx.Annotate(
		f,
		fx.ResultTags(`group:"health.liveness"`),
	)
}

func AsReadinessCheck(f any) any {
	return fx.Annotate(
		f,
		fx.ResultTags(`group:"health.readiness"`),
	)
}
