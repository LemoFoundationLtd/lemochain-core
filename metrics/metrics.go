// Package metrics provides general system and process level metrics collection.
package metrics

import (
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"os"
	"runtime"
	"strings"
	"time"
)

// MetricsEnabledFlag is the CLI flag name to use to enable metrics collections.
var MetricsEnabledFlag = "metrics"

// Enabled is the flag specifying if metrics are enable or not.
var Enabled = false

// Init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.
func init() {
	for _, arg := range os.Args {
		if strings.TrimLeft(arg, "-") == MetricsEnabledFlag {
			log.Info("Enabling metrics collection")
			Enabled = true
		}
	}
	Enabled = true
	exp.Exp(metrics.DefaultRegistry)
}

func NewGauge(name string) metrics.Gauge {
	if !Enabled {
		return new(metrics.NilGauge)
	}
	return metrics.GetOrRegisterGauge(name, metrics.DefaultRegistry)
}

// NewCounter create a new metrics Counter, either a real one of a NOP stub depending
// on the metrics flag.
func NewCounter(name string) metrics.Counter {
	if !Enabled {
		return new(metrics.NilCounter)
	}
	return metrics.GetOrRegisterCounter(name, metrics.DefaultRegistry)
}

// NewMeter create a new metrics Meter, either a real one of a NOP stub depending
// on the metrics flag.
func NewMeter(name string) metrics.Meter {
	if !Enabled {
		return new(metrics.NilMeter)
	}
	return metrics.GetOrRegisterMeter(name, metrics.DefaultRegistry)
}

// NewTimer create a new metrics Timer, either a real one of a NOP stub depending
// on the metrics flag.
func NewTimer(name string) metrics.Timer {
	if !Enabled {
		return new(metrics.NilTimer)
	}
	return metrics.GetOrRegisterTimer(name, metrics.DefaultRegistry)
}

// CollectProcessMetrics periodically collects various metrics about the running
// process.
func CollectProcessMetrics(refresh time.Duration) {
	// Short circuit if the metrics system is disabled
	if !Enabled {
		return
	}
	// Create the various data collectors
	memstats := make([]*runtime.MemStats, 2)
	diskstats := make([]*DiskStats, 2)
	for i := 0; i < len(memstats); i++ {
		memstats[i] = new(runtime.MemStats)
		diskstats[i] = new(DiskStats)
	}
	// Define the various metrics to collect
	memAllocs := metrics.GetOrRegisterMeter("system/memory/allocs", metrics.DefaultRegistry)
	memFrees := metrics.GetOrRegisterMeter("system/memory/frees", metrics.DefaultRegistry)
	memInuse := metrics.GetOrRegisterMeter("system/memory/inuse", metrics.DefaultRegistry)
	memPauses := metrics.GetOrRegisterMeter("system/memory/pauses", metrics.DefaultRegistry)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes metrics.Meter
	if err := ReadDiskStats(diskstats[0]); err == nil {
		diskReads = metrics.GetOrRegisterMeter("system/disk/readcount", metrics.DefaultRegistry)
		diskReadBytes = metrics.GetOrRegisterMeter("system/disk/readdata", metrics.DefaultRegistry)
		diskWrites = metrics.GetOrRegisterMeter("system/disk/writecount", metrics.DefaultRegistry)
		diskWriteBytes = metrics.GetOrRegisterMeter("system/disk/writedata", metrics.DefaultRegistry)
	} else {
		log.Debug("Failed to read disk metrics", "err", err)
	}
	// Iterate loading the different stats and updating the meters
	for i := 1; ; i++ {
		runtime.ReadMemStats(memstats[i%2])
		memAllocs.Mark(int64(memstats[i%2].Mallocs - memstats[(i-1)%2].Mallocs))
		memFrees.Mark(int64(memstats[i%2].Frees - memstats[(i-1)%2].Frees))
		memInuse.Mark(int64(memstats[i%2].Alloc - memstats[(i-1)%2].Alloc))
		memPauses.Mark(int64(memstats[i%2].PauseTotalNs - memstats[(i-1)%2].PauseTotalNs))

		if ReadDiskStats(diskstats[i%2]) == nil {
			diskReads.Mark(int64(diskstats[i%2].ReadCount - diskstats[(i-1)%2].ReadCount))
			diskReadBytes.Mark(int64(diskstats[i%2].ReadBytes - diskstats[(i-1)%2].ReadBytes))
			diskWrites.Mark(int64(diskstats[i%2].WriteCount - diskstats[(i-1)%2].WriteCount))
			diskWriteBytes.Mark(int64(diskstats[i%2].WriteBytes - diskstats[(i-1)%2].WriteBytes))
		}
		time.Sleep(refresh)
	}
}

func PointMetricsLog() {
	// metrics.Log(metrics.DefaultRegistry, 10*time.Second, logger.New(os.Stderr, "metrics: ", logger.Lmicroseconds))
	WriteMetricsData(metrics.DefaultRegistry, 5*time.Second)
}

// WriteMetricsData 收集统计数据
func WriteMetricsData(r metrics.Registry, refresh time.Duration) {
	du := float64(time.Nanosecond)
	duSuffix := time.Nanosecond.String()[1:]

	for range time.Tick(refresh) {
		r.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case metrics.Gauge:
				log.Infof("gauge %s\n", name)
				log.Infof("  value:       %9d\n", metric.Value())
			case metrics.Counter:
				log.Infof("counter %s\n", name)
				log.Infof("  count:       %9d\n", metric.Count())
			case metrics.Meter:
				m := metric.Snapshot()
				log.Infof("meter %s\n", name)
				log.Infof("  count:       %9d\n", m.Count())
				log.Infof("  1-min rate:  %12.2f\n", m.Rate1())
				log.Infof("  5-min rate:  %12.2f\n", m.Rate5())
				log.Infof("  15-min rate: %12.2f\n", m.Rate15())
				log.Infof("  mean rate:   %12.2f\n", m.RateMean())
			case metrics.Timer:
				t := metric.Snapshot()
				ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				log.Infof("timer %s\n", name)
				log.Infof("  count:       %9d\n", t.Count())
				log.Infof("  min:         %12.2f%s\n", float64(t.Min())/du, duSuffix)
				log.Infof("  max:         %12.2f%s\n", float64(t.Max())/du, duSuffix)
				log.Infof("  mean:        %12.2f%s\n", t.Mean()/du, duSuffix)
				log.Infof("  stddev:      %12.2f%s\n", t.StdDev()/du, duSuffix)
				log.Infof("  median:      %12.2f%s\n", ps[0]/du, duSuffix)
				log.Infof("  75%%:         %12.2f%s\n", ps[1]/du, duSuffix)
				log.Infof("  95%%:         %12.2f%s\n", ps[2]/du, duSuffix)
				log.Infof("  99%%:         %12.2f%s\n", ps[3]/du, duSuffix)
				log.Infof("  99.9%%:       %12.2f%s\n", ps[4]/du, duSuffix)
				log.Infof("  1-min rate:  %12.2f\n", t.Rate1())
				log.Infof("  5-min rate:  %12.2f\n", t.Rate5())
				log.Infof("  15-min rate: %12.2f\n", t.Rate15())
				log.Infof("  mean rate:   %12.2f\n", t.RateMean())
			}
		})
	}
}

// GetModuleMetrics 返回指定模块的的metrics
func GetModuleMetrics(moduleName string) map[string]interface{} {
	m := make(map[string]interface{})
	metrics.DefaultRegistry.Each(func(name string, i interface{}) {
		if strings.HasPrefix(name, moduleName) {
			m[name] = i
		}
	})
	return m
}
