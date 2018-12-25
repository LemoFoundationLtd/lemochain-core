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

type HandleConnFunc func(fd net.Conn, nodeID *NodeID) error

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

// Start
func (m *DialManager) Start() error {
	if atomic.LoadInt32(&m.state) == 1 {
		log.Info("dial manager has already started")
		return ErrHasStared
	}
	atomic.StoreInt32(&m.state, 1)
	go m.loop()
	log.Infof("dial manager start")
	return nil
}

// Stop
func (m *DialManager) Stop() error {
	if atomic.LoadInt32(&m.state) < 1 {
		log.Info("dial manager not start")
		return ErrNotStart
	}
	atomic.StoreInt32(&m.state, -1)
	log.Infof("dial manager stop")
	return nil
}

// runDialTask run dial task
func (m *DialManager) runDialTask(node string) int {
	// check
	nodeID, endpoint := checkNodeString(node)
	if nodeID == nil {
		log.Warnf("dial: invalid node. node: %s", node)
		return -3
	}
	// dial
	conn, err := net.DialTimeout("tcp", endpoint, dialTimeout)
	if err != nil {
		m.discover.SetConnectResult(nodeID, false)
		log.Warnf("dial node error: %s", err.Error())
		return -1
	}
	// handle connection
	if err = m.handleConn(conn, nodeID); err != nil {
		log.Warnf("node first connect error: %s", err.Error())
		return -2
	}
	return 0
}

// loop
func (m *DialManager) loop() {
	for {
		list := m.discover.connectingNodes()
		for _, n := range list {
			log.Debugf("start dial: %s", n)
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
