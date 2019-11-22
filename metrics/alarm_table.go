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
		AlarmReason:  "最近的一分钟时间内有大于30笔交易执行失败了",
		MetricsType:  TypeRate1,
		AlarmValue:   0.5,
		AlarmMsgCode: textMsgCode,
	},
	TxpoolNumber_counterName: {
		AlarmReason:  "交易池中的交易大于5000笔了",
		MetricsType:  TypeCount,
		AlarmValue:   5000,
		AlarmMsgCode: textMsgCode,
	},
	// network
	HandleBlocksMsg_meterName: {
		AlarmReason:  "最近一分钟时间内收到其他节点广播过来的blocks消息次数大于20次",
		MetricsType:  TypeRate1,
		AlarmValue:   0.33,
		AlarmMsgCode: textMsgCode,
	},
	HandleGetBlocksMsg_meterName: {
		AlarmReason:  "最近一分钟时间内收到其他节点请求拉取指定高度的block消息次数大于100次",
		MetricsType:  TypeRate1,
		AlarmValue:   1.66,
		AlarmMsgCode: textMsgCode,
	},
	HandleBlockHashMsg_meterName: {
		AlarmReason:  "最近一分钟时间内普通节点收到广播的稳定块hash的次数大于2000次", // 普通节点收到广播的稳定块hash，按照连接100个peer和一分钟出20个块来计算
		MetricsType:  TypeRate1,
		AlarmValue:   33.33,
		AlarmMsgCode: textMsgCode,
	},
	HandleGetConfirmsMsg_meterName: {
		AlarmReason:  "最近一分钟时间内收到其他节点请求拉取block确认包消息次数大于1600次", // 极端情况另外16个deputy都来拉,每个节点请求100次，则最多请求1600次
		MetricsType:  TypeRate1,
		AlarmValue:   26.6,
		AlarmMsgCode: textMsgCode,
	},
	HandleConfirmMsg_meterName: {
		AlarmReason:  "最近一分钟时间内收到其他节点广播过来的区块确认包的次数大于320", // 按照一分钟最多出块20个，16个deputyNode peers计算
		MetricsType:  TypeRate1,
		AlarmValue:   5.33,
		AlarmMsgCode: textMsgCode,
	},
	HandleGetBlocksWithChangeLogMsg_meterName: {
		AlarmReason:  "最近一分钟时间内收到调用handleGetBlocksWithChangeLogMsg请求的次数大于100次", // lemochain-distribution端同步区块调用的接口，按照连接5个distribution节点计算
		MetricsType:  TypeRate1,
		AlarmValue:   1.66,
		AlarmMsgCode: textMsgCode,
	},
	HandleDiscoverReqMsg_meterName: {
		AlarmReason:  "最近一分钟时间内收到调用handleDiscoverReqMsg的次数大于600次", // 节点发现请求为10s发送一次，按照连接100个peer来算一分钟最多收到600次
		MetricsType:  TypeRate1,
		AlarmValue:   10,
		AlarmMsgCode: textMsgCode,
	},
	HandleDiscoverResMsg_meterName: {
		AlarmReason:  "最近一分钟时间内收到调用handleDiscoverReqMsg的次数大于600次", // 同上
		MetricsType:  TypeRate1,
		AlarmValue:   10,
		AlarmMsgCode: textMsgCode,
	},
	// p2p
	PeerConnFailed_meterName: {
		AlarmReason:  "最近一分钟时间内节点连接断开的次数大于5次",
		MetricsType:  TypeRate1,
		AlarmValue:   0.083,
		AlarmMsgCode: textMsgCode,
	},
	ReadMsgSuccess_timerName: {
		AlarmReason:  "读取接收节点的Msg所用的平均时间大于6s，有必要升级网络带宽", // 节点之间的心跳包是每隔5s发送一次，最多允许1s的传输时间。
		MetricsType:  TypeMean,
		AlarmValue:   6,
		AlarmMsgCode: textMsgCode,
	},
	ReadMsgFailed_timerName: {
		AlarmReason:  "最近一分钟时间内读取接收节点的Msg失败的次数大于5次", // 读取节点消息失败会断开连接，允许一分钟内重连失败次数为5
		MetricsType:  TypeRate1,
		AlarmValue:   0.083,
		AlarmMsgCode: textMsgCode,
	},
	WriteMsgSuccess_timerName: {
		AlarmReason:  "发送Msg给其他节点的平均用时超过5s，有必要升级网络带宽",
		MetricsType:  TypeMean,
		AlarmValue:   5,
		AlarmMsgCode: textMsgCode,
	},
	WriteMsgFailed_timerName: {
		AlarmReason:  "最近一分钟时间内发送Msg给其他节点失败的次数超过5次",
		MetricsType:  TypeRate1,
		AlarmValue:   0.083,
		AlarmMsgCode: textMsgCode,
	},
	// tx
	VerifyFailedTx_meterName: {
		AlarmReason:  "最近一分钟时间内交易验证失败的的次数超过了30笔",
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
		AlarmReason:  "Mine Block 所用平均时间大于15s",
		MetricsType:  TypeMean,
		AlarmValue:   15,
		AlarmMsgCode: textMsgCode,
	},
	VerifyBlock_meterName: {
		AlarmReason:  "最近一分钟时间内收到2个以上InsertBlock校验不通过的block",
		MetricsType:  TypeRate1,
		AlarmValue:   0.033,
		AlarmMsgCode: textMsgCode,
	},
	// levelDB
	LevelDb_miss_meterName: {
		AlarmReason:  "最近一分钟时间内从leveldb中读取数据失败次数大于10次",
		MetricsType:  TypeRate1,
		AlarmValue:   0.16,
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
