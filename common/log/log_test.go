package log

import (
	"testing"
	"time"
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

// TestDoRotate 测试滚动日志
func TestDoRotate(t *testing.T) {
	Setup(LevelInfo, true, false)
	go func() {
		DoRotate(LevelInfo, true, false)
	}()

	go func() {
		for {
			Info("测试info")
			Debug("测试debug")
			Warn("测试warn")
			Error("测试error")
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {}
}
