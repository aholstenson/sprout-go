package sprout

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Tracer returns a function that can be used with fx.Provide to make a tracer
// available to the application.
func Tracer(name string, opts ...trace.TracerOption) any {
	return func(tp trace.TracerProvider) trace.Tracer {
		return tp.Tracer(name, opts...)
	}
}

// Meter returns a function that can be used with fx.Provide to make a meter
// available to the application.
func Meter(name string, opts ...metric.MeterOption) any {
	return func(mp metric.MeterProvider) metric.Meter {
		return mp.Meter(name, opts...)
	}
}
