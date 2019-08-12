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

func TestEventf(t *testing.T) {
	Setup(LevelWarn, false, false)
	for i := 0; i < 10; i++ {
		Eventf(TxEvent, "test log %d", i)
		Event(TxEvent, "test")
	}

}
