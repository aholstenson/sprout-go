package config

import (
	"reflect"

	"github.com/caarlos0/env/v7"
	"github.com/go-logr/logr"
)

// Config will read configuration from the environment and provide the
// specified type to the application.
func Config[T any](prefix string, value T) any {
	if prefix != "" {
		prefix += "_"
	}

	return func(logger logr.Logger) (T, error) {
		var config = value

		opts := env.Options{
			Prefix: prefix,
			OnSet: func(tag string, value interface{}, isDefault bool) {
				if !isDefault {
					logger.V(1).Info("Read config value", "key", tag, "value", value)
				} else {
					logger.V(1).Info("Config value set to default", "key", tag, "value", value)
				}
			},
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
