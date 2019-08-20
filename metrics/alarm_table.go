package metrics

import (
	"github.com/rcrowley/go-metrics"
	"time"
)

// 表示用于告警系统比较的metrics度量类型
type metricsType uint8

const (
	TypeCount metricsType = iota // 0
	TypeRate1
	TypeRate5
	TypeRate15
	TypeRateMean
	TypeMean
	TypeMax
	TypeMin
	TypeStdDev
	TypeSum
	TypeVariance
	TypeValue
)

// 缓存注册的metrics方法
type MetricsMap map[string]interface{}

// GetMapMetrics 返回所有注册是metrics方法
func GetMapMetrics() MetricsMap {
	m := make(MetricsMap)
	metrics.DefaultRegistry.Each(func(name string, i interface{}) {
		m[name] = i
	})
	return m
}

// 验证触发告警条件
type Condition struct {
	AlarmReason  string      // 告警的理由
	MetricsType  metricsType // 需要告警的度量类型
	AlarmValue   float64     // 触发告警的临界度量值
	TimeStamp    time.Time   // 用于记录上次告警时间
	AlarmMsgCode uint32      // 发送告警消息类型,目前只支持text类型
}

// 告警规则表
var AlarmRuleTable = map[string]*Condition{
	// txpool
	InvalidTx_meterName: {
		AlarmReason:  "平均两秒有1笔交易执行失败了",
		MetricsType:  TypeRate1,
		AlarmValue:   0.5,
		AlarmMsgCode: textMsgCode,
	},
	TxpoolNumber_counterName: {
		AlarmReason:  "交易池中的交易大于10000笔了",
		MetricsType:  TypeCount,
		AlarmValue:   10000,
		AlarmMsgCode: textMsgCode,
	},
	// network
	HandleBlocksMsg_meterName: {
		AlarmReason:  "调用handleBlocksMsg的速率大于50次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   50,
		AlarmMsgCode: textMsgCode,
	},
	HandleGetBlocksMsg_meterName: {
		AlarmReason:  "调用handleGetBlocksMsg的速率大于100次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   100,
		AlarmMsgCode: textMsgCode,
	},
	HandleBlockHashMsg_meterName: {
		AlarmReason:  "调用handleBlockHashMsg的速率大于5次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	HandleGetConfirmsMsg_meterName: {
		AlarmReason:  "调用handleGetConfirmsMsg的速率大于50次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   50,
		AlarmMsgCode: textMsgCode,
	},
	HandleConfirmMsg_meterName: {
		AlarmReason:  "调用handleConfirmMsg的速率大于10次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   10,
		AlarmMsgCode: textMsgCode,
	},
	HandleGetBlocksWithChangeLogMsg_meterName: {
		AlarmReason:  "调用handleGetBlocksWithChangeLogMsg的速率大于50次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   50,
		AlarmMsgCode: textMsgCode,
	},
	HandleDiscoverReqMsg_meterName: {
		AlarmReason:  "调用handleDiscoverReqMsg的速率大于5次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	HandleDiscoverResMsg_meterName: {
		AlarmReason:  "调用handleDiscoverReqMsg的速率大于5次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	// p2p
	PeerConnFailed_meterName: {
		AlarmReason:  "远程peer连接失败的频率大于5次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	ReadMsgSuccess_timerName: {
		AlarmReason:  "读取接收到的message所用的平均时间大于20s",
		MetricsType:  TypeMean,
		AlarmValue:   20,
		AlarmMsgCode: textMsgCode,
	},
	ReadMsgFailed_timerName: {
		AlarmReason:  "读取接收到的message失败的频率大于5次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	WriteMsgSuccess_timerName: {
		AlarmReason:  "写操作的平均用时超过15s",
		MetricsType:  TypeMean,
		AlarmValue:   15,
		AlarmMsgCode: textMsgCode,
	},
	WriteMsgFailed_timerName: {
		AlarmReason:  "写操作失败的频率超过了5次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	// tx
	VerifyFailedTx_meterName: {
		AlarmReason:  "验证交易失败的频率超过了0.5次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   0.5,
		AlarmMsgCode: textMsgCode,
	},
	// consensus
	BlockInsert_timerName: {
		AlarmReason:  "Insert chain 所用平均时间大于5s",
		MetricsType:  TypeMean,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	MineBlock_timerName: {
		AlarmReason:  "Mine Block 所用平均时间大于8s",
		MetricsType:  TypeMean,
		AlarmValue:   8,
		AlarmMsgCode: textMsgCode,
	},
	VerifyBlock_meterName: {
		AlarmReason:  "VerifyAndSeal 校验收到的block失败的频率大于5s一次",
		MetricsType:  TypeRate15,
		AlarmValue:   0.2,
		AlarmMsgCode: textMsgCode,
	},
	// levelDB
	LevelDb_miss_meterName: {
		AlarmReason:  "从leveldb中读取数据失败的频率大于10次/s",
		MetricsType:  TypeRate1,
		AlarmValue:   10,
		AlarmMsgCode: textMsgCode,
	},
	// system
	// System_memory_allocs: {
	// 	AlarmReason: "申请内存次数超过了释放内存次数的1.5倍",
	// 	MetricsType: "TypeCount",
	// 	AlarmValue:  float64(MetricsMap[System__memory_frees].(metrics.Meter).Snapshot().TypeCount()*3/2),
	// 	AlarmMsgCode: textMsgCode,
	// },
}
