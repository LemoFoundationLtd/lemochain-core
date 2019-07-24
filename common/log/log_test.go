package log

import (
	"testing"
)

func TestWarn(t *testing.T) {
	Setup(LevelWarn, false, false)
	Debug("should be invisible")
	Info("should be invisible")
	Warn("汉字")
	Error("👽🚀")

	Setup(LevelDebug, false, false)
	Debug("debug level visible")
	Info("info level visible")

	Setup(LevelDebug, false, true)
	Debug("show code line")
	Info("show code line")

	// Crit("critical error")
}
