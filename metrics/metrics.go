// Package metrics provides general system and process level metrics collection.
package metrics

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"runtime"
	"time"
)

// MetricsEnabledFlag is the CLI flag name to use to enable metrics collections.
// var MetricsEnabledFlag = "metrics"

// Enabled is the flag specifying if metrics are enable or not.
var (
	Enabled  = false // 是否激活metrics,通过检测到配置文件是否配置了告警server的url来判断是否激活
	AlarmUrl string  // 告警系统server的url,通过配置文件传进来
)

// Init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.
func init() {
	// for _, arg := range os.Args {
	// 	if strings.TrimLeft(arg, "-") == MetricsEnabledFlag {
	// 		log.Info("Enabling metrics collection")
	// 		Enabled = true
	// 	}
	// }
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
		log.Info("Metrics is not start, cannot collect process metrics.")
		return
	} else {
		log.Infof("Start collect process Metrics. Refresh time: %f", refresh.Seconds())
	}

	// Create the various data collectors
	memstats := make([]*runtime.MemStats, 2)
	diskstats := make([]*DiskStats, 2)
	for i := 0; i < len(memstats); i++ {
		memstats[i] = new(runtime.MemStats)
		diskstats[i] = new(DiskStats)
	}
	// Define the various metrics to collect
	memAllocs := metrics.GetOrRegisterMeter(System_memory_allocs, metrics.DefaultRegistry)
	memFrees := metrics.GetOrRegisterMeter(System__memory_frees, metrics.DefaultRegistry)
	memInuse := metrics.GetOrRegisterMeter(System_memory_inuse, metrics.DefaultRegistry)
	memPauses := metrics.GetOrRegisterMeter(System_memory_pauses, metrics.DefaultRegistry)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes metrics.Meter
	if err := ReadDiskStats(diskstats[0]); err == nil {
		diskReads = metrics.GetOrRegisterMeter(System_disk_readCount, metrics.DefaultRegistry)
		diskReadBytes = metrics.GetOrRegisterMeter(System_disk_readData, metrics.DefaultRegistry)
		diskWrites = metrics.GetOrRegisterMeter(System_disk_writeCount, metrics.DefaultRegistry)
		diskWriteBytes = metrics.GetOrRegisterMeter(System_disk_writeData, metrics.DefaultRegistry)
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

// GetMapMetrics 返回所有注册是metrics方法
func GetMapMetrics() MapMetr {
	m := make(MapMetr)
	metrics.DefaultRegistry.Each(func(name string, i interface{}) {
		m[name] = i
	})
	return m
}

// 返回出给定name的metrics的[]string
func SprintMetrics(metricsName string, i interface{}) []string {
	du := float64(time.Second)
	duSuffix := time.Second.String()[1:]

	switch metric := i.(type) {
	case metrics.Gauge:
		str0 := fmt.Sprintf("\ngauge %s\n", metricsName)
		str1 := fmt.Sprintf("  value:		%9d\n", metric.Value())
		return ToStrings(str0, str1)
	case metrics.Counter:
		str0 := fmt.Sprintf("\ncounter %s\n", metricsName)
		str1 := fmt.Sprintf("  count:		%9d\n", metric.Count())
		return ToStrings(str0, str1)
	case metrics.Meter:
		m := metric.Snapshot()
		str0 := fmt.Sprintf("\nmeter %s\n", metricsName)
		str1 := fmt.Sprintf("  count:     %9d\n", m.Count())
		str2 := fmt.Sprintf("  1-min rate:  %12.2f\n", m.Rate1())
		str3 := fmt.Sprintf("  5-min rate:  %12.2f\n", m.Rate5())
		str4 := fmt.Sprintf("  15-min rate: %12.2f\n", m.Rate15())
		str5 := fmt.Sprintf("  mean rate:   %12.2f\n", m.RateMean())
		return ToStrings(str0, str1, str2, str3, str4, str5)
	case metrics.Timer:
		t := metric.Snapshot()
		ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		str0 := fmt.Sprintf("\ntimer %s\n", metricsName)
		str1 := fmt.Sprintf("  count:       %9d\n", t.Count())
		str2 := fmt.Sprintf("  min:         %12.2f%s\n", float64(t.Min())/du, duSuffix)
		str3 := fmt.Sprintf("  max:         %12.2f%s\n", float64(t.Max())/du, duSuffix)
		str4 := fmt.Sprintf("  mean:        %12.2f%s\n", t.Mean()/du, duSuffix)
		str5 := fmt.Sprintf("  stddev:      %12.2f%s\n", t.StdDev()/du, duSuffix)
		str6 := fmt.Sprintf("  median:      %12.2f%s\n", ps[0]/du, duSuffix)
		str7 := fmt.Sprintf("  75%%:         %12.2f%s\n", ps[1]/du, duSuffix)
		str8 := fmt.Sprintf("  95%%:         %12.2f%s\n", ps[2]/du, duSuffix)
		str9 := fmt.Sprintf("  99%%:         %12.2f%s\n", ps[3]/du, duSuffix)
		str10 := fmt.Sprintf("  99.9%%:       %12.2f%s\n", ps[4]/du, duSuffix)
		str11 := fmt.Sprintf("  1-min rate:  %12.2f\n", t.Rate1())
		str12 := fmt.Sprintf("  5-min rate:  %12.2f\n", t.Rate5())
		str13 := fmt.Sprintf("  15-min rate: %12.2f\n", t.Rate15())
		str14 := fmt.Sprintf("  mean rate:   %12.2f\n", t.RateMean())
		return ToStrings(str0, str1, str2, str3, str4, str5, str6, str7, str8, str9, str10, str11, str12, str13, str14)
	default:
		return nil
	}
	return nil
}

// ToStrings 把多个string拼接成一个[]string
func ToStrings(str ...string) []string {
	var ss = make([]string, 0)
	for _, s := range str {
		ss = append(ss, s)
	}
	return ss
}
