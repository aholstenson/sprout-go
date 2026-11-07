package sprout

import (
	"github.com/aholstenson/sprout-go/internal/config"
)

// Config will read configuration from the environment and provide the
// specified type to the application.
//
// Example:
//
//	type Config struct {
//		Host string `env:"HOST" envDefault:"localhost"`
//		Port int    `env:"PORT" envDefault:"8080"`
//	}
//
//	sprout.New("my-service", "1.0.0").With(
//		fx.Provide(sprout.Config("HTTP", Config{}), fx.Private),
//		fx.Invoke(func(config Config) {
//			// ...
//		}),
//	).Run()
//
// If the type implements ConfigValidator, its Validate method is called after
// the values are read. An error stops the application from starting.
func Config[T any](prefix string, value T) any {
	return config.Config(prefix, value)
}

// ConfigValidator is implemented by configuration structs that check their own
// values. Config, BindConfig and ReadConfig call Validate after they have read
// the values from the environment.
//
// Return several problems with errors.Join, so that each one is logged.
//
// Example:
//
//	type Config struct {
//		MinConns int `env:"MIN_CONNS" envDefault:"1"`
//		MaxConns int `env:"MAX_CONNS" envDefault:"10"`
//	}
//
//	func (c Config) Validate() error {
//		if c.MinConns > c.MaxConns {
//			return errors.New("MIN_CONNS must not be more than MAX_CONNS")
//		}
//		return nil
//	}
type ConfigValidator = config.Validator

// BindConfig is an on-demand version of Config. It will read configuration
// from the environment and bind them to the specified struct.
func BindConfig(prefix string, value any) error {
	return config.BindConfig(prefix, value)
}

// ReadConfig will read configuration from the environment and return the
// specified type.
func ReadConfig[T any](prefix string) (T, error) {
	return config.ReadConfig[T](prefix)
}
