package sprout

import (
	"log/slog"
	"os"

	"github.com/aholstenson/sprout-go/internal"
	"github.com/aholstenson/sprout-go/internal/health"
	"github.com/aholstenson/sprout-go/internal/logging"
	"github.com/aholstenson/sprout-go/internal/runtime"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

type Sprout struct {
	rootLogger *logging.RootLogger

	serviceInfo internal.ServiceInfo
}

// New creates a new Sprout application. The name and version will be used to
// identify the application in logs, traces and metrics.
func New(name string, version string) *Sprout {
	serviceInfo := internal.ServiceInfo{
		Name:        name,
		Version:     version,
		Development: internal.CheckIfDevelopment(),
		Testing:     false,
	}

	rootLogger, err := logging.CreateRootLogger(serviceInfo)
	if err != nil {
		_, _ = os.Stderr.WriteString("Unable to bootstrap: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := rootLogger.Logger()
	zap.ReplaceGlobals(logger)

	// Integrate with log/slog
	slogLogger := slog.New(zapslog.NewHandler(logger.Core()))
	slog.SetDefault(slogLogger)

	// Continue bootstrapping
	logger.Info("Starting application", zap.String("name", name), zap.String("version", version))
	runtime.Setup(logger)
	return &Sprout{
		rootLogger:  rootLogger,
		serviceInfo: serviceInfo,
	}
}

// With lets you specify Fx options to be used when creating the application.
func (s *Sprout) With(options ...fx.Option) *fx.App {
	logger := s.rootLogger.Logger()

	allOptions := []fx.Option{
		fx.WithLogger(func() fxevent.Logger {
			fxLogLevel := os.Getenv("FX_ENABLE_DETAILED_LOGGING")
			return logging.NewFxLogger(logger, fxLogLevel == "true")
		}),
		fx.Supply(s.serviceInfo),
		s.rootLogger.Module(),
		otelModule,
		health.Module,
	}

	allOptions = append(allOptions, options...)
	allOptions = append(allOptions, fx.Invoke(enableHealthServer))
	return fx.New(allOptions...)
}

func enableHealthServer(checks Health) {
	// Do nothing, only here to make health server always start
}
