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

	// g := NewGauge("gu")
	go func() {
		start := time.Now()
		for {
			// start := time.Now()
			time.Sleep(1 * time.Second)
			// counter.Inc(1)
			// meter.Mark(1)
			timer.UpdateSince(start)
			// g.Update(1)
		}
	}()
	go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	select {}
}
