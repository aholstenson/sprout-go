package logging

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type FxLogger struct {
	zapLogger       *fxevent.ZapLogger
	detailedLogging bool
}

func NewFxLogger(logger *zap.Logger, detailedLogging bool) fxevent.Logger {
	return &FxLogger{
		zapLogger:       &fxevent.ZapLogger{Logger: CreateLogger(logger, []string{"fx"})},
		detailedLogging: detailedLogging,
	}
}

func (l *FxLogger) LogEvent(event fxevent.Event) {
	if !l.detailedLogging {
		isNoisy := isNoisyEvent(event)
		hasError := hasError(event)
		if isNoisy && !hasError {
			return
		}
	}

	l.zapLogger.LogEvent(event)
}

func isNoisyEvent(event fxevent.Event) bool {
	switch event.(type) {
	case *fxevent.Provided:
		return true
	case *fxevent.Supplied:
		return true
	case *fxevent.Invoking:
		return false
	case *fxevent.BeforeRun:
		return true
	case *fxevent.Decorated:
		return true
	case *fxevent.Replaced:
		return true
	case *fxevent.LoggerInitialized:
		return true
	default:
		return false
	}
}

func hasError(event fxevent.Event) bool {
	switch e := event.(type) {
	case *fxevent.Provided:
		return e.Err != nil
	case *fxevent.Supplied:
		return e.Err != nil
	case *fxevent.Invoked:
		return e.Err != nil
	case *fxevent.Run:
		return e.Err != nil
	case *fxevent.LoggerInitialized:
		return e.Err != nil
	case *fxevent.Started:
		return e.Err != nil
	case *fxevent.Stopped:
		return e.Err != nil
	case *fxevent.OnStartExecuted:
		return e.Err != nil
	case *fxevent.OnStopExecuted:
		return e.Err != nil
	case *fxevent.RolledBack:
		return e.Err != nil
	default:
		return false
	}
}
