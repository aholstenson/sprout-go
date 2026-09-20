package config_test

import (
	"github.com/aholstenson/sprout-go/internal/config"
	"github.com/aholstenson/sprout-go/internal/logging"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

type Config struct {
	Host string `env:"HOST" envDefault:"localhost"`
	Port int    `env:"PORT" envDefault:"8080"`
}

var _ = Describe("Config", func() {
	It("should be able to provide config", func() {
		var readConfig Config
		app := fxtest.New(
			GinkgoT(),
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Provide(config.Config("TEST", Config{})),
			fx.Populate(&readConfig),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(readConfig.Host).To(Equal("localhost"))
		Expect(readConfig.Port).To(Equal(8080))
	})

	It("environment variables set config", func() {
		t := GinkgoT()
		t.Setenv("TEST_HOST", "test")
		t.Setenv("TEST_PORT", "1234")

		var readConfig Config
		app := fxtest.New(
			t,
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Provide(config.Config("TEST", Config{})),
			fx.Populate(&readConfig),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(readConfig.Host).To(Equal("test"))
		Expect(readConfig.Port).To(Equal(1234))
	})

	It("can read config via reference", func() {
		t := GinkgoT()
		t.Setenv("TEST_HOST", "test")
		t.Setenv("TEST_PORT", "1234")

		var readConfig *Config
		app := fxtest.New(
			t,
			logging.Module(zaptest.NewLogger(GinkgoT())),
			fx.Provide(config.Config("TEST", &Config{})),
			fx.Populate(&readConfig),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(readConfig.Host).To(Equal("test"))
		Expect(readConfig.Port).To(Equal(1234))
	})

	It("reports the values it read to the application logger", func() {
		t := GinkgoT()
		t.Setenv("TEST_HOST", "test")

		core, logs := observer.New(zapcore.InfoLevel)

		var readConfig Config
		app := fxtest.New(
			t,
			logging.Module(zap.New(core)),
			fx.Provide(config.Config("TEST", Config{})),
			fx.Populate(&readConfig),
		)
		app.RequireStart()
		defer app.RequireStop()

		Expect(logs.FilterMessage("Read config value from environment").Len()).To(Equal(1))
		Expect(logs.FilterMessage("Config value set to default").Len()).To(Equal(1))
	})
})
