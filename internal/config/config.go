package config

import (
	"reflect"

	"github.com/caarlos0/env/v7"
	"github.com/go-logr/logr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ConfigIn struct {
	fx.In

	Logger     *zap.Logger  `optional:"true"`
	LogrLogger *logr.Logger `optional:"true"`
}

// Config will read configuration from the environment and provide the
// specified type to the application.
func Config[T any](prefix string, value T) any {
	if prefix != "" {
		prefix += "_"
	}

	return func(in ConfigIn) (T, error) {
		var config = value

		opts := env.Options{
			Prefix: prefix,
			OnSet:  logFunc(in),
		}

		var err error
		if reflect.TypeOf(config).Kind() == reflect.Ptr {
			err = env.Parse(config, opts)
		} else {
			err = env.Parse(&config, opts)
		}

		if err != nil {
			return config, err
		}

		return config, nil
	}
}

func logFunc(in ConfigIn) func(tag string, value interface{}, isDefault bool) {
	if in.Logger != nil {
		logger := in.Logger
		return func(tag string, value interface{}, isDefault bool) {
			if !isDefault {
				logger.Info("Read config value", zap.String("key", tag), zap.Any("value", value))
			} else {
				logger.Debug("Config value set to default", zap.String("key", tag), zap.Any("value", value))
			}
		}
	} else if in.LogrLogger != nil {
		logger := in.LogrLogger
		return func(tag string, value interface{}, isDefault bool) {
			if !isDefault {
				logger.Info("Read config value", "key", tag, "value", value)
			} else {
				logger.V(1).Info("Config value set to default", "key", tag, "value", value)
			}
		}
	}

	return func(tag string, value interface{}, isDefault bool) {}
}

// BindConfig is an on-demand version of Config. It will read configuration
// from the environment and bind them to the specified struct.
func BindConfig(prefix string, value any) any {
	if prefix != "" {
		prefix += "_"
	}

	err := env.Parse(value, env.Options{
		Prefix: prefix,
	})
	if err != nil {
		return err
	}

	return nil
}
