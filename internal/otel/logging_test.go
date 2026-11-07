package otel_test

import (
	"context"

	"github.com/aholstenson/sprout-go/internal"
	"github.com/aholstenson/sprout-go/internal/otel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
)

var serviceInfo = internal.ServiceInfo{
	Name:    "test",
	Version: "dev",
	Testing: true,
}

var _ = Describe("Logging", func() {
	It("should not export when no endpoint is configured", func() {
		provider, err := otel.InitLogging(serviceInfo)

		Expect(err).ToNot(HaveOccurred())
		Expect(provider).To(BeNil())
	})

	It("should return a provider when an endpoint is configured", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://localhost:4317")

		provider, err := otel.InitLogging(serviceInfo)

		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeNil())

		Expect(provider.ForceFlush(context.Background())).To(Succeed())
		Expect(provider.Shutdown(context.Background())).To(Succeed())
	})

	It("should send logs with OTLP over HTTP", func() {
		server := newOTLPServer()
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "http/protobuf")

		provider, err := otel.InitLogging(serviceInfo)
		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeNil())

		var record log.Record
		record.SetBody(attribute.StringValue("example"))
		provider.Logger("test").Emit(context.Background(), record)

		Expect(provider.Shutdown(context.Background())).To(Succeed())

		Expect(server.Paths()).To(ContainElement("/v1/logs"))
	})

	It("should return a provider when OTEL_LOGS_EXPORTER is console", func() {
		GinkgoT().Setenv("OTEL_LOGS_EXPORTER", "console")

		provider, err := otel.InitLogging(serviceInfo)

		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeNil())
		Expect(provider.Shutdown(context.Background())).To(Succeed())
	})

	It("should not export when OTEL_LOGS_EXPORTER is none", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://localhost:4317")
		GinkgoT().Setenv("OTEL_LOGS_EXPORTER", "none")

		provider, err := otel.InitLogging(serviceInfo)

		Expect(err).ToNot(HaveOccurred())
		Expect(provider).To(BeNil())
	})

	It("should not export when OTEL_SDK_DISABLED is true", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://localhost:4317")
		GinkgoT().Setenv("OTEL_SDK_DISABLED", "true")

		provider, err := otel.InitLogging(serviceInfo)

		Expect(err).ToNot(HaveOccurred())
		Expect(provider).To(BeNil())
	})

	It("should fail when the protocol is unknown", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://localhost:4317")
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "grpc/json")

		_, err := otel.InitLogging(serviceInfo)

		Expect(err).To(MatchError(ContainSubstring(`invalid OTEL_EXPORTER_OTLP_LOGS_PROTOCOL "grpc/json"`)))
	})
})
