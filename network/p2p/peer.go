package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/mclock"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"io"
	"net"
	"sync"
	"time"
)

const (
	StatusNormal int32 = iota
	StatusHardFork
	StatusManualDisconnect
	StatusFailedHandshake
	StatusBadData
	StatusDifferentGenesis
)

var (
	readMsgSuccessTimer  = metrics.NewTimer(metrics.ReadMsgSuccess_timerName)  // 统计成功读取msg的timer
	readMsgFailedTimer   = metrics.NewTimer(metrics.ReadMsgFailed_timerName)   // 统计读取msg失败的timer
	writeMsgSuccessTimer = metrics.NewTimer(metrics.WriteMsgSuccess_timerName) // 统计写msg成功的timer
	writeMsgFailedTimer  = metrics.NewTimer(metrics.WriteMsgFailed_timerName)  // 统计写msg失败的timer
)

type IPeer interface {
	ReadMsg() (msg *Msg, err error)
	WriteMsg(code MsgCode, msg []byte) (err error)
	SetWriteDeadline(duration time.Duration)
	RNodeID() *NodeID
	RAddress() string
	LAddress() string
	DoHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) error
	Run() (err error)
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

	status   int32
	wmu      sync.Mutex
	newMsgCh chan *Msg
	wg       sync.WaitGroup
	stopCh   chan struct{}
}

// NewPeer
func NewPeer(fd net.Conn) IPeer {
	return &Peer{
		conn:          fd,
		created:       mclock.Now(),
		writeDeadline: frameWriteTimeout,
		// closed:   false,
		newMsgCh: make(chan *Msg, 10),
		stopCh:   make(chan struct{}),
	}
}

// DoHandshake do handshake when connection
func (p *Peer) DoHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) (err error) {
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
	p.wmu.Lock()
	p.safeClose()
	p.wmu.Unlock()
}

// safeClose
func (p *Peer) safeClose() {
	select {
	case <-p.stopCh:
		return
	default:
	}

	close(p.stopCh)
	subscribe.Send(subscribe.SrvDeletePeer, p)
	log.Info("Close peer connection")
	if err := p.conn.Close(); err != nil {
		log.Infof("Close peer connection failed: %v", err)
	}
}

// Run  Run peer and block this
func (p *Peer) Run() (err error) {
	p.wg.Add(2)
	go p.heartbeatLoop()
	go p.readLoop()
	// block this and wait for stop
	p.wg.Wait()
	log.Debugf("Peer.Run finished.p: %s", p.RAddress())
	return err
}

// readLoop
func (p *Peer) readLoop() {
	defer func() {
		p.wg.Done()
		log.Debugf("ReadLoop finished: %s", p.RNodeID().String()[:16])
	}()
	for {
		start := time.Now()
		content, err := p.readConn()
		if err != nil {
			log.Debugf("Read conn err: %v", err)
			readMsgFailedTimer.UpdateSince(start)
			p.Close()
			return
		}

		// handle content
		err = p.handle(content)
		if err != nil {
			log.Debugf("Handle conn content err: %v", err)
			readMsgFailedTimer.UpdateSince(start)
			p.Close()
			return
		}
		readMsgSuccessTimer.UpdateSince(start)
	}
}

// ReadMsg read message for call of outside
func (p *Peer) ReadMsg() (msg *Msg, err error) {
	select {
	case <-p.stopCh:
		log.Debugf("Stop Peer at ReadMsg function, peer nodeId:%s", p.rNodeID.String()[:16])
		err = io.EOF
	case msg = <-p.newMsgCh:
		err = nil
	}
	return msg, err
}

// readMsg read message from conn
func (p *Peer) readConn() ([]byte, error) {
	// set read outTime
	if err := p.conn.SetReadDeadline(time.Now().Add(frameReadTimeout)); err != nil {
		return nil, err
	}
	// read PackagePrefix and package length
	headBuf := make([]byte, len(PackagePrefix)+PackageLength) // 6 bytes
	if _, err := io.ReadFull(p.conn, headBuf); err != nil {
		return nil, err
	}
	// compare PackagePrefix
	if bytes.Compare(PackagePrefix[:], headBuf[:2]) != 0 {
		log.Debug("ReadMsg: recv invalid stream data")
		return nil, ErrUnavailablePackage
	}
	// package length
	length := binary.BigEndian.Uint32(headBuf[2:])
	if length == 0 {
		return nil, ErrUnavailablePackage
	}
	if length > params.MaxPackageLength {
		return nil, ErrLengthOverflow
	}
	// read actual encoded content
	content := make([]byte, length)
	if _, err := io.ReadFull(p.conn, content); err != nil {
		return nil, err
	}
	return content, nil
}

// handle
func (p *Peer) handle(content []byte) (err error) {
	// unpack frame
	code, buf, err := p.unpackFrame(content)
	if err != nil {
		return err
	}
	msg := &Msg{
		Code:       code,
		Content:    buf,
		ReceivedAt: time.Now(),
	}
	// check code
	if msg.CheckCode() == false {
		return ErrUnavailablePackage
	}
	switch {
	case msg.Code == HeartbeatMsg:
		return nil
	default:
		select {
		case <-p.stopCh:
			log.Info("Read'peer has stopped")
			return io.EOF
		case p.newMsgCh <- msg:
			log.Debug("← Receive network message success", "Code", msg.Code, "NodeID", common.ToHex(p.rNodeID[:4]))
			return nil
		}
	}
	return nil
}

// WriteMsg send message to net stream
func (p *Peer) WriteMsg(code MsgCode, msg []byte) (err error) {
	p.wmu.Lock()
	defer p.wmu.Unlock()

	start := time.Now()
	defer func() {
		if err != nil {
			writeMsgFailedTimer.UpdateSince(start)
		} else {
			writeMsgSuccessTimer.UpdateSince(start)
		}
	}()

	// pack message frame
	buf, err := p.packFrame(code, msg)
	if err != nil {
		return err
	}
	p.conn.SetWriteDeadline(time.Now().Add(p.writeDeadline))
	_, err = p.conn.Write(buf)
	p.writeDeadline = frameWriteTimeout

	log.Debug("→ Send network message success", "Code", code, "NodeID", common.ToHex(p.rNodeID[:4]))
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
	heartbeat := time.NewTicker(heartbeatInterval)
	defer func() {
		heartbeat.Stop()
		p.wg.Done()
		log.Debugf("HeartbeatLoop finished: %s", p.RNodeID().String()[:16])
	}()

	var count = 3
	for {
		select {
		case <-p.stopCh:
			log.Debugf("Peer stopch from heartbeat. nodeID:%s", p.RNodeID().String()[:16])
			return
		case <-heartbeat.C:
			for i := 1; ; i++ {
				// send heartbeat data
				err := p.WriteMsg(HeartbeatMsg, nil)
				if err != nil {
					if i <= count {
						continue
					} else {
						log.Debugf("HeartbeatLoop error: nodeID: %s, err: %v", p.RNodeID().String()[:16], err)
						p.Close()
						return
					}
				} else {
					break
				}
			}
		}
	}
}

// packFrame pack message to net stream
func (p *Peer) packFrame(code MsgCode, msg []byte) ([]byte, error) {
	// message code to bytes
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(code))
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
func (p *Peer) unpackFrame(content []byte) (MsgCode, []byte, error) {
	// AES Decrypt
	originData, err := crypto.AesDecrypt(content, p.aes)
	if err != nil {
		return 0, nil, err
	}
	code := binary.BigEndian.Uint32(originData[:4])
	if len(originData) == 4 {
		return MsgCode(code), nil, nil
	}
	return MsgCode(code), originData[4:], nil
}

// SetStatus set peer's status
func (p *Peer) SetStatus(status int32) {
	p.status = status
}

// NeedReConnect
func (p *Peer) NeedReConnect() bool {
	switch p.status {
	case StatusHardFork:
		return false
	case StatusDifferentGenesis:
		return false
	case StatusManualDisconnect:
		return false
	case StatusFailedHandshake:
		return false
	case StatusBadData:
		return false

	case StatusNormal:
		return true
	default:
		return true
	}
}
