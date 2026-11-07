// Package shutdown controls how a Sprout application stops.
//
// When the application receives a signal to stop, it first makes /readyz
// report that the service is unavailable. It then waits for the shutdown
// delay, so that load balancers can stop sending new requests. Only after that
// do the other components stop.
package shutdown

import (
	"context"
	"fmt"
	"time"

	"github.com/aholstenson/sprout-go/internal/config"
	"github.com/aholstenson/sprout-go/internal/health"
	"github.com/aholstenson/sprout-go/internal/logging"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Config controls how the application stops.
type Config struct {
	// Delay is how long the application waits after it reports that it is
	// not ready, before the other components stop.
	Delay time.Duration `env:"DELAY" envDefault:"0s"`

	// Timeout limits how long the application can take to stop. The delay is
	// part of this time.
	Timeout time.Duration `env:"TIMEOUT" envDefault:"15s"`
}

// ReadConfig reads the configuration from the SHUTDOWN_ variables and
// validates it.
func ReadConfig() (Config, error) {
	cfg, err := config.ReadConfig[Config]("SHUTDOWN")
	if err != nil {
		return cfg, fmt.Errorf("failed to load shutdown configuration: %w", err)
	}

	return cfg, cfg.Validate()
}

// Validate returns an error when the values cannot work together.
func (c Config) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be more than 0, got %s", c.Timeout)
	}

	if c.Delay < 0 {
		return fmt.Errorf("SHUTDOWN_DELAY must not be negative, got %s", c.Delay)
	}

	if c.Delay >= c.Timeout {
		return fmt.Errorf(
			"SHUTDOWN_DELAY (%s) must be shorter than SHUTDOWN_TIMEOUT (%s), so that the components have time to stop",
			c.Delay, c.Timeout,
		)
	}

	return nil
}

type params struct {
	fx.In

	Lifecycle  fx.Lifecycle
	RootLogger *zap.Logger `name:"logging.zap"`
	Health     *health.Server
}

// Options returns the Fx options that make the application stop gracefully.
//
// Put the options after all other options of the application, and not in an
// fx.Module. Fx stops the hooks in the reverse order that they were added, and
// it runs the invokes of modules before the invokes at the top level. The hook
// that these options add must be the last one, so that it runs first when the
// application stops.
//
// The invoke also makes sure that the health server is created, also when no
// module uses it.
func Options(cfg Config) fx.Option {
	return fx.Options(
		fx.StopTimeout(cfg.Timeout),
		fx.Invoke(func(in params) {
			register(in, cfg.Delay)
		}),
	)
}

func register(in params, delay time.Duration) {
	logger := logging.CreateLogger(in.RootLogger, []string{"shutdown"})

	in.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			in.Health.MarkNotReady()

			if delay <= 0 {
				return nil
			}

			logger.Info("Waiting before the components stop", zap.Duration("delay", delay))

			timer := time.NewTimer(delay)
			defer timer.Stop()

			select {
			case <-timer.C:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}
