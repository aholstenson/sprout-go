package health

import (
	"github.com/levelfourab/sprout-go/internal/config"
	"github.com/levelfourab/sprout-go/internal/logging"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"sprout:health",
	fx.Provide(config.Config("HEALTH_SERVER", Config{}), fx.Private),
	fx.Provide(logging.Logger("health"), fx.Private),
	fx.Provide(fx.Annotate(NewServer, fx.As(new(Checks)))),
)
