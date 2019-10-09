package p2p

import (
	"crypto/ecdsa"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

type testPeer struct {
	conn         net.Conn
	localNodeID  NodeID
	errHandshake bool

	disconnect bool

	stopCh chan struct{}
	closed bool
}

func newTestPeer(conn net.Conn) IPeer {
	return &testPeer{
		conn:         conn,
		errHandshake: false,
		stopCh:       make(chan struct{}),
	}
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
func (p *testPeer) SetWriteDeadline(duration time.Duration) {

}
func (p *testPeer) RNodeID() *NodeID {
	return &NodeID{0x01, 0x02, 0x03, 0x04, 0x05}
}
func (p *testPeer) RAddress() string {
	if p.disconnect {
		return "127.0.0.1:7001"
	} else if p.conn != nil {
		return p.conn.RemoteAddr().String()
	}
	return "123"
}
func (p *testPeer) LAddress() string {
	return p.conn.LocalAddr().String()
}
func (p *testPeer) DoHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) error {
	if p.errHandshake {
		return errors.New("halt ha ha")
	}
	return nil
}
func (p *testPeer) Run() (err error) {
	time.Sleep(3 * time.Second)
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

	server := initServer(7027)
	assert.NoError(t, server.Start())

	server.newPeer = newTestPeerErrHandshake
	_, err := dial(":7027")
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	server.newPeer = newTestPeer
	_, err = dial(":7027")
	assert.NoError(t, err)
	<-pCh
	time.Sleep(1 * time.Second)
	_, err = dial(":7027")
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

type testDialManger struct {
}

func (d *testDialManger) Start() error {
	return nil
}

func (d *testDialManger) Stop() error {
	return nil
}

func (d *testDialManger) runDialTask(node string) int {
	return 0
}

func Test_Connect(t *testing.T) {
	server := initServer(7007)
	server.newPeer = newTestPeerErrHandshake
	server.dialManager = &testDialManger{}
	assert.NoError(t, server.Start())

	addNewPeerCh := make(chan IPeer)
	removePeerCh := make(chan IPeer)
	subscribe.Sub(subscribe.AddNewPeer, addNewPeerCh)
	subscribe.Sub(subscribe.DeletePeer, removePeerCh)

	res := server.Connect("dba86efb88a96acd81b8f4b13ec9a1a033a7d56edda619c743b5c9911958914e94475716b61d236530368043d379c4c8e5a2107604d63b74e4fe4257a6ce1c25@127.0.0.1:8984")
	assert.Equal(t, res, "Connect success")
}

func Test_Disconnect(t *testing.T) {
	removePeerCh := make(chan IPeer, 1)
	subscribe.ClearSub()
	subscribe.Sub(subscribe.DeletePeer, removePeerCh)

	srv := initServer(5009)
	prv1, _ := crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
	pub1 := &prv1.PublicKey
	nodeID1 := PubKeyToNodeID(pub1)

	prv2, _ := crypto.ToECDSA(common.FromHex("0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"))
	pub2 := &prv2.PublicKey
	nodeID2 := PubKeyToNodeID(pub2)

	conn1, conn2 := net.Pipe()
	p1 := newTestPeer(conn1)
	p1.(*testPeer).disconnect = true
	srv.peersMux.Lock()
	srv.connectedNodes[nodeID1] = p1
	srv.connectedNodes[nodeID2] = newTestPeer(conn2)
	srv.peersMux.Unlock()

	go func() {
		<-removePeerCh
	}()

	assert.Equal(t, true, srv.Disconnect("5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0@127.0.0.1:7001"))
	assert.Len(t, srv.connectedNodes, 1)
}

func Test_server_run(t *testing.T) {
	server := initServer(7007)
	server.newPeer = newTestPeerErrHandshake
	go server.run()

	addNewPeerCh := make(chan IPeer)
	removePeerCh := make(chan IPeer)
	subscribe.ClearSub()
	subscribe.Sub(subscribe.AddNewPeer, addNewPeerCh)
	subscribe.Sub(subscribe.DeletePeer, removePeerCh)

	connCli, _ := net.Pipe()
	peer := server.newPeer(connCli)
	server.addPeerCh <- peer

	timer := time.NewTimer(3 * time.Second)
	select {
	case p := <-addNewPeerCh:
		if p != peer {
			t.Fatal("not match")
		}
	case <-timer.C:
		t.Fatal("can't recv addpeer event")
	}
	server.running = 1
	deleteTimer := time.NewTimer(5 * time.Second)
	select {
	case <-removePeerCh:
		break
	case <-deleteTimer.C:
		t.Fatal("not recv delete event")
	}
}

func Test_Start_error(t *testing.T) {
	srv := initServer(100)
	srv.running = 1
	assert.Equal(t, ErrAlreadyRunning, srv.Start())

	srv.running = 0
	srv.PrivateKey = nil
	assert.Panics(t, func() { srv.Start() })

	srv.running = 0
	srv.PrivateKey = prvSrv
	assert.Panics(t, func() { srv.Start() })
}
