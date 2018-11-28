package p2p

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func newDial() *DialManager {
	handleConn := func(fd net.Conn, isSelfServer bool) error {
		if isSelfServer {
			return errors.New("not dial")
		}
		return nil
	}
	discover := NewDiscoverManager("data")
	return NewDialManager(handleConn, discover)
}

func newDialWithErrHandle() *DialManager {
	handleConn := func(fd net.Conn, isSelfServer bool) error {
		return errors.New("handle failed")
	}
	discover := NewDiscoverManager("data")
	return NewDialManager(handleConn, discover)
}

func startListen(startCh, stopCh chan struct{}) {
	listener, err := net.Listen("tcp", "127.0.0.1:7002")
	if err != nil {
		return
	}
	defer listener.Close()

	startCh <- struct{}{}
	listener.Accept()
	fmt.Println("recv new conn")
	<-stopCh
}

func Test_runDialTask_ok(t *testing.T) {
	m := newDial()
	startCh := make(chan struct{})
	stopCh := make(chan struct{})
	go func() {
		startListen(startCh, stopCh)
	}()

	<-startCh
	res := m.runDialTask("127.0.0.1:7002")
	stopCh <- struct{}{}
	assert.Equal(t, 0, res)
}

func Test_runDialTask_no_server(t *testing.T) {
	m := newDial()
	res := m.runDialTask("127.0.0.1:7002")
	assert.Equal(t, -1, res)
}

func Test_runDialTask_err_handle(t *testing.T) {
	m := newDialWithErrHandle()
	startCh := make(chan struct{})
	stopCh := make(chan struct{})
	go func() {
		startListen(startCh, stopCh)
	}()

	<-startCh
	res := m.runDialTask("127.0.0.1:7002")
	stopCh <- struct{}{}
	assert.Equal(t, -2, res)
}

func Test_loop(t *testing.T) {
	startCh := make(chan struct{})
	stopCh := make(chan struct{})
	go func() {
		startListen(startCh, stopCh)
	}()
	<-startCh

	resCh := make(chan bool)
	handleConn := func(fd net.Conn, isSelfServer bool) error {
		resCh <- true
		return nil
	}
	dis := newDiscover()

	dial := NewDialManager(handleConn, dis)
	assert.NoError(t, dial.Start())
	assert.Error(t, dial.Start(), ErrHasStared)

	list := []string{
		"127.0.0.1:7001",
		"127.0.0.1:7002",
		"127.0.0.1:7003",
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

	stopCh <- struct{}{}
}
