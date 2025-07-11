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

type Config struct {
	// Port is the port to bind to
	Port int `env:"PORT" envDefault:"8088"`
}

type Server struct {
	logger *zap.Logger

	httpListener net.Listener
	httpServer   *http.Server
	httpPort     int

	livenessChecks  []Check
	readinessChecks []Check
}

func NewServer(lifecycle fx.Lifecycle, logger *zap.Logger, config Config) Checks {
	s := &Server{
		logger:   logger,
		httpPort: config.Port,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return s.Start()
		},
		OnStop: func(ctx context.Context) error {
			return s.Stop(ctx)
		},
	})
	return s
}

func (s *Server) AddLivenessCheck(check Check) {
	s.livenessChecks = append(s.livenessChecks, check)
}

func (s *Server) AddReadinessCheck(check Check) {
	s.readinessChecks = append(s.readinessChecks, check)
}

func (s *Server) Start() error {
	s.logger.Info("Starting health server", zap.Int("port", s.httpPort))

	mux := &http.ServeMux{}
	mux.HandleFunc(
		"/healthz",
		health.NewHandler(newChecker(
			s.logger.With(zap.String("type", "liveness")),
			s.livenessChecks,
		)),
	)
	mux.HandleFunc(
		"/readyz",
		health.NewHandler(newChecker(
			s.logger.With(zap.String("type", "readiness")),
			s.readinessChecks,
		)),
	)

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(s.httpPort))
	if err != nil {
		return err
	}

	s.httpListener = ln

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	go func() {
		err2 := s.httpServer.Serve(ln)
		if err2 != nil && err2 != http.ErrServerClosed {
			s.logger.Error("Error starting health server", zap.Error(err2))
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping health server")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

func newChecker(logger *zap.Logger, checks []Check) health.Checker {
	options := []health.CheckerOption{
		health.WithTimeout(5 * time.Second),
		health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
			switch state.Status {
			case health.StatusDown:
				logger.Info("Health status changed", zap.String("state", "down"))
			case health.StatusUp:
				logger.Info("Health status changed", zap.String("state", "up"))
			case health.StatusUnknown:
				// Unknown should not be logged
			}
		}),
		health.WithInterceptors(func(next health.InterceptorFunc) health.InterceptorFunc {
			return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
				currentStatus := state.Status
				result := next(ctx, name, state)

				if currentStatus != result.Status {
					switch result.Status {
					case health.StatusUp:
						logger.Info("Health check marked as healthy", zap.String("name", name))
					case health.StatusDown:
						logger.Info("Health check marked as unhealthy", zap.String("name", name))
					case health.StatusUnknown:
						// Unknown should not be logged
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
