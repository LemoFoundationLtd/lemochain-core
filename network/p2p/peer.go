package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/mclock"
	"io"
	"net"
	"sync"
	"time"
)

const CodeHeartbeat = uint32(0x01)

const (
	StatusNormal int32 = iota
	StatusHardFork
	StatusManualDisconnect
	StatusFailedHandshake
	StatusBadData
)

type IPeer interface {
	ReadMsg() (msg *Msg, err error)
	WriteMsg(code uint32, msg []byte) (err error)
	SetWriteDeadline(duration time.Duration)
	RNodeID() *NodeID
	RAddress() string
	LAddress() string
	doHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) error
	run() (err error)
	NeedReConnect() bool
	SetStatus(status int32)
	Close()
}

// Peer represents a connected remote node.
type Peer struct {
	conn          net.Conn
	rNodeID       NodeID // remote NodeID
	aes           []byte // AES key
	created       mclock.AbsTime
	writeDeadline time.Duration

	status         int32
	heartbeatTimer *time.Timer
	wmu            sync.Mutex
	newMsgCh       chan *Msg
	wg             sync.WaitGroup
	stopCh         chan struct{}
}

// newPeer
func newPeer(fd net.Conn) IPeer {
	return &Peer{
		conn:          fd,
		created:       mclock.Now(),
		writeDeadline: frameWriteTimeout,
		// closed:   false,
		newMsgCh: make(chan *Msg),
		stopCh:   make(chan struct{}),
	}
}

// doHandshake do handshake when connection
func (p *Peer) doHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) (err error) {
	// as server
	if nodeID == nil {
		s, err := serverEncHandshake(p.conn, prv, nil)
		if err != nil {
			return err
		}
		p.aes = s.Aes
		p.rNodeID = s.RemoteID
	} else { // as client
		s, err := clientEncHandshake(p.conn, prv, nodeID)
		if err != nil {
			return err
		}
		p.aes = s.Aes
		p.rNodeID = s.RemoteID
	}
	return err
}

// Close close peer
func (p *Peer) Close() {
	p.safeClose()
}

// safeClose
func (p *Peer) safeClose() {
	needClose := false
	select {
	case _, needClose = <-p.stopCh:
	default:
		needClose = true
	}
	if needClose {
		close(p.stopCh)
		p.conn.Close()
	}
}

// run  run peer and block this
func (p *Peer) run() (err error) {
	p.wg.Add(2)
	p.heartbeatTimer = time.NewTimer(heartbeatInterval)
	go p.heartbeatLoop()
	go p.readLoop()
	// block this and wait for stop
	p.wg.Wait()
	log.Debugf("peer.run finished.p: %s", p.RAddress())
	return err
}

// readLoop
func (p *Peer) readLoop() {
	defer p.wg.Done()

	for {
		msg, err := p.readMsg()
		if err != nil {
			log.Debugf("read message error: %v", err)
			p.safeClose()
			return
		}
		if msg.Code == CodeHeartbeat {
			continue
		}
		p.newMsgCh <- msg
	}
}

// ReadMsg read message for call of outside
func (p *Peer) ReadMsg() (msg *Msg, err error) {
	select {
	case <-p.stopCh:
		err = io.EOF
	case msg = <-p.newMsgCh:
		err = nil
	}
	return msg, err
}

// readMsg read message from net stream
func (p *Peer) readMsg() (msg *Msg, err error) {
	p.conn.SetReadDeadline(time.Now().Add(frameReadTimeout))
	// read PackagePrefix and package length
	headBuf := make([]byte, len(PackagePrefix)+PackageLength) // 6 bytes
	if _, err := io.ReadFull(p.conn, headBuf); err != nil {
		return msg, err
	}
	// compare PackagePrefix
	if bytes.Compare(PackagePrefix[:], headBuf[:2]) != 0 {
		log.Debug("readMsg: recv invalid stream data")
		return msg, ErrUnavailablePackage
	}
	// package length
	length := binary.BigEndian.Uint32(headBuf[2:])
	if length == 0 {
		return msg, ErrUnavailablePackage
	}
	// read actual encoded content
	content := make([]byte, length)
	if _, err := io.ReadFull(p.conn, content); err != nil {
		return msg, err
	}
	// unpack frame
	code, buf, err := p.unpackFrame(content)
	if err != nil {
		return nil, err
	}
	msg = &Msg{
		Code:       code,
		Content:    buf,
		ReceivedAt: time.Now(),
	}
	// check code
	if msg.CheckCode() == false {
		return msg, ErrUnavailablePackage
	}
	return msg, nil
}

// WriteMsg send message to net stream
func (p *Peer) WriteMsg(code uint32, msg []byte) (err error) {
	p.wmu.Lock()
	defer p.wmu.Unlock()
	// pack message frame
	buf, err := p.packFrame(code, msg)
	if err != nil {
		return err
	}
	p.conn.SetWriteDeadline(time.Now().Add(p.writeDeadline))
	_, err = p.conn.Write(buf)
	// reset heartbeatTimer
	if code != CodeHeartbeat && p.heartbeatTimer != nil {
		p.heartbeatTimer.Reset(heartbeatInterval)
	}
	p.writeDeadline = frameWriteTimeout
	return err
}

// SetWriteDeadline
func (p *Peer) SetWriteDeadline(duration time.Duration) {
	p.wmu.Lock()
	defer p.wmu.Unlock()
	p.writeDeadline = duration
}

// RNodeID
func (p *Peer) RNodeID() *NodeID {
	return &p.rNodeID
}

// RAddress remote address (ipv4:port)
func (p *Peer) RAddress() string {
	return p.conn.RemoteAddr().String()
}

// LAddress local address (ipv4:port)
func (p *Peer) LAddress() string {
	return p.conn.LocalAddr().String()
}

// heartbeatLoop send heartbeat info when after special internal of no data sending
func (p *Peer) heartbeatLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.heartbeatTimer.C:
			// send heartbeat data
			if err := p.WriteMsg(CodeHeartbeat, nil); err != nil {
				log.Debugf("heartbeatLoop: send heartbeat data failed and stopped: %v", err)
				return
			}
			// reset heartbeatTimer
			p.heartbeatTimer.Reset(heartbeatInterval)
		case <-p.stopCh:
			log.Debug("heartbeatLoop: stopped. p: %s", p.RAddress())
			return
		}
	}
}

// packFrame pack message to net stream
func (p *Peer) packFrame(code uint32, msg []byte) ([]byte, error) {
	// message code to bytes
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, code)
	// combine code and message buffer
	if msg != nil {
		buf = append(buf, msg...)
	}
	// AES encrypt
	content, err := crypto.AesEncrypt(buf, p.aes)
	if err != nil {
		return nil, err
	}
	// make length bytes
	length := make([]byte, PackageLength)
	binary.BigEndian.PutUint32(length, uint32(len(content)))
	// make header
	buf = append(PackagePrefix, length...)
	// combine header and body
	buf = append(buf, content...)
	return buf, nil
}

// unpackFrame unpack net stream
func (p *Peer) unpackFrame(content []byte) (uint32, []byte, error) {
	// AES Decrypt
	originData, err := crypto.AesDecrypt(content, p.aes)
	if err != nil {
		return 0, nil, err
	}
	code := binary.BigEndian.Uint32(originData[:4])
	if len(originData) == 4 {
		return code, nil, nil
	}
	return code, originData[4:], nil
}

// SetStatus set peer's status
func (p *Peer) SetStatus(status int32) {
	p.status = status
}

// NeedReConnect
func (p *Peer) NeedReConnect() bool {
	if p.status == StatusHardFork || p.status == StatusManualDisconnect || p.status == StatusFailedHandshake || p.status == StatusBadData {
		return false
	}
	return true
}
