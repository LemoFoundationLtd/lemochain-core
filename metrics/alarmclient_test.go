package metrics

import (
	"fmt"
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
func TestAlarmManager_Start(t *testing.T) {
	func(n interface{}) {
		if v, ok := n.(int); ok {
			fmt.Println(v)
		}
		if v, ok := n.(string); ok {
			fmt.Println(v)
		}
		switch t := n.(type) {

		}
	}()

}
