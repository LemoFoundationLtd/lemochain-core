package metrics

import (
	"github.com/rcrowley/go-metrics"
	"time"
)

const (
	CountFunc = "Count()"
	Rate1Func = "Rate1()"
	MeanFunc  = "Mean()"
)

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
type Condition struct {
	AlarmReason  string
	MetricsType  string
	AlarmValue   float64
	TimeStamp    time.Time
	AlarmMsgCode uint32
}

// 告警规则表
var AlarmRuleTable = map[string]*Condition{
	// txpool
	InvalidTx_meterName: {
		AlarmReason:  "平均两秒有1笔交易执行失败了",
		MetricsType:  Rate1Func,
		AlarmValue:   0.5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	TxpoolNumber_counterName: {
		AlarmReason:  "交易池中的交易大于10000笔了",
		MetricsType:  CountFunc,
		AlarmValue:   10000,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	// network
	HandleBlocksMsg_meterName: {
		AlarmReason:  "调用handleBlocksMsg的速率大于50次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   50,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	HandleGetBlocksMsg_meterName: {
		AlarmReason:  "调用handleGetBlocksMsg的速率大于100次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   100,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	HandleBlockHashMsg_meterName: {
		AlarmReason:  "调用handleBlockHashMsg的速率大于5次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	HandleGetConfirmsMsg_meterName: {
		AlarmReason:  "调用handleGetConfirmsMsg的速率大于50次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   50,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	HandleConfirmMsg_meterName: {
		AlarmReason:  "调用handleConfirmMsg的速率大于10次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   10,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	HandleGetBlocksWithChangeLogMsg_meterName: {
		AlarmReason:  "调用handleGetBlocksWithChangeLogMsg的速率大于50次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   50,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	HandleDiscoverReqMsg_meterName: {
		AlarmReason:  "调用handleDiscoverReqMsg的速率大于5次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	HandleDiscoverResMsg_meterName: {
		AlarmReason:  "调用handleDiscoverReqMsg的速率大于5次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	// p2p
	PeerConnFailed_meterName: {
		AlarmReason:  "远程peer连接失败的频率大于5次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	ReadMsgSuccess_timerName: {
		AlarmReason:  "读取接收到的message所用的平均时间大于20s",
		MetricsType:  MeanFunc,
		AlarmValue:   20,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	ReadMsgFailed_timerName: {
		AlarmReason:  "读取接收到的message失败的频率大于5次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	WriteMsgSuccess_timerName: {
		AlarmReason:  "写操作的平均用时超过15s",
		MetricsType:  MeanFunc,
		AlarmValue:   15,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	WriteMsgFailed_timerName: {
		AlarmReason:  "写操作失败的频率超过了5次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	// tx
	VerifyFailedTx_meterName: {
		AlarmReason:  "验证交易失败的频率超过了0.5次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   0.5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	// consensus
	BlockInsert_timerName: {
		AlarmReason:  "Insert chain 所用平均时间大于5s",
		MetricsType:  MeanFunc,
		AlarmValue:   5,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	MineBlock_timerName: {
		AlarmReason:  "Mine Block 所用平均时间大于8s",
		MetricsType:  MeanFunc,
		AlarmValue:   8,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	// levelDB
	LevelDb_miss_meterName: {
		AlarmReason:  "从leveldb中读取数据失败的频率大于2次/s",
		MetricsType:  Rate1Func,
		AlarmValue:   2,
		TimeStamp:    time.Now(),
		AlarmMsgCode: textMsgCode,
	},
	// system
	// System_memory_allocs: {
	// 	AlarmReason: "申请内存次数超过了释放内存次数的1.5倍",
	// 	MetricsType: "Count()",
	// 	AlarmValue:  float64(MapMetr[System__memory_frees].(metrics.Meter).Snapshot().Count()*3/2),
	// 	TimeStamp:   time.Now(),
	// 	AlarmMsgCode: textMsgCode,
	// },
}
