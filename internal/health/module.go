package health

import (
	"github.com/alexliesenfeld/health"
	"github.com/levelfourab/sprout-go/internal/config"
	"github.com/levelfourab/sprout-go/internal/logging"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"sprout:health",
	fx.Provide(config.Config("HEALTH_SERVER", Config{}), fx.Private),
	fx.Provide(logging.Logger("health"), fx.Private),
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
