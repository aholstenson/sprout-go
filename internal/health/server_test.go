package health_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/aholstenson/sprout-go/internal/health"
	"github.com/aholstenson/sprout-go/internal/logging"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// environment supplies the flags that the health server uses to decide whether
// it should run.
func environment(testing bool) fx.Option {
	return fx.Options(
		fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
		fx.Supply(fx.Annotate(testing, fx.ResultTags(`name:"env:testing"`))),
	)
}

// newApp creates an application with the health module. It does not start it.
func newApp(testing bool, server **health.Server, options ...fx.Option) *fxtest.App {
	all := make([]fx.Option, 0, 4+len(options))
	all = append(all,
		logging.Module(zaptest.NewLogger(GinkgoT())),
		environment(testing),
		health.Module,
		fx.Populate(server),
	)

	return fxtest.New(GinkgoT(), append(all, options...)...)
}

// statusOf returns the status code of a request to a path of the server.
func statusOf(server *health.Server, path string) int {
	url := fmt.Sprintf("http://localhost:%d%s", server.Port(), path)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	Expect(err).ToNot(HaveOccurred())

	res, err := http.DefaultClient.Do(request)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode
}

var _ = Describe("Health", func() {
	BeforeEach(func() {
		// A free port, so that the tests do not compete for a port.
		GinkgoT().Setenv("HEALTH_SERVER_PORT", "0")
	})

	It("server can be started", func() {
		var server *health.Server
		app := newApp(false, &server)
		app.RequireStart()
		defer app.RequireStop()

		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusOK))
		Expect(statusOf(server, "/readyz")).To(Equal(http.StatusOK))
	})

	It("server can be started with custom port", func() {
		GinkgoT().Setenv("HEALTH_SERVER_PORT", "8089")

		var server *health.Server
		app := newApp(false, &server)
		app.RequireStart()
		defer app.RequireStop()

		Expect(server.Port()).To(Equal(8089))
		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusOK))
		Expect(statusOf(server, "/readyz")).To(Equal(http.StatusOK))
	})

	It("selects a free port when the port is 0", func() {
		var first, second *health.Server

		firstApp := newApp(false, &first)
		firstApp.RequireStart()
		defer firstApp.RequireStop()

		secondApp := newApp(false, &second)
		secondApp.RequireStart()
		defer secondApp.RequireStop()

		Expect(first.Port()).To(BeNumerically(">", 0))
		Expect(second.Port()).To(BeNumerically(">", 0))
		Expect(first.Port()).ToNot(Equal(second.Port()))
		Expect(statusOf(second, "/healthz")).To(Equal(http.StatusOK))
	})

	It("reports port 0 after it has stopped", func() {
		var server *health.Server
		app := newApp(false, &server)
		app.RequireStart()
		Expect(server.Port()).To(BeNumerically(">", 0))

		app.RequireStop()
		Expect(server.Port()).To(Equal(0))
	})

	It("rejects a port that is out of range", func() {
		GinkgoT().Setenv("HEALTH_SERVER_PORT", "70000")

		app := fx.New(
			fx.NopLogger,
			logging.Module(zap.NewNop()),
			environment(false),
			health.Module,
			fx.Invoke(func(health.Checks) {}),
		)

		Expect(app.Err()).To(MatchError(ContainSubstring("HEALTH_SERVER_PORT must be from 0 to 65535")))
	})

	It("failing liveness check returns 503", func() {
		var server *health.Server
		app := newApp(false, &server, fx.Invoke(func(checks health.Checks) {
			checks.AddLivenessCheck(health.Check{
				Name: "test",
				Check: func(ctx context.Context) error {
					return errors.New("failed")
				},
			})
		}))
		app.RequireStart()
		defer app.RequireStop()

		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusServiceUnavailable))
		Expect(statusOf(server, "/readyz")).To(Equal(http.StatusOK))
	})

	It("failing readiness check returns 503", func() {
		var server *health.Server
		app := newApp(false, &server, fx.Invoke(func(checks health.Checks) {
			checks.AddReadinessCheck(health.Check{
				Name: "test",
				Check: func(ctx context.Context) error {
					return errors.New("failed")
				},
			})
		}))
		app.RequireStart()
		defer app.RequireStop()

		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusOK))
		Expect(statusOf(server, "/readyz")).To(Equal(http.StatusServiceUnavailable))
	})

	It("keeps every check added from several goroutines", func() {
		const goroutines = 8
		const perGoroutine = 16

		var ran atomic.Int64
		var server *health.Server
		app := newApp(false, &server)

		var wg sync.WaitGroup
		for i := range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := range perGoroutine {
					server.AddLivenessCheck(health.Check{
						Name: fmt.Sprintf("check-%d-%d", i, j),
						Check: func(ctx context.Context) error {
							ran.Add(1)
							return nil
						},
					})
				}
			}()
		}
		wg.Wait()

		app.RequireStart()
		defer app.RequireStop()

		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusOK))

		// Every check that was added must have run exactly once.
		Expect(ran.Load()).To(Equal(int64(goroutines * perGoroutine)))
	})

	It("runs checks that are added after the server has started", func() {
		var ran atomic.Int64
		var server *health.Server
		app := newApp(false, &server)
		app.RequireStart()
		defer app.RequireStop()

		server.AddLivenessCheck(health.Check{
			Name: "added-after-start",
			Check: func(ctx context.Context) error {
				ran.Add(1)
				return nil
			},
		})

		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusOK))
		Expect(ran.Load()).To(Equal(int64(1)))
	})

	It("reports a failing check that is added after the server has started", func() {
		var server *health.Server
		app := newApp(false, &server)
		app.RequireStart()
		defer app.RequireStop()

		Expect(statusOf(server, "/readyz")).To(Equal(http.StatusOK))

		server.AddReadinessCheck(health.Check{
			Name: "added-after-start",
			Check: func(ctx context.Context) error {
				return errors.New("failed")
			},
		})

		Expect(statusOf(server, "/readyz")).To(Equal(http.StatusServiceUnavailable))
	})

	It("reports not ready but stays live after MarkNotReady", func() {
		var server *health.Server
		app := newApp(false, &server)
		app.RequireStart()
		defer app.RequireStop()

		server.MarkNotReady()

		Expect(statusOf(server, "/readyz")).To(Equal(http.StatusServiceUnavailable))
		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusOK))
	})

	It("does not bind a port while testing", func() {
		var server *health.Server
		app := newApp(true, &server)
		app.RequireStart()
		defer app.RequireStop()

		Expect(server.Port()).To(Equal(0))
	})

	It("can be enabled while testing", func() {
		GinkgoT().Setenv("HEALTH_SERVER_ENABLED", "true")

		var server *health.Server
		app := newApp(true, &server)
		app.RequireStart()
		defer app.RequireStop()

		Expect(statusOf(server, "/healthz")).To(Equal(http.StatusOK))
	})
})
