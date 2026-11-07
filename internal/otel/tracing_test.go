package otel_test

import (
	"context"

	"github.com/aholstenson/sprout-go/internal/otel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Tracing", func() {
	var (
		res  *resource.Resource
		core zapcore.Core
		logs *observer.ObservedLogs
	)

	BeforeEach(func() {
		var err error
		res, err = otel.CreateResource(otel.ServiceInfo{Name: "test", Version: "dev", Testing: true})
		Expect(err).ToNot(HaveOccurred())

		core, logs = observer.New(zapcore.InfoLevel)
	})

	setup := func() (trace.TracerProvider, *fxtest.Lifecycle, error) {
		lifecycle := fxtest.NewLifecycle(GinkgoT())
		provider, err := otel.SetupTracing(res, lifecycle, zap.New(core))
		return provider, lifecycle, err
	}

	createSpan := func(provider trace.TracerProvider) {
		_, span := provider.Tracer("test").Start(context.Background(), "example span")
		span.End()
	}

	It("should log spans when trace logging is enabled", func() {
		GinkgoT().Setenv("OTEL_TRACING_LOG", "true")

		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())

		createSpan(provider)

		Expect(logs.FilterMessage("example span").Len()).To(Equal(1))
	})

	It("should log spans when OTEL_TRACES_EXPORTER is console", func() {
		GinkgoT().Setenv("OTEL_TRACES_EXPORTER", "console")

		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())

		createSpan(provider)

		Expect(logs.FilterMessage("example span").Len()).To(Equal(1))
	})

	It("should not trace when neither logging nor an endpoint is configured", func() {
		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())

		createSpan(provider)

		Expect(logs.FilterMessage("example span").Len()).To(Equal(0))
	})

	It("should ignore OTEL_TRACING_LOG when OTEL_TRACES_EXPORTER is none", func() {
		GinkgoT().Setenv("OTEL_TRACING_LOG", "true")
		GinkgoT().Setenv("OTEL_TRACES_EXPORTER", "none")

		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeAssignableToTypeOf(&sdktrace.TracerProvider{}))

		createSpan(provider)

		Expect(logs.FilterMessage("example span").Len()).To(Equal(0))
		Expect(logs.FilterMessageSnippet("OTEL_TRACING_LOG is ignored").Len()).To(Equal(1))
	})

	It("should not trace when OTEL_SDK_DISABLED is true", func() {
		GinkgoT().Setenv("OTEL_SDK_DISABLED", "true")
		GinkgoT().Setenv("OTEL_TRACES_EXPORTER", "console")

		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeAssignableToTypeOf(&sdktrace.TracerProvider{}))

		createSpan(provider)

		Expect(logs.FilterMessage("example span").Len()).To(Equal(0))
	})

	It("should not record spans when OTEL_TRACES_SAMPLER is always_off", func() {
		GinkgoT().Setenv("OTEL_TRACES_EXPORTER", "console")
		GinkgoT().Setenv("OTEL_TRACES_SAMPLER", "always_off")

		provider, _, err := setup()
		Expect(err).ToNot(HaveOccurred())

		createSpan(provider)

		Expect(logs.FilterMessage("example span").Len()).To(Equal(0))
	})

	It("should send spans with OTLP over HTTP", func() {
		server := newOTLPServer()
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

		provider, lifecycle, err := setup()
		Expect(err).ToNot(HaveOccurred())
		lifecycle.RequireStart()

		createSpan(provider)

		sdkProvider, ok := provider.(*sdktrace.TracerProvider)
		Expect(ok).To(BeTrue())
		Expect(sdkProvider.ForceFlush(context.Background())).To(Succeed())
		lifecycle.RequireStop()

		Expect(server.Paths()).To(ContainElement("/v1/traces"))
	})

	It("should fail when the protocol is unknown", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/json")

		_, _, err := setup()
		Expect(err).To(MatchError(ContainSubstring("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL")))
	})

	It("should fail when the sampler is unknown", func() {
		GinkgoT().Setenv("OTEL_TRACES_SAMPLER", "sometimes")

		_, _, err := setup()
		Expect(err).To(MatchError(ContainSubstring("OTEL_TRACES_SAMPLER")))
	})
})
