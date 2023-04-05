package test_test

import (
	"github.com/go-logr/logr"
	"github.com/levelfourab/sprout-go"
	"github.com/levelfourab/sprout-go/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type TestConf struct {
	Host string `env:"HOST"`
}

var _ = Describe("Test", func() {
	It("testing module bootstraps as expected", func() {
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
		)
		app.RequireStart()
		defer app.RequireStop()
	})

	It("sprout.Config works as expected", func() {
		t := GinkgoT()
		t.Setenv("TEST_HOST", "test")

		var c TestConf

		app := fxtest.New(
			t,
			test.Module(t),
			fx.Provide(sprout.Config("TEST", TestConf{})),
			fx.Populate(&c),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(c.Host).To(Equal("test"))
	})

	It("sprout.Logger works as expected", func() {
		var logger *zap.Logger
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.Logger("test")),
			fx.Populate(&logger),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(logger).NotTo(BeNil())
	})

	It("sprout.SugaredLogger works as expected", func() {
		var logger *zap.SugaredLogger
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.SugaredLogger("test")),
			fx.Populate(&logger),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(logger).NotTo(BeNil())
	})

	It("sprout.LogrLogger works as expected", func() {
		var logger logr.Logger
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.LogrLogger("test")),
			fx.Populate(&logger),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(logger).NotTo(BeNil())
	})

	It("sprout.Tracer works as expected", func() {
		var tracer trace.Tracer
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.Tracer("test")),
			fx.Populate(&tracer),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(tracer).NotTo(BeNil())
	})

	It("sprout.Meter works as expected", func() {
		var meter metric.Meter
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.Meter("test")),
			fx.Populate(&meter),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(meter).NotTo(BeNil())
	})

	It("sprout.Health is available", func() {
		var health sprout.Health
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Populate(&health),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(health).NotTo(BeNil())
	})
})
