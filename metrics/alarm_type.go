package metrics

// 缓存注册的metrics方法
type MapMetr map[string]interface{}

// 验证触发告警条件
type VerifyCondition struct {
	AlarmReason    string
	AlarmCondition float64
}
