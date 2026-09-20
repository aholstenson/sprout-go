# Sprout

Sprout is a module to build microservices in Go. It provides a way to set up
shared things such as configuration, logging, tracing, metrics and health
checks.

## Features

- 💉 Dependency injection and lifecycle management via [Fx](https://github.com/uber-go/fx)
- 🛠️ Configuration via environment variables using [env](https://github.com/caarlos0/env)
- 📝 Logging via [Zap](https://github.com/uber-go/zap) and [logr](https://github.com/go-logr/logr)
- 🔍 Tracing and metrics via [OpenTelemetry](https://opentelemetry.io/)
- 🩺 Liveness and readiness checks via [Health](https://github.com/alexliesenfeld/health)
- 📤 OTLP exporting of traces, metrics and logs

## Usage

Sprout provides a small wrapper around [Fx](https://github.com/uber-go/fx) that
bootstraps the application. Sprout encourages the use of modules to keep
things organized.

The main of the application may look something like this:

```go
package main

import "github.com/aholstenson/sprout-go"

func main() {
  sprout.New("ExampleApp", "v1.0.0").With(
    example.Module
  ).Run()
}
```

The module can then be defined like this:

```go
package example

import (
  "github.com/aholstenson/sprout-go"
  "go.uber.org/fx"
  "go.uber.org/zap"
)

type Config struct {
  Name string `env:"NAME" envDefault:"Test"`
}

var Module = fx.Module(
  "example",
  fx.Provide(sprout.Config("", &Config{}), fx.Private),
  fx.Provide(sprout.Logger("example"), fx.Private),
  fx.Invoke(func(cfg *Config, logger *zap.Logger) {
    logger.Info("Hello", zap.String("name", cfg.Name))
  })
)
```

## Development mode

Sprout will act differently if the environment variable `DEVELOPMENT` is set
to `true`. This is intended for local development. It pretty prints logs to
stderr and keeps the health server off.

Development mode does not control OTLP exporting. Traces, metrics and logs are
only exported when an endpoint is configured, in any mode. See
[Observability](#observability).

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `DEVELOPMENT` | Set to `true` for development mode: pretty printed logs, and the health server stays off | unset |

A quick way to enable development mode is to use the `DEVELOPMENT=true` prefix
when running the application:

```sh
DEVELOPMENT=true go run .
```

As Sprout applications use environment variables for configuration a tool such
as [direnv](https://direnv.net/) can be used to automatically set variables
when entering the project directory.

A basic `.envrc` for use with direnv would look like this:

```sh
# .envrc
export DEVELOPMENT=true
```

Entering the project directory will then use this file:

```sh
$ cd example
direnv: loading .envrc
direnv: export +DEVELOPMENT
$ go run .
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
  fx.Provide(sprout.Config("PREFIX_IF_ANY", &Config{}), fx.Private),
  fx.Invoke(func(cfg *Config) {
    // Config is now available for use with Fx
  })
)
```

## Logging

Sprout provides logging via [Zap](https://github.com/uber-go/zap) and
[Logr](https://github.com/go-logr/logr). Sprout will automatically configure
logging based on if the application is running in development or production
mode. In development mode, logs are pretty printed to stderr. In production
mode, logs are formatted as JSON and sent to stderr.

`sprout.Logger` will create a function that returns a logger, which can be used
with `fx.Provide` to create a `*zap.Logger` for a certain module. It is
recommended to use the `fx.Private` option to make the logger private to the
module.

Example:

```go
var Module = fx.Module(
  "example",
  fx.Provide(sprout.Logger("example"), fx.Private),
  fx.Invoke(func(logger *zap.Logger) {
    // Logger is now available for use with Fx
  })
)
```

Variants of `sprout.Logger` are also available to create a `*zap.SugaredLogger`,
a `logr.Logger`, or a `*slog.Logger` with `sprout.SlogLogger`. `logr.Logger` is
a value type, not a pointer.

Example:
  
```go
fx.Provide(sprout.SugaredLogger("example"), fx.Private)
fx.Provide(sprout.LogrLogger("example"), fx.Private)
fx.Provide(sprout.SlogLogger("example"), fx.Private)
```

### Logging configuration

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `LOG_CONSOLE_OUTPUT` | Write logs to stderr | `true` |
| `LOG_FILE_OUTPUT` | Path of a file to append JSON logs to. Empty means no file | empty |
| `LOG_SAMPLING_INITIAL` | Number of identical messages logged each second before sampling starts. `0` turns sampling off | `100` |
| `LOG_SAMPLING_THEREAFTER` | After the initial count, log every n-th identical message in that second | `100` |

Sampling counts identical messages at the same level within a one second
window. It applies to every output.

### Log levels

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `LOG_LEVEL` | Level of the root logger | `info` |
| `LOG_LEVEL_<NAME>` | Level of one named logger | falls back to `LOG_LEVEL` |

Accepted values: `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`.

To build the variable name, take the parts of the logger name, join them with
`_`, replace every `.` with `_`, and put the result in upper case after
`LOG_LEVEL_`. A logger created with `sprout.Logger("database", "connection")`
is therefore set with `LOG_LEVEL_DATABASE_CONNECTION`.

The most specific variable wins. A logger named `service.api.v1.endpoint` looks
at `LOG_LEVEL_SERVICE_API_V1_ENDPOINT`, then `LOG_LEVEL_SERVICE_API_V1`, then
`LOG_LEVEL_SERVICE_API`, then `LOG_LEVEL_SERVICE`, and last `LOG_LEVEL`.

Sprout itself uses these logger names: `config`, `health`, `otel` and `fx`.

## Observability

Sprout integrates with [OpenTelemetry](https://opentelemetry.io/) and will push
data to an OTLP compatible backend such as [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/).
This decouples the application from the telemetry backend, allowing for easy
migration to other backends.

The following environment variables are used to configure the OpenTelemetry
integration:

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `OTEL_PROPAGATORS` | The default propagators to use | `tracecontext,baggage` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | The endpoint to send traces, metrics and logs to |  |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | The timeout in seconds for sending data | `10` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Custom endpoint to send traces to, overrides `OTEL_EXPORTER_OTLP_ENDPOINT` |  |
| `OTEL_EXPORTER_OTLP_TRACES_TIMEOUT` | Custom timeout in seconds for sending traces | `10` |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | Custom endpoint to send metrics to, overrides `OTEL_EXPORTER_OTLP_ENDPOINT` |  |
| `OTEL_EXPORTER_OTLP_METRICS_TIMEOUT` | Custom timeout in seconds for sending metrics | `10` |
| `OTEL_METRIC_EXPORT_INTERVAL` | The interval in seconds to export metrics | `60` |
| `OTEL_METRIC_EXPORT_TIMEOUT` | The timeout in seconds for exporting metrics | `30` |
| `OTEL_TRACING_LOG` | Enable logging mode for tracing | `false` |
| `OTEL_TRACING_SAMPLE_RATE` | Part of the traces to sample, between 0 and 1. `0` or less turns tracing off. `1` or more samples everything. The decision follows the parent span when there is one | `1.0` |

OTLP exporting is disabled by default. You can enable logging of traces which
can be useful for development by setting the `OTEL_TRACING_LOG` environment 
variable to `true`.

### Tracing

Sprout provides an easy way to make a [`trace.Tracer`](https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer)
available to a module:

```go
var Module = fx.Module(
  "example",
  fx.Provide(sprout.Tracer("example"), fx.Private),
  fx.Invoke(func(tracer trace.Tracer) {
    // Tracer is now available for use with Fx
  })
)
```

If the module is internal to the service, you can use `sprout.ServiceTracer` to
create a tracer based on the service name and version:
  
```go
var Module = fx.Module(
  "internalModule",
  fx.Provide(sprout.ServiceTracer(), fx.Private),
  fx.Invoke(func(tracer trace.Tracer) {
    // Tracer is now available for use with Fx
  })
)
```

### Metrics

Sprout provides an easy way to make a [`metric.Meter`](https://pkg.go.dev/go.opentelemetry.io/otel/metric#Meter)
available to a module:

```go
var Module = fx.Module(
  "example",
  fx.Provide(sprout.Meter("example"), fx.Private),
  fx.Invoke(func(meter metric.Meter) {
    // Meter is now available for use with Fx
  })
)
```

For modules that are internal to the service, you can use `sprout.ServiceMeter`
to create a meter based on the service name and version:

```go
var Module = fx.Module(
  "internalModule",
  fx.Provide(sprout.ServiceMeter(), fx.Private),
  fx.Invoke(func(meter metric.Meter) {
    // Meter is now available for use with Fx
  })
)
```

## Health checks

Sprout will start a HTTP server on port 8088 that exposes a `/healthz` and
`/readyz` endpoint. Requests to these will run checks and return a `200` status
code if all checks pass, or a `503` status code if any check fails. The port
that the server listens on can be configured via the `HEALTH_SERVER_PORT`
environment variable.

The server runs in production, but not in development mode or in tests, where
it would bind a port that you did not ask for. Set `HEALTH_SERVER_ENABLED` to
`true` or `false` to decide for yourself.

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `HEALTH_SERVER_ENABLED` | Whether to start the health server | On outside of development and tests |
| `HEALTH_SERVER_PORT` | The port that the health server listens on | `8088` |

Health checks are implemented using [Health](https://github.com/alexliesenfeld/health)
with checks being defined via `sprout.HealthCheck` structs. Checks can then
be added by calling `AddLivenessCheck` or `AddReadinessCheck` on the
`sprout.Health` service.

Example:

```go
var Module = fx.Module(
  "example",
  fx.Invoke(func(checks sprout.Health) {
    checks.AddLivenessCheck(sprout.HealthCheck{
      Name: "nameOfCheck",
      Check: func(ctx context.Context) error {
        // Check health here
        return nil
      },
    })
  })
)
```

Checks may be added at any time, including while the application runs. It is
recommended to add checks either using `fx.Invoke` for simple cases or in a
provide function of a service, so that they are in place before the first
request arrives.

Example with a fictional `RemoteService`:

```go
var Module = fx.Module(
  "healthCheckWithProvide",
  fx.Provide(func(lifecycle fx.Lifecycle, checks sprout.Health) *RemoteService {
    service := &RemoteService{
      ...
    }

    checks.AddReadinessCheck(sprout.HealthCheck{
      Name: "nameOfCheck",
      Check: func(ctx context.Context) error {
        return service.Ping()
      },
    })

    lifecycle.Append(fx.Hook{
      OnStart: func(ctx context.Context) error {
        return service.Start()
      },
      OnStop: func(ctx context.Context) error {
        return service.Stop()
      },
    })
    return service
  }),
)
```

## Fx logging

Sprout hides the Fx events that are only noise during startup, such as every
provided type, but always shows the events that carry an error. Set the
variable below to see them all, which helps when a dependency fails to resolve.

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `FX_ENABLE_DETAILED_LOGGING` | Set to `true` to log every Fx event. Without it Sprout hides the noisy events but always shows errors | unset |

## Working with the code

### Pre-commit hooks

[pre-commit](https://pre-commit.com/) is used to run various checks on the
code before it is committed. To install the hooks, run:

```bash
pre-commit install -t pre-commit -t pre-commit-msg
```

Commits will fail if any of the checks fail. If files are modified during the
checks, such as for formatting, you will need to add the modified files to the
commit again.

### Commit messages

[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) is used
for commit messages. This allows for automatic generation of changelogs and
version numbers. [commitlint](https://commitlint.js.org/#/) is used to enforce
the commit message format as a pre-commit hook.

### Code style

`gofmt` and [goimports](http://godoc.org/golang.org/x/tools/cmd/goimports) is
used for code formatting. Code formatting will run automatically as part of the
pre-commit hooks.

In addition to this [EditorConfig](https://editorconfig.org/) is used to ensure
consistent code style across editors.

### Linting

[golangci-lint](https://golangci-lint.run/) is used for linting. Linters will
run automatically as part of the pre-commit hooks. To run the linters manually:

```bash
golangci-lint run
```

### Running tests

[Ginkgo](https://onsi.github.io/ginkgo/) is used for testing. Tests can be run
via `go test` but the `ginkgo` CLI provides an improved experience:

```bash
ginkgo run ./...
```

## License

Sprout is licensed under the MIT License. See [LICENSE](LICENSE) for the full
license text.
