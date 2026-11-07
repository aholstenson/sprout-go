package test_test

import (
	"context"

	"github.com/aholstenson/sprout-go"
	"github.com/aholstenson/sprout-go/test"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
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

	It("sprout.ServiceTracer works as expected", func() {
		var tracer trace.Tracer
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.ServiceTracer()),
			fx.Populate(&tracer),
		)
		app.RequireStart()
		defer app.RequireStop()
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

	It("sprout.ServiceMeter works as expected", func() {
		var meter metric.Meter
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.ServiceMeter()),
			fx.Populate(&meter),
		)
		app.RequireStart()
		defer app.RequireStop()
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

	It("records spans", func() {
		var recorder *tracetest.SpanRecorder
		var tracer trace.Tracer
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.Tracer("test")),
			fx.Populate(&recorder, &tracer),
		)
		app.RequireStart()
		defer app.RequireStop()

		_, span := tracer.Start(context.Background(), "example span")
		span.End()

		spans := recorder.Ended()
		Expect(spans).To(HaveLen(1))
		Expect(spans[0].Name()).To(Equal("example span"))
	})

	It("collects metrics", func() {
		var reader *sdkmetric.ManualReader
		var meter metric.Meter
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.Meter("test")),
			fx.Populate(&reader, &meter),
		)
		app.RequireStart()
		defer app.RequireStop()

		counter, err := meter.Int64Counter("example.counter")
		Expect(err).ToNot(HaveOccurred())
		counter.Add(context.Background(), 2)

		var rm metricdata.ResourceMetrics
		Expect(reader.Collect(context.Background(), &rm)).To(Succeed())
		Expect(rm.ScopeMetrics).To(HaveLen(1))
		Expect(rm.ScopeMetrics[0].Metrics).To(HaveLen(1))

		m := rm.ScopeMetrics[0].Metrics[0]
		Expect(m.Name).To(Equal("example.counter"))
		sum, ok := m.Data.(metricdata.Sum[int64])
		Expect(ok).To(BeTrue())
		Expect(sum.DataPoints).To(HaveLen(1))
		Expect(sum.DataPoints[0].Value).To(Equal(int64(2)))
	})

	It("keeps the telemetry of each application apart", func() {
		var first, second *tracetest.SpanRecorder
		var tracer trace.Tracer
		firstApp := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Provide(sprout.Tracer("test")),
			fx.Populate(&first, &tracer),
		)
		secondApp := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT()),
			fx.Populate(&second),
		)

		_, span := tracer.Start(context.Background(), "example span")
		span.End()

		Expect(first.Ended()).To(HaveLen(1))
		Expect(second.Ended()).To(BeEmpty())

		firstApp.RequireStart().RequireStop()
		secondApp.RequireStart().RequireStop()
	})

	It("uses the service name and version from the options", func() {
		var name, version string
		var recorder *tracetest.SpanRecorder
		var tracer trace.Tracer
		app := fxtest.New(
			GinkgoT(),
			test.Module(
				GinkgoT(),
				test.WithServiceName("orders"),
				test.WithServiceVersion("1.2.3"),
			),
			fx.Provide(sprout.ServiceTracer()),
			fx.Populate(
				fx.Annotate(&name, fx.ParamTags(`name:"service:name"`)),
				fx.Annotate(&version, fx.ParamTags(`name:"service:version"`)),
				&recorder,
				&tracer,
			),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(name).To(Equal("orders"))
		Expect(version).To(Equal("1.2.3"))

		_, span := tracer.Start(context.Background(), "example span")
		span.End()

		spans := recorder.Ended()
		Expect(spans).To(HaveLen(1))
		Expect(spans[0].InstrumentationScope().Name).To(Equal("orders"))
		Expect(spans[0].Resource().Attributes()).To(ContainElements(
			semconv.ServiceNameKey.String("orders"),
			semconv.ServiceVersionKey.String("1.2.3"),
		))
	})

	It("uses noop telemetry with WithNoopTelemetry", func() {
		var tracer trace.Tracer
		app := fxtest.New(
			GinkgoT(),
			test.Module(GinkgoT(), test.WithNoopTelemetry()),
			fx.Provide(sprout.Tracer("test")),
			fx.Populate(&tracer),
		)
		app.RequireStart()
		defer app.RequireStop()

		_, span := tracer.Start(context.Background(), "example span")
		defer span.End()
		Expect(span.IsRecording()).To(BeFalse())
	})
})
