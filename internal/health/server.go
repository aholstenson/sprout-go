package health

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	// Enabled controls if the health server should be started.
	// When not explicitly set, defaults to true in production and false in
	// development and in tests.
	Enabled *bool `env:"ENABLED"`
	// Port is the port to bind to
	Port int `env:"PORT" envDefault:"8088"`
}

type Server struct {
	logger *zap.Logger

	httpServer *http.Server
	httpPort   int

	liveness  *checkSet
	readiness *checkSet
}

type ServiceInfo struct {
	fx.In

	Development bool `name:"env:development"`
	Testing     bool `name:"env:testing"`
}

func NewServer(lifecycle fx.Lifecycle, logger *zap.Logger, serviceInfo ServiceInfo, config Config) Checks {
	s := &Server{
		logger:   logger,
		httpPort: config.Port,

		liveness:  newCheckSet(logger.With(zap.String("type", "liveness"))),
		readiness: newCheckSet(logger.With(zap.String("type", "readiness"))),
	}

	// Determine if health server should be enabled
	enabled := false
	if config.Enabled != nil {
		// User explicitly set the enabled flag
		enabled = *config.Enabled
	} else {
		// Default: enabled in production. In development and in tests a port
		// would be bound that the developer did not ask for.
		enabled = !serviceInfo.Development && !serviceInfo.Testing
	}

	if enabled {
		lifecycle.Append(fx.Hook{
			OnStart: s.Start,
			OnStop:  s.Stop,
		})
	} else {
		logger.Info("Health server is disabled")
	}
	return s
}

func (s *Server) AddLivenessCheck(check Check) {
	s.liveness.add(check)
}

func (s *Server) AddReadinessCheck(check Check) {
	s.readiness.add(check)
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting health server", zap.Int("port", s.httpPort))

	mux := &http.ServeMux{}
	mux.Handle("/healthz", s.liveness)
	mux.Handle("/readyz", s.readiness)

	listenConfig := &net.ListenConfig{}
	ln, err := listenConfig.Listen(ctx, "tcp", ":"+strconv.Itoa(s.httpPort))
	if err != nil {
		return err
	}

	s.liveness.start()
	s.readiness.start()

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	go func() {
		err2 := s.httpServer.Serve(ln)
		if err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			s.logger.Error("Error starting health server", zap.Error(err2))
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping health server")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.httpServer.Shutdown(ctx)

	s.liveness.stop()
	s.readiness.stop()

	return err
}
