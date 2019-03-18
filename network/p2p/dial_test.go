package p2p

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func init() {
	prvSrv, _ = crypto.ToECDSA(common.FromHex("0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"))
	deputynode.SetSelfNodeKey(prvSrv)
}

func newDial() *DialManager {
	handleConn := func(fd net.Conn, nodeID *NodeID) error {
		if nodeID == nil {
			return errors.New("not dial")
		}
		return nil
	}
	discover := NewDiscoverManager("data")
	return NewDialManager(handleConn, discover)
}

func newDialWithErrHandle() *DialManager {
	handleConn := func(fd net.Conn, nodeID *NodeID) error {
		return errors.New("handle failed")
	}
	discover := NewDiscoverManager("data")
	return NewDialManager(handleConn, discover)
}

func startListen(startCh chan struct{}, port string) {
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		return
	}
	defer listener.Close()
	log.Info("start listen ok...")
	startCh <- struct{}{}
	listener.Accept()
	fmt.Println("recv new conn")
	// <-stopCh
}

func Test_runDialTask_ok(t *testing.T) {
	m := newDial()
	startCh := make(chan struct{})
	go func() {
		startListen(startCh, "7003")
	}()

	<-startCh
	log.Info("start dial")
	res := m.runDialTask("fe6c44dc5e2f690e6b087ed094875d8f3e49ce03cab9782b1ea25fe676abf3fa81b508929cb13f4cf412ee7150c6a92dc65b86adb5a2e40ad8fe25efbdd12312@127.0.0.1:7003")
	log.Info("dial complete")
	assert.Equal(t, 0, res)
}

func Test_runDialTask_no_server(t *testing.T) {
	m := newDial()
	res := m.runDialTask("fe6c44dc5e2f690e6b087ed094875d8f3e49ce03cab9782b1ea25fe676abf3fa81b508929cb13f4cf412ee7150c6a92dc65b86adb5a2e40ad8fe25efbdd12312@127.0.0.1:7002")
	assert.Equal(t, -1, res)
}

func Test_runDialTask_err_handle(t *testing.T) {
	m := newDialWithErrHandle()
	startCh := make(chan struct{})
	go func() {
		startListen(startCh, "7007")
	}()

	<-startCh
	res := m.runDialTask("fe6c44dc5e2f690e6b087ed094875d8f3e49ce03cab9782b1ea25fe676abf3fa81b508929cb13f4cf412ee7150c6a92dc65b86adb5a2e40ad8fe25efbdd12312@127.0.0.1:7007")
	assert.Equal(t, -2, res)
}

func Test_loop(t *testing.T) {
	startCh := make(chan struct{})
	go func() {
		startListen(startCh, "7002")
	}()
	<-startCh

	resCh := make(chan bool)
	handleConn := func(fd net.Conn, nodeID *NodeID) error {
		resCh <- true
		return nil
	}
	dis := newDiscover()

	dial := NewDialManager(handleConn, dis)
	assert.NoError(t, dial.Start())
	assert.Error(t, dial.Start(), ErrHasStared)

	list := []string{
		"fe6c44dc5e2f690e6b087ed094875d8f3e49ce03cab9782b1ea25fe676abf3fa81b508929cb13f4cf412ee7150c6a92dc65b86adb5a2e40ad8fe25efbdd12312@127.0.0.1:7002",
	}
	dis.AddNewList(list)

	timer := time.NewTimer(5 * time.Second)
	select {
	case r := <-resCh:
		assert.Equal(t, r, true)
	case <-timer.C:
		t.Fatalf("timeout")
		break
	}

	assert.NoError(t, dial.Stop())
	assert.Error(t, dial.Stop(), ErrNotStart)

	time.Sleep(3 * time.Second)
}
