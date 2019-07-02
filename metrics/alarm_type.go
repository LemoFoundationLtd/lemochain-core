package metrics

import "github.com/rcrowley/go-metrics"

// 缓存注册的metrics方法
type MapMetr map[string]interface{}

// GetMapMetrics 返回所有注册是metrics方法
func GetMapMetrics() MapMetr {
	m := make(MapMetr)
	metrics.DefaultRegistry.Each(func(name string, i interface{}) {
		m[name] = i
	})
	return m
}

// 验证触发告警条件
type VerifyCondition struct {
	AlarmReason string
	MetricsType string
	AlarmValue  float64
}

// 缓存告警的触发规则
type AlarmRuleMap map[string]VerifyCondition

func GetAlarmRuleMap() AlarmRuleMap {

}
