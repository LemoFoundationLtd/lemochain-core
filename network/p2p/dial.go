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

func (m *DialManager) Start() {
	if atomic.LoadInt32(&m.state) == 1 {
		log.Info("dial manager has started")
		return
	}
	atomic.StoreInt32(&m.state, 1)
	go m.loop()
}

func (m *DialManager) Stop() {
	if atomic.LoadInt32(&m.state) < 1 {
		log.Info("dial manager not start")
		return
	}
	atomic.StoreInt32(&m.state, -1)
}

func (m *DialManager) runDialTask(node string) int {
	conn, err := net.DialTimeout("tcp", node, dialTimeout)
	if err != nil {
		m.discover.SetConnectResult(node, false)
		log.Warnf("node first dial error: %s", err.Error())
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
