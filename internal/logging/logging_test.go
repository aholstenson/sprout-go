package logging_test

import (
	"log/slog"

	"github.com/aholstenson/sprout-go/internal/logging"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Logging", func() {
	Describe("*zap.Logger", func() {
		It("should be able to provide named *zap.Logger", func() {
			t := GinkgoT()

			var logger *zap.Logger
			app := fxtest.New(
				t,
				logging.Module(zaptest.NewLogger(t)),
				fx.Provide(logging.Logger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})

	Describe("*zap.SugaredLogger", func() {
		It("should be able to provide named *zap.SugaredLogger", func() {
			t := GinkgoT()

			var logger *zap.SugaredLogger
			app := fxtest.New(
				t,
				logging.Module(zaptest.NewLogger(t)),
				fx.Provide(logging.SugaredLogger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})

	Describe("logr.Logger", func() {
		It("should be able to provide named logr.Logger", func() {
			t := GinkgoT()

			var logger logr.Logger
			app := fxtest.New(
				t,
				logging.Module(zaptest.NewLogger(t)),
				fx.Provide(logging.LogrLogger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})

	Describe("slog.Logger", func() {
		It("should be able to provide named *slog.Logger", func() {
			t := GinkgoT()

			var logger *slog.Logger
			app := fxtest.New(
				t,
				logging.Module(zaptest.NewLogger(t)),
				fx.Provide(logging.SlogLogger("test")),
				fx.Populate(&logger),
			)

			app.RequireStart()
			defer app.RequireStop()

			Expect(logger).ToNot(BeNil())
		})
	})

	Describe("Level Management", func() {
		Describe("Environment Variable Level Detection", func() {
			It("should use DEBUG level when LOG_LEVEL is set to debug for root logger", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				// Root logger (empty name array)
				logger := logging.CreateLogger(rootLogger, []string{})

				logger.Debug("root debug message")
				logger.Info("root info message")

				Expect(logs.Len()).To(Equal(3)) // Setting log level message + debug + info

				// Find our actual log messages (skip the level setting message)
				var debugFound, infoFound bool
				for _, log := range logs.All() {
					if log.Message == "root debug message" {
						debugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					}
					if log.Message == "root info message" {
						infoFound = true
						Expect(log.Level).To(Equal(zapcore.InfoLevel))
					}
				}
				Expect(debugFound).To(BeTrue())
				Expect(infoFound).To(BeTrue())
			})

			It("should override root LOG_LEVEL with more specific environment variables", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL", "error")
				t.Setenv("LOG_LEVEL_SERVICE", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				// Root logger should use error level
				rootLoggerTest := logging.CreateLogger(rootLogger, []string{})
				// Service logger should use debug level (overrides root)
				serviceLogger := logging.CreateLogger(rootLogger, []string{"service"})

				rootLoggerTest.Info("root info")     // Should not appear (below error)
				rootLoggerTest.Error("root error")   // Should appear
				serviceLogger.Debug("service debug") // Should appear
				serviceLogger.Info("service info")   // Should appear

				var rootErrorFound, serviceDebugFound, serviceInfoFound bool
				for _, log := range logs.All() {
					switch log.Message {
					case "root error":
						rootErrorFound = true
						Expect(log.Level).To(Equal(zapcore.ErrorLevel))
					case "service debug":
						serviceDebugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					case "service info":
						serviceInfoFound = true
						Expect(log.Level).To(Equal(zapcore.InfoLevel))
					}
					// root info should not appear
					Expect(log.Message).ToNot(Equal("root info"))
				}
				Expect(rootErrorFound).To(BeTrue())
				Expect(serviceDebugFound).To(BeTrue())
				Expect(serviceInfoFound).To(BeTrue())
			})

			It("should use INFO level when no environment variable is set", func() {
				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				logger := logging.CreateLogger(rootLogger, []string{"test"})

				logger.Debug("debug message")
				logger.Info("info message")

				Expect(logs.Len()).To(Equal(1))
				Expect(logs.All()[0].Message).To(Equal("info message"))
				Expect(logs.All()[0].Level).To(Equal(zapcore.InfoLevel))
			})

			It("should use DEBUG level when LOG_LEVEL_TEST is set to debug", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_TEST", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				logger := logging.CreateLogger(rootLogger, []string{"test"})

				logger.Debug("debug message")
				logger.Info("info message")

				Expect(logs.Len()).To(Equal(3)) // Setting log level message + debug + info

				// Find our actual log messages (skip the level setting message)
				var debugFound, infoFound bool
				for _, log := range logs.All() {
					if log.Message == "debug message" {
						debugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					}
					if log.Message == "info message" {
						infoFound = true
						Expect(log.Level).To(Equal(zapcore.InfoLevel))
					}
				}
				Expect(debugFound).To(BeTrue())
				Expect(infoFound).To(BeTrue())
			})

			It("should use ERROR level when LOG_LEVEL_TEST is set to error", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_TEST", "error")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				logger := logging.CreateLogger(rootLogger, []string{"test"})

				logger.Debug("debug message")
				logger.Info("info message")
				logger.Error("error message")

				// Should only see the level setting message and the error message
				Expect(logs.Len()).To(Equal(2))

				var errorFound bool
				for _, log := range logs.All() {
					if log.Message == "error message" {
						errorFound = true
						Expect(log.Level).To(Equal(zapcore.ErrorLevel))
					}
				}
				Expect(errorFound).To(BeTrue())
			})

			It("should prioritize more specific environment variables", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_TEST", "error")
				t.Setenv("LOG_LEVEL_TEST_COMPONENT", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				// Logger with more specific name should use debug level
				specificLogger := logging.CreateLogger(rootLogger, []string{"test", "component"})

				// Logger with less specific name should use error level
				generalLogger := logging.CreateLogger(rootLogger, []string{"test"})

				specificLogger.Debug("specific debug")
				generalLogger.Debug("general debug")
				generalLogger.Error("general error")

				var specificDebugFound, generalErrorFound bool
				for _, log := range logs.All() {
					if log.Message == "specific debug" {
						specificDebugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					}
					if log.Message == "general error" {
						generalErrorFound = true
						Expect(log.Level).To(Equal(zapcore.ErrorLevel))
					}
					// general debug should not be found since it's below error level
					Expect(log.Message).ToNot(Equal("general debug"))
				}
				Expect(specificDebugFound).To(BeTrue())
				Expect(generalErrorFound).To(BeTrue())
			})

			It("should handle nested namespaces correctly", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_DATABASE", "warn")
				t.Setenv("LOG_LEVEL_DATABASE_CONNECTION", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				dbLogger := logging.CreateLogger(rootLogger, []string{"database"})
				connLogger := logging.CreateLogger(rootLogger, []string{"database", "connection"})

				dbLogger.Info("db info")       // Should not appear (below warn)
				dbLogger.Warn("db warning")    // Should appear
				connLogger.Debug("conn debug") // Should appear
				connLogger.Info("conn info")   // Should appear

				var dbWarnFound, connDebugFound, connInfoFound bool
				for _, log := range logs.All() {
					switch log.Message {
					case "db warning":
						dbWarnFound = true
						Expect(log.Level).To(Equal(zapcore.WarnLevel))
					case "conn debug":
						connDebugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					case "conn info":
						connInfoFound = true
						Expect(log.Level).To(Equal(zapcore.InfoLevel))
					}
					// db info should not appear
					Expect(log.Message).ToNot(Equal("db info"))
				}
				Expect(dbWarnFound).To(BeTrue())
				Expect(connDebugFound).To(BeTrue())
				Expect(connInfoFound).To(BeTrue())
			})

			It("should handle names with dots correctly", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_COM_EXAMPLE_SERVICE", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				logger := logging.CreateLogger(rootLogger, []string{"com.example", "service"})

				logger.Debug("debug with dots")

				var debugFound bool
				for _, log := range logs.All() {
					if log.Message == "debug with dots" {
						debugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					}
				}
				Expect(debugFound).To(BeTrue())
			})
		})

		Describe("Integration with Different Logger Types", func() {
			It("should apply level changes to SugaredLogger", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_SUGAR", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				var logger *zap.SugaredLogger
				app := fxtest.New(
					GinkgoT(),
					logging.Module(rootLogger),
					fx.Provide(logging.SugaredLogger("sugar")),
					fx.Populate(&logger),
				)

				app.RequireStart()
				defer app.RequireStop()

				logger.Debug("sugar debug message")
				logger.Info("sugar info message")

				var debugFound, infoFound bool
				for _, log := range logs.All() {
					if log.Message == "sugar debug message" {
						debugFound = true
					}
					if log.Message == "sugar info message" {
						infoFound = true
					}
				}
				Expect(debugFound).To(BeTrue())
				Expect(infoFound).To(BeTrue())
			})

			It("should apply level changes to LogrLogger", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_LOGR", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				var logger logr.Logger
				app := fxtest.New(
					GinkgoT(),
					logging.Module(rootLogger),
					fx.Provide(logging.LogrLogger("logr")),
					fx.Populate(&logger),
				)

				app.RequireStart()
				defer app.RequireStop()

				// logr uses V() levels where V(1) is debug level
				logger.V(1).Info("logr debug message")
				logger.Info("logr info message")

				Expect(logs.Len()).To(BeNumerically(">=", 2)) // At least our messages (might have level setting message too)
			})
		})

		Describe("Level Fallback Behavior", func() {
			It("should fall back to parent namespace levels", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_PARENT", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				// Child logger should inherit parent's debug level
				childLogger := logging.CreateLogger(rootLogger, []string{"parent", "child", "grandchild"})

				childLogger.Debug("inherited debug")

				var debugFound bool
				for _, log := range logs.All() {
					if log.Message == "inherited debug" {
						debugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					}
				}
				Expect(debugFound).To(BeTrue())
			})

			It("should prioritize most specific match", func() {
				t := GinkgoT()
				t.Setenv("LOG_LEVEL_SERVICE", "error")
				t.Setenv("LOG_LEVEL_SERVICE_API", "warn")
				t.Setenv("LOG_LEVEL_SERVICE_API_V1", "debug")

				core, logs := observer.New(zapcore.InfoLevel)
				rootLogger := zap.New(core)

				logger := logging.CreateLogger(rootLogger, []string{"service", "api", "v1", "endpoint"})

				logger.Debug("most specific debug")

				var debugFound bool
				for _, log := range logs.All() {
					if log.Message == "most specific debug" {
						debugFound = true
						Expect(log.Level).To(Equal(zapcore.DebugLevel))
					}
				}
				Expect(debugFound).To(BeTrue())
			})
		})
	})
})
