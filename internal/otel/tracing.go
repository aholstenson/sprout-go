package otel

import (
	"context"

	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// traceConfig holds the variables that Sprout used before it supported the
// standard OpenTelemetry variables. They continue to work, but the standard
// variables have priority.
type traceConfig struct {
	// Log is a flag that writes traces to the log. It is used only when
	// OTEL_TRACES_EXPORTER is not set.
	Log bool `env:"LOG" envDefault:"false"`

	// SampleRate is the rate at which traces should be sampled. It is used
	// only when OTEL_TRACES_SAMPLER is not set.
	SampleRate float64 `env:"SAMPLE_RATE" envDefault:"1.0"`
}

// SetupTracing configures OpenTelemetry tracing.
func SetupTracing(
	resource *resource.Resource,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) (trace.TracerProvider, error) {
	// Set a better default for propagators, since the default is a no-op
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	legacy, err := env.ParseAsWithOptions[traceConfig](env.Options{
		Prefix: "OTEL_TRACING_",
	})
	if err != nil {
		return nil, err
	}

	config, err := readSignalConfig(moduleTracing, legacy.Log)
	if err != nil {
		return nil, err
	}

	if config.ExporterSet && legacy.Log {
		logger.Warn("OTEL_TRACING_LOG is ignored because OTEL_TRACES_EXPORTER is set")
	}

	if config.SDKDisabled {
		logger.Warn("Tracing disabled", zap.String("reason", config.DisabledReason))
		return noopTracing()
	}

	// The sampler is read also when no exporter is set, so that an incorrect
	// value is found before an exporter is set.
	sampler, err := readSampler(logger, legacy.SampleRate)
	if err != nil {
		return nil, err
	}

	if !config.enabled() {
		logger.Warn("Tracing disabled", zap.String("reason", config.DisabledReason))
		return noopTracing()
	}

	if sampler == nil {
		logger.Warn("Sample rate is less than or equal to 0, disabling tracing")
		return noopTracing()
	}

	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sampler),
	}

	if config.Console {
		logger.Info("Writing traces to the log")
		options = append(options, sdktrace.WithSyncer(&loggingTraceExporter{
			logger: logger.Named("trace"),
		}))
	}

	if config.OTLP {
		exporter := newTraceExporter(config.Protocol)
		options = append(options, sdktrace.WithBatcher(exporter))

		lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return exporter.Start(ctx)
			},
		})
	}

	logger.Info(
		"Tracing enabled",
		zap.String("sampler", sampler.Description()),
		zap.Bool("otlp", config.OTLP),
		zap.String("protocol", string(config.Protocol)),
		zap.Bool("console", config.Console),
	)
	tp := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(tp)

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
	})

	return tp, nil
}

func noopTracing() (trace.TracerProvider, error) {
	provider := noop.NewTracerProvider()
	otel.SetTracerProvider(provider)
	return provider, nil
}

type loggingTraceExporter struct {
	logger *zap.Logger
}

func (e *loggingTraceExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		// Collect the the context as fields
		fields := make([]zap.Field, 0, 7+len(span.Attributes()))
		fields = append(fields,
			zap.String("traceID", span.SpanContext().TraceID().String()),
			zap.String("spanID", span.SpanContext().SpanID().String()),
			zap.String("parentSpanID", span.Parent().SpanID().String()),
			zap.String("status", span.Status().Code.String()),
			zap.Time("startTime", span.StartTime()),
			zap.Duration("duration", span.EndTime().Sub(span.StartTime())),
			zap.Namespace("attributes"),
		)

		// Add the attributes as fields
		for _, attr := range span.Attributes() {
			// AsInterface gives the plain Go value for each attribute type,
			// which logs more clearly than the attribute.Value struct.
			fields = append(fields, zap.Any(string(attr.Key), attr.Value.AsInterface()))
		}

		// This exporter only exists when console output of traces was asked
		// for, so the spans are logged at info and need no further opt-in.
		e.logger.Info(span.Name(), fields...)
	}

	return nil
}

func (e *loggingTraceExporter) Shutdown(ctx context.Context) error {
	return nil
}
