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
	var core zapcore.Core
	if internal.CheckIfDevelopment() {
		core = createDevelopmentCore()
	} else {
		core = createProductionCore()
	}

	logger := zap.New(core)
	return logger, nil
}

func createDevelopmentCore() zapcore.Core {
	config := prettyconsole.NewEncoderConfig()
	encoder := prettyconsole.NewEncoder(config)
	return zapcore.NewCore(encoder, os.Stderr, zap.InfoLevel)
}

func createProductionCore() zapcore.Core {
	// TODO: Connect to OpenTelemetry Collector
	config := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(config)
	return zapcore.NewCore(encoder, os.Stderr, zap.InfoLevel)
}
