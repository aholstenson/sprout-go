package otel_test

import (
	"context"

	"github.com/aholstenson/sprout-go/internal/otel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Tracing", func() {
	It("should log spans when trace logging is enabled", func() {
		t := GinkgoT()
		t.Setenv("OTEL_TRACING_LOG", "true")

		core, logs := observer.New(zapcore.InfoLevel)

		resource, err := otel.CreateResource(otel.ServiceInfo{Name: "test", Version: "dev", Testing: true})
		Expect(err).ToNot(HaveOccurred())

		provider, err := otel.SetupTracing(resource, fxtest.NewLifecycle(t), zap.New(core))
		Expect(err).ToNot(HaveOccurred())

		_, span := provider.Tracer("test").Start(context.Background(), "example span")
		span.End()

		Expect(logs.FilterMessage("example span").Len()).To(Equal(1))
	})

	It("should not trace when neither logging nor an endpoint is configured", func() {
		t := GinkgoT()

		core, logs := observer.New(zapcore.InfoLevel)

		resource, err := otel.CreateResource(otel.ServiceInfo{Name: "test", Version: "dev", Testing: true})
		Expect(err).ToNot(HaveOccurred())

		provider, err := otel.SetupTracing(resource, fxtest.NewLifecycle(t), zap.New(core))
		Expect(err).ToNot(HaveOccurred())

		_, span := provider.Tracer("test").Start(context.Background(), "example span")
		span.End()

		Expect(logs.FilterMessage("example span").Len()).To(Equal(0))
	})
})
