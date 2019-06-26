package metrics

import (
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	gometrics "github.com/rcrowley/go-metrics"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	dialTimeOut       = 3 * time.Second
	frameWriteTimeout = 20 * time.Second
	heartbeatInterval = 5 * time.Second
	alarmLoopInterval = 5 * time.Second
	CodeHeartbeat     = uint32(0x01) // 心跳包msg code
	textMsgCode       = uint32(0x02) // 普通文本msg code
)

var (
	alarmServerIP = "127.0.0.1:8088"
	PackagePrefix = []byte{0x77, 0x88} // package flag
	PackageLength = 4                  // msg长度所占字节的个数
)

type alarmManager struct {
}

func NewAlarmManager() *alarmManager {
	return &alarmManager{}
}

func (m *alarmManager) Start() {
	for {
		m.dialAlarmServer(alarmServerIP)
		time.Sleep(5 * time.Second)
	}
}

// dialAlarmServer
func (m *alarmManager) dialAlarmServer(ip string) {
	conn, err := net.DialTimeout("tcp", ip, dialTimeOut)
	if err != nil {
		log.Errorf("Dial alarm system server error. err: %v", err)
		return
	}
	c := &client{
		Conn:   conn,
		stopCh: make(chan struct{}),
		wg:     sync.WaitGroup{},
	}
	c.run() // 阻塞
}

type client struct {
	Conn   net.Conn
	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func (c *client) run() {
	c.wg.Add(7)
	log.Info("start run alarm system")
	go c.heartbeatLoop()
	go c.txpoolAlarm()
	go c.handleMsgAlarm()
	go c.p2pAlarm()
	go c.verifyTxAlarm()
	go c.consensusAlarm()
	go c.leveldbAlarm()
	c.wg.Wait()
}

func (c *client) Close() {
	c.Lock()
	defer c.Unlock()
	select {
	case <-c.stopCh:
		return
	default:

	}
	close(c.stopCh)
	log.Debug("Close client")
}

func (c *client) heartbeatLoop() {
	heartbeatTimer := time.NewTicker(heartbeatInterval)
	defer func() {
		c.wg.Done()
		heartbeatTimer.Stop()
		log.Debug("Heartbeat loop end.")
	}()

	for {
		select {
		case <-c.stopCh:
			return
		case <-heartbeatTimer.C:
			err := c.WriteMsg(CodeHeartbeat, nil)
			if err != nil {
				c.Close()
				return
			}
			log.Info("send a heartbeat msg to alarm server")
			// for test
			time.Sleep(1 * time.Second)
			textMsg := fmt.Sprintf("你好,\nlemochain alarm system.")
			err = c.WriteMsg(textMsgCode, []byte(textMsg))
			if err != nil {
				log.Errorf("write msg error. err: %v", err)
			}
		}
	}
}

// 通过socket发送消息到server端
func (c *client) WriteMsg(msgCode uint32, content []byte) error {
	c.Lock()
	defer c.Unlock()
	pack := packFrame(msgCode, content)
	c.Conn.SetWriteDeadline(time.Now().Add(frameWriteTimeout))
	_, err := c.Conn.Write(pack)
	return err
}

// packFrame
func packFrame(code uint32, msg []byte) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, code) // uint32转为[]byte
	if msg != nil {
		buf = append(buf, msg...)
	}
	length := make([]byte, PackageLength)
	binary.BigEndian.PutUint32(length, uint32(len(buf)))
	pack := make([]byte, 0)
	pack = append(append(PackagePrefix, length...), buf...)
	return pack
}

func (c *client) sendMsgToAlarmServer(msgCode uint32, content []byte) {
	select {
	case <-c.stopCh:
		return
	default:
	}

	// 给3次的冗余
	count := 3
	for i := 0; ; i++ {
		err := c.WriteMsg(msgCode, []byte(content))
		if err != nil {
			if i < count {
				continue
			} else {
				log.Debugf("write msg to alarm server error. err: %v", err)
				c.Close()
				return
			}
		} else {
			return
		}
	}
}

// txpoolAlarm 对交易池报警
func (c *client) txpoolAlarm() {
	ticker := time.NewTicker(alarmLoopInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("Txpool alarm end.")
	}()

	m := GetModuleMetrics("txpool")
	if len(m) == 0 {
		return
	}

	metricsName01 := "txpool/totalTxNumber"
	metricsName02 := "txpool/RecvTx/receiveTx"
	metricsName03 := "txpool/DelInvalidTxs/invalid"

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			// 1. 对交易池中剩下的交易数量大于10000进行报警
			if _, ok := m[metricsName01]; ok {
				txpoolTotalNumberGauge := m[metricsName01].(gometrics.Gauge).Snapshot()
				if txpoolTotalNumberGauge.Value() > int64(10000) {

					alarmReason := "交易池中的交易大于10000笔了\n"
					details := strings.Join(SprintMetrics(metricsName01, txpoolTotalNumberGauge), "")
					content := alarmReason + details

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}

			// 2. 对交易池中接收到的交易速率进行报警处理，当每1秒超过1000次调用则进行报警
			if _, ok := m[metricsName02]; ok {
				recvTxMeter := m[metricsName02].(gometrics.Meter).Snapshot()
				if recvTxMeter.Rate1() > float64(1000) { // 最近一分钟的平均速度

					alarmReason := "最近一分钟平均每秒调用接收交易进交易池的次数大于1000次了\n"
					details := strings.Join(SprintMetrics(metricsName02, recvTxMeter), "")
					content := alarmReason + details

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}

			// 3. 对交易池中对执行失败的交易累计总数大于5000笔进行报警
			if _, ok := m[metricsName03]; ok {
				invalidTxCounter := m[metricsName03].(gometrics.Counter)
				if invalidTxCounter.Count() > int64(5000) {

					alarmReason := "此节点执行失败的交易数量累计大于5000笔了"
					details := strings.Join(SprintMetrics(metricsName03, invalidTxCounter), "")
					content := alarmReason + details

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}

		}
	}
}

// handleMsgAlarm 对处理网络模块的message进行报警
func (c *client) handleMsgAlarm() {
	ticker := time.NewTicker(alarmLoopInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("HandleMsg alarm end.")
	}()

	m := GetModuleMetrics("network")
	if len(m) == 0 {
		return
	}
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
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
	}

}

// p2pAlarm 对p2p模块的报警
func (c *client) p2pAlarm() {
	ticker := time.NewTicker(alarmLoopInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("P2p alarm end.")
	}()

	m := GetModuleMetrics("p2p")
	if len(m) == 0 {
		return
	}

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
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
	}

}

// 交易验证失败报警
func (c *client) verifyTxAlarm() {
	ticker := time.NewTicker(alarmLoopInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("VerifyTx alarm end.")
	}()

	m := GetModuleMetrics("types")
	if len(m) == 0 {
		return
	}
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			verifyFailedTxMeter := m["types/VerifyTx/verifyFailed"].(gometrics.Meter).Snapshot()
			if verifyFailedTxMeter.Rate1() > float64(5) { // 最近一分钟中每秒有5次交易验证失败的情况则报警
				// todo 进行报警处理
			}
		}
	}
}

// 共识模块中mineBlock和insertChain
func (c *client) consensusAlarm() {
	ticker := time.NewTicker(alarmLoopInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("Consensus alarm end.")
	}()

	m := GetModuleMetrics("consensus")
	if len(m) == 0 {
		return
	}

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
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
	}
}

// 对leveldb的状态进行报警
func (c *client) leveldbAlarm() {
	ticker := time.NewTicker(alarmLoopInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("leveldb alarm end.")
	}()

	m := GetModuleMetrics("glemo/db/chaindata/")
	if len(m) == 0 {
		return
	}
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			// todo
		}
	}

}
