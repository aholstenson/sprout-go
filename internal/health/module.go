package health

import (
	"github.com/aholstenson/sprout-go/internal/config"
	"github.com/aholstenson/sprout-go/internal/logging"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"sprout:health",
	fx.Provide(config.Config("HEALTH_SERVER", Config{}), fx.Private),
	fx.Provide(logging.Logger("health"), fx.Private),
	// The server is provided as itself so that the application can control it
	// when it stops, and as Checks for the modules that add checks.
	fx.Provide(NewServer),
	fx.Provide(func(server *Server) Checks { return server }),
)
