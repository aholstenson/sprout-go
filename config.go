package sprout

import (
	"github.com/levelfourab/sprout-go/internal/config"
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
//		fx.Provide(sprout.Config("HTTP", Config{})),
//		fx.Invoke(func(config Config) {
//			// ...
//		}),
//	).Run()
func Config[T any](prefix string, value T) any {
	return config.Config(prefix, value)
}

// BindConfig is an on-demand version of Config. It will read configuration
// from the environment and bind them to the specified struct.
func BindConfig(prefix string, value any) any {
	return config.BindConfig(prefix, value)
}
