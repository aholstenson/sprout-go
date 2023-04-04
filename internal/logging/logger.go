package logging

import (
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Logger(name ...string) any {
	return fx.Annotate(func(logger *zap.Logger) *zap.Logger {
		return logger.Named(strings.Join(name, "."))
	}, fx.ParamTags(`name:"logging.zap"`))
}

func SugaredLogger(name ...string) any {
	return fx.Annotate(func(logger *zap.Logger) *zap.SugaredLogger {
		return logger.Named(strings.Join(name, ".")).Sugar()
	}, fx.ParamTags(`name:"logging.zap"`))
}

func LogrLogger(name ...string) any {
	return fx.Annotate(func(logger logr.Logger) logr.Logger {
		return logger.WithName(strings.Join(name, "."))
	}, fx.ParamTags(`name:"logging.logr"`))
}
