package otel

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// setenv sets the environment variables for the current spec. Ginkgo restores
// them after the spec.
func setenv(vars map[string]string) {
	for key, value := range vars {
		GinkgoT().Setenv(key, value)
	}
}

var _ = Describe("Configuration", func() {
	Describe("readProtocol", func() {
		DescribeTable("selects the protocol",
			func(vars map[string]string, expected protocol) {
				setenv(vars)

				p, err := readProtocol(moduleTracing)
				Expect(err).ToNot(HaveOccurred())
				Expect(p).To(Equal(expected))
			},
			Entry("defaults to grpc", map[string]string{}, protocolGRPC),
			Entry("uses the general variable",
				map[string]string{"OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf"},
				protocolHTTPProtobuf),
			Entry("uses the signal variable",
				map[string]string{"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL": "http/protobuf"},
				protocolHTTPProtobuf),
			Entry("prefers the signal variable",
				map[string]string{
					"OTEL_EXPORTER_OTLP_PROTOCOL":        "http/protobuf",
					"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL": "grpc",
				},
				protocolGRPC),
			Entry("ignores the variable of another signal",
				map[string]string{"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL": "http/protobuf"},
				protocolGRPC),
		)

		DescribeTable("rejects unknown values",
			func(vars map[string]string, message string) {
				setenv(vars)

				_, err := readProtocol(moduleTracing)
				Expect(err).To(MatchError(ContainSubstring(message)))
			},
			Entry("unknown general value",
				map[string]string{"OTEL_EXPORTER_OTLP_PROTOCOL": "tcp"},
				`invalid OTEL_EXPORTER_OTLP_PROTOCOL "tcp"`),
			Entry("unknown signal value",
				map[string]string{"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL": "http"},
				`invalid OTEL_EXPORTER_OTLP_TRACES_PROTOCOL "http"`),
			Entry("http/json is not supported",
				map[string]string{"OTEL_EXPORTER_OTLP_PROTOCOL": "http/json"},
				"use grpc or http/protobuf"),
		)
	})

	Describe("readSignalConfig", func() {
		DescribeTable("selects the exporters",
			func(vars map[string]string, defaultConsole bool, expected signalConfig) {
				setenv(vars)

				cfg, err := readSignalConfig(moduleMetrics, defaultConsole)
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg.OTLP).To(Equal(expected.OTLP))
				Expect(cfg.Console).To(Equal(expected.Console))
				Expect(cfg.SDKDisabled).To(Equal(expected.SDKDisabled))
			},
			Entry("exports nothing without an endpoint",
				map[string]string{}, false, signalConfig{}),
			Entry("uses OTLP when an endpoint is set",
				map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317"},
				false, signalConfig{OTLP: true}),
			Entry("uses the default console when no exporter is set",
				map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317"},
				true, signalConfig{Console: true}),
			Entry("uses OTLP when it is set without an endpoint",
				map[string]string{"OTEL_METRICS_EXPORTER": "otlp"},
				false, signalConfig{OTLP: true}),
			Entry("uses console",
				map[string]string{"OTEL_METRICS_EXPORTER": "console"},
				false, signalConfig{Console: true}),
			Entry("uses more than one exporter",
				map[string]string{"OTEL_METRICS_EXPORTER": " otlp , console "},
				false, signalConfig{OTLP: true, Console: true}),
			Entry("exports nothing for none even with an endpoint",
				map[string]string{
					"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
					"OTEL_METRICS_EXPORTER":       "none",
				},
				false, signalConfig{}),
			Entry("none has priority over the default console",
				map[string]string{"OTEL_METRICS_EXPORTER": "none"},
				true, signalConfig{}),
			Entry("exports nothing when the SDK is disabled",
				map[string]string{
					"OTEL_SDK_DISABLED":           "true",
					"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
					"OTEL_METRICS_EXPORTER":       "otlp,console",
				},
				true, signalConfig{SDKDisabled: true}),
			Entry("accepts any case for OTEL_SDK_DISABLED",
				map[string]string{"OTEL_SDK_DISABLED": "TRUE", "OTEL_METRICS_EXPORTER": "otlp"},
				false, signalConfig{SDKDisabled: true}),
			Entry("exports when OTEL_SDK_DISABLED is false",
				map[string]string{"OTEL_SDK_DISABLED": "false", "OTEL_METRICS_EXPORTER": "otlp"},
				false, signalConfig{OTLP: true}),
			Entry("ignores the exporter of another signal",
				map[string]string{"OTEL_TRACES_EXPORTER": "console"},
				false, signalConfig{}),
		)

		DescribeTable("rejects incorrect values",
			func(vars map[string]string, message string) {
				setenv(vars)

				_, err := readSignalConfig(moduleMetrics, false)
				Expect(err).To(MatchError(ContainSubstring(message)))
			},
			Entry("unknown exporter",
				map[string]string{"OTEL_METRICS_EXPORTER": "prometheus"},
				`invalid OTEL_METRICS_EXPORTER "prometheus": "prometheus" is not supported`),
			Entry("none with other exporters",
				map[string]string{"OTEL_METRICS_EXPORTER": "none,otlp"},
				"none cannot be used with other exporters"),
			Entry("unknown protocol",
				map[string]string{"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL": "udp"},
				`invalid OTEL_EXPORTER_OTLP_METRICS_PROTOCOL "udp"`),
			Entry("incorrect OTEL_SDK_DISABLED",
				map[string]string{"OTEL_SDK_DISABLED": "yes"},
				`invalid OTEL_SDK_DISABLED "yes"`),
		)

		It("ignores other variables when the SDK is disabled", func() {
			setenv(map[string]string{
				"OTEL_SDK_DISABLED":           "true",
				"OTEL_METRICS_EXPORTER":       "unknown",
				"OTEL_EXPORTER_OTLP_PROTOCOL": "unknown",
			})

			cfg, err := readSignalConfig(moduleMetrics, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.enabled()).To(BeFalse())
		})
	})

	Describe("readSampler", func() {
		DescribeTable("selects the sampler",
			func(vars map[string]string, legacyRate float64, expected string) {
				setenv(vars)

				sampler, err := readSampler(zap.NewNop(), legacyRate)
				Expect(err).ToNot(HaveOccurred())
				Expect(sampler).ToNot(BeNil())
				Expect(sampler.Description()).To(Equal(expected))
			},
			Entry("defaults to parentbased_always_on",
				map[string]string{}, 1.0,
				"ParentBased{root:AlwaysOnSampler,remoteParentSampled:AlwaysOnSampler,remoteParentNotSampled:AlwaysOffSampler,localParentSampled:AlwaysOnSampler,localParentNotSampled:AlwaysOffSampler}"),
			Entry("always_on",
				map[string]string{"OTEL_TRACES_SAMPLER": "always_on"}, 1.0,
				"AlwaysOnSampler"),
			Entry("always_off",
				map[string]string{"OTEL_TRACES_SAMPLER": "always_off"}, 1.0,
				"AlwaysOffSampler"),
			Entry("traceidratio without an argument",
				map[string]string{"OTEL_TRACES_SAMPLER": "traceidratio"}, 1.0,
				"TraceIDRatioBased{1}"),
			Entry("traceidratio with an argument",
				map[string]string{"OTEL_TRACES_SAMPLER": "traceidratio", "OTEL_TRACES_SAMPLER_ARG": "0.25"}, 1.0,
				"TraceIDRatioBased{0.25}"),
			Entry("parentbased_always_off",
				map[string]string{"OTEL_TRACES_SAMPLER": "parentbased_always_off"}, 1.0,
				"ParentBased{root:AlwaysOffSampler,remoteParentSampled:AlwaysOnSampler,remoteParentNotSampled:AlwaysOffSampler,localParentSampled:AlwaysOnSampler,localParentNotSampled:AlwaysOffSampler}"),
			Entry("parentbased_traceidratio",
				map[string]string{"OTEL_TRACES_SAMPLER": "parentbased_traceidratio", "OTEL_TRACES_SAMPLER_ARG": "0.5"}, 1.0,
				"ParentBased{root:TraceIDRatioBased{0.5},remoteParentSampled:AlwaysOnSampler,remoteParentNotSampled:AlwaysOffSampler,localParentSampled:AlwaysOnSampler,localParentNotSampled:AlwaysOffSampler}"),
			Entry("uses OTEL_TRACING_SAMPLE_RATE as a parent-based ratio",
				map[string]string{"OTEL_TRACING_SAMPLE_RATE": "0.1"}, 0.1,
				"ParentBased{root:TraceIDRatioBased{0.1},remoteParentSampled:AlwaysOnSampler,remoteParentNotSampled:AlwaysOffSampler,localParentSampled:AlwaysOnSampler,localParentNotSampled:AlwaysOffSampler}"),
			Entry("prefers OTEL_TRACES_SAMPLER over OTEL_TRACING_SAMPLE_RATE",
				map[string]string{"OTEL_TRACES_SAMPLER": "always_on", "OTEL_TRACING_SAMPLE_RATE": "0"}, 0.0,
				"AlwaysOnSampler"),
		)

		It("returns no sampler when OTEL_TRACING_SAMPLE_RATE is 0", func() {
			setenv(map[string]string{"OTEL_TRACING_SAMPLE_RATE": "0"})

			sampler, err := readSampler(zap.NewNop(), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(sampler).To(BeNil())
		})

		It("warns when OTEL_TRACING_SAMPLE_RATE is ignored", func() {
			setenv(map[string]string{"OTEL_TRACES_SAMPLER": "always_on", "OTEL_TRACING_SAMPLE_RATE": "0.5"})
			core, logs := observer.New(zapcore.WarnLevel)

			_, err := readSampler(zap.New(core), 0.5)
			Expect(err).ToNot(HaveOccurred())
			Expect(logs.FilterMessageSnippet("OTEL_TRACING_SAMPLE_RATE is ignored").Len()).To(Equal(1))
		})

		DescribeTable("rejects incorrect values",
			func(vars map[string]string, message string) {
				setenv(vars)

				_, err := readSampler(zap.NewNop(), 1)
				Expect(err).To(MatchError(ContainSubstring(message)))
			},
			Entry("unknown sampler",
				map[string]string{"OTEL_TRACES_SAMPLER": "jaeger_remote"},
				`invalid OTEL_TRACES_SAMPLER "jaeger_remote"`),
			Entry("argument that is not a number",
				map[string]string{"OTEL_TRACES_SAMPLER": "traceidratio", "OTEL_TRACES_SAMPLER_ARG": "half"},
				`invalid OTEL_TRACES_SAMPLER_ARG "half"`),
			Entry("argument above 1",
				map[string]string{"OTEL_TRACES_SAMPLER": "parentbased_traceidratio", "OTEL_TRACES_SAMPLER_ARG": "1.5"},
				`invalid OTEL_TRACES_SAMPLER_ARG "1.5"`),
			Entry("argument below 0",
				map[string]string{"OTEL_TRACES_SAMPLER": "traceidratio", "OTEL_TRACES_SAMPLER_ARG": "-0.1"},
				`invalid OTEL_TRACES_SAMPLER_ARG "-0.1"`),
			Entry("argument that is NaN",
				map[string]string{"OTEL_TRACES_SAMPLER": "traceidratio", "OTEL_TRACES_SAMPLER_ARG": "NaN"},
				`invalid OTEL_TRACES_SAMPLER_ARG "NaN"`),
		)
	})
})
