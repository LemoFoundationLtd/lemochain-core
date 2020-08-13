// Copyright 2016 The lemochain-core Authors
// This file is part of the lemochain-core library.
//
// The lemochain-core library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lemochain-core library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lemochain-core library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/davecgh/go-spew/spew"
)

func TestClientRequest(t *testing.T) {
	server := newTestServer("service", new(Service))
	defer server.Stop()
	client := DialInProc(server)
	defer client.Close()

	var resp Result
	if err := client.Call(&resp, "service_echo", "hello", 10, &Args{"world"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resp, Result{"hello", 10, &Args{"world"}}) {
		t.Errorf("incorrect result %#v", resp)
	}
}

func TestClientBatchRequest(t *testing.T) {
	server := newTestServer("service", new(Service))
	defer server.Stop()
	client := DialInProc(server)
	defer client.Close()

	batch := []BatchElem{
		{
			Method: "service_echo",
			Args:   []interface{}{"hello", 10, &Args{"world"}},
			Result: new(Result),
		},
		{
			Method: "service_echo",
			Args:   []interface{}{"hello2", 11, &Args{"world"}},
			Result: new(Result),
		},
		{
			Method: "no_such_method",
			Args:   []interface{}{1, 2, 3},
			Result: new(int),
		},
	}
	if err := client.BatchCall(batch); err != nil {
		t.Fatal(err)
	}
	wantResult := []BatchElem{
		{
			Method: "service_echo",
			Args:   []interface{}{"hello", 10, &Args{"world"}},
			Result: &Result{"hello", 10, &Args{"world"}},
		},
		{
			Method: "service_echo",
			Args:   []interface{}{"hello2", 11, &Args{"world"}},
			Result: &Result{"hello2", 11, &Args{"world"}},
		},
		{
			Method: "no_such_method",
			Args:   []interface{}{1, 2, 3},
			Result: new(int),
			Error:  &jsonError{Code: -32601, Message: "The method no_such_method_ does not exist/is not available"},
		},
	}
	if !reflect.DeepEqual(batch, wantResult) {
		t.Errorf("batch results mismatch:\ngot %swant %s", spew.Sdump(batch), spew.Sdump(wantResult))
	}
}

// func TestClientCancelInproc(t *testing.T) { testClientCancel("inproc", t) }
func TestClientCancelIPC(t *testing.T) { testClientCancel("ipc", t) }

// This test checks that requests made through CallContext can be canceled by canceling
// the context.
func testClientCancel(transport string, t *testing.T) {
	server := newTestServer("service", new(Service))
	defer server.Stop()

	// What we want to achieve is that the context gets canceled
	// at various stages of request processing. The interesting cases
	// are:
	//  - cancel during dial
	//  - cancel while performing a HTTP request
	//  - cancel while waiting for a response
	//
	// To trigger those, the times are chosen such that connections
	// are killed within the deadline for every other call (maxKillTimeout
	// is 2x maxCancelTimeout).
	//
	// Once a connection is dead, there is a fair chance it won't connect
	// successfully because the accept is delayed by 1s.
	maxContextCancelTimeout := 300 * time.Millisecond
	fl := &flakeyListener{
		maxAcceptDelay: 1 * time.Second,
		maxKillTimeout: 600 * time.Millisecond,
	}

	var client *Client
	switch transport {
	case "ipc":
		c, l := ipcTestClient(server, fl)
		defer l.Close()
		client = c
	default:
		panic("unknown transport: " + transport)
	}

	// These tests take a lot of time, run them all at once.
	// You probably want to run with -parallel 1 or comment out
	// the call to t.Parallel if you enable the logging.
	t.Parallel()

	// The actual test starts here.
	var (
		wg       sync.WaitGroup
		nreqs    = 10
		ncallers = 6
	)
	caller := func(index int) {
		defer wg.Done()
		for i := 0; i < nreqs; i++ {
			var (
				ctx     context.Context
				cancel  func()
				timeout = time.Duration(rand.Int63n(int64(maxContextCancelTimeout)))
			)
			if index < ncallers/2 {
				// For half of the callers, create a context without deadline
				// and cancel it later.
				ctx, cancel = context.WithCancel(context.Background())
				time.AfterFunc(timeout, cancel)
			} else {
				// For the other half, create a context with a deadline instead. This is
				// different because the context deadline is used to set the socket write
				// deadline.
				ctx, cancel = context.WithTimeout(context.Background(), timeout)
			}
			// Now perform a call with the context.
			// The key thing here is that no call will ever complete successfully.
			err := client.CallContext(ctx, nil, "service_sleep", 2*maxContextCancelTimeout)
			if err != nil {
				log.Debug(fmt.Sprint("got expected error:", err))
			} else {
				t.Errorf("no error for call with %v wait time", timeout)
			}
			cancel()
		}
	}
	wg.Add(ncallers)
	for i := 0; i < ncallers; i++ {
		go caller(i)
	}
	wg.Wait()
}

func TestClientReconnect(t *testing.T) {
	// Listen on a random endpoint.
	endpoint := fmt.Sprintf("lemochain-core-test-ipc-%d-%d", os.Getpid(), rand.Int63())
	if runtime.GOOS == "windows" {
		endpoint = `\\.\pipe\` + endpoint
	} else {
		endpoint = os.TempDir() + "/" + endpoint
	}
	startServer := func() (*Server, net.Listener) {
		srv := newTestServer("service", new(Service))

		l, err := ipcListen(endpoint)
		if err != nil {
			panic(err)
		}
		l = &flakeyListener{
			maxAcceptDelay: 1 * time.Second,
			maxKillTimeout: 600 * time.Millisecond,
			Listener:       l,
		}
		go srv.ServeListener(l)
		return srv, l
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start a server and corresponding client.
	s1, l1 := startServer()
	client, err := DialIPC(ctx, l1.Addr().String())
	if err != nil {
		t.Fatal("can't dial", err)
	}

	// Perform a call. This should work because the server is up.
	var resp Result
	if err := client.CallContext(ctx, &resp, "service_echo", "", 1, nil); err != nil {
		t.Fatal(err)
	}

	// Shut down the server and try calling again. It shouldn't work.
	l1.Close()
	s1.Stop()
	if err := client.CallContext(ctx, &resp, "service_echo", "", 2, nil); err == nil {
		t.Error("successful call while the server is down")
		t.Logf("resp: %#v", resp)
	}

	// Allow for some cool down time so we can listen on the same address again.
	time.Sleep(2 * time.Second)

	// Start it up again and call again. The connection should be reestablished.
	// We spawn multiple calls here to check whether this hangs somehow.
	s2, l2 := startServer()
	defer l2.Close()
	defer s2.Stop()

	start := make(chan struct{})
	errors := make(chan error, 20)
	for i := 0; i < cap(errors); i++ {
		go func() {
			<-start
			var resp Result
			errors <- client.CallContext(ctx, &resp, "service_echo", "", 3, nil)
		}()
	}
	close(start)
	errcount := 0
	for i := 0; i < cap(errors); i++ {
		if err = <-errors; err != nil {
			errcount++
		}
	}
	t.Log("err:", err)
	if errcount > 1 {
		t.Errorf("expected one error after disconnect, got %d", errcount)
	}
}

func newTestServer(serviceName string, service interface{}) *Server {
	server := NewServer()
	if err := server.RegisterName(serviceName, service); err != nil {
		panic(err)
	}
	return server
}

func ipcTestClient(srv *Server, fl *flakeyListener) (*Client, net.Listener) {
	// Listen on a random endpoint.
	endpoint := fmt.Sprintf("lemochain-core-test-ipc-%d-%d", os.Getpid(), rand.Int63())
	if runtime.GOOS == "windows" {
		endpoint = `\\.\pipe\` + endpoint
	} else {
		endpoint = os.TempDir() + "/" + endpoint
	}
	l, err := ipcListen(endpoint)
	if err != nil {
		panic(err)
	}
	// Connect the listener to the server.
	if fl != nil {
		fl.Listener = l
		l = fl
	}
	go srv.ServeListener(l)
	// Connect the client.
	client, err := DialIPC(context.Background(), endpoint)
	if err != nil {
		panic(err)
	}
	return client, l
}

// flakeyListener kills accepted connections after a random timeout.
type flakeyListener struct {
	net.Listener
	maxKillTimeout time.Duration
	maxAcceptDelay time.Duration
}

func (l *flakeyListener) Accept() (net.Conn, error) {
	delay := time.Duration(rand.Int63n(int64(l.maxAcceptDelay)))
	time.Sleep(delay)

	c, err := l.Listener.Accept()
	if err == nil {
		timeout := time.Duration(rand.Int63n(int64(l.maxKillTimeout)))
		time.AfterFunc(timeout, func() {
			log.Debug(fmt.Sprintf("killing conn %v after %v", c.LocalAddr(), timeout))
			c.Close()
		})
	}
	return c, err
}
