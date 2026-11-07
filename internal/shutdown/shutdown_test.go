package shutdown_test

import (
	"context"
	"net/http"
	"time"

	"github.com/aholstenson/sprout-go/internal/health"
	"github.com/aholstenson/sprout-go/internal/logging"
	"github.com/aholstenson/sprout-go/internal/shutdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

// The port is not the default port of the health server, so that these tests
// do not collide with the tests of the health server.
const healthPort = "8090"

// environment supplies the flags that the health server uses to decide whether
// it should run. The server runs, as these tests read /readyz.
func environment() fx.Option {
	return fx.Options(
		fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
		fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:testing"`))),
	)
}

// statusOf returns the status code of a request to a path of the health server.
func statusOf(ctx context.Context, path string) (int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:"+healthPort+path, nil)
	if err != nil {
		return 0, err
	}

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}

	return res.StatusCode, res.Body.Close()
}

func readyzStatus(ctx context.Context) (int, error) {
	return statusOf(ctx, "/readyz")
}

func livezStatus(ctx context.Context) (int, error) {
	return statusOf(ctx, "/healthz")
}

// statusAtStop records the health status codes when the component stops.
type statusAtStop struct {
	readiness int
	liveness  int
	stoppedAt time.Time
}

// component is a module like the ones of an application. It stops after the
// shutdown hook, as it is created first.
func component(result *statusAtStop) fx.Option {
	return fx.Module(
		"component",
		fx.Invoke(func(lifecycle fx.Lifecycle, _ health.Checks) {
			lifecycle.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					var err error
					result.readiness, err = readyzStatus(ctx)
					if err != nil {
						return err
					}

					result.liveness, err = livezStatus(ctx)
					if err != nil {
						return err
					}

					result.stoppedAt = time.Now()
					return nil
				},
			})
		}),
	)
}

var _ = Describe("Shutdown", func() {
	Describe("Config", func() {
		It("reads the defaults", func() {
			cfg, err := shutdown.ReadConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Delay).To(Equal(time.Duration(0)))
			Expect(cfg.Timeout).To(Equal(15 * time.Second))
		})

		It("reads the delay and the timeout from the environment", func() {
			t := GinkgoT()
			t.Setenv("SHUTDOWN_DELAY", "5s")
			t.Setenv("SHUTDOWN_TIMEOUT", "30s")

			cfg, err := shutdown.ReadConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Delay).To(Equal(5 * time.Second))
			Expect(cfg.Timeout).To(Equal(30 * time.Second))
		})

		It("rejects a delay that is not shorter than the timeout", func() {
			t := GinkgoT()
			t.Setenv("SHUTDOWN_DELAY", "15s")

			_, err := shutdown.ReadConfig()
			Expect(err).To(MatchError(ContainSubstring("must be shorter than SHUTDOWN_TIMEOUT")))
		})

		It("rejects a negative delay", func() {
			err := shutdown.Config{Delay: -time.Second, Timeout: time.Second}.Validate()
			Expect(err).To(MatchError(ContainSubstring("must not be negative")))
		})

		It("rejects a timeout of 0", func() {
			err := shutdown.Config{Timeout: 0}.Validate()
			Expect(err).To(MatchError(ContainSubstring("must be more than 0")))
		})

		It("rejects a value that is not a duration", func() {
			t := GinkgoT()
			t.Setenv("SHUTDOWN_DELAY", "five")

			_, err := shutdown.ReadConfig()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Stop", func() {
		BeforeEach(func() {
			GinkgoT().Setenv("HEALTH_SERVER_PORT", healthPort)
		})

		It("reports not ready before the components stop", func() {
			var result statusAtStop

			app := fxtest.New(
				GinkgoT(),
				logging.Module(zaptest.NewLogger(GinkgoT())),
				environment(),
				health.Module,
				component(&result),
				shutdown.Options(shutdown.Config{Timeout: 5 * time.Second}),
			)
			app.RequireStart()

			status, err := readyzStatus(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))

			app.RequireStop()

			Expect(result.readiness).To(Equal(http.StatusServiceUnavailable))
			Expect(result.liveness).To(Equal(http.StatusOK))
		})

		It("waits for the delay before the components stop", func() {
			const delay = 200 * time.Millisecond
			var result statusAtStop

			app := fxtest.New(
				GinkgoT(),
				logging.Module(zaptest.NewLogger(GinkgoT())),
				environment(),
				health.Module,
				component(&result),
				shutdown.Options(shutdown.Config{Delay: delay, Timeout: 5 * time.Second}),
			)
			app.RequireStart()

			stopStarted := time.Now()
			app.RequireStop()

			Expect(result.stoppedAt.Sub(stopStarted)).To(BeNumerically(">=", delay))
		})

		It("creates the health server when no module uses it", func() {
			app := fxtest.New(
				GinkgoT(),
				logging.Module(zaptest.NewLogger(GinkgoT())),
				environment(),
				health.Module,
				shutdown.Options(shutdown.Config{Timeout: 5 * time.Second}),
			)
			app.RequireStart()
			defer app.RequireStop()

			status, err := readyzStatus(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))
		})
	})
})
