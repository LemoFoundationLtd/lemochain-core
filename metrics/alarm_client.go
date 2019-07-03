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
	alarmInterval     = 5 * time.Second  // 报警监听间隔时间

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
	c.wg.Add(2)
	log.Info("Start run alarm system")
	go c.heartbeatLoop()
	go c.alarmLoop(alarmInterval)
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

// alarmLoop 告警
func (c *client) alarmLoop(alarmTimeInterval time.Duration) {
	ticker := time.NewTicker(alarmTimeInterval)
	defer func() {
		c.wg.Done()
		ticker.Stop()
		log.Debug("Alarm loop close")
	}()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			mm := GetMapMetrics() // 获取所有的注册metrics方法
			if len(mm) == 0 {
				break
			}
			for name, condition := range AlarmRuleTable {
				if time.Since(condition.TimeStamp).Seconds() > 60 {
					if c.listenAndAlarm(mm, name, condition) {
						condition.TimeStamp = time.Now()
					}
				}
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

// listenAndAlarm
func (c *client) listenAndAlarm(m map[string]interface{}, metricsName string, condition *Condition) bool {
	var (
		enabled        = false
		metricsDetails string
	)

	if i, ok := m[metricsName]; ok {
		switch metr := i.(type) {
		case gometrics.Gauge:
			gauge := metr.Snapshot()
			// float64转int64
			if gauge.Value() > int64(condition.AlarmValue) {
				// 满足告警条件
				enabled = true
				metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, gauge), "")
			}

		case gometrics.Counter:
			counter := metr.Snapshot()

			if counter.Count() > int64(condition.AlarmValue) {
				// 满足告警条件
				enabled = true
				metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, counter), "")
			}

		case gometrics.Meter:
			meter := metr.Snapshot()

			if condition.MetricsType == TypeCount {
				if meter.Count() > int64(condition.AlarmValue) {
					// 满足告警条件
					enabled = true
				}
			} else if condition.MetricsType == TypeRate1 {
				if meter.Rate1() > condition.AlarmValue {
					// 满足告警条件
					enabled = true
				}
			} else {
				log.Errorf("This type of measurement is not currently supported. error metricsType: %s", condition.MetricsType)
				return false
			}

			metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, meter), "")

		case gometrics.Timer:
			timer := metr.Snapshot()
			if condition.MetricsType == TypeMean {
				if timer.Mean()/float64(time.Second) > condition.AlarmValue {
					// 满足告警条件
					enabled = true
				}
			} else if condition.MetricsType == TypeCount {
				if timer.Count() > int64(condition.AlarmValue) {
					// 满足告警条件
					enabled = true
				}
			} else {
				log.Errorf("This type of measurement is not currently supported. error metricsType: %s", condition.MetricsType)
				return false
			}
			metricsDetails = "Detail: " + strings.Join(SprintMetrics(metricsName, timer), "")

		default:
			log.Errorf("Metrics Type error. error type: %T", metr)
			return false
		}
	}

	// 发送告警消息到告警server
	if enabled {
		alarmReason := fmt.Sprintf("AlarmReason: %s\n", condition.AlarmReason)
		alarmTime := fmt.Sprintf("AlarmTime: \n%s\n", time.Now().Format("2006/01/02 15:04:05"))
		content := alarmReason + metricsDetails + alarmTime

		go c.sendMsgToAlarmServer(condition.AlarmMsgCode, []byte(content))
	}
	return enabled
}

// // 对system状态进行告警 todo
// func (c *client) systemStateAlarm(alarmTimeInterval time.Duration) {
// 	ticker := time.NewTicker(alarmTimeInterval)
// 	defer func() {
// 		c.wg.Done()
// 		ticker.Stop()
// 		log.Debug("System state alarm end.")
// 	}()
//
// 	var (
// 		now = time.Now()
// 	)
//
// 	for {
// 		select {
// 		case <-c.stopCh:
// 			return
// 		case <-ticker.C:
// 			m := GetMapMetrics(systemModule)
// 			if len(m) == 0 {
// 				break
// 			}
//
// 			// 1. 申请内存次数和释放内存次数比较
// 			if _, ok := m[System__memory_frees]; ok {
// 				freesMemMeter := m[System__memory_frees].(gometrics.Meter).Snapshot()
// 				if time.Since(now).Seconds() > 60 {
// 					if c.listenAndAlarm(m, System_memory_allocs, "申请内存次数超过了释放内存次数的1.5倍", freesMemMeter.TypeCount()*3/2, textMsgCode) {
// 						now = time.Now()
// 					}
// 				}
// 			}
// 		}
// 	}
// }
