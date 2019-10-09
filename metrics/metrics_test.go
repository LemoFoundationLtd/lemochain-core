package metrics

import (
	"strings"
	"testing"
)

func TestNewCounter(t *testing.T) {

	// // counter := NewCounter("co:")
	// // meter := NewMeter("me:")
	// timer := NewTimer("metrics")
	//
	// g := NewGauge("lemochain")
	// go func() {
	// 	// start := time.Now()
	// 	for {
	// 		start := time.Now()
	// 		time.Sleep(1 * time.Second)
	// 		// counter.Inc(1)
	// 		// meter.Mark(1)
	// 		timer.UpdateSince(start)
	// 		g.Update(1)
	// 	}
	// }()
	// // go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	// go WriteMetricsData(metrics.DefaultRegistry, 5*time.Second)
	// // cfg := falconmetrics.DefaultFalconConfig
	// // // cfg.Debug = true
	// // cfg.Step = 5
	// // falcon := falconmetrics.NewFalcon(&cfg)
	// // go falcon.ReportRegistry(metrics.DefaultRegistry)
	// select {}
}

func TestCollectProcessMetrics(t *testing.T) {
	str := "\n中和asc\n转\n换\n\n\n"
	sstr := strings.SplitAfter(str, "\n")
	for _, v := range sstr {
		if v != "\n" {
			t.Log(v)
		}
	}
}
