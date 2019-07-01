package metrics

import (
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"testing"
)

func init() {
	log.Setup(log.LevelDebug, false, true)
}

func TestNewAlarmManager(t *testing.T) {
	am := NewAlarmManager()
	am.Start()
}
