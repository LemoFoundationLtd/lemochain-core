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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/LemoFoundationLtd/lemochain-core/common/log"
)

var (
	ErrClientQuit                = errors.New("client is closed")
	ErrNoResult                  = errors.New("no result in JSON-RPC response")
	ErrSubscriptionQueueOverflow = errors.New("subscription queue overflow")
)

const (
	// Timeouts
	tcpKeepAliveInterval = 30 * time.Second
	defaultWriteTimeout  = 10 * time.Second // used for calls if the context has no deadline
	subscribeTimeout     = 5 * time.Second  // overall timeout lemo_subscribe, rpc_modules calls
)

const (
	// Subscriptions are removed when the subscriber cannot keep up.
	//
	// This can be worked around by supplying a channel with sufficiently sized buffer,
	// but this can be inconvenient and hard to explain in the docs. Another issue with
	// buffered channels is that the buffer is static even though it might not be needed
	// most of the time.
	//
	// The approach taken here is to maintain a per-subscription linked list buffer
	// shrinks on demand. If the buffer reaches the size below, the subscription is
	// dropped.
	maxClientSubscriptionBuffer = 8000
)

// BatchElem is an element in a batch request.
type BatchElem struct {
	Method string
	Args   []interface{}
	// The result is unmarshaled into this field. Result must be set to a
	// non-nil pointer value of the desired type, otherwise the response will be
	// discarded.
	Result interface{}
	// Error is set if the server returns an error for this request, or if
	// unmarshaling into Result fails. It is not set for I/O errors.
	Error error
}

// A value of this type can a JSON-RPC request, notification, successful response or
// error response. Which one it is depends on the fields.
type jsonrpcMessage struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (msg *jsonrpcMessage) isNotification() bool {
	return msg.ID == nil && msg.Method != ""
}

func (msg *jsonrpcMessage) isResponse() bool {
	return msg.hasValidID() && msg.Method == "" && len(msg.Params) == 0
}

func (msg *jsonrpcMessage) hasValidID() bool {
	return len(msg.ID) > 0 && msg.ID[0] != '{' && msg.ID[0] != '['
}

func (msg *jsonrpcMessage) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

// Client represents a connection to an RPC server.
type Client struct {
	idCounter   uint32
	connectFunc func(ctx context.Context) (net.Conn, error)

	// writeConn is only safe to access outside dispatch, with the
	// write lock held. The write lock is taken by sending on
	// requestOp and released by sending on sendDone.
	writeConn net.Conn

	// for dispatch
	close       chan struct{}
	didQuit     chan struct{}          // closed when client quits
	reconnected chan net.Conn          // where write/reconnect sends the new connection
	readErr     chan error             // errors from read
	readResp    chan []*jsonrpcMessage // valid messages from read
	requestOp   chan *requestOp        // for registering response IDs
	sendDone    chan error             // signals write completion, releases write lock
	respWait    map[string]*requestOp  // active requests
}

type requestOp struct {
	ids  []json.RawMessage
	err  error
	resp chan *jsonrpcMessage // receives up to len(ids) responses
}

func (op *requestOp) wait(ctx context.Context) (*jsonrpcMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, op.err
	}
}

// Dial creates a new client for the given URL.
//
// The currently supported URL schemes is a file name with no URL scheme, a local socket connection is established using UNIX domain sockets on supported platforms and named pipes on Windows. If you want to configure transport options, use DialIPC instead.
//
// The client reconnects automatically if the connection is lost.
func Dial(rawurl string) (*Client, error) {
	return DialIPC(context.Background(), rawurl)
}

func newClient(initctx context.Context, connectFunc func(context.Context) (net.Conn, error)) (*Client, error) {
	conn, err := connectFunc(initctx)
	if err != nil {
		return nil, err
	}

	c := &Client{
		writeConn:   conn,
		connectFunc: connectFunc,
		close:       make(chan struct{}),
		didQuit:     make(chan struct{}),
		reconnected: make(chan net.Conn),
		readErr:     make(chan error),
		readResp:    make(chan []*jsonrpcMessage),
		requestOp:   make(chan *requestOp),
		sendDone:    make(chan error, 1),
		respWait:    make(map[string]*requestOp),
	}
	go c.dispatch(conn)
	return c, nil
}

func (c *Client) nextID() json.RawMessage {
	id := atomic.AddUint32(&c.idCounter, 1)
	return []byte(strconv.FormatUint(uint64(id), 10))
}

// SupportedModules calls the rpc_modules method, retrieving the list of
// APIs that are available on the server.
func (c *Client) SupportedModules() ([]string, error) {
	var result []string
	ctx, cancel := context.WithTimeout(context.Background(), subscribeTimeout)
	defer cancel()
	err := c.CallContext(ctx, &result, "rpc_modules")
	return result, err
}

// Close closes the client, aborting any in-flight requests.
func (c *Client) Close() {
	select {
	case c.close <- struct{}{}:
		<-c.didQuit
	case <-c.didQuit:
	}
}

// Call performs a JSON-RPC call with the given arguments and unmarshals into
// result if no error occurred.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
func (c *Client) Call(result interface{}, method string, args ...interface{}) error {
	ctx := context.Background()
	return c.CallContext(ctx, result, method, args...)
}

// CallContext performs a JSON-RPC call with the given arguments. If the context is
// canceled before the call has successfully returned, CallContext returns immediately.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
func (c *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	msg, err := c.newMessage(method, args...)
	if err != nil {
		return err
	}
	op := &requestOp{ids: []json.RawMessage{msg.ID}, resp: make(chan *jsonrpcMessage, 1)}

	err = c.send(ctx, op, msg)
	if err != nil {
		return err
	}

	// dispatch has accepted the request and will close the channel it when it quits.
	switch resp, err := op.wait(ctx); {
	case err != nil:
		return err
	case resp.Error != nil:
		return resp.Error
	case len(resp.Result) == 0:
		return ErrNoResult
	default:
		return json.Unmarshal(resp.Result, &result)
	}
}

// BatchCall sends all given requests as a single batch and waits for the server
// to return a response for all of them.
//
// In contrast to Call, BatchCall only returns I/O errors. Any error specific to
// a request is reported through the Error field of the corresponding BatchElem.
//
// Note that batch calls may not be executed atomically on the server side.
func (c *Client) BatchCall(b []BatchElem) error {
	ctx := context.Background()
	return c.BatchCallContext(ctx, b)
}

// BatchCall sends all given requests as a single batch and waits for the server
// to return a response for all of them. The wait duration is bounded by the
// context's deadline.
//
// In contrast to CallContext, BatchCallContext only returns errors that have occurred
// while sending the request. Any error specific to a request is reported through the
// Error field of the corresponding BatchElem.
//
// Note that batch calls may not be executed atomically on the server side.
func (c *Client) BatchCallContext(ctx context.Context, b []BatchElem) error {
	msgs := make([]*jsonrpcMessage, len(b))
	op := &requestOp{
		ids:  make([]json.RawMessage, len(b)),
		resp: make(chan *jsonrpcMessage, len(b)),
	}
	for i, elem := range b {
		msg, err := c.newMessage(elem.Method, elem.Args...)
		if err != nil {
			return err
		}
		msgs[i] = msg
		op.ids[i] = msg.ID
	}

	var err error
	err = c.send(ctx, op, msgs)

	// Wait for all responses to come back.
	for n := 0; n < len(b) && err == nil; n++ {
		var resp *jsonrpcMessage
		resp, err = op.wait(ctx)
		if err != nil {
			break
		}
		// Find the element corresponding to this response.
		// The element is guaranteed to be present because dispatch
		// only sends valid IDs to our channel.
		var elem *BatchElem
		for i := range msgs {
			if bytes.Equal(msgs[i].ID, resp.ID) {
				elem = &b[i]
				break
			}
		}
		if resp.Error != nil {
			elem.Error = resp.Error
			continue
		}
		if len(resp.Result) == 0 {
			elem.Error = ErrNoResult
			continue
		}
		elem.Error = json.Unmarshal(resp.Result, elem.Result)
	}
	return err
}

func (c *Client) newMessage(method string, paramsIn ...interface{}) (*jsonrpcMessage, error) {
	params, err := json.Marshal(paramsIn)
	if err != nil {
		return nil, err
	}
	return &jsonrpcMessage{Version: "2.0", ID: c.nextID(), Method: method, Params: params}, nil
}

// send registers op with the dispatch loop, then sends msg on the connection.
// if sending fails, op is deregistered.
func (c *Client) send(ctx context.Context, op *requestOp, msg interface{}) error {
	select {
	case c.requestOp <- op:
		// log.Debug("rpc", "msg", log.Lazy{Fn: func() string {
		// 	return fmt.Sprint("sending ", msg)
		// }})
		err := c.write(ctx, msg)
		c.sendDone <- err
		return err
	case <-ctx.Done():
		// This can happen if the client is overloaded or unable to keep up with
		// subscription notifications.
		return ctx.Err()
	case <-c.didQuit:
		return ErrClientQuit
	}
}

func (c *Client) write(ctx context.Context, msg interface{}) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultWriteTimeout)
	}
	// The previous write failed. Try to establish a new connection.
	if c.writeConn == nil {
		if err := c.reconnect(ctx); err != nil {
			return err
		}
	}
	c.writeConn.SetWriteDeadline(deadline)
	err := json.NewEncoder(c.writeConn).Encode(msg)
	if err != nil {
		c.writeConn = nil
	}
	return err
}

func (c *Client) reconnect(ctx context.Context) error {
	newconn, err := c.connectFunc(ctx)
	if err != nil {
		log.Debug(fmt.Sprintf("reconnect failed: %v", err))
		return err
	}
	select {
	case c.reconnected <- newconn:
		c.writeConn = newconn
		return nil
	case <-c.didQuit:
		newconn.Close()
		return ErrClientQuit
	}
}

// dispatch is the main loop of the client.
// It sends read messages to waiting calls to Call and BatchCall
// and subscription notifications to registered subscriptions.
func (c *Client) dispatch(conn net.Conn) {
	// Spawn the initial read loop.
	go c.read(conn)

	var (
		lastOp        *requestOp    // tracks last send operation
		requestOpLock = c.requestOp // nil while the send lock is held
		reading       = true        // if true, a read loop is running
	)
	defer close(c.didQuit)
	defer func() {
		c.closeRequestOps(ErrClientQuit)
		conn.Close()
		if reading {
			// Empty read channels until read is dead.
			for {
				select {
				case <-c.readResp:
				case <-c.readErr:
					return
				}
			}
		}
	}()

	for {
		select {
		case <-c.close:
			return

			// Read path.
		case batch := <-c.readResp:
			for _, msg := range batch {
				switch {
				case msg.isNotification():
					// log.Debug("rpc", "msg", log.Lazy{Fn: func() string {
					// 	return fmt.Sprint("<-readResp: notification ", msg)
					// }})
					c.handleNotification(msg)
				case msg.isResponse():
					// log.Debug("rpc", "msg", log.Lazy{Fn: func() string {
					// 	return fmt.Sprint("<-readResp: response ", msg)
					// }})
					c.handleResponse(msg)
				default:
					log.Debug("rpc", "msg", log.Lazy{Fn: func() string {
						return fmt.Sprint("<-readResp: dropping weird message", msg)
					}})
					// TODO: maybe close
				}
			}

		case err := <-c.readErr:
			log.Debug(fmt.Sprintf("<-readErr: %v", err))
			c.closeRequestOps(err)
			conn.Close()
			reading = false

		case newconn := <-c.reconnected:
			log.Debug(fmt.Sprintf("<-reconnected: (reading=%t) %v", reading, conn.RemoteAddr()))
			if reading {
				// Wait for the previous read loop to exit. This is a rare case.
				conn.Close()
				<-c.readErr
			}
			go c.read(newconn)
			reading = true
			conn = newconn

			// Send path.
		case op := <-requestOpLock:
			// Stop listening for further send ops until the current one is done.
			requestOpLock = nil
			lastOp = op
			for _, id := range op.ids {
				c.respWait[string(id)] = op
			}

		case err := <-c.sendDone:
			if err != nil {
				// Remove response handlers for the last send. We remove those here
				// because the error is already handled in Call or BatchCall. When the
				// read loop goes down, it will signal all other current operations.
				for _, id := range lastOp.ids {
					delete(c.respWait, string(id))
				}
			}
			// Listen for send ops again.
			requestOpLock = c.requestOp
			lastOp = nil
		}
	}
}

// closeRequestOps unblocks pending send ops and active subscriptions.
func (c *Client) closeRequestOps(err error) {
	didClose := make(map[*requestOp]bool)

	for id, op := range c.respWait {
		// Remove the op so that later calls will not close op.resp again.
		delete(c.respWait, id)

		if !didClose[op] {
			op.err = err
			close(op.resp)
			didClose[op] = true
		}
	}
}

func (c *Client) handleNotification(msg *jsonrpcMessage) {
	if !strings.HasSuffix(msg.Method, notificationMethodSuffix) {
		log.Debug(fmt.Sprint("dropping non-subscription message: ", msg))
		return
	}
	var subResult struct {
		ID     string          `json:"subscription"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(msg.Params, &subResult); err != nil {
		log.Debug(fmt.Sprint("dropping invalid subscription message: ", msg))
		return
	}
}

func (c *Client) handleResponse(msg *jsonrpcMessage) {
	op := c.respWait[string(msg.ID)]
	if op == nil {
		log.Debug(fmt.Sprintf("unsolicited response %v", msg))
		return
	}
	delete(c.respWait, string(msg.ID))
	// forward the reply to Call/BatchCall.
	op.resp <- msg
}

// Reading happens on a dedicated goroutine.

func (c *Client) read(conn net.Conn) error {
	var (
		buf json.RawMessage
		dec = json.NewDecoder(conn)
	)
	readMessage := func() (rs []*jsonrpcMessage, err error) {
		buf = buf[:0]
		if err = dec.Decode(&buf); err != nil {
			return nil, err
		}
		if isBatch(buf) {
			err = json.Unmarshal(buf, &rs)
		} else {
			rs = make([]*jsonrpcMessage, 1)
			err = json.Unmarshal(buf, &rs[0])
		}
		return rs, err
	}

	for {
		resp, err := readMessage()
		if err != nil {
			c.readErr <- err
			return err
		}
		c.readResp <- resp
	}
}
