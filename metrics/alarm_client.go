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
		m.dialAlarmServer(AlarmUrl) // 阻塞
		// 如果与server端断开，休眠5s再连接
		time.Sleep(5 * time.Second)
		log.Debug("Restart alarm system")
	}
}

// dialAlarmServer
func (m *alarmManager) dialAlarmServer(alarmUrl string) {
	log.Debugf("Start dial alarm server. alarmUrl: %s", alarmUrl)
	conn, err := net.DialTimeout("tcp", alarmUrl, dialTimeOut)
	if err != nil {
		log.Errorf("Dial alarm system server error. Please check alarm url configuration correct. err: %v. alarmUrl: %s.", err, alarmUrl)
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

// heartbeatLoop
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

// ListenAndAlarm 参数说明 m: 注册的metrics方法; metricsName: 注册的metrics方法名; alarmReason: 告警理由; alarmCondition: 告警触发条件; alarmMsgCode: 发送告警消息的类型
func (c *client) ListenAndAlarm(m map[string]interface{}, metricsName string, alarmReason string, alarmCondition interface{}, alarmMsgCode uint32) bool {
	var (
		enabled        = false
		metricsDetails string
	)

	if i, ok := m[metricsName]; ok {
		switch metr := i.(type) {
		case gometrics.Gauge:
			gauge := metr.Snapshot()
			if value, ok := alarmCondition.(int64); ok {
				if gauge.Value() > value {
					// 满足告警条件
					enabled = true
					metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, gauge), "")
				}
			}

		case gometrics.Counter:
			counter := metr.Snapshot()
			if value, ok := alarmCondition.(int64); ok {
				if counter.Count() > value {
					// 满足告警条件
					enabled = true
					metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, counter), "")
				}
			}
		case gometrics.Meter:
			meter := metr.Snapshot()
			if value, ok := alarmCondition.(int64); ok {
				if meter.Count() > value {
					// 满足告警条件
					enabled = true
				}
			}
			if value, ok := alarmCondition.(float64); ok {
				if meter.Rate1() > value {
					// 满足告警条件
					enabled = true
				}
			}
			metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, meter), "")

		case gometrics.Timer:
			timer := metr.Snapshot()
			if value, ok := alarmCondition.(float64); ok {
				if timer.Mean()/float64(time.Second) > value {
					// 满足告警条件
					enabled = true

				}
			}
			if value, ok := alarmCondition.(int64); ok {
				if timer.Count() > value {
					// 满足告警条件
					enabled = true
				}
			}
			metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, timer), "")

		default:
		}
	}

	// 发送告警消息到告警server
	if enabled {
		alarmReason := fmt.Sprintf("AlarmReason: %s\n", alarmReason)
		alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().UTC().String())
		content := alarmReason + metricsDetails + alarmTime

		go c.sendMsgToAlarmServer(alarmMsgCode, []byte(content))
	}
	return enabled
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
		count int64 = 100        // 交易执行失败的累计交易数量
		incr  int64 = 100        // 增量
		now01       = time.Now() // 限制交易池交易数量告警的时间间隔
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
			if time.Since(now01).Seconds() > 30 {
				if c.ListenAndAlarm(m, TxpoolNumber_counterName, "交易池中的交易大于10000笔了", int64(10000), textMsgCode) {
					now01 = time.Now()
				}
			}

			// 2. 对交易池中对执行失败的交易每增加100笔报警一次
			if c.ListenAndAlarm(m, InvalidTx_counterName, fmt.Sprintf("此节点执行失败的交易数量累计大于%d笔了", count), count, textMsgCode) {
				count = count + incr
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
		now01 = time.Now()
		now02 = time.Now()
		now03 = time.Now()
		now04 = time.Now()
		now05 = time.Now()
		now06 = time.Now()
		now07 = time.Now()
		now08 = time.Now()
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
			if time.Since(now01).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleBlocksMsg_meterName, "调用handleBlocksMsg的速率大于50次/s", float64(50), textMsgCode) {
					now01 = time.Now()
				}
			}

			// 2. 对调用handleGetBlocksMsg的速率大于100次/秒进行报警
			if time.Since(now02).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleGetBlocksMsg_meterName, "调用handleGetBlocksMsg的速率大于100次/s", float64(100), textMsgCode) {
					now02 = time.Now()
				}
			}

			// 3. 对调用handleBlockHashMsg的速率大于5次/每秒进行报警
			if time.Since(now03).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleBlockHashMsg_meterName, "调用handleBlockHashMsg的速率大于5次/s", float64(5), textMsgCode) {
					now03 = time.Now()
				}
			}

			// 4. 对调用handleGetConfirmsMsg的速率大于50次/秒进行报警
			if time.Since(now04).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleGetConfirmsMsg_meterName, "调用handleGetConfirmsMsg的速率大于50次/s", float64(50), textMsgCode) {
					now04 = time.Now()
				}
			}

			// 5. 对调用handleConfirmMsg的速率大于10次/秒进行报警
			if time.Since(now05).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleConfirmMsg_meterName, "调用handleConfirmMsg的速率大于10次/s", float64(10), textMsgCode) {
					now05 = time.Now()
				}
			}

			// 6. 对调用handleGetBlocksWithChangeLogMsg的速率大于50次/秒进行报警
			if time.Since(now06).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleGetBlocksWithChangeLogMsg_meterName, "调用handleGetBlocksWithChangeLogMsg的速率大于50次/s", float64(50), textMsgCode) {
					now06 = time.Now()
				}
			}

			// 7. 对调用handleDiscoverReqMsg的速率大于5次/秒进行报警
			if time.Since(now07).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleDiscoverReqMsg_meterName, "调用handleDiscoverReqMsg的速率大于5次/s", float64(5), textMsgCode) {
					now07 = time.Now()
				}
			}

			// 8. 对调用handleDiscoverResMsg的速率大于5次/秒进行报警
			if time.Since(now08).Seconds() > 60 {
				if c.ListenAndAlarm(m, HandleDiscoverResMsg_meterName, "调用handleDiscoverReqMsg的速率大于5次/s", float64(5), textMsgCode) {
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
		now01 = time.Now()
		now02 = time.Now()
		now03 = time.Now()
		now04 = time.Now()
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
			if time.Since(now01).Seconds() > 30 {
				if c.ListenAndAlarm(m, PeerConnFailed_meterName, "远程peer连接失败的频率大于5次/s", float64(5), textMsgCode) {
					now01 = time.Now()
				}
			}

			// 2. 统计成功读取一次msg所用时间的分布情况和调用读取msg的频率
			if time.Since(now02).Seconds() > 60 {
				if c.ListenAndAlarm(m, ReadMsgSuccess_timerName, "读取接收到的message所用的平均时间大于20s", float64(20), textMsgCode) {
					now02 = time.Now()
				}
			}

			// 3. 统计读取失败msg的时间分布和频率情况
			if time.Since(now03).Seconds() > 30 {
				if c.ListenAndAlarm(m, ReadMsgFailed_timerName, "读取接收到的message失败的频率大于5次/s", float64(5), textMsgCode) {
					now03 = time.Now()
				}
			}

			// 4. 统计写入操作成功的时间分布和频率
			if time.Since(now04).Seconds() > 60 {
				if c.ListenAndAlarm(m, WriteMsgSuccess_timerName, "写操作的平均用时超过15s", float64(15), textMsgCode) {
					now04 = time.Now()
				}
			}

			// 5. 统计写入操作失败的时间分布和频率
			if time.Since(now05).Seconds() > 60 {
				if c.ListenAndAlarm(m, WriteMsgFailed_timerName, "写操作失败的频率超过了5次/s", float64(5), textMsgCode) {
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
		now01 = time.Now()
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
			// 1. 验证交易失败的频率
			if time.Since(now01).Seconds() > 60 {
				if c.ListenAndAlarm(m, VerifyFailedTx_meterName, "验证交易失败的频率超过了0.5次/s", float64(0.5), textMsgCode) {
					now01 = time.Now()
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
		now01 = time.Now()
		now02 = time.Now()
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
			if time.Since(now01).Seconds() > 60 {
				if c.ListenAndAlarm(m, BlockInsert_timerName, "Insert chain 所用平均时间大于5s", float64(5), textMsgCode) {
					now01 = time.Now()
				}
			}

			// 2. miner
			if time.Since(now02).Seconds() > 60 {
				if c.ListenAndAlarm(m, MineBlock_timerName, "Mine Block 所用平均时间大于8s", float64(8), textMsgCode) {
					now02 = time.Now()
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
		temp int64 = 0
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
			if c.ListenAndAlarm(m, LevelDb_miss_meterName, "从leveldb中读取数据失败", temp, textMsgCode) {
				missDBMeter := m[LevelDb_miss_meterName].(gometrics.Meter).Snapshot()
				temp = missDBMeter.Count() // 更新temp
			}
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
		now = time.Now()
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

			// 1. 申请内存次数和释放内存次数比较
			if _, ok := m[System__memory_frees]; ok {
				freesMemMeter := m[System__memory_frees].(gometrics.Meter).Snapshot()
				if time.Since(now).Seconds() > 60 {
					if c.ListenAndAlarm(m, System_memory_allocs, "申请内存次数超过了释放内存次数的1.5倍", freesMemMeter.Count()*3/2, textMsgCode) {
						now = time.Now()
					}
				}
			}
		}
	}
}
