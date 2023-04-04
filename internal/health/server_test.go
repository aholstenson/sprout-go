package health_test

import (
	"context"
	"errors"
	"net/http"

	"github.com/levelfourab/sprout-go/internal/health"
	"github.com/levelfourab/sprout-go/internal/logging"
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
			health.Module,
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
			health.Module,
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
			health.Module,
			fx.Provide(health.AsLivenessCheck(func() health.Check {
				return health.Check{
					Name: "test",
					Check: func(ctx context.Context) error {
						return errors.New("failed")
					},
				}
			})),
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
			health.Module,
			fx.Provide(health.AsReadinessCheck(func() health.Check {
				return health.Check{
					Name: "test",
					Check: func(ctx context.Context) error {
						return errors.New("failed")
					},
				}
			})),
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
})
