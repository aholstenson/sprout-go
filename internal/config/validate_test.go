package config_test

import (
	"errors"

	"github.com/aholstenson/sprout-go/internal/config"
	"github.com/aholstenson/sprout-go/internal/logging"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var (
	errTooSmall = errors.New("MIN must not be more than MAX")
	errNoName   = errors.New("NAME must not be empty")
)

// RangeConfig validates with a value receiver.
type RangeConfig struct {
	Min int `env:"MIN" envDefault:"1"`
	Max int `env:"MAX" envDefault:"10"`
}

func (c RangeConfig) Validate() error {
	if c.Min > c.Max {
		return errTooSmall
	}

	return nil
}

// NamedConfig validates with a pointer receiver.
type NamedConfig struct {
	Name string `env:"NAME"`
}

func (c *NamedConfig) Validate() error {
	if c.Name == "" {
		return errNoName
	}

	return nil
}

// MultiConfig reports more than one problem.
type MultiConfig struct {
	Name string `env:"NAME"`
	Min  int    `env:"MIN" envDefault:"5"`
	Max  int    `env:"MAX" envDefault:"1"`
}

func (c MultiConfig) Validate() error {
	var errs []error
	if c.Name == "" {
		errs = append(errs, errNoName)
	}

	if c.Min > c.Max {
		errs = append(errs, errTooSmall)
	}

	return errors.Join(errs...)
}

// RequiredValidatedConfig records if Validate was called.
type RequiredValidatedConfig struct {
	Host string `env:"HOST,required"`
}

var requiredValidated bool

func (c RequiredValidatedConfig) Validate() error {
	requiredValidated = true
	return nil
}

var _ = Describe("Validation", func() {
	It("accepts a valid configuration", func() {
		provider, ok := config.Config("TEST", RangeConfig{}).(func(config.In) (RangeConfig, error))
		Expect(ok).To(BeTrue())

		cfg, err := provider(config.In{})
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Max).To(Equal(10))
	})

	It("calls Validate with a value receiver", func() {
		GinkgoT().Setenv("TEST_MIN", "20")

		provider, ok := config.Config("TEST", RangeConfig{}).(func(config.In) (RangeConfig, error))
		Expect(ok).To(BeTrue())

		_, err := provider(config.In{})
		Expect(err).To(MatchError(errTooSmall))
		Expect(err).To(MatchError(ContainSubstring("configuration is not valid")))
	})

	It("calls Validate with a pointer receiver for a value type", func() {
		provider, ok := config.Config("TEST", NamedConfig{}).(func(config.In) (NamedConfig, error))
		Expect(ok).To(BeTrue())

		_, err := provider(config.In{})
		Expect(err).To(MatchError(errNoName))
	})

	It("calls Validate for a pointer type", func() {
		provider, ok := config.Config("TEST", &NamedConfig{}).(func(config.In) (*NamedConfig, error))
		Expect(ok).To(BeTrue())

		_, err := provider(config.In{})
		Expect(err).To(MatchError(errNoName))
	})

	It("does not call Validate when the values can not be read", func() {
		requiredValidated = false

		provider, ok := config.Config("TEST", RequiredValidatedConfig{}).(func(config.In) (RequiredValidatedConfig, error))
		Expect(ok).To(BeTrue())

		_, err := provider(config.In{})
		Expect(err).To(HaveOccurred())
		Expect(requiredValidated).To(BeFalse())
	})

	It("logs each joined problem on its own", func() {
		core, logs := observer.New(zapcore.InfoLevel)

		app := fx.New(
			fx.NopLogger,
			logging.Module(zap.New(core)),
			fx.Provide(config.Config("TEST", MultiConfig{})),
			fx.Invoke(func(MultiConfig) {}),
		)

		err := app.Err()
		Expect(err).To(MatchError(errNoName))
		Expect(err).To(MatchError(errTooSmall))

		entries := logs.FilterMessage("Configuration is not valid").All()
		Expect(entries).To(HaveLen(2))
		Expect(entries[0].ContextMap()).To(HaveKeyWithValue("type", "*config_test.MultiConfig"))
		Expect(entries[0].ContextMap()).To(HaveKeyWithValue("error", errNoName.Error()))
		Expect(entries[1].ContextMap()).To(HaveKeyWithValue("error", errTooSmall.Error()))
	})

	It("stops the application from starting", func() {
		GinkgoT().Setenv("TEST_MIN", "20")

		app := fx.New(
			fx.NopLogger,
			logging.Module(zap.NewNop()),
			fx.Provide(config.Config("TEST", RangeConfig{})),
			fx.Invoke(func(RangeConfig) {}),
		)

		Expect(app.Err()).To(MatchError(errTooSmall))
	})

	It("validates with BindConfig", func() {
		var cfg NamedConfig
		err := config.BindConfig("TEST", &cfg)
		Expect(err).To(MatchError(errNoName))
	})

	It("validates with ReadConfig", func() {
		GinkgoT().Setenv("TEST_MIN", "20")

		_, err := config.ReadConfig[RangeConfig]("TEST")
		Expect(err).To(MatchError(errTooSmall))
	})
})
