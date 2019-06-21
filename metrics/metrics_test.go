package metrics

import (
	"github.com/rcrowley/go-metrics"
	"log"
	"testing"
	"time"
)

func TestNewCounter(t *testing.T) {

	// counter := NewCounter("co:")
	// meter := NewMeter("me:")
	timer := NewTimer("metrics")

	g := NewGauge("lemochain")
	go func() {
		// start := time.Now()
		for {
			start := time.Now()
			time.Sleep(1 * time.Second)
			// counter.Inc(1)
			// meter.Mark(1)
			timer.UpdateSince(start)
			g.Update(1)
		}
	}()
	// go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	go printMetrics(metrics.DefaultRegistry, 5*time.Second)
	// cfg := falconmetrics.DefaultFalconConfig
	// // cfg.Debug = true
	// cfg.Step = 5
	// falcon := falconmetrics.NewFalcon(&cfg)
	// go falcon.ReportRegistry(metrics.DefaultRegistry)
	select {}
}

func printMetrics(r metrics.Registry, refresh time.Duration) {
	du := float64(time.Nanosecond)
	duSuffix := time.Nanosecond.String()[1:]

	for range time.Tick(refresh) {
		r.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case metrics.Counter:
				log.Printf("counter %s\n", name)
				log.Printf("  count:       %9d\n", metric.Count())
			case metrics.Gauge:
				log.Printf("gauge %s\n", name)
				log.Printf("  value:       %9d\n", metric.Value())
			case metrics.GaugeFloat64:
				log.Printf("gauge %s\n", name)
				log.Printf("  value:       %f\n", metric.Value())
			case metrics.Healthcheck:
				metric.Check()
				log.Printf("healthcheck %s\n", name)
				log.Printf("  error:       %v\n", metric.Error())
			case metrics.Histogram:
				h := metric.Snapshot()
				ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				log.Printf("histogram %s\n", name)
				log.Printf("  count:       %9d\n", h.Count())
				log.Printf("  min:         %9d\n", h.Min())
				log.Printf("  max:         %9d\n", h.Max())
				log.Printf("  mean:        %12.2f\n", h.Mean())
				log.Printf("  stddev:      %12.2f\n", h.StdDev())
				log.Printf("  median:      %12.2f\n", ps[0])
				log.Printf("  75%%:         %12.2f\n", ps[1])
				log.Printf("  95%%:         %12.2f\n", ps[2])
				log.Printf("  99%%:         %12.2f\n", ps[3])
				log.Printf("  99.9%%:       %12.2f\n", ps[4])
			case metrics.Meter:
				m := metric.Snapshot()
				log.Printf("meter %s\n", name)
				log.Printf("  count:       %9d\n", m.Count())
				log.Printf("  1-min rate:  %12.2f\n", m.Rate1())
				log.Printf("  5-min rate:  %12.2f\n", m.Rate5())
				log.Printf("  15-min rate: %12.2f\n", m.Rate15())
				log.Printf("  mean rate:   %12.2f\n", m.RateMean())
			case metrics.Timer:
				t := metric.Snapshot()
				ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				log.Printf("timer %s\n", name)
				log.Printf("  count:       %9d\n", t.Count())
				log.Printf("  min:         %12.2f%s\n", float64(t.Min())/du, duSuffix)
				log.Printf("  max:         %12.2f%s\n", float64(t.Max())/du, duSuffix)
				log.Printf("  mean:        %12.2f%s\n", t.Mean()/du, duSuffix)
				log.Printf("  stddev:      %12.2f%s\n", t.StdDev()/du, duSuffix)
				log.Printf("  median:      %12.2f%s\n", ps[0]/du, duSuffix)
				log.Printf("  75%%:         %12.2f%s\n", ps[1]/du, duSuffix)
				log.Printf("  95%%:         %12.2f%s\n", ps[2]/du, duSuffix)
				log.Printf("  99%%:         %12.2f%s\n", ps[3]/du, duSuffix)
				log.Printf("  99.9%%:       %12.2f%s\n", ps[4]/du, duSuffix)
				log.Printf("  1-min rate:  %12.2f\n", t.Rate1())
				log.Printf("  5-min rate:  %12.2f\n", t.Rate5())
				log.Printf("  15-min rate: %12.2f\n", t.Rate15())
				log.Printf("  mean rate:   %12.2f\n", t.RateMean())
			}
		})
	}
}
