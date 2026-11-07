package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	// Enabled controls if the health server should be started.
	// When not explicitly set, defaults to true in production and false in
	// development and in tests.
	Enabled *bool `env:"ENABLED"`
	// Port is the port to bind to. 0 lets the operating system select a free
	// port, see Server.Port.
	Port int `env:"PORT" envDefault:"8088"`
}

// Validate makes sure that the port is in the range of TCP ports.
func (c Config) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("HEALTH_SERVER_PORT must be from 0 to 65535, got %d", c.Port)
	}

	return nil
}

type Server struct {
	logger *zap.Logger

	httpServer *http.Server
	httpPort   int

	// boundPort is the port that the server listens on, or 0 when it does
	// not run.
	boundPort atomic.Int32

	liveness  *checkSet
	readiness *checkSet
}

type ServiceInfo struct {
	fx.In

	Development bool `name:"env:development"`
	Testing     bool `name:"env:testing"`
}

func NewServer(lifecycle fx.Lifecycle, logger *zap.Logger, serviceInfo ServiceInfo, config Config) *Server {
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

// MarkNotReady makes /readyz report that the service is unavailable, whatever
// the result of the readiness checks. /healthz continues to run its checks, so
// that the service is not restarted while it stops.
//
// The application calls this when it starts to stop. Load balancers then stop
// sending new requests before the components stop.
func (s *Server) MarkNotReady() {
	if s.readiness.markStopping() {
		s.logger.Info("Reporting not ready, as the application stops")
	}
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

	// The port can be different from the configured port, when the configured
	// port is 0.
	if addr, ok := ln.Addr().(*net.TCPAddr); ok {
		s.boundPort.Store(int32(addr.Port)) //nolint:gosec // A TCP port is at most 65535.
		if s.httpPort == 0 {
			s.logger.Info("Health server got a free port", zap.Int("port", addr.Port))
		}
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
	s.boundPort.Store(0)

	s.liveness.stop()
	s.readiness.stop()

	return err
}

// Port returns the port that the server listens on. It is 0 when the server
// does not run, for example because it is disabled or has stopped.
func (s *Server) Port() int {
	return int(s.boundPort.Load())
}
