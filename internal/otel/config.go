package otel

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

// protocol is the transport that an OTLP exporter uses.
type protocol string

const (
	protocolGRPC         protocol = "grpc"
	protocolHTTPProtobuf protocol = "http/protobuf"
)

// defaultProtocol is the OTLP protocol when no variable sets one. The
// OpenTelemetry specification recommends http/protobuf, but Sprout always
// used gRPC. Sprout keeps gRPC so that current applications continue to work.
const defaultProtocol = protocolGRPC

// Values of the OTEL_<SIGNAL>_EXPORTER variables.
const (
	exporterOTLP    = "otlp"
	exporterConsole = "console"
	exporterNone    = "none"
)

// signalConfig tells how a signal (traces, metrics or logs) is exported.
type signalConfig struct {
	// OTLP is true when the signal is sent to an OTLP endpoint.
	OTLP bool

	// Console is true when the signal is written to the console.
	Console bool

	// Protocol is the OTLP protocol to use.
	Protocol protocol

	// SDKDisabled is true when OTEL_SDK_DISABLED turns off all signals.
	SDKDisabled bool

	// ExporterSet is true when OTEL_<SIGNAL>_EXPORTER has a value.
	ExporterSet bool

	// DisabledReason tells why the signal is not exported. It is empty when
	// the signal has an exporter.
	DisabledReason string
}

// enabled returns true when the signal has one or more exporters.
func (c signalConfig) enabled() bool {
	return c.OTLP || c.Console
}

// envName returns the part of the variable names that identifies the signal.
func (m module) envName() string {
	switch m {
	case moduleTracing:
		return "TRACES"
	case moduleMetrics:
		return "METRICS"
	case moduleLogging:
		return "LOGS"
	}

	return ""
}

// sdkDisabled reads OTEL_SDK_DISABLED. Only "true" and "false" are accepted,
// in any case. An empty value is the same as "false".
func sdkDisabled() (bool, error) {
	value := strings.TrimSpace(os.Getenv("OTEL_SDK_DISABLED"))
	switch {
	case value == "", strings.EqualFold(value, "false"):
		return false, nil
	case strings.EqualFold(value, "true"):
		return true, nil
	}

	return false, fmt.Errorf("invalid OTEL_SDK_DISABLED %q: use true or false", value)
}

// readSignalConfig reads the exporters and the OTLP protocol of a signal from
// the environment.
//
// When OTEL_<SIGNAL>_EXPORTER is not set, the signal is written to the console
// if defaultConsole is true. If not, it is sent with OTLP when an OTLP
// endpoint is set, and else it is not exported.
func readSignalConfig(m module, defaultConsole bool) (signalConfig, error) {
	disabled, err := sdkDisabled()
	if err != nil {
		return signalConfig{}, err
	}

	if disabled {
		return signalConfig{SDKDisabled: true, DisabledReason: "OTEL_SDK_DISABLED is true"}, nil
	}

	exporterVar := "OTEL_" + m.envName() + "_EXPORTER"
	value := strings.TrimSpace(os.Getenv(exporterVar))

	var cfg signalConfig
	if value == "" {
		switch {
		case defaultConsole:
			cfg.Console = true
		case hasExporterEndpoint(m):
			cfg.OTLP = true
		default:
			cfg.DisabledReason = "no OTLP endpoint is set"
		}
	} else {
		cfg, err = parseExporters(exporterVar, value)
		if err != nil {
			return signalConfig{}, err
		}

		cfg.ExporterSet = true
	}

	// The protocol is read also when OTLP is not used, so that an incorrect
	// value is found before an endpoint is set.
	cfg.Protocol, err = readProtocol(m)
	if err != nil {
		return signalConfig{}, err
	}

	return cfg, nil
}

// parseExporters parses a comma-separated list of exporters, such as
// "otlp,console".
func parseExporters(name string, value string) (signalConfig, error) {
	var cfg signalConfig
	none := false
	for item := range strings.SplitSeq(value, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		switch item {
		case "":
			continue
		case exporterOTLP:
			cfg.OTLP = true
		case exporterConsole:
			cfg.Console = true
		case exporterNone:
			none = true
		default:
			return signalConfig{}, fmt.Errorf("invalid %s %q: %q is not supported, use otlp, console or none", name, value, item)
		}
	}

	if none && cfg.enabled() {
		return signalConfig{}, fmt.Errorf("invalid %s %q: none cannot be used with other exporters", name, value)
	}

	if !cfg.enabled() {
		cfg.DisabledReason = name + " is none"
	}

	return cfg, nil
}

// readProtocol reads the OTLP protocol of a signal. The variable for the
// signal, such as OTEL_EXPORTER_OTLP_TRACES_PROTOCOL, has priority over
// OTEL_EXPORTER_OTLP_PROTOCOL.
func readProtocol(m module) (protocol, error) {
	name := "OTEL_EXPORTER_OTLP_" + m.envName() + "_PROTOCOL"
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		name = "OTEL_EXPORTER_OTLP_PROTOCOL"
		value = strings.TrimSpace(os.Getenv(name))
	}

	switch protocol(strings.ToLower(value)) {
	case "":
		return defaultProtocol, nil
	case protocolGRPC:
		return protocolGRPC, nil
	case protocolHTTPProtobuf:
		return protocolHTTPProtobuf, nil
	}

	return "", fmt.Errorf("invalid %s %q: use grpc or http/protobuf", name, value)
}

// Values of OTEL_TRACES_SAMPLER.
const (
	samplerAlwaysOn                = "always_on"
	samplerAlwaysOff               = "always_off"
	samplerTraceIDRatio            = "traceidratio"
	samplerParentBasedAlwaysOn     = "parentbased_always_on"
	samplerParentBasedAlwaysOff    = "parentbased_always_off"
	samplerParentBasedTraceIDRatio = "parentbased_traceidratio"
)

// readSampler returns the sampler for traces.
//
// OTEL_TRACES_SAMPLER and OTEL_TRACES_SAMPLER_ARG have priority. When
// OTEL_TRACES_SAMPLER is not set, OTEL_TRACING_SAMPLE_RATE gives the ratio of
// a parent-based sampler. The result is nil when the sample rate is 0 or less,
// which turns tracing off.
func readSampler(logger *zap.Logger, legacyRate float64) (sdktrace.Sampler, error) {
	_, legacyRateSet := os.LookupEnv("OTEL_TRACING_SAMPLE_RATE")

	name := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER")))
	if name == "" {
		if legacyRate <= 0 {
			return nil, nil //nolint:nilnil // A nil sampler means that tracing is off.
		}

		if legacyRate >= 1 {
			return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
		}

		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(legacyRate)), nil
	}

	if legacyRateSet {
		logger.Warn("OTEL_TRACING_SAMPLE_RATE is ignored because OTEL_TRACES_SAMPLER is set")
	}

	switch name {
	case samplerAlwaysOn:
		return sdktrace.AlwaysSample(), nil
	case samplerAlwaysOff:
		return sdktrace.NeverSample(), nil
	case samplerParentBasedAlwaysOn:
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case samplerParentBasedAlwaysOff:
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	case samplerTraceIDRatio, samplerParentBasedTraceIDRatio:
		ratio, err := readSamplerRatio()
		if err != nil {
			return nil, err
		}

		if name == samplerTraceIDRatio {
			return sdktrace.TraceIDRatioBased(ratio), nil
		}

		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio)), nil
	}

	return nil, fmt.Errorf(
		"invalid OTEL_TRACES_SAMPLER %q: use %s, %s, %s, %s, %s or %s",
		name,
		samplerAlwaysOn,
		samplerAlwaysOff,
		samplerTraceIDRatio,
		samplerParentBasedAlwaysOn,
		samplerParentBasedAlwaysOff,
		samplerParentBasedTraceIDRatio,
	)
}

// readSamplerRatio reads the ratio in OTEL_TRACES_SAMPLER_ARG. The ratio must
// be between 0 and 1. When the variable is not set, the ratio is 1.
func readSamplerRatio() (float64, error) {
	value := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG"))
	if value == "" {
		return 1, nil
	}

	ratio, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(ratio) || ratio < 0 || ratio > 1 {
		return 0, fmt.Errorf("invalid OTEL_TRACES_SAMPLER_ARG %q: use a number between 0 and 1", value)
	}

	return ratio, nil
}

// newTraceExporter creates an OTLP trace exporter that is not started. The
// exporter reads the endpoint, headers and timeout from the environment.
func newTraceExporter(p protocol) *otlptrace.Exporter {
	if p == protocolHTTPProtobuf {
		return otlptracehttp.NewUnstarted()
	}

	return otlptracegrpc.NewUnstarted()
}

// newMetricExporter creates an OTLP metric exporter. The exporter reads the
// endpoint, headers and timeout from the environment.
func newMetricExporter(ctx context.Context, p protocol) (sdkmetric.Exporter, error) {
	if p == protocolHTTPProtobuf {
		return otlpmetrichttp.New(ctx)
	}

	return otlpmetricgrpc.New(ctx)
}

// newLogExporter creates an OTLP log exporter. The exporter reads the
// endpoint, headers and timeout from the environment.
func newLogExporter(ctx context.Context, p protocol) (sdklog.Exporter, error) {
	if p == protocolHTTPProtobuf {
		return otlploghttp.New(ctx)
	}

	return otlploggrpc.New(ctx)
}
