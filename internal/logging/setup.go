package logging

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/aholstenson/sprout-go/internal"
	"github.com/aholstenson/sprout-go/internal/otel"
	"github.com/caarlos0/env/v11"
	prettyconsole "github.com/thessem/zap-prettyconsole"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logConfig struct {
	ConsoleOutput bool   `env:"LOG_CONSOLE_OUTPUT" envDefault:"true"`
	FileOutput    string `env:"LOG_FILE_OUTPUT"`
	Sampling      struct {
		Initial    int `env:"LOG_SAMPLING_INITIAL" envDefault:"100"`
		Thereafter int `env:"LOG_SAMPLING_THEREAFTER" envDefault:"100"`
	} `env:"LOG_SAMPLING"`
}

// CreateRootLogger creates the root logger of the application.
func CreateRootLogger(serviceInfo internal.ServiceInfo) (_ *RootLogger, err error) {
	opts := []zap.Option{zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel)}
	var cores []zapcore.Core

	config, err := env.ParseAs[logConfig]()
	if err != nil {
		return nil, err
	}

	var file *logFile
	defer func() {
		// Release the file if a later step fails, as no RootLogger will own it.
		if err != nil && file != nil {
			_ = file.Close()
		}
	}()

	if internal.CheckIfDevelopment() {
		if config.ConsoleOutput {
			cores = append(cores, createDevelopmentCore())
		}
		opts = append(opts, zap.Development())
	} else {
		if config.ConsoleOutput {
			cores = append(cores, createProductionCore())
		}
	}

	if config.FileOutput != "" {
		file, err = openLogFile(config.FileOutput)
		if err != nil {
			return nil, err
		}

		cores = append(cores, createFileCore(file))
	}

	// Connect to OpenTelemetry
	provider, otelShutdown, err := otel.InitLogging(serviceInfo)
	if err != nil {
		return nil, err
	} else if otelShutdown != nil {
		otelCore := otelzap.NewCore("global", otelzap.WithLoggerProvider(provider))
		// Limit the otelzap core to info, so that debug logs are not exported
		// even when a logger below is more verbose.
		cores = append(cores, withLevel(otelCore, zap.InfoLevel))
	}

	core := zapcore.NewTee(cores...)

	if config.Sampling.Initial > 0 {
		// Setup sampling if not disabled
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second,
			config.Sampling.Initial,
			config.Sampling.Thereafter,
		)
	}

	// The cores above accept every level. This decides what the root logger
	// emits, and CreateLogger replaces it for each named logger.
	core = withLevel(core, determineLevel(nil))

	return NewRootLogger(zap.New(core, opts...), releaseOutputs(otelShutdown, file)), nil
}

// releaseOutputs returns a function that releases the outputs that need it.
// It returns nil when no output needs to be released.
func releaseOutputs(otelShutdown func(ctx context.Context) error, file *logFile) func(ctx context.Context) error {
	if otelShutdown == nil && file == nil {
		return nil
	}

	return func(ctx context.Context) error {
		var errs []error
		if otelShutdown != nil {
			errs = append(errs, otelShutdown(ctx))
		}

		if file != nil {
			errs = append(errs, file.Close())
		}

		return errors.Join(errs...)
	}
}

func createDevelopmentCore() zapcore.Core {
	config := prettyconsole.NewEncoderConfig()
	encoder := prettyconsole.NewEncoder(config)
	return zapcore.NewCore(encoder, os.Stderr, zap.DebugLevel)
}

func createProductionCore() zapcore.Core {
	config := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(config)
	return zapcore.NewCore(encoder, os.Stderr, zap.DebugLevel)
}

func createFileCore(file *logFile) zapcore.Core {
	config := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(config)
	return zapcore.NewCore(encoder, file, zap.DebugLevel)
}

// logFile is the file that LOG_FILE_OUTPUT names.
//
// Fx logs some events after the logger has stopped and closed the file. A
// write to a closed file makes zap report an error on stderr, so logFile drops
// writes after it is closed.
type logFile struct {
	mu     sync.Mutex
	file   *os.File
	closed bool
}

func openLogFile(path string) (*logFile, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666) //nolint:gosec
	if err != nil {
		return nil, err
	}

	return &logFile{file: file}, nil
}

func (f *logFile) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return len(p), nil
	}

	return f.file.Write(p)
}

func (f *logFile) Sync() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return nil
	}

	return f.file.Sync()
}

// Close closes the file. Calls after the first do nothing.
func (f *logFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return nil
	}

	f.closed = true
	return f.file.Close()
}
