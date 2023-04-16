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
- 📤 OTLP exporting of traces and metrics

## Usage

Sprout provides a small wrapper around [Fx](https://github.com/uber-go/fx) that
bootstraps the application. Sprout encourages the use of modules to keep
things organized.

The main of the application may look something like this:

```go
package main

import "github.com/levelfourab/sprout-go"

func main() {
  sprout.New("ExampleApp", "v1.0.0").With(
    example.Module
  ).Run()
}
```

The module can then be defined like this:

```go
package example

import "github.com/levelfourab/sprout-go"
import "go.uber.org/fx"

type Config struct {
  Name string `env:"NAME" envDefault:"Test"`
}

var Module = fx.Module(
  "example",
  fx.Provide(sprout.Config("", &Config{}), fx.Private),
  fx.Provide(sprout.Logger("example"), fx.Private),
  fx.Invoke(func(cfg *Config, logger *logr.Logger) {
    logger.Info("Hello", "name", cfg.Name)
  })
)
```

## Development mode

Sprout will act differently if the environment variable `DEVELOPMENT` is set
to `true`. This is intended for local development, and will enable things such
as pretty printing logs and disable sending of traces and metrics to an OTLP
backend.

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

Variants of `sprout.Logger` are also available to create a `*zap.SugaredLogger`
or a `logr.Logger`.

Example:
  
```go
fx.Provide(sprout.SugaredLogger("example"), fx.Private)
fx.Provide(sprout.LogrLogger("example"), fx.Private)
```

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
| `OTEL_EXPORTER_OTLP_ENDPOINT` | The endpoint to send traces, metrics and logs to | `https://localhost:4317` |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | The timeout for sending data | `10s` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Custom endpoint to send traces to, overrides `OTEL_EXPORTER_OTLP_ENDPOINT` | `https://localhost:4317` |
| `OTEL_EXPORTER_OTLP_TRACES_TIMEOUT` | Custom timeout for sending traces | `10s` |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | Custom endpoint to send metrics to, overrides `OTEL_EXPORTER_OTLP_ENDPOINT` | `https://localhost:4317` |
| `OTEL_EXPORTER_OTLP_METRICS_TIMEOUT` | Custom timeout for sending metrics | `10s` |
| `OTEL_TRACING_DEVELOPMENT` | Enable development mode for tracing | `false` |

If Sprout is in development mode, the OTLP exporter will be disabled. You can
enable logging of traces using the `OTEL_TRACING_DEVELOPMENT` environment
variable.

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

Checks can not be added after the application has started. It is recommended to
add checks either using `fx.Invoke` for simple cases or in a provide function
of a service.

Example with a fictional `RemoteService`:

```go
var Module = fx.Module(
  "healthCheckWithProvide",
  fx.Provide(func(lifecycle fx.Lifecycle, checks sprout.HealthChecks) *RemoteService {
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
