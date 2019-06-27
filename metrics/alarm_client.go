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
	dialTimeOut       = 3 * time.Second  // 拨号超时时间
	frameWriteTimeout = 20 * time.Second // 写socket超时时间
	heartbeatInterval = 5 * time.Second  // 发送心跳包间隔时间

	// 报警监听间隔时间
	alarmTxpoolInterval       = 3 * time.Second
	alarmHandleMsgInterval    = 4 * time.Second
	alarmP2pInterval          = 5 * time.Second
	alarmTxInterval           = 6 * time.Second
	alarmConsensusInterval    = 5 * time.Second
	alarmLeveldbStateInterval = 5 * time.Second
	alarmSystemStateInterval  = 5 * time.Second

	CodeHeartbeat = uint32(0x01) // 心跳包msg code
	textMsgCode   = uint32(0x02) // 普通文本msg code
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
	// 如果metrics未开启，则不能启动alarm system
	if !Enabled {
		log.Info("The metrics not open. So alarm system start failed.")
		return
	}

	for {
		m.dialAlarmServer(alarmServerIP) // 阻塞
		// 如果与server端断开，休眠5s再连接
		time.Sleep(5 * time.Second)
		log.Debug("Restart alarm system")
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
	log.Info("Dial alarm system server success")
	c.run() // 阻塞
}

type client struct {
	Conn   net.Conn
	stopCh chan struct{}
	wg     sync.WaitGroup
	sync.RWMutex
}

func (c *client) run() {
	c.wg.Add(8)
	log.Info("Start run alarm system")
	go c.heartbeatLoop()
	go c.txpoolAlarm(alarmTxpoolInterval)
	go c.handleMsgAlarm(alarmHandleMsgInterval)
	go c.p2pAlarm(alarmP2pInterval)
	go c.verifyTxAlarm(alarmTxInterval)
	go c.consensusAlarm(alarmConsensusInterval)
	go c.leveldbStateAlarm(alarmLeveldbStateInterval)
	go c.systemStateAlarm(alarmSystemStateInterval)
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
			// // for test
			// time.Sleep(1 * time.Second)
			// textMsg := fmt.Sprintf("你好,\nlemochain alarm system.")
			// err = c.WriteMsg(textMsgCode, []byte(textMsg))
			// if err != nil {
			// 	log.Errorf("write msg error. err: %v", err)
			// }
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
func (c *client) txpoolAlarm(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("Txpool alarm end.")
	}()

	var (
		metricsName01 = TxpoolNumber_gaugeName
		metricsName02 = RecvTx_meterName
		metricsName03 = InvalidTx_counterName

		count int64 = 2000       // 交易执行失败的累计交易数量
		incr  int64 = 1000       // 增量
		now01       = time.Now() // 限制交易池交易数量告警的时间间隔
		now02       = time.Now() // 限制调用接收交易的交易池函数的速率的时间间隔
	)

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			m := GetModuleMetrics(txpoolModule)
			if len(m) == 0 {
				break
			}
			// 1. 对交易池中剩下的交易数量大于10000进行报警
			if _, ok := m[metricsName01]; ok {
				txpoolTotalNumberGauge := m[metricsName01].(gometrics.Gauge).Snapshot()

				if txpoolTotalNumberGauge.Value() > int64(10000) && time.Since(now01).Seconds() > 10 { // 告警条件,并满足距离上次告警时间间隔必须大于10s

					alarmReason := "AlarmReason: 交易池中的交易大于10000笔了\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName01, txpoolTotalNumberGauge), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now01 = time.Now()
				}
			}

			// 2. 对交易池中接收到的交易速率进行报警处理，当每1秒超过50次调用则进行报警
			if _, ok := m[metricsName02]; ok {
				recvTxMeter := m[metricsName02].(gometrics.Meter).Snapshot()
				if recvTxMeter.Rate1() > float64(50) && time.Since(now02).Seconds() > 60 { // 最近一分钟的平均速度,并满足距离上次告警时间间隔为60s

					alarmReason := "AlarmReason: 最近一分钟平均每秒调用接收交易进交易池的次数大于50次了\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName02, recvTxMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now02 = time.Now()
				}
			}

			// 3. 对交易池中对执行失败的交易累计总数大于2000笔进行报警
			if _, ok := m[metricsName03]; ok {
				invalidTxCounter := m[metricsName03].(gometrics.Counter)
				if invalidTxCounter.Count() > count { // count为动态调整参数，每报警一次则增加一定的增量值

					alarmReason := fmt.Sprintf("AlarmReason: 此节点执行失败的交易数量累计大于%d笔了", count)
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName03, invalidTxCounter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					// 修改告警参数值
					count = count + incr
				}
			}
		}
	}
}

// handleMsgAlarm 对处理网络模块的message进行报警
func (c *client) handleMsgAlarm(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("HandleMsg alarm end.")
	}()

	var (
		metricsName01 = HandleBlocksMsg_meterName
		metricsName02 = HandleGetBlocksMsg_meterName
		metricsName03 = HandleBlockHashMsg_meterName
		metricsName04 = HandleGetConfirmsMsg_meterName
		metricsName05 = HandleConfirmMsg_meterName
		metricsName06 = HandleGetBlocksWithChangeLogMsg_meterName
		metricsName07 = HandleDiscoverReqMsg_meterName
		metricsName08 = HandleDiscoverResMsg_meterName
		now01         = time.Now()
		now02         = time.Now()
		now03         = time.Now()
		now04         = time.Now()
		now05         = time.Now()
		now06         = time.Now()
		now07         = time.Now()
		now08         = time.Now()
	)

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			m := GetModuleMetrics(networkModule)
			if len(m) == 0 {
				break
			}

			// 1. 对调用handleBlocksMsg的速率大于50次/秒进行报警
			if _, ok := m[metricsName01]; ok {
				handleBlocksMsgMeter := m[metricsName01].(gometrics.Meter).Snapshot()
				if handleBlocksMsgMeter.Rate1() > float64(50) && time.Since(now01).Seconds() > 60 { // 调用速率要大于50并且距离上次告警时间间隔必须大于60s

					alarmReason := "AlarmReason: 调用handleBlocksMsg的速率大于50次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName01, handleBlocksMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now01 = time.Now()
				}
			}

			// 2. 对调用handleGetBlocksMsg的速率大于100次/秒进行报警
			if _, ok := m[metricsName02]; ok {
				handleGetBlocksMsgMeter := m[metricsName02].(gometrics.Meter).Snapshot()
				if handleGetBlocksMsgMeter.Rate1() > float64(100) && time.Since(now02).Seconds() > 60 {

					alarmReason := "AlarmReason: 调用handleGetBlocksMsg的速率大于100次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName02, handleGetBlocksMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now02 = time.Now()
				}
			}

			// 3. 对调用handleBlockHashMsg的速率大于5次/每秒进行报警
			if _, ok := m[metricsName03]; ok {
				handleBlockHashMsgMeter := m[metricsName03].(gometrics.Meter).Snapshot()
				if handleBlockHashMsgMeter.Rate1() > float64(5) && time.Since(now03).Seconds() > 60 {

					alarmReason := "AlarmReason: 调用handleBlockHashMsg的速率大于5次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName03, handleBlockHashMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now03 = time.Now()
				}
			}

			// 4. 对调用handleGetConfirmsMsg的速率大于50次/秒进行报警
			if _, ok := m[metricsName04]; ok {
				handleGetConfirmsMsgMeter := m[metricsName04].(gometrics.Meter).Snapshot()
				if handleGetConfirmsMsgMeter.Rate1() > float64(50) && time.Since(now04).Seconds() > 60 {

					alarmReason := "AlarmReason: 调用handleGetConfirmsMsg的速率大于50次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName04, handleGetConfirmsMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now04 = time.Now()
				}
			}

			// 5. 对调用handleConfirmMsg的速率大于10次/秒进行报警
			if _, ok := m[metricsName05]; ok {
				handleConfirmMsgMeter := m[metricsName05].(gometrics.Meter).Snapshot()
				if handleConfirmMsgMeter.Rate1() > float64(10) && time.Since(now05).Seconds() > 60 {

					alarmReason := "AlarmReason: 调用handleConfirmMsg的速率大于10次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName05, handleConfirmMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now05 = time.Now()
				}
			}

			// 6. 对调用handleGetBlocksWithChangeLogMsg的速率大于50次/秒进行报警
			if _, ok := m[metricsName06]; ok {
				handleGetBlocksWithChangeLogMsgMeter := m[metricsName06].(gometrics.Meter).Snapshot()
				if handleGetBlocksWithChangeLogMsgMeter.Rate1() > float64(50) && time.Since(now06).Seconds() > 60 {

					alarmReason := "AlarmReason: 调用handleGetBlocksWithChangeLogMsg的速率大于50次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName06, handleGetBlocksWithChangeLogMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now06 = time.Now()
				}
			}

			// 7. 对调用handleDiscoverReqMsg的速率大于5次/秒进行报警
			if _, ok := m[metricsName07]; ok {
				handleDiscoverReqMsgMeter := m[metricsName07].(gometrics.Meter).Snapshot()
				if handleDiscoverReqMsgMeter.Rate1() > float64(5) && time.Since(now07).Seconds() > 60 {

					alarmReason := "AlarmReason: 调用handleDiscoverReqMsg的速率大于5次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName07, handleDiscoverReqMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now07 = time.Now()
				}
			}

			// 8. 对调用handleDiscoverResMsg的速率大于5次/秒进行报警
			if _, ok := m[metricsName08]; ok {
				handleDiscoverResMsgMeter := m[metricsName08].(gometrics.Meter).Snapshot()
				if handleDiscoverResMsgMeter.Rate1() > float64(5) && time.Since(now08).Seconds() > 60 {

					alarmReason := "AlarmReason: 调用handleDiscoverReqMsg的速率大于5次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName08, handleDiscoverResMsgMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now08 = time.Now()
				}
			}
		}
	}
}

// p2pAlarm 对p2p模块的报警
func (c *client) p2pAlarm(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("P2p alarm end.")
	}()

	var (
		metricsName01 = PeerConnFailed_meterName
		metricsName02 = ReadMsgSuccess_timerName
		metricsName03 = ReadMsgFailed_timerName
		metricsName04 = WriteMsgSuccess_timerName
		metricsName05 = WriteMsgFailed_timerName

		now01 = time.Now()
		now03 = time.Now()
		now05 = time.Now()
	)
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			m := GetModuleMetrics(p2pModule)
			if len(m) == 0 {
				break
			}

			// 1. 统计peer连接失败的频率,当每秒钟连接失败的频率大于5次，则报警
			if _, ok := m[metricsName01]; ok {
				handleConnFailedMeter := m[metricsName01].(gometrics.Meter).Snapshot()
				if handleConnFailedMeter.Rate1() > float64(5) && time.Since(now01).Seconds() > 30 {

					alarmReason := "AlarmReason: 远程peer连接失败的频率大于5次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName01, handleConnFailedMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now01 = time.Now()
				}
			}

			// 2. 统计成功读取一次msg所用时间的分布情况和调用读取msg的频率
			if _, ok := m[metricsName02]; ok {
				readMsgSuccessTimer := m[metricsName02].(gometrics.Timer).Snapshot()
				if readMsgSuccessTimer.Mean()/float64(time.Second) > float64(20) { // 读取msg所用平均时间大于20s则报警

					alarmReason := "AlarmReason: 读取接收到的message所用的平均时间大于20s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName02, readMsgSuccessTimer), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}

			// 3. 统计读取失败msg的时间分布和频率情况
			if _, ok := m[metricsName03]; ok {
				readMsgFailedTimer := m[metricsName03].(gometrics.Timer).Snapshot()
				if readMsgFailedTimer.Rate1() > float64(5) && time.Since(now03).Seconds() > 30 {

					alarmReason := "AlarmReason: 读取接收到的message失败的频率大于5次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName03, readMsgFailedTimer), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now03 = time.Now()
				}
			}

			// 4. 统计写入操作成功的时间分布和频率
			if _, ok := m[metricsName04]; ok {
				writeMsgSuccessTimer := m[metricsName04].(gometrics.Timer).Snapshot()
				if writeMsgSuccessTimer.Mean()/float64(time.Second) > float64(15) { // 如果写入操作所用平均时间超过15s则需要报警

					alarmReason := "AlarmReason: 写操作的平均用时超过15s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName04, writeMsgSuccessTimer), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}

			// 5. 统计写入操作失败的时间分布和频率
			if _, ok := m[metricsName05]; ok {
				writeMsgFailedTimer := m[metricsName05].(gometrics.Timer).Snapshot()
				if writeMsgFailedTimer.Rate1() > float64(5) && time.Since(now05).Seconds() > 60 {

					alarmReason := "AlarmReason: 写操作失败的频率超过了5次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName05, writeMsgFailedTimer), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now05 = time.Now()
				}
			}
		}
	}
}

// 交易验证失败报警
func (c *client) verifyTxAlarm(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("VerifyTx alarm end.")
	}()

	var (
		metricsName01 = VerifyFailedTx_meterName
	)
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			m := GetModuleMetrics(txModule)
			if len(m) == 0 {
				break
			}

			if _, ok := m[metricsName01]; ok {
				verifyFailedTxMeter := m[metricsName01].(gometrics.Meter).Snapshot()
				if verifyFailedTxMeter.Rate1() > float64(0.5) { // 最近一分钟中每秒有0.5次交易验证失败的情况则报警

					alarmReason := "AlarmReason: 验证交易失败的频率超过了0.5次/s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName01, verifyFailedTxMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}
		}
	}
}

// 共识模块中mineBlock和insertChain
func (c *client) consensusAlarm(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("Consensus alarm end.")
	}()

	var (
		metricsName01 = BlockInsert_timerName
		metricsName02 = MineBlock_timerName
	)

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			m := GetModuleMetrics(consensusModule)
			if len(m) == 0 {
				break
			}

			// 1. inertChain时间分布和频率
			if _, ok := m[metricsName01]; ok {
				blockInsertTimer := m[metricsName01].(gometrics.Timer).Snapshot()
				if blockInsertTimer.Mean()/float64(time.Second) > float64(0.01) { // insertChain所用平均时间大于3秒则报警

					alarmReason := "AlarmReason: Insert chain 所用平均时间大于3s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName01, blockInsertTimer), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}

			// 2. miner
			if _, ok := m[metricsName02]; ok {
				mineBlockTimer := m[metricsName02].(gometrics.Timer).Snapshot() // 挖块所用平均时间大于8s则报警
				if mineBlockTimer.Mean()/float64(time.Second) > 8 {

					alarmReason := "AlarmReason: Mine Block 所用平均时间大于3s\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName02, mineBlockTimer), "")
					alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
				}
			}
		}
	}
}

// 对leveldb的状态进行报警
func (c *client) leveldbStateAlarm(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("Leveldb state alarm end.")
	}()

	var (
		metricsName01       = LevelDb_miss_meterName
		temp          int64 = 0
	)
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			m := GetModuleMetrics(leveldbModule)
			if len(m) == 0 {
				break
			}

			// 1. 对leveldb进行get操作失败进行告警
			if _, ok := m[metricsName01]; ok {
				missDBMeter := m[metricsName01].(gometrics.Meter).Snapshot()
				if missDBMeter.Count() > temp { // 此次快照的count大于上一次，表示有get db失败，则告警

					alarmReason := "AlarmReason: 从leveldb中读取数据失败\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName01, missDBMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					temp = missDBMeter.Count() // 更新temp
				}
			}
			// todo 暂时还没有leveldb的其他参数告警需求
		}
	}
}

// 对system状态进行告警
func (c *client) systemStateAlarm(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("System state alarm end.")
	}()

	var (
		metricsName01 = System__memory_frees
		now           = time.Now()
	)

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			m := GetModuleMetrics(systemModule)
			if len(m) == 0 {
				break
			}

			// 1. 对系统内存不足进行告警
			if _, ok := m[metricsName01]; ok {
				freesMemMeter := m[metricsName01].(gometrics.Meter).Snapshot()
				if freesMemMeter.Count() < 100*1024*1024 && time.Since(now).Seconds() > 60 { // 内存小于100M并且距离上次报警时间间隔1分钟则报警

					alarmReason := "AlarmReason: 系统内存小于100M\n"
					metricsDetails := "Detail: " + strings.Join(SprintMetrics(metricsName01, freesMemMeter), "")
					alarmTime := fmt.Sprintf("AlarmTime:\n %s\n", time.Now().UTC().String())
					content := alarmReason + metricsDetails + alarmTime

					go c.sendMsgToAlarmServer(textMsgCode, []byte(content))
					now = time.Now()
				}
			}
		}
	}
}
