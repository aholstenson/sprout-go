# Sprout

Sprout is a library to build microservices in Go. It provides a way to set up
shared things such as configuration, logging, telemetry and metrics.

## Usage

Sprout provides a small wrapper around [Fx](https://github.com/uber-go/fx) that
bootstraps the application. Sprout encourages the use of modules to keep
things organized.

The main application may look something like this:

```go
package main

import "github.com/levelfourab/sprout-go"

func main() {
  sprout.New("ExampleApp", "v1.0.0").With(
    example.Module
  ).Run()
}
```

A module can then be defined like this:

```go
package main

import "github.com/levelfourab/sprout-go"
import "go.uber.org/fx"

func main() {
  sprout.New("ExampleApp", "v1.0.0").With(
    example.Module
  ).Run()
}

type Config struct {
  Name string `env:"NAME" envDefault:"Test"`
}

var Module = fx.Module(
  "example",
  fx.Provide(sprout.Config("", &Config{})),
  fx.Provide(sprout.Logger("example")),
  fx.Invoke(func(cfg *Config, logger *logr.Logger) {
    logger.Info("Hello", "name", cfg.Name)
  })
)
```

## Configuration

Sprout uses environment variables to configure the application. Variables are
read via [env](https://github.com/caarlos0/env) into structs.

`sprout.Config` will create a function that reads the environment variables,
which can be used with `fx.Provide`.

Example:

```go
type Config struct {
  Host string `env:"HOST" envDefault:"localhost"`
  Port int    `env:"PORT" envDefault:"8080"`
}

var Module = fx.Module(
  "example",
  fx.Provide(sprout.Config("", &Config{})),
  fx.Invoke(func(cfg *Config) {
    // Config is now available for use with Fx
  })
)
```

## Logging

Sprout provides logging via [Logr](https://github.com/go-logr/logr). Sprout
will automatically configure logging based on if the application is running
in development or production mode. In development mode, logs are pretty printed
to stderr. In production mode, logs are formatted as JSON and sent to
stderr.

`sprout.Logger` will create a function that returns a logger, which can be used
with `fx.Decorate` to create a logger for a certain module.

Example:

```go
var Module = fx.Module(
  "example",
  fx.Decorate(sprout.Logger("example")),
  fx.Invoke(func(logger *logr.Logger) {
    // Logger is now available for use with Fx
  })
)
```

## Observability

Sprout integrates with [OpenTelemetry](https://opentelemetry.io/) and will push
data to an [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
instance. This decouples the application from the telemetry backend, allowing
for easy migration to other backends.

The following environment variables are used to configure the OpenTelemetry
integration:

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `OTEL_PROPAGATORS` | The default propagators to use | `tracecontext,baggage` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | The endpoint to send traces, metrics and logs to | `https://localhost:4317` |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | The timeout for sending data | `10s` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Custom endpoint to send traces to, overrides `OTEL_EXPORTER_OTLP_ENDPOINT` | `https://localhost:4317` |
| `OTEL_EXPORTER_OTLP_TRACES_TIMEOUT` | Custom timeout for sending traces | `10s` |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | Custom endpoint to send metrics to, overrides `OTEL_EXPORTER_OTLP_ENDPOINT` | `https://localhost:4317` |
| `OTEL_EXPORTER_OTLP_METRICS_TIMEOUT` | Custom timeout for sending metrics | `10s` |

### Tracing

Sprout provides tracing via [OpenTelemetry](https://opentelemetry.io/). Sprout
will automatically configure tracing based on if the application is running
in development or production mode. In development mode, traces are currently
not sent anywhere. In production mode, traces are sent to an OpenTelemetry
Collector instance.

```go
var Module = fx.Module(
  "example",
  fx.Provide(sprout.Tracer("example")),
  fx.Invoke(func(tracer *trace.Tracer) {
    // Tracer is now available for use with Fx
  })
)
```

### Metrics

Sprout provides metrics via [OpenTelemetry](https://opentelemetry.io/). Sprout
will automatically configure metrics based on if the application is running
in development or production mode. In development mode, metrics are currently
not sent anywhere. In production mode, metrics are sent to an OpenTelemetry
Collector instance.

```go
var Module = fx.Module(
  "example",
  fx.Provide(sprout.Meter("example")),
  fx.Invoke(func(meter *otel.Meter) {
    // Meter is now available for use with Fx
  })
)
```

## License

Sprout is licensed under the MIT License. See [LICENSE](LICENSE) for the full
license text.
