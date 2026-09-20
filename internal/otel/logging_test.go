package otel_test

import (
	"context"

	"github.com/aholstenson/sprout-go/internal"
	"github.com/aholstenson/sprout-go/internal/otel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var serviceInfo = internal.ServiceInfo{
	Name:    "test",
	Version: "dev",
	Testing: true,
}

var _ = Describe("Logging", func() {
	It("should not export when no endpoint is configured", func() {
		provider, shutdown, err := otel.InitLogging(serviceInfo)

		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeNil())
		Expect(shutdown).To(BeNil())
	})

	It("should return a shutdown function when an endpoint is configured", func() {
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://localhost:4317")

		provider, shutdown, err := otel.InitLogging(serviceInfo)

		Expect(err).ToNot(HaveOccurred())
		Expect(provider).ToNot(BeNil())
		Expect(shutdown).ToNot(BeNil())

		Expect(shutdown(context.Background())).To(Succeed())
	})
})
