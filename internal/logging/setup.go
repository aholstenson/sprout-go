package logging

import (
	"os"

	"github.com/aholstenson/sprout-go/internal"
	prettyconsole "github.com/thessem/zap-prettyconsole"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CreateRootLogger creates the root logger of the application.
func CreateRootLogger() (*zap.Logger, error) {
	var cores []zapcore.Core
	if internal.CheckIfDevelopment() {
		cores = append(cores, createDevelopmentCore())
	} else {
		cores = append(cores, createProductionCore())
	}

	if logFile := os.Getenv("LOG_FILE_OUTPUT"); logFile != "" {
		fileCore, err := createFileCore(logFile)
		if err != nil {
			return nil, err
		}

		cores = append(cores, fileCore)
	}

	// TODO: Connect to OpenTelemetry Collector
	logger := zap.New(zapcore.NewTee(cores...))
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
