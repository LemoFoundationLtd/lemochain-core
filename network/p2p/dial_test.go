package p2p

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
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
