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
only exported when an endpoint or an exporter is configured, in any mode. See
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

### Validation

A configuration struct can check its own values. Give it a `Validate() error`
method (the `sprout.ConfigValidator` interface). Sprout calls it after it has
read the values, and only when the values were read without errors. The method
can have a value or a pointer receiver. `sprout.Config`, `sprout.BindConfig`
and `sprout.ReadConfig` all call it.

If `Validate` returns an error, the application does not start, and Sprout logs
`Configuration is not valid` with the type and the error. To report more than
one problem, return them with `errors.Join`. Sprout then logs each problem on
its own line.

Example:

```go
type Config struct {
  MinConns int `env:"MIN_CONNS" envDefault:"1"`
  MaxConns int `env:"MAX_CONNS" envDefault:"10"`
}

func (c Config) Validate() error {
  if c.MinConns > c.MaxConns {
    return errors.New("MIN_CONNS must not be more than MAX_CONNS")
  }
  return nil
}
```

Sprout validates only the struct that it reads. If the struct contains other
structs that have a `Validate` method, call them from the `Validate` of the
outer struct.

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

When the application ends, Sprout writes the logs that are still buffered and
then closes the log file and the OpenTelemetry log exporter. This also occurs
when the application fails to start, so that the error is exported. The Fx
logger of Sprout does this after the last Fx event. If you replace it with
`fx.WithLogger`, Sprout only flushes the logs when the application stops.

### Log levels

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `LOG_LEVEL` | Level of the root logger | `info` |
| `LOG_LEVEL_<NAME>` | Level of one named logger | falls back to `LOG_LEVEL` |

Accepted values: `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`.

To build the variable name, take the parts of the logger name, join them with
`_`, replace every `.` with `_`, and put the result in upper case after
`LOG_LEVEL_`. A logger created with `sprout.Logger("database", "connection")`
is therefore set with `LOG_LEVEL_DATABASE_CONNECTION`. Each character that is
not a letter, digit or `_`, such as `-`, also becomes `_`, so the logger
`my-module` is set with `LOG_LEVEL_MY_MODULE`.

The most specific variable wins. A logger named `service.api.v1.endpoint` looks
at `LOG_LEVEL_SERVICE_API_V1_ENDPOINT`, then `LOG_LEVEL_SERVICE_API_V1`, then
`LOG_LEVEL_SERVICE_API`, then `LOG_LEVEL_SERVICE`, and last `LOG_LEVEL`.

Sprout itself uses these logger names: `config`, `health`, `otel`, `shutdown`
and `fx`.

## Observability

Sprout integrates with [OpenTelemetry](https://opentelemetry.io/) and will push
data to an OTLP compatible backend such as [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/).
This decouples the application from the telemetry backend, allowing for easy
migration to other backends.

The following environment variables are used to configure the OpenTelemetry
integration. `<SIGNAL>` is `TRACES`, `METRICS` or `LOGS`.

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `OTEL_SDK_DISABLED` | Set to `true` to turn off traces, metrics and logs. Sprout then ignores the exporter, protocol and sampler variables | `false` |
| `OTEL_TRACES_EXPORTER` | Where to send traces: `otlp`, `console` or `none`. Use a comma to set more than one, such as `otlp,console` | see below |
| `OTEL_METRICS_EXPORTER` | Where to send metrics: `otlp`, `console` or `none` | see below |
| `OTEL_LOGS_EXPORTER` | Where to send logs: `otlp`, `console` or `none` | see below |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | The OTLP protocol: `grpc` or `http/protobuf` | `grpc` |
| `OTEL_EXPORTER_OTLP_<SIGNAL>_PROTOCOL` | The OTLP protocol for one signal, overrides `OTEL_EXPORTER_OTLP_PROTOCOL` | |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | The endpoint to send traces, metrics and logs to | |
| `OTEL_EXPORTER_OTLP_<SIGNAL>_ENDPOINT` | The endpoint for one signal, overrides `OTEL_EXPORTER_OTLP_ENDPOINT` | |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | The timeout in milliseconds for sending data | `10000` |
| `OTEL_EXPORTER_OTLP_<SIGNAL>_TIMEOUT` | The timeout for one signal, overrides `OTEL_EXPORTER_OTLP_TIMEOUT` | |
| `OTEL_EXPORTER_OTLP_HEADERS` | Headers to send with each request, such as `api-key=secret` | |
| `OTEL_EXPORTER_OTLP_<SIGNAL>_HEADERS` | Headers for one signal, overrides `OTEL_EXPORTER_OTLP_HEADERS` | |
| `OTEL_METRIC_EXPORT_INTERVAL` | The interval in milliseconds to export metrics | `60000` |
| `OTEL_METRIC_EXPORT_TIMEOUT` | The timeout in milliseconds for exporting metrics | `30000` |
| `OTEL_TRACES_SAMPLER` | The sampler for traces, see [Sampling](#sampling) | `parentbased_always_on` |
| `OTEL_TRACES_SAMPLER_ARG` | The ratio for `traceidratio` and `parentbased_traceidratio`, between 0 and 1 | `1.0` |
| `OTEL_PROPAGATORS` | The default propagators to use | `tracecontext,baggage` |
| `OTEL_RESOURCE_ATTRIBUTES` | More resource attributes, such as `deployment.environment=prod` | |
| `OTEL_TRACING_LOG` | Older name for `OTEL_TRACES_EXPORTER=console`. Ignored when `OTEL_TRACES_EXPORTER` is set | `false` |
| `OTEL_TRACING_SAMPLE_RATE` | Older way to set the sampler. Ignored when `OTEL_TRACES_SAMPLER` is set, see [Sampling](#sampling) | `1.0` |

Sprout stops at startup with an error that names the variable when a value
is not valid, such as an unknown protocol, exporter or sampler.

### Exporters

When `OTEL_<SIGNAL>_EXPORTER` is not set, Sprout exports the signal with OTLP
only when `OTEL_EXPORTER_OTLP_ENDPOINT` or `OTEL_EXPORTER_OTLP_<SIGNAL>_ENDPOINT`
is set. Without an endpoint the signal is not exported. Thus OTLP exporting is
off by default.

When `OTEL_<SIGNAL>_EXPORTER` is set, it decides:

* `otlp` sends the signal with OTLP. Without an endpoint the exporter uses the
  default endpoint of the protocol: `localhost:4317` for `grpc` and
  `localhost:4318` for `http/protobuf`.
* `console` writes the signal to the console:
  * Traces go to the Sprout log at info level with the logger name
    `otel.trace`. Each span is one log entry.
  * Metrics go to stdout as JSON, one line for each export interval.
  * Logs go to stdout as JSON, one line for each OpenTelemetry log record.
    The normal log output to stderr continues.
* `none` does not export the signal, also when an endpoint is set.

The OTLP protocol is `grpc` by default. The OpenTelemetry specification
recommends `http/protobuf` as the default, but Sprout keeps `grpc` so that
applications that use gRPC today continue to work. Set
`OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf` to use HTTP. With HTTP, the
endpoint in `OTEL_EXPORTER_OTLP_ENDPOINT` is a base URL such as
`http://collector:4318`, and the exporters add the path for each signal, such
as `/v1/traces`. `http/json` is not supported.

To write traces to the log during development, set `OTEL_TRACES_EXPORTER` to
`console`:

```sh
DEVELOPMENT=true OTEL_TRACES_EXPORTER=console go run .
```

### Sampling

`OTEL_TRACES_SAMPLER` accepts these values:

| Value | Description |
| ----- | ----------- |
| `always_on` | Sample all traces |
| `always_off` | Sample no traces |
| `traceidratio` | Sample the part of the traces that `OTEL_TRACES_SAMPLER_ARG` gives |
| `parentbased_always_on` | Follow the parent span. Sample all traces that have no parent |
| `parentbased_always_off` | Follow the parent span. Sample no traces that have no parent |
| `parentbased_traceidratio` | Follow the parent span. For traces that have no parent, sample the part that `OTEL_TRACES_SAMPLER_ARG` gives |

When `OTEL_TRACES_SAMPLER` is not set, Sprout uses `OTEL_TRACING_SAMPLE_RATE`
as the ratio of a parent-based sampler. `1` or more samples all traces. `0` or
less turns tracing off. When both variables are set, `OTEL_TRACES_SAMPLER`
wins and Sprout logs a warning.

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

### Testing telemetry

`test.Module` sets up an application for tests. It records spans and metrics
in memory, so that a test can examine them. Get the recorders with
`fx.Populate`:

* `*tracetest.SpanRecorder` records all spans. `Ended` returns the spans that
  have ended.
* `*sdkmetric.ManualReader` collects the metrics when the test calls `Collect`.

```go
var recorder *tracetest.SpanRecorder
var reader *sdkmetric.ManualReader

app := fxtest.New(
  t,
  test.Module(t),
  myModule,
  fx.Populate(&recorder, &reader),
)
app.RequireStart()
defer app.RequireStop()

// Run the code under test, then examine the telemetry.
spans := recorder.Ended()

var rm metricdata.ResourceMetrics
err := reader.Collect(context.Background(), &rm)
```

Options change the test application:

| Option | Description |
| ------ | ----------- |
| `test.WithServiceName(name)` | Set the service name. The default is `test` |
| `test.WithServiceVersion(version)` | Set the service version. The default is `dev` |
| `test.WithNoopTelemetry()` | Use providers that do nothing and do not provide the recorders. Use it for benchmarks, as the span recorder keeps all spans in memory |

Each call to `test.Module` creates new providers and recorders. It does not
change the OpenTelemetry globals, so tests that run in parallel do not share
telemetry. Code that gets a tracer or a meter from the globals, such as with
`otel.Tracer`, does not send telemetry to the recorders. Get the
`trace.TracerProvider` and the `metric.MeterProvider` from Fx instead. Logs
are not recorded.

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

## Shutdown

When the application gets `SIGINT` or `SIGTERM`, it stops in this sequence:

1. `/readyz` returns `503`, whatever the result of the readiness checks.
   `/healthz` continues to run its checks, so that the service is not restarted
   while it stops.
2. The application waits for `SHUTDOWN_DELAY`.
3. The components stop, in the reverse sequence of their start.

The delay gives load balancers time to stop sending new requests before the
components stop. In Kubernetes, a pod continues to get traffic for a short time
after it starts to terminate, until all nodes and ingresses know that the pod
is gone. A delay of 5 to 10 seconds is usually sufficient.

`SHUTDOWN_TIMEOUT` limits the total time that the application can use to stop,
the delay included. The delay must be shorter than the timeout. In Kubernetes,
keep the timeout shorter than `terminationGracePeriodSeconds` of the pod, which
is 30 seconds if you do not set it.

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `SHUTDOWN_DELAY` | Time to wait after `/readyz` starts to fail, before the components stop. For example `5s` | `0s` |
| `SHUTDOWN_TIMEOUT` | Maximum time the application can take to stop, the delay included | `15s` |

Example for Kubernetes:

```yaml
env:
  - name: SHUTDOWN_DELAY
    value: 5s
  - name: SHUTDOWN_TIMEOUT
    value: 25s
```

## Fx logging

Sprout hides the Fx events that are only noise during startup, such as every
provided type, but always shows the events that carry an error. Set the
variable below to see them all, which helps when a dependency fails to resolve.

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `FX_ENABLE_DETAILED_LOGGING` | Set to `true` to log every Fx event. Without it Sprout hides the noisy events but always shows errors | unset |

## Working with the code

### Tooling

[mise](https://mise.jdx.dev/) manages the development tools and the versions
they are pinned to. Install the tools with:

```bash
mise install
```

Common jobs are available as mise tasks, which run the tools at the pinned
versions. List them with `mise tasks`:

| Task | Description |
| ---- | ----------- |
| `mise run fmt` | Format the code |
| `mise run lint` | Run the linters |
| `mise run lint-fix` | Run the linters and apply the fixes that are available |
| `mise run test` | Run the tests |
| `mise run tidy` | Tidy the Go module |
| `mise run build` | Build a release snapshot |

The same tasks run in CI, so a task that passes locally runs the same commands
with the same tool versions as the pipeline.

### Pre-commit hooks

[pre-commit](https://pre-commit.com/) is used to run various checks on the
code before it is committed. It is installed by `mise install`. To install the
hooks, run:

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
mise run lint
```

### Running tests

[Ginkgo](https://onsi.github.io/ginkgo/) is used for testing. The `ginkgo` CLI
is installed by `mise install`. Run the tests with:

```bash
mise run test
```

This runs `ginkgo run -r`. The CLI can also be called directly for things like
focusing on a single suite:

```bash
ginkgo run ./internal/health
```

## License

Sprout is licensed under the MIT License. See [LICENSE](LICENSE) for the full
license text.
