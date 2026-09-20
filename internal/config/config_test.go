package config_test

import (
	"errors"

	"github.com/aholstenson/sprout-go/internal/config"
	"github.com/aholstenson/sprout-go/internal/logging"
	"github.com/caarlos0/env/v11"
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

type RequiredConfig struct {
	Host string `env:"HOST,required"`
}

type NumberConfig struct {
	Port int `env:"PORT"`
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

	It("keeps the cause when a required variable is not set", func() {
		provider, ok := config.Config("TEST", RequiredConfig{}).(func(config.In) (RequiredConfig, error))
		Expect(ok).To(BeTrue())

		_, err := provider(config.In{})
		Expect(err).To(HaveOccurred())

		var notSet env.VarIsNotSetError
		Expect(errors.As(err, &notSet)).To(BeTrue())
		Expect(notSet.Key).To(Equal("TEST_HOST"))
	})

	It("keeps the cause when a value can not be parsed", func() {
		t := GinkgoT()
		t.Setenv("TEST_PORT", "not-a-number")

		provider, ok := config.Config("TEST", NumberConfig{}).(func(config.In) (NumberConfig, error))
		Expect(ok).To(BeTrue())

		_, err := provider(config.In{})
		Expect(err).To(HaveOccurred())

		var parseError env.ParseError
		Expect(errors.As(err, &parseError)).To(BeTrue())
		Expect(parseError.Name).To(Equal("Port"))
	})
})
