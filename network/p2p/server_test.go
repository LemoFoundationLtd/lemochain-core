package p2p

import (
	"crypto/ecdsa"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/common/subscribe"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

type testPeer struct {
	conn         net.Conn
	localNodeID  NodeID
	errHandshake bool
}

func newTestPeer(conn net.Conn) IPeer {
	return &testPeer{conn: conn, errHandshake: false}
}

func newTestPeerErrHandshake(conn net.Conn) IPeer {
	return &testPeer{conn: conn, errHandshake: true}
}

func (p *testPeer) ReadMsg() (msg *Msg, err error) {
	return nil, nil
}
func (p *testPeer) WriteMsg(code uint32, msg []byte) (err error) {
	return nil
}
func (p *testPeer) RNodeID() *NodeID {
	return &NodeID{0x01, 0x02, 0x03, 0x04, 0x05}
}
func (p *testPeer) RAddress() string {
	return p.conn.RemoteAddr().String()
}
func (p *testPeer) LAddress() string {
	return p.conn.LocalAddr().String()
}
func (p *testPeer) doHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) error {
	if p.errHandshake {
		return errors.New("halt ha ha")
	}
	return nil
}
func (p *testPeer) run() (err error) {
	time.Sleep(5 * time.Second)
	return nil
}
func (p *testPeer) NeedReConnect() bool {
	return true
}
func (p *testPeer) SetStatus(status int32) {
	return
}
func (p *testPeer) Close() {

}

func initServer(port int) *Server {
	config := Config{
		PrivateKey: prvSrv,
		Port:       port,
	}
	discover := newDiscover()
	server := NewServer(config, discover)
	return server
}

func Test_Listen_failed(t *testing.T) {
	server := initServer(70707)
	assert.Panics(t, func() {
		server.Start()
	})
}

func Test_HandleConn(t *testing.T) {
	pCh := make(chan IPeer)
	subscribe.Sub(subscribe.AddNewPeer, pCh)

	server := initServer(7007)
	assert.NoError(t, server.Start())

	server.newPeer = newTestPeerErrHandshake
	_, err := dial(":7007")
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	server.newPeer = newTestPeer
	_, err = dial(":7007")
	assert.NoError(t, err)
	<-pCh
	time.Sleep(1 * time.Second)
	_, err = dial(":7007")
	assert.NoError(t, err)
	assert.Len(t, server.Connections(), 1)

	time.Sleep(5 * time.Second)
	server.Stop()
}

func dial(dst string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", dst, dialTimeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func Test_Connect(t *testing.T) {
	server := initServer(7007)
	server.dialManager = NewDialManager(func(fd net.Conn, nodeID *NodeID) error {
		return nil
	}, server.discover)
	assert.NoError(t, server.Start())

	res := server.Connect(nodeIDCli.String()[:] + "@127.0.0.1:7007")
	if len(res) < 0 {
		t.Error("dial failed")
	}
	server.Stop()
}
