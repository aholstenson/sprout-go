package config

import (
	"errors"
	"reflect"

	"github.com/caarlos0/env/v11"
	"github.com/aholstenson/sprout-go/internal/logging"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type In struct {
	fx.In

	Logger *zap.Logger `optional:"true"`
}

// Config will read configuration from the environment and provide the
// specified type to the application.
func Config[T any](prefix string, value T) any {
	if prefix != "" {
		prefix += "_"
	}

	return func(in In) (T, error) {
		config := value

		logger := in.Logger
		if logger == nil {
			// No logger provided, use the default logger
			logger = logging.CreateLogger(zap.L(), []string{"config"})
		}

		opts := env.Options{
			Prefix: prefix,
			OnSet:  logFunc(logger),
		}

		var err error
		if reflect.TypeOf(config).Kind() == reflect.Ptr {
			err = env.ParseWithOptions(config, opts)
		} else {
			err = env.ParseWithOptions(&config, opts)
		}

		var aggregateError env.AggregateError
		if errors.As(err, &aggregateError) {
			for _, err := range aggregateError.Errors {
				logError(logger, err)
			}

			return config, errors.New("failed to load configuration")
		} else if err != nil {
			return config, err
		}

		return config, nil
	}
}

func logFunc(logger *zap.Logger) func(tag string, value interface{}, isDefault bool) {
	return func(tag string, value interface{}, isDefault bool) {
		if !isDefault {
			logger.Info("Read config value from environment", zap.String("key", tag))
		} else {
			logger.Info("Config value set to default", zap.String("key", tag), zap.Any("value", value))
		}
	}
}

func logError(logger *zap.Logger, err error) {
	var parseError env.ParseError
	if errors.As(err, &parseError) {
		logger.Error("Failed to parse configuration value", zap.String("key", parseError.Name), zap.Error(parseError.Err))
		return
	}

	var envVarIsNotSetError env.EnvVarIsNotSetError
	if errors.As(err, &envVarIsNotSetError) {
		logger.Error("Required environment variable is not set", zap.String("key", envVarIsNotSetError.Key))
		return
	}

	var emptyEnvVarError env.EmptyEnvVarError
	if errors.As(err, &emptyEnvVarError) {
		logger.Error("Environment variable should not be empty", zap.String("key", emptyEnvVarError.Key))
		return
	}

	logger.Error("Failed to parse configuration", zap.Error(err))
}

// BindConfig is an on-demand version of Config. It will read configuration
// from the environment and bind them to the specified struct.
func BindConfig(prefix string, value any) any {
	if prefix != "" {
		prefix += "_"
	}

	err := env.ParseWithOptions(value, env.Options{
		Prefix: prefix,
	})
	if err != nil {
		return err
	}

	return nil
}
