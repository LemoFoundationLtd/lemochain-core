package p2p

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

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

func startListen(startCh chan struct{}) {
	listener, err := net.Listen("tcp", "127.0.0.1:7002")
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
		startListen(startCh)
	}()

	<-startCh
	log.Info("start dial")
	res := m.runDialTask("ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0@127.0.0.1:7002")
	log.Info("dial complete")
	assert.Equal(t, 0, res)
}

func Test_runDialTask_no_server(t *testing.T) {
	m := newDial()
	res := m.runDialTask("ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0@127.0.0.1:7002")
	assert.Equal(t, -1, res)
}

func Test_runDialTask_err_handle(t *testing.T) {
	m := newDialWithErrHandle()
	startCh := make(chan struct{})
	go func() {
		startListen(startCh)
	}()

	<-startCh
	res := m.runDialTask("ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0@127.0.0.1:7002")
	assert.Equal(t, -2, res)
}

func Test_loop(t *testing.T) {
	startCh := make(chan struct{})
	go func() {
		startListen(startCh)
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
		"ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0@127.0.0.1:7002",
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
