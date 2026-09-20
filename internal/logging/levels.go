package logging

import (
	"os"
	"strings"

	"go.uber.org/zap/zapcore"
)

// determineLevel determines the log level by checking for environment
// variables like LOG_LEVEL_NAMEPART1_NAMEPART2, LOG_LEVEL_NAMEPART1 etc.
// Lastly checking LOG_LEVEL for the root logger.
//
// If no environment variable is found, it returns the INFO level.
func determineLevel(name []string) zapcore.Level {
	for i := len(name); i > 0; i-- {
		level := levelFromEnv(name[:i])
		if level != zapcore.InvalidLevel {
			return level
		}
	}

	// Check for root LOG_LEVEL environment variable
	value := os.Getenv("LOG_LEVEL")
	if value != "" {
		level, err := zapcore.ParseLevel(value)
		if err == nil {
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

// levelChangingCore decides which levels reach the core it wraps. Entries that
// pass the level check are handed to the wrapped core, so sampling and the
// levels of individual outputs below still apply.
//
// The cores that write to an output accept every level. This core is what
// applies a level, which lets a named logger be more or less verbose than the
// root logger.
type levelChangingCore struct {
	core  zapcore.Core
	level zapcore.Level
}

// withLevel applies level to core. A level applied earlier is replaced instead
// of nested, so that a named logger can be more verbose than the root logger.
func withLevel(core zapcore.Core, level zapcore.Level) zapcore.Core {
	if applied, ok := core.(*levelChangingCore); ok {
		core = applied.core
	}

	return &levelChangingCore{core: core, level: level}
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
