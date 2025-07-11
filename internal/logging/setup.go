package logging

import (
	"os"
	"time"

	"github.com/aholstenson/sprout-go/internal"
	"github.com/caarlos0/env/v11"
	prettyconsole "github.com/thessem/zap-prettyconsole"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logConfig struct {
	FileOutput string `env:"LOG_FILE_OUTPUT"`
	Sampling   struct {
		Initial    int `env:"LOG_SAMPLING_INITIAL" envDefault:"100"`
		Thereafter int `env:"LOG_SAMPLING_THEREAFTER" envDefault:"100"`
	} `env:"LOG_SAMPLING"`
}

// CreateRootLogger creates the root logger of the application.
func CreateRootLogger() (*zap.Logger, error) {
	opts := []zap.Option{zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel)}
	var cores []zapcore.Core

	if internal.CheckIfDevelopment() {
		cores = append(cores, createDevelopmentCore())
		opts = append(opts, zap.Development())
	} else {
		cores = append(cores, createProductionCore())
	}

	config, err := env.ParseAs[logConfig]()
	if err != nil {
		return nil, err
	}

	if config.FileOutput != "" {
		fileCore, err := createFileCore(config.FileOutput)
		if err != nil {
			return nil, err
		}

		cores = append(cores, fileCore)
	}

	// TODO: Connect to OpenTelemetry Collector

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

	logger := zap.New(core, opts...)
	return logger, nil
}

func createDevelopmentCore() zapcore.Core {
	config := prettyconsole.NewEncoderConfig()
	encoder := prettyconsole.NewEncoder(config)
	return zapcore.NewCore(encoder, os.Stderr, zap.InfoLevel)
}

func createProductionCore() zapcore.Core {
	config := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(config)
	return zapcore.NewCore(encoder, os.Stderr, zap.InfoLevel)
}

func createFileCore(logFile string) (zapcore.Core, error) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return nil, err
	}

	config := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(config)
	return zapcore.NewCore(encoder, zapcore.AddSync(file), zap.InfoLevel), nil
}
