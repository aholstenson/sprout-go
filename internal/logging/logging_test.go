package logging_test

import (
	"github.com/aholstenson/sprout-go/internal/logging"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("Logging", func() {
	Describe("*zap.Logger", func() {
		It("should be able to provide named *zap.Logger", func() {
			t := GinkgoT()

			var logger *zap.Logger
			app := fxtest.New(
				t,
				logging.Module(zaptest.NewLogger(t)),
				fx.Provide(logging.Logger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})

	Describe("*zap.SugaredLogger", func() {
		It("should be able to provide named *zap.SugaredLogger", func() {
			t := GinkgoT()

			var logger *zap.SugaredLogger
			app := fxtest.New(
				t,
				logging.Module(zaptest.NewLogger(t)),
				fx.Provide(logging.SugaredLogger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})

	Describe("logr.Logger", func() {
		It("should be able to provide named logr.Logger", func() {
			t := GinkgoT()

			var logger logr.Logger
			app := fxtest.New(
				t,
				logging.Module(zaptest.NewLogger(t)),
				fx.Provide(logging.LogrLogger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})
})
