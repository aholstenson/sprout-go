package logging

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type FxLogger struct {
	zapLogger       *fxevent.ZapLogger
	detailedLogging bool

	// onEnd runs after the logger has logged the last event of the
	// application. It may be nil.
	onEnd func()
}

func NewFxLogger(logger *zap.Logger, detailedLogging bool) fxevent.Logger {
	return newFxLogger(logger, detailedLogging)
}

func newFxLogger(logger *zap.Logger, detailedLogging bool) *FxLogger {
	return &FxLogger{
		zapLogger:       &fxevent.ZapLogger{Logger: CreateLogger(logger, []string{"fx"})},
		detailedLogging: detailedLogging,
	}
}

func (l *FxLogger) LogEvent(event fxevent.Event) {
	if l.shouldLog(event) {
		l.zapLogger.LogEvent(event)
	}

	if l.onEnd != nil && isEndEvent(event) {
		l.onEnd()
	}
}

func (l *FxLogger) shouldLog(event fxevent.Event) bool {
	if l.detailedLogging {
		return true
	}

	return !isNoisyEvent(event) || hasError(event)
}

// isEndEvent returns true for the last event that Fx logs for an application:
// a failed start, which also covers an application that could not be created,
// or a stop.
func isEndEvent(event fxevent.Event) bool {
	switch e := event.(type) {
	case *fxevent.Started:
		return e.Err != nil
	case *fxevent.Stopped:
		return true
	default:
		return false
	}
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
	case *fxevent.Decorated:
		return e.Err != nil
	case *fxevent.Replaced:
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
