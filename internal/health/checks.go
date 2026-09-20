package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/alexliesenfeld/health"
	"go.uber.org/zap"
)

// checkSet holds the checks of one endpoint and serves their result. Checks
// may be added at any time, also while the server runs, in which case the next
// request includes them.
type checkSet struct {
	logger *zap.Logger

	mu      sync.RWMutex
	checks  []Check
	checker health.Checker
	handler http.HandlerFunc
}

func newCheckSet(logger *zap.Logger) *checkSet {
	return &checkSet{logger: logger}
}

// add registers a check.
func (c *checkSet) add(check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks = append(c.checks, check)

	if c.checker != nil {
		// The endpoint already serves requests, so replace the checker to
		// include the check that was just added.
		c.build()
	}
}

// start makes the endpoint serve the checks that were added.
func (c *checkSet) start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.build()
}

// stop releases the checker. The endpoint then reports that it is unavailable.
func (c *checkSet) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.checker == nil {
		return
	}

	c.checker.Stop()
	c.checker = nil
	c.handler = nil
}

func (c *checkSet) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	c.mu.RLock()
	handler := c.handler
	c.mu.RUnlock()

	if handler == nil {
		// The HTTP server runs but the checks do not, which is the case while
		// the application starts and after it has stopped.
		http.Error(writer, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	handler(writer, request)
}

// build replaces the checker with one that runs the checks added so far. The
// caller must hold the write lock.
//
// A checker keeps the state of the checks it runs, such as how long a check
// has been failing. Replacing it resets that state, so a check that is added
// while the application runs also resets the checks next to it.
func (c *checkSet) build() {
	if c.checker != nil {
		c.checker.Stop()
	}

	c.checker = newChecker(c.logger, c.checks)
	c.handler = health.NewHandler(c.checker)
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
