package runtime

import (
	"fmt"
	"math"
	"runtime/debug"
	"strings"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
)

func Setup(logger *zap.Logger) {
	// Setup GOMAXPROCS to be the number of CPUs available.
	_, err := maxprocs.Set(maxprocs.Logger(func(s string, i ...any) {
		s = strings.TrimPrefix(s, "maxprocs: ")
		s = fmt.Sprintf(s, i...)
		logger.Info(s)
	}))
	if err != nil {
		logger.Warn("Unable to set GOMAXPROCS", zap.Error(err))
	}

	setupMemoryLimit(logger)
}

func setupMemoryLimit(logger *zap.Logger) {
	if memLimit := debug.SetMemoryLimit(-1); memLimit != math.MaxInt64 {
		logger.Info("Leaving GOMEMLIMIT=" + formatBytes(memLimit))
		return
	}

	memLimit, err := memlimit.SetGoMemLimit(0.9)
	if err != nil {
		logger.Info("Unable to set GOMEMLIMIT: " + err.Error())
		return
	}
	logger.Info("Setting GOMEMLIMIT=" + formatBytes(memLimit))
}

// formatBytes formats a byte count into MiB.
func formatBytes(bytes int64) string {
	return fmt.Sprintf("%.0fMiB", float64(bytes)/1024/1024)
}
