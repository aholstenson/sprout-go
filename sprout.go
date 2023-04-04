package sprout

import (
	"os"

	"github.com/levelfourab/sprout-go/internal"
	"github.com/levelfourab/sprout-go/internal/health"
	"github.com/levelfourab/sprout-go/internal/logging"
	"github.com/levelfourab/sprout-go/internal/otel"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type Sprout struct {
	logger *zap.Logger

	name    string
	version string
}

// New creates a new Sprout application. The name and version will be used to
// identify the application in logs, traces and metrics.
func New(name string, version string) *Sprout {
	logger, err := logging.CreateRootLogger()
	if err != nil {
		os.Stderr.WriteString("Unable to bootstrap: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger.Info("Starting application", zap.String("name", name), zap.String("version", version))
	return &Sprout{
		logger:  logger,
		name:    name,
		version: version,
	}
}

// With lets you specify Fx options to be used when creating the application.
func (s *Sprout) With(options ...fx.Option) *fx.App {
	logger := s.logger
	coreModules := []fx.Option{
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger.Named("fx")}
		}),
		fx.Supply(internal.ServiceInfo{
			Name:        s.name,
			Version:     s.version,
			Development: internal.CheckIfDevelopment(),
			Testing:     false,
		}),
		logging.Module(logger),
		otel.Module,
		health.Module,
	}

	return fx.New(append(coreModules, options...)...)
}
