package alarm

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/alarm/push_server/dingding"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	gometrics "github.com/rcrowley/go-metrics"
	"strings"
)

/*
	告警系统
*/
var webHook = "https://oapi.dingtalk.com/robot/send?access_token=099784dbbdc904f1a8a7236d8efcc2facf53367ad5d82638c8bd43023308b97c" // 通过配置进去

// sendAlarmMsg 发送告警消息到ding ding
func sendAlarmMsg(alarmReason string, metricsName string, i interface{}) error {

	robot := dingding.NewRobot(webHook)
	sstr := metrics.SprintMetrics(metricsName, i)
	title := alarmReason
	markdownText := fmt.Sprintf("> #### %s\n> ", alarmReason) + strings.Join(sstr, "> ")
	err := robot.SendMarkdown(title, markdownText, nil, true)
	return err
}

// txpoolAlarm 对交易池报警
func txpoolAlarm() {
	m := metrics.GetModuleMetrics("txpool")
	if len(m) == 0 {
		return
	}

	metricsName01 := "txpool/totalTxNumber"
	metricsName02 := "txpool/RecvTx/receiveTx"
	metricsName03 := "txpool/DelInvalidTxs/invalid"

	for {
		// 1. 对交易池中剩下的交易数量大于10000进行报警
		if _, ok := m[metricsName01]; ok {
			txpoolTotalNumberGauge := m[metricsName01].(gometrics.Gauge).Snapshot()
			if txpoolTotalNumberGauge.Value() > int64(10000) {

				alarmReason := "交易池中的交易大于10000笔了"
				go func() {
					err := sendAlarmMsg(alarmReason, metricsName01, txpoolTotalNumberGauge)
					if err != nil {
						log.Errorf("Send alarm message failed. error: %s", err)
					}
				}()
				// todo 设置报警次数，因为一旦满足告警的条件之后，会很长时间都会满足此条件，导致会一直报警
			}
		}

		// 2. 对交易池中接收到的交易速率进行报警处理，当每1秒超过1000次调用则进行报警
		if _, ok := m[metricsName02]; ok {
			recvTxMeter := m[metricsName02].(gometrics.Meter).Snapshot()
			if recvTxMeter.Rate1() > float64(1000) { // 最近一分钟的平均速度

				alarmReason := "最近一分钟平均每秒调用接收交易进交易池的次数大于1000次了"
				go func() {
					err := sendAlarmMsg(alarmReason, metricsName02, recvTxMeter)
					if err != nil {
						log.Errorf("Send alarm message failed. error: %s", err)
					}
				}()

			}
		}

		// 3. 对交易池中对执行失败的交易累计总数大于5000笔进行报警
		if _, ok := m[metricsName03]; ok {
			invalidTxCounter := m[metricsName03].(gometrics.Counter)
			if invalidTxCounter.Count() > int64(5000) {

				alarmReason := "此节点执行失败的交易数量累计大于5000笔了"
				go func() {
					err := sendAlarmMsg(alarmReason, metricsName03, invalidTxCounter)
					if err != nil {
						log.Errorf("Send alarm message failed. error: %s", err)
					}
				}()

			}
		}
	}
}

// handleMsgAlarm 对处理网络模块的message进行报警
func handleMsgAlarm() {
	m := metrics.GetModuleMetrics("network")
	if len(m) == 0 {
		return
	}
	// 1. 对调用handleBlocksMsg的速率大于50次/秒进行报警
	handleBlocksMsgMeter := m["network/protocol_manager/handleBlocksMsg"].(gometrics.Meter).Snapshot()
	if handleBlocksMsgMeter.Rate1() > float64(50) {
		// todo 进行报警处理
	}
	// 2. 对调用handleGetBlocksMsg的速率大于100次/秒进行报警
	handleGetBlocksMsgMeter := m["network/protocol_manager/handleGetBlocksMsg"].(gometrics.Meter).Snapshot()
	if handleGetBlocksMsgMeter.Rate1() > float64(100) {
		// todo 进行报警处理
	}
	// 3. 对调用handleBlockHashMsg的速率大于5次/每秒进行报警
	handleBlockHashMsgMeter := m["network/protocol_manager/handleBlockHashMsg"].(gometrics.Meter).Snapshot()
	if handleBlockHashMsgMeter.Rate1() > float64(5) {
		// todo 进行报警处理
	}
	// 4. 对调用handleGetConfirmsMsg的速率大于50次/秒进行报警
	handleGetConfirmsMsgMeter := m["network/protocol_manager/handleGetConfirmsMsg"].(gometrics.Meter).Snapshot()
	if handleGetConfirmsMsgMeter.Rate1() > float64(50) {
		// todo 进行报警处理
	}
	// 5. 对调用handleConfirmMsg的速率大于10次/秒进行报警
	handleConfirmMsgMeter := m["network/protocol_manager/handleConfirmMsg"].(gometrics.Meter).Snapshot()
	if handleConfirmMsgMeter.Rate1() > float64(10) {
		// todo 进行报警处理
	}
	// 6. 对调用handleGetBlocksWithChangeLogMsg的速率大于50次/秒进行报警
	handleGetBlocksWithChangeLogMsgMeter := m["network/protocol_manager/handleGetBlocksWithChangeLogMsg"].(gometrics.Meter).Snapshot()
	if handleGetBlocksWithChangeLogMsgMeter.Rate1() > float64(50) {
		// todo 进行报警处理
	}
	// 7. 对调用handleDiscoverReqMsg的速率大于5次/秒进行报警
	handleDiscoverReqMsgMeter := m["network/protocol_manager/handleDiscoverReqMsg"].(gometrics.Meter).Snapshot()
	if handleDiscoverReqMsgMeter.Rate1() > float64(5) {
		// todo 进行报警处理
	}
	// 8. 对调用handleDiscoverResMsg的速率大于5次/秒进行报警
	handleDiscoverResMsgMeter := m["network/protocol_manager/handleDiscoverResMsg"].(gometrics.Meter).Snapshot()
	if handleDiscoverResMsgMeter.Rate1() > float64(5) {
		// todo 进行报警处理
	}
}

// p2pAlarm 对p2p模块的报警
func p2pAlarm() {
	m := metrics.GetModuleMetrics("p2p")
	if len(m) == 0 {
		return
	}
	// 1. 统计peer连接失败的频率,当每秒钟连接失败的频率大于5次，则报警
	handleConnFailedMeter := m["p2p/listenLoop/failedHandleConn"].(gometrics.Meter).Snapshot()
	if handleConnFailedMeter.Rate1() > float64(5) {
		// todo 进行报警处理
	}
	// 2. 统计成功读取一次msg所用时间的分布情况和调用读取msg的频率,
	readMsgSuccessTimer := m["p2p/readLoop/readMsgSuccess"].(gometrics.Timer).Snapshot()
	if readMsgSuccessTimer.Max() > int64(15) { // 读取msg所用最大时间大于15s则报警 todo 15s需要修改，综合规定的读取时间来设置
		// todo 进行报警处理
	}
	// 3. 统计读取失败msg的时间分布情况和频率
	readMsgFailedTimer := m["p2p/readLoop/readMsgFailed"].(gometrics.Timer).Snapshot()
	if readMsgFailedTimer.Rate1() > float64(20) { // 最近一分钟中每秒调用次数超过20次则报警
		// todo 进行报警处理
	}
	// 4. 统计写入操作成功的时间分布和频率
	writeMsgSuccessTimer := m["p2p/WriteMsg/writeMsgSuccess"].(gometrics.Timer).Snapshot()
	if writeMsgSuccessTimer.Max() > int64(15) { // 如果写入操作所用时间超过15s则需要报警
		// todo 进行报警处理
	}
	// 5. 统计写入操作失败的时间分布和频率
	writeMsgFailedTimer := m["p2p/WriteMsg/writeMsgFailed"].(gometrics.Timer).Snapshot()
	if writeMsgFailedTimer.Rate1() > float64(5) {
		// todo 进行报警处理
	}
}

// 交易验证失败报警
func verifyTxAlarm() {
	m := metrics.GetModuleMetrics("types")
	if len(m) == 0 {
		return
	}
	verifyFailedTxMeter := m["types/VerifyTx/verifyFailed"].(gometrics.Meter).Snapshot()
	if verifyFailedTxMeter.Rate1() > float64(5) { // 最近一分钟中每秒有5次交易验证失败的情况则报警
		// todo 进行报警处理
	}
}

// 共识模块中mineBlock和insertChain
func consensusAlarm() {
	m := metrics.GetModuleMetrics("consensus")
	if len(m) == 0 {
		return
	}
	// 1. inertChain时间分布和频率
	blockInsertTimer := m["consensus/InsertBlock/insertBlock"].(gometrics.Timer).Snapshot()
	if blockInsertTimer.Max() > int64(3) { // insertChain所用时间大于3秒则报警
		// todo 进行报警处理
	}
	// 2. miner
	mineBlockTimer := m["consensus/MineBlock/mineBlock"].(gometrics.Timer).Snapshot()
	if mineBlockTimer.Max() > 10 { // todo
		// todo 进行报警处理
	}
}

// 对leveldb的状态进行报警
func leveldbAlarm() {
	m := metrics.GetModuleMetrics(store.Prefix)
	if len(m) == 0 {
		return
	}

}
