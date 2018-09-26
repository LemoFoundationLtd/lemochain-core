package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/mclock"
	//"github.com/LemoFoundationLtd/lemochain-go/sync"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"io"
	"net"
	"sync"
	"time"
)

// Peer represents a connected remote node.
type Peer struct {
	rw      *conn
	created mclock.AbsTime
	wg      sync.WaitGroup
	closed  chan struct{}
	nodeId  NodeID // sman 远程节点公钥

	rmu sync.Mutex // 读锁
	wmu sync.Mutex // 写锁

	newMsgCh chan Msg // 新消息
}

func newPeer(fd net.Conn) transport {
	c := &conn{fd: fd, cont: make(chan error)}
	return &Peer{
		rw:       c,
		created:  mclock.Now(),
		closed:   make(chan struct{}),
		newMsgCh: make(chan Msg),
	}
}

// 发送NodeID格式，不直接使用
type authMsg struct {
	Version uint32 // 版本
	NodeID  NodeID // nodeid
}

func (p *Peer) doHandshake(prv *ecdsa.PrivateKey, isSelfServer bool) (err error) {
	if isSelfServer { // 本地为服务端
		err = p.receiverHandshake(prv)
	} else { // 本地为客户端
		err = p.initiatorEncHandshake(prv)
	}
	return err
}

func (p *Peer) Close() {
	p.rw.fd.Close()
	close(p.closed)
}

// 作为服务端处理流程
func (p *Peer) receiverHandshake(prv *ecdsa.PrivateKey) error {
	conn := p.rw.fd
	// 读取对方的NodeID
	buf := make([]byte, 4+64) // 4个字节的版本，64个字节的公钥
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	v := binary.BigEndian.Uint32(buf[:4])
	if v != 1 {
		return errors.New("version not match")
	}
	copy(p.nodeId[:], buf[4:])

	// 发送自己的NodeID
	nodeID := PubkeyID(&prv.PublicKey)
	copy(buf[4:], nodeID[:])
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("server: receive peer:%s conn", common.ToHex(p.nodeId[:8])))
	return nil
}

// 作为客户端处理流程
func (p *Peer) initiatorEncHandshake(prv *ecdsa.PrivateKey) error {
	buf := make([]byte, 4+64)
	// 发送自己的NodeID
	v := make([]byte, 4)
	binary.BigEndian.PutUint32(v, uint32(1))
	copy(buf[:4], v)
	nodeID := PubkeyID(&prv.PublicKey)
	copy(buf[4:], nodeID[:])
	conn := p.rw.fd
	conn.Write(buf)

	// 读取对方的NodeID
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	version := binary.BigEndian.Uint32(buf[:4])
	if version != 1 {
		return errors.New("version not match")
	}
	copy(p.nodeId[:], buf[4:])
	log.Info(fmt.Sprintf("client: connect peer:%s success", common.ToHex(p.nodeId[:8])))
	return nil
}

// 节点运行起来 读取
func (p *Peer) run() (err error) {
	var (
		readErr = make(chan error)
	)
	p.wg.Add(2)

	go p.readLoop(readErr)
	go p.heartbeatLoop()

loop:
	for {
		select {
		case err = <-readErr:
			log.Info(fmt.Sprintf("%v", err))
			break loop
		case <-p.closed:
			break loop
		}
	}

	p.Close()
	p.wg.Wait()
	return err
}

// 节点读取循环
func (p *Peer) readLoop(errCh chan<- error) {
	defer p.wg.Done()
	for {
		// 读取数据
		msg, err := p.readMsg()
		if err != nil {
			errCh <- err
			p.newMsgCh <- Msg{}
			return
		}
		if msg.Code == 0x01 { // 心跳包
			continue
		}
		// 处理数据
		p.newMsgCh <- msg
	}
}

// 数据帧结构 数据解析格式 结构体不直接使用
type frameHeader struct {
	version uint32 // 版本，标识数据是否可用
	code    uint32 // code
	size    uint32 // size
	content []byte
}

// 发送心跳循环
func (p *Peer) heartbeatLoop() {
	defer p.wg.Done()

	heartbeatTimer := time.NewTimer(heartbeatInterval)
	defer heartbeatTimer.Stop()

	for {
		select {
		case <-heartbeatTimer.C:
			if err := p.sendHeartbeatMsg(); err != nil {
				return
			}
			heartbeatTimer.Reset(heartbeatInterval)
		case <-p.closed:
			return
		}
	}
}

func (p *Peer) readMsg() (msg Msg, err error) {
	p.rmu.Lock()
	defer p.rmu.Unlock()

	p.rw.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))

	headBuf := make([]byte, 12) // 帧头12个字节
	if _, err := io.ReadFull(p.rw.fd, headBuf); err != nil {
		return msg, err
	}
	if binary.BigEndian.Uint32(headBuf[:4]) != uint32(baseFrameVersion) {
		str := fmt.Sprintf("remote node's frame version not match. nodeid:%s", common.ToHex(p.nodeId[:]))
		err = errors.New(str)
		log.Warn(str)
		return msg, err
	}
	msg.Code = binary.BigEndian.Uint32(headBuf[4:8])
	if msg.CheckCode() == false {
		return Msg{}, errors.New("recv unavaliable message")
	}
	msg.Size = binary.BigEndian.Uint32(headBuf[8:])
	msg.ReceivedAt = time.Now()
	// 非心跳数据
	if msg.Size > 0 {
		frameBuf := make([]byte, msg.Size)
		if _, err := io.ReadFull(p.rw.fd, frameBuf); err != nil {
			return msg, err
		}
		msg.Payload = bytes.NewReader(frameBuf)
	}
	return msg, nil
}

// 对外提供 供读取节点数据用
func (p *Peer) ReadMsg() (msg Msg) {
	msg = <-p.newMsgCh
	return
}

// 对外提供 提供写入数据到节点
func (p *Peer) WriteMsg(code uint32, content []byte) error {
	p.wmu.Lock()
	defer p.wmu.Unlock()

	buf := make([]byte, len(content)+12)
	headBuf := p.sealFrameHead(code, uint32(len(content)))
	copy(buf[:12], headBuf)
	copy(buf[12:], content)

	_, err := p.rw.fd.Write(buf)
	return err
}

// 发送心跳数据
func (p *Peer) sendHeartbeatMsg() error {
	p.wmu.Lock()
	defer p.wmu.Unlock()

	buf := p.sealFrameHead(0x01, 0)
	_, err := p.rw.fd.Write(buf)
	return err
}

// 封装帧头
func (p *Peer) sealFrameHead(code, size uint32) []byte {
	var (
		buf = make([]byte, 12)
		tmp = make([]byte, 4)
	)
	binary.BigEndian.PutUint32(tmp, uint32(baseFrameVersion))
	copy(buf[:4], tmp)
	binary.BigEndian.PutUint32(tmp, code)
	copy(buf[4:8], tmp)
	binary.BigEndian.PutUint32(tmp, size) // size: 0
	copy(buf[8:], tmp)
	return buf
}

// 获取Peer ID
func (p *Peer) NodeID() NodeID {
	return p.nodeId
}
