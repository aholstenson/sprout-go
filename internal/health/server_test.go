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
	"go.uber.org/zap/zaptest"
)

var _ = Describe("Health", func() {
	It("server can be started", func() {
		app := fxtest.New(
			GinkgoT(),
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
			health.Module,
			fx.Invoke(func(checks health.Checks) {
				// Do nothing, only here to make server always start
			}),
		)
		app.RequireStart()
		defer app.RequireStop()

		// Check that /healthz and /readyz are available
		res, err := http.Get("http://localhost:8088/healthz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		res, err = http.Get("http://localhost:8088/readyz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
	})

	It("server can be started with custom port", func() {
		t := GinkgoT()
		t.Setenv("HEALTH_SERVER_PORT", "8089")
		app := fxtest.New(
			t,
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
			health.Module,
			fx.Invoke(func(checks health.Checks) {
				// Do nothing, only here to make server always start
			}),
		)

		app.RequireStart()
		defer app.RequireStop()

		// Check that /healthz and /readyz are available
		res, err := http.Get("http://localhost:8089/healthz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		res, err = http.Get("http://localhost:8089/readyz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
	})

	It("failing liveness check returns 503", func() {
		app := fxtest.New(
			GinkgoT(),
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
			health.Module,
			fx.Invoke(func(checks health.Checks) {
				checks.AddLivenessCheck(health.Check{
					Name: "test",
					Check: func(ctx context.Context) error {
						return errors.New("failed")
					},
				})
			}),
		)
		app.RequireStart()
		defer app.RequireStop()

		// Check that /healthz and /readyz are available
		res, err := http.Get("http://localhost:8088/healthz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusServiceUnavailable))

		res, err = http.Get("http://localhost:8088/readyz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
	})

	It("failing readiness check returns 503", func() {
		app := fxtest.New(
			GinkgoT(),
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
			health.Module,
			fx.Invoke(func(checks health.Checks) {
				checks.AddReadinessCheck(health.Check{
					Name: "test",
					Check: func(ctx context.Context) error {
						return errors.New("failed")
					},
				})
			}),
		)
		app.RequireStart()
		defer app.RequireStop()

		// Check that /healthz and /readyz are available
		res, err := http.Get("http://localhost:8088/healthz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		res, err = http.Get("http://localhost:8088/readyz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusServiceUnavailable))
	})

	It("keeps every check added from several goroutines", func() {
		const goroutines = 8
		const perGoroutine = 16

		var ran atomic.Int64
		var checks health.Checks

		app := fxtest.New(
			GinkgoT(),
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
			health.Module,
			fx.Populate(&checks),
		)

		var wg sync.WaitGroup
		for i := range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := range perGoroutine {
					checks.AddLivenessCheck(health.Check{
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

		res, err := http.Get("http://localhost:8088/healthz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		// Every check that was added must have run exactly once.
		Expect(ran.Load()).To(Equal(int64(goroutines * perGoroutine)))
	})

	It("runs checks that are added after the server has started", func() {
		var ran atomic.Int64
		var checks health.Checks

		app := fxtest.New(
			GinkgoT(),
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
			health.Module,
			fx.Populate(&checks),
		)
		app.RequireStart()
		defer app.RequireStop()

		checks.AddLivenessCheck(health.Check{
			Name: "added-after-start",
			Check: func(ctx context.Context) error {
				ran.Add(1)
				return nil
			},
		})

		res, err := http.Get("http://localhost:8088/healthz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		Expect(ran.Load()).To(Equal(int64(1)))
	})

	It("reports a failing check that is added after the server has started", func() {
		var checks health.Checks

		app := fxtest.New(
			GinkgoT(),
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Supply(fx.Annotate(false, fx.ResultTags(`name:"env:development"`))),
			health.Module,
			fx.Populate(&checks),
		)
		app.RequireStart()
		defer app.RequireStop()

		res, err := http.Get("http://localhost:8088/readyz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		checks.AddReadinessCheck(health.Check{
			Name: "added-after-start",
			Check: func(ctx context.Context) error {
				return errors.New("failed")
			},
		})

		res, err = http.Get("http://localhost:8088/readyz")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusServiceUnavailable))
	})
})
