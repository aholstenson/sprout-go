package sprout

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// Tracer returns a function that can be used with fx.Provide to make a tracer
// available to the application.
func Tracer(name string, opts ...trace.TracerOption) any {
	return func(tp trace.TracerProvider) trace.Tracer {
		return tp.Tracer(name, opts...)
	}
}

// ServiceTracer returns a function that can be used with fx.Provide to make a
// tracer available to the application. The tracer will use the name and version
// of the service.
func ServiceTracer(opts ...trace.TracerOption) any {
	return fx.Annotate(func(serviceName string, serviceVersion string, tp trace.TracerProvider) trace.Tracer {
		optsWithServiceVersion := append([]trace.TracerOption{
			trace.WithInstrumentationVersion(serviceVersion),
		}, opts...)
		return tp.Tracer(serviceName, optsWithServiceVersion...)
	}, fx.ParamTags(`name:"service:name"`, `name:"service:version"`))
}

// Meter returns a function that can be used with fx.Provide to make a meter
// available to the application.
func Meter(name string, opts ...metric.MeterOption) any {
	return func(mp metric.MeterProvider) metric.Meter {
		return mp.Meter(name, opts...)
	}
}

// ServiceMeter returns a function that can be used with fx.Provide to make a
// meter available to the application. The meter will use the name and version
// of the service.
func ServiceMeter(opts ...metric.MeterOption) any {
	return fx.Annotate(func(serviceName string, serviceVersion string, mp metric.MeterProvider) metric.Meter {
		optsWithServiceVersion := append([]metric.MeterOption{
			metric.WithInstrumentationVersion(serviceVersion),
		}, opts...)
		return mp.Meter(serviceName, optsWithServiceVersion...)
	}, fx.ParamTags(`name:"service:name"`, `name:"service:version"`))
}
