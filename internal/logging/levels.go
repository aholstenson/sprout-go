package logging

import (
	"os"
	"strings"

	"go.uber.org/zap/zapcore"
)

// determineLevel determines the log level by checking for environment
// variables like LOG_LEVEL_NAMEPART1_NAMEPART2, LOG_LEVEL_NAMEPART1 etc.
// If no environment variable is found, it returns the INFO level.
func determineLevel(name []string) zapcore.Level {
	for i := len(name); i > 0; i-- {
		level := levelFromEnv(name[:i])
		if level != zapcore.InvalidLevel {
			return level
		}
	}

	return zapcore.InfoLevel
}

// levelFromEnv returns the log level from the environment variable
func levelFromEnv(name []string) zapcore.Level {
	levelName := "LOG_LEVEL_" + strings.ReplaceAll(strings.ToUpper(strings.Join(name, "_")), ".", "_")
	value := os.Getenv(levelName)
	if value == "" {
		return zapcore.InvalidLevel
	}

	level, err := zapcore.ParseLevel(value)
	if err != nil {
		return zapcore.InvalidLevel
	}

	return level
}

type levelChangingCore struct {
	core  zapcore.Core
	level zapcore.Level
}

func (c *levelChangingCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level)
}

func (c *levelChangingCore) With(fields []zapcore.Field) zapcore.Core {
	return &levelChangingCore{c.core.With(fields), c.level}
}

func (c *levelChangingCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return checkedEntry
	}

	return c.core.Check(entry, checkedEntry)
}

func (c *levelChangingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.core.Write(entry, fields)
}

func (c *levelChangingCore) Sync() error {
	return c.core.Sync()
}
