package config

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/aholstenson/sprout-go/internal/logging"
	"github.com/caarlos0/env/v11"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type In struct {
	fx.In

	// RootLogger is the root logger of the application. It is optional, so
	// that configuration can also be read without the logging module.
	RootLogger *zap.Logger `name:"logging.zap" optional:"true"`
}

// Config will read configuration from the environment and provide the
// specified type to the application.
func Config[T any](prefix string, value T) any {
	if prefix != "" {
		prefix += "_"
	}

	return func(in In) (T, error) {
		config := value

		logger := configLogger(in.RootLogger)

		opts := env.Options{
			Prefix: prefix,
			OnSet:  logFunc(logger),
		}

		// target is the pointer that the values are read into.
		var target any = &config
		if reflect.TypeOf(config).Kind() == reflect.Pointer {
			target = config
		}

		err := env.ParseWithOptions(target, opts)
		if err != nil {
			// Report each problem on its own, so that a reader of the log sees
			// which variable caused it.
			var aggregateError env.AggregateError
			if errors.As(err, &aggregateError) {
				for _, err := range aggregateError.Errors {
					logError(logger, err)
				}
			} else {
				logError(logger, err)
			}

			// Keep the cause, so that a caller can inspect it with errors.As.
			return config, fmt.Errorf("failed to load configuration: %w", err)
		}

		err = validate(target)
		if err != nil {
			for _, err := range splitJoined(err) {
				logger.Error("Configuration is not valid", zap.String("type", fmt.Sprintf("%T", target)), zap.Error(err))
			}

			return config, err
		}

		return config, nil
	}
}

// Validator is implemented by configuration structs that check their own
// values. Validate is called after the values are read from the environment,
// and only when they were read without errors. A Validate that finds more than
// one problem can return them with errors.Join, so that each is reported.
//
// Only the struct that is read is validated. A struct that contains other
// structs must call their Validate itself.
type Validator interface {
	Validate() error
}

// validate calls Validate on the struct that target points to, if the struct
// implements Validator. Both value and pointer receivers are found, as target
// is a pointer.
func validate(target any) error {
	validator, ok := target.(Validator)
	if !ok {
		return nil
	}

	err := validator.Validate()
	if err != nil {
		// Keep the cause, so that a caller can inspect it with errors.As.
		return fmt.Errorf("configuration is not valid: %w", err)
	}

	return nil
}

// splitJoined returns the errors that errors.Join combined, or err itself.
func splitJoined(err error) []error {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		return joined.Unwrap()
	}

	return []error{err}
}

// configLogger creates the logger that reports which values were read. It
// falls back to the global logger when the application has no root logger,
// which is the case when configuration is read on demand.
func configLogger(rootLogger *zap.Logger) *zap.Logger {
	if rootLogger == nil {
		rootLogger = zap.L()
	}

	return logging.CreateLogger(rootLogger, []string{"config"})
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

	var envVarIsNotSetError env.VarIsNotSetError
	if errors.As(err, &envVarIsNotSetError) {
		logger.Error("Required environment variable is not set", zap.String("key", envVarIsNotSetError.Key))
		return
	}

	var emptyEnvVarError env.EmptyVarError
	if errors.As(err, &emptyEnvVarError) {
		logger.Error("Environment variable should not be empty", zap.String("key", emptyEnvVarError.Key))
		return
	}

	logger.Error("Failed to parse configuration", zap.Error(err))
}

// BindConfig is an on-demand version of Config. It will read configuration
// from the environment and bind them to the specified struct. If the struct
// implements Validator, BindConfig validates it after it is read.
func BindConfig(prefix string, value any) error {
	if prefix != "" {
		prefix += "_"
	}

	logger := configLogger(nil)

	err := env.ParseWithOptions(value, env.Options{
		Prefix: prefix,
		OnSet:  logFunc(logger),
	})
	if err != nil {
		return err
	}

	return validate(value)
}

// ReadConfig will read configuration from the environment and return the
// specified type.
func ReadConfig[T any](prefix string) (T, error) {
	var value T

	err := BindConfig(prefix, &value)
	if err != nil {
		return value, err
	}

	return value, nil
}
