package health

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/alexliesenfeld/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *zap.Logger
	Config    Config

	LivenessChecks  []Check `group:"health.liveness"`
	ReadinessChecks []Check `group:"health.readiness"`
}

type Config struct {
	// Port is the port to bind to
	Port int `env:"PORT" envDefault:"8088"`
}

func server(params Params) {
	var httpServer *http.Server
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			mux := &http.ServeMux{}
			mux.HandleFunc(
				"/healthz",
				health.NewHandler(newChecker(
					params.Logger.With(zap.String("type", "liveness")),
					params.LivenessChecks,
				),
				))
			mux.HandleFunc(
				"/readyz",
				health.NewHandler(newChecker(
					params.Logger.With(zap.String("type", "readiness")),
					params.ReadinessChecks,
				),
				))

			ln, err := net.Listen("tcp", ":"+strconv.Itoa(params.Config.Port))
			if err != nil {
				return err
			}

			httpServer = &http.Server{
				Handler: mux,
			}

			params.Logger.Info("Starting health server", zap.Int("port", params.Config.Port))
			go func() {
				err2 := httpServer.Serve(ln)
				if err2 != nil && err2 != http.ErrServerClosed {
					params.Logger.Error("Error starting health server", zap.Error(err2))
				}
			}()
			return nil
		},

		OnStop: func(ctx context.Context) error {
			return httpServer.Shutdown(ctx)
		},
	})
}

func newChecker(logger *zap.Logger, checks []Check) health.Checker {
	options := []health.CheckerOption{
		health.WithTimeout(5 * time.Second),
		health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
			if state.Status == health.StatusDown {
				logger.Info("Health status changed", zap.String("state", "down"))
			} else if state.Status == health.StatusUp {
				logger.Info("Health status changed", zap.String("state", "up"))
			}
		}),
		health.WithInterceptors(func(next health.InterceptorFunc) health.InterceptorFunc {
			return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
				currentStatus := state.Status
				result := next(ctx, name, state)

				if currentStatus != result.Status {
					if result.Status == health.StatusUp {
						logger.Info("Health check marked as healthy", zap.String("name", name))
					} else if result.Status == health.StatusDown {
						logger.Info("Health check marked as unhealthy", zap.String("name", name))
					}
				}
				return result
			}
		}),
	}

	for _, check := range checks {
		options = append(options, health.WithCheck(check))
	}

	return health.NewChecker(options...)
}
