package metrics

import (
	"github.com/rcrowley/go-metrics"
	"log"
	"os"
	"testing"
	"time"
)

func TestNewCounter(t *testing.T) {
	// counter := NewCounter("co:")
	// meter := NewMeter("me:")
	timer := NewTimer("metrics")
	start := time.Now()
	// g := NewGauge("gu")
	go func() {
		for {
			time.Sleep(1 * time.Second)
			// counter.Inc(2)
			// meter.Mark(2)
			timer.UpdateSince(start)
			// g.Update(1)
		}
	}()
	go metrics.Log(metrics.DefaultRegistry, 4*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	select {}
}
