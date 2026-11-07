package otel_test

import (
	"context"

	"github.com/aholstenson/sprout-go/internal/otel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

var _ = Describe("Metrics", func() {
	var res *resource.Resource

	BeforeEach(func() {
		var err error
		res, err = otel.CreateResource(otel.ServiceInfo{Name: "test", Version: "dev", Testing: true})
		Expect(err).ToNot(HaveOccurred())
	})

	setup := func() (metric.MeterProvider, *fxtest.Lifecycle, error) {
		lifecycle := fxtest.NewLifecycle(GinkgoT())
		provider, err := otel.SetupMetrics(res, lifecycle, zap.NewNop())
		return provider, lifecycle, err
	}

	It("should not export when no endpoint is configured", func() {
		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeAssignableToTypeOf(&sdkmetric.MeterProvider{}))
	})

	It("should not export when OTEL_METRICS_EXPORTER is none", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
		GinkgoT().Setenv("OTEL_METRICS_EXPORTER", "none")

		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeAssignableToTypeOf(&sdkmetric.MeterProvider{}))
	})

	It("should not export when OTEL_SDK_DISABLED is true", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
		GinkgoT().Setenv("OTEL_SDK_DISABLED", "true")

		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeAssignableToTypeOf(&sdkmetric.MeterProvider{}))
	})

	It("should send metrics with OTLP over HTTP", func() {
		server := newOTLPServer()
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", server.URL+"/v1/metrics")
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL", "http/protobuf")

		provider, lifecycle, err := setup()
		Expect(err).ToNot(HaveOccurred())
		lifecycle.RequireStart()

		counter, err := provider.Meter("test").Int64Counter("example")
		Expect(err).ToNot(HaveOccurred())
		counter.Add(context.Background(), 1)

		sdkProvider, ok := provider.(*sdkmetric.MeterProvider)
		Expect(ok).To(BeTrue())
		Expect(sdkProvider.ForceFlush(context.Background())).To(Succeed())
		lifecycle.RequireStop()

		Expect(server.Paths()).To(ContainElement("/v1/metrics"))
	})

	It("should fail when the protocol is unknown", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "websocket")

		_, _, err := setup()
		Expect(err).To(MatchError(ContainSubstring(`invalid OTEL_EXPORTER_OTLP_PROTOCOL "websocket"`)))
	})

	It("should fail when the exporter is unknown", func() {
		GinkgoT().Setenv("OTEL_METRICS_EXPORTER", "prometheus")

		_, _, err := setup()
		Expect(err).To(MatchError(ContainSubstring("OTEL_METRICS_EXPORTER")))
	})
})
