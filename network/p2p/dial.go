package p2p

import (
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"net"
	"sync/atomic"
	"time"
)

const (
	dialTimeout = 3 * time.Second
)

type HandleConnFunc func(fd net.Conn, isSelfServer bool) error

type DialManager struct {
	handleConn HandleConnFunc
	discover   *DiscoverManager
	state      int32
}

func NewDialManager(handleConn HandleConnFunc, discover *DiscoverManager) *DialManager {
	return &DialManager{
		handleConn: handleConn,
		discover:   discover,
		state:      0,
	}
}

func (m *DialManager) Start() error {
	if atomic.LoadInt32(&m.state) == 1 {
		log.Info("dial manager has started")
		return ErrHasStared
	}
	atomic.StoreInt32(&m.state, 1)
	go m.loop()
	log.Infof("dial manager start")
	return nil
}

func (m *DialManager) Stop() error {
	if atomic.LoadInt32(&m.state) < 1 {
		log.Info("dial manager not start")
		return ErrNotStart
	}
	atomic.StoreInt32(&m.state, -1)
	log.Infof("dial manager stop")
	return nil
}

func (m *DialManager) runDialTask(node string) int {
	conn, err := net.DialTimeout("tcp", node, dialTimeout)
	if err != nil {
		m.discover.SetConnectResult(node, false)
		log.Warnf("dial node error: %s", err.Error())
		return -1
	}
	if err = m.handleConn(conn, false); err != nil {
		log.Warnf("node first connect error: %s", err.Error())
		return -2
	}
	return 0
}

func (m *DialManager) loop() {
	for {
		list := m.discover.connectingNodes()
		for _, n := range list {
			log.Debugf("dial:%s", n)
			if atomic.LoadInt32(&m.state) == -1 {
				return
			}
			m.runDialTask(n)
		}
		if atomic.LoadInt32(&m.state) == -1 {
			return
		}
		time.Sleep(3 * time.Second)
	}
}
