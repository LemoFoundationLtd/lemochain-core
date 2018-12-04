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
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
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
	closeCh chan struct{}
	closed  bool
	nodeID  NodeID // 远程节点公钥

	needReConnect bool

	rmu sync.Mutex // 读锁
	wmu sync.Mutex // 写锁

	newMsgCh chan Msg // 新消息

	heartbeatTimer *time.Timer

	aes []byte
}

func newPeer(fd net.Conn) transport {
	c := &conn{fd: fd, cont: make(chan error)}
	return &Peer{
		rw:            c,
		created:       mclock.Now(),
		closeCh:       make(chan struct{}),
		closed:        false,
		needReConnect: true,
		newMsgCh:      make(chan Msg),
	}
}

func (p *Peer) doHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) (err error) {
	if nodeID == nil { // 本地为服务端
		s, err := serverEncHandshake(p.rw.fd, prv)
		if err != nil {
			return err
		}
		p.aes = make([]byte, len(s.Aes))
		copy(p.aes, s.Aes)
		p.nodeID = s.RemoteID
	} else { // 本地为客户端
		s, err := clientEncHandshake(p.rw.fd, prv, *nodeID)
		if err != nil {
			return err
		}
		p.aes = make([]byte, len(s.Aes))
		copy(p.aes, s.Aes)
		p.nodeID = s.RemoteID
	}
	return err
}

func (p *Peer) Close() {
	p.rw.fd.Close()
	p.closed = true
	close(p.closeCh)
}

func (p *Peer) DisableReConnect() {
	p.needReConnect = false
}

func (p *Peer) NeedReConnect() bool {
	return p.needReConnect
}

// 节点运行起来 读取
func (p *Peer) run() (err error) {
	var (
		readErr = make(chan error)
	)
	p.heartbeatTimer = time.NewTimer(heartbeatInterval)
	go p.readLoop(readErr)
	go p.heartbeatLoop()

	select {
	case err = <-readErr:
		log.Infof("read error: %v", err)
		break
	case <-p.closeCh:
		break
	}

	p.wg.Wait()
	log.Debug("peer.run finished")
	return err
}

// 节点读取循环
func (p *Peer) readLoop(errCh chan<- error) {
	p.wg.Add(1)
	defer func() {
		p.wg.Done()
		log.Debug("peer.readLoop finished.")
	}()
	for {
		// 读取数据
		msg, err := p.readMsg()
		if err != nil {
			if p.closed == false {
				errCh <- err
				p.newMsgCh <- Msg{}
			}
			select {
			case _, ok := <-p.closeCh:
				if ok && err == io.EOF {
					p.closeCh <- struct{}{}
				}
			default:

			}
			// if err == io.EOF {
			// 	p.closeCh <- struct{}{}
			// }
			return
		}
		if msg.Code == 0x01 { // 心跳包
			continue
		}
		// 处理数据
		// log.Debugf("receive message from: %s, msg: %v", common.ToHex(p.nodeID[:8]), msg)
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
	p.wg.Add(1)
	defer func() {
		p.heartbeatTimer.Stop()
		p.heartbeatTimer = nil
		p.wg.Done()
		log.Debug("peer.heartbeatLoop finished.")
	}()

	for {
		select {
		case <-p.heartbeatTimer.C:
			if err := p.sendHeartbeatMsg(); err != nil {
				return
			}
			p.heartbeatTimer.Reset(heartbeatInterval)
		case <-p.closeCh:
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
		str := fmt.Sprintf("remote node's frame version not match. nodeid:%s", common.ToHex(p.nodeID[:]))
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
func (p *Peer) ReadMsg() Msg {
	return <-p.newMsgCh
}

// 对外提供 提供写入数据到节点
func (p *Peer) WriteMsg(code uint32, msg []byte) (err error) {
	p.wmu.Lock()
	defer p.wmu.Unlock()

	buf, err := p.packFrame(code, msg)
	if err != nil {
		return err
	}
	_, err = p.rw.fd.Write(buf)

	p.heartbeatTimer.Reset(heartbeatInterval)
	return err
}

// 发送心跳数据
func (p *Peer) sendHeartbeatMsg() error {
	p.wmu.Lock()
	defer p.wmu.Unlock()

	buf, err := p.packFrame(0x01, nil)
	if err != nil {
		return err
	}
	_, err = p.rw.fd.Write(buf)
	return err
}

// 获取Peer ID
func (p *Peer) NodeID() *NodeID {
	return &p.nodeID
}

func (p *Peer) RemoteAddr() string {
	return p.rw.fd.RemoteAddr().String()
}

func (p *Peer) packFrame(code uint32, msg []byte) ([]byte, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, code)
	if msg != nil {
		buf = append(buf, msg...)
	}
	content, err := crypto.AesEncrypt(buf, p.aes)
	if err != nil {
		return nil, err
	}
	length := make([]byte, PackageLength)
	binary.BigEndian.PutUint32(length, uint32(len(content)))
	buf = append(PackagePrefix, length...)
	buf = append(buf, content...)
	return buf, nil
}

func (p *Peer) unpackFrame(content []byte) (uint32, []byte, error) {
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
