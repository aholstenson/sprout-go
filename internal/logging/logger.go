package logging

import (
	"log/slog"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

func CreateLogger(rootLogger *zap.Logger, name []string) *zap.Logger {
	result := rootLogger.Named(strings.Join(name, "."))

	level := determineLevel(name)
	if level != zapcore.InfoLevel {
		rootLogger.Info("Setting log level", zap.String("name", strings.Join(name, ".")), zap.String("level", level.String()))
		result = result.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return &levelChangingCore{core: core, level: level}
		}))
	}

	return result
}

func Logger(name ...string) any {
	return fx.Annotate(func(logger *zap.Logger) *zap.Logger {
		return CreateLogger(logger, name)
	}, fx.ParamTags(`name:"logging.zap"`))
}

func SugaredLogger(name ...string) any {
	return fx.Annotate(func(logger *zap.Logger) *zap.SugaredLogger {
		return CreateLogger(logger, name).Sugar()
	}, fx.ParamTags(`name:"logging.zap"`))
}

func LogrLogger(name ...string) any {
	return fx.Annotate(func(logger *zap.Logger) logr.Logger {
		childLogger := CreateLogger(logger, name)
		return zapr.NewLogger(childLogger)
	}, fx.ParamTags(`name:"logging.zap"`))
}

func SlogLogger(name ...string) any {
	return fx.Annotate(func(logger *zap.Logger) *slog.Logger {
		logger = CreateLogger(logger, name)
		return slog.New(zapslog.NewHandler(logger.Core(), &zapslog.HandlerOptions{
			LoggerName: logger.Name(),
		}))
	}, fx.ParamTags(`name:"logging.zap"`))
}
