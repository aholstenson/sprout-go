package logging_test

import (
	"github.com/go-logr/logr"
	"github.com/levelfourab/sprout-go/internal/logging"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("Logging", func() {
	Describe("Named loggers", func() {
		It("should be able to provide named loggers", func() {
			t := GinkgoT()

			var logger logr.Logger
			app := fxtest.New(
				t,
				fx.Supply(zaptest.NewLogger(t)),
				logging.Module,
				fx.Decorate(logging.Logger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})
})
