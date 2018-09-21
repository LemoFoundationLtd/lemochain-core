// Package p2p implements the Lemochain p2p network protocols.
package p2p

import (
	"bufio"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseProtocolVersion = 1

	baseFrameVersion = 1

	heartbeatInterval = 15 * time.Second
	frameReadTimeout  = 30 * time.Second
	retryConnTimeout  = 30 * time.Second
)

// Config holds Server options.
type Config struct {
	// 私钥
	PrivateKey *ecdsa.PrivateKey

	// 最大可连接节点数 须大于0
	MaxPeerNum int

	// 最大连接中的节点数
	MaxPendingPeerNum int // reserve

	// server的Name
	Name string

	// 黑名单
	NetRestrict *Netlist

	// 节点数据库路径
	NodeDatabase string

	// 监听地址与端口
	ListenAddr string
}

// 对外节点通知类型
type PeerEventFlag int

const (
	AddPeerFlag PeerEventFlag = iota
	DropPeerFlag
)

type PeerEventFn func(peer *Peer, flag PeerEventFlag) error

// Server manages all peer connections.
type Server struct {
	Config // server的一些基本配置

	lock    sync.Mutex // running 读写保护
	running bool       // 标识server是否在运行

	listener net.Listener // TCP监听

	nodeList []string // nodedatabase配置的节点列表

	quit      chan struct{}
	addPeerCh chan *Peer
	delPeerCh chan *Peer

	loopWG sync.WaitGroup

	needConnectNodeCh chan string // 需要立即拨号通道

	newTransport func(net.Conn) transport // 目前只有Peer使用到

	PeerEvent PeerEventFn // 外界注册使用
}

type transport interface {
	doHandshake(prv *ecdsa.PrivateKey, isSelfServer bool) error
	Close()
}

var errServerStopped = errors.New("server has stopped")

// conn wraps a network connection with information gathered
// during the two handshakes.
type conn struct {
	fd net.Conn
	transport
	cont chan error // The run loop uses cont to signal errors to SetupConn.
	id   NodeID     // valid after the encryption handshake
	name string     // valid after the protocol handshake
}

// 启动服务器
func (srv *Server) Start() error {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return fmt.Errorf("server already running")
	}
	if srv.PrivateKey == nil {
		return fmt.Errorf("server.PrivateKey can't be nil")
	}
	if srv.ListenAddr == "" { // 默认强制开始服务器，前期防止搭建都不启动服务
		return fmt.Errorf("server.ListenAddr can't be empty")
	}
	if err := srv.startListening(); err != nil {
		return err
	}
	if srv.addPeerCh == nil {
		srv.addPeerCh = make(chan *Peer)
	}
	if srv.delPeerCh == nil {
		srv.delPeerCh = make(chan *Peer)
	}
	if srv.needConnectNodeCh == nil {
		srv.needConnectNodeCh = make(chan string)
	}
	if srv.newTransport == nil {
		srv.newTransport = newPeer
	}
	if srv.NodeDatabase != "" {
		if err := srv.readNodeDatabaseFile(); err != nil {
			log.Error(err.Error())
		}
	}
	go srv.run()
	srv.running = true
	return nil
}

func (srv *Server) Stop() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if !srv.running {
		return
	}
	srv.running = false
	if srv.listener != nil {
		srv.listener.Close()
	}
	close(srv.quit)
	srv.loopWG.Wait()
}

// 从本地读取节点列表
func (srv *Server) readNodeDatabaseFile() error {
	exePath, err := os.Executable()
	if err != nil {
		return errors.New("can't get executable")
	}
	dir := filepath.Dir(exePath)
	filename := filepath.Join(dir, srv.NodeDatabase)
	fmt.Println(filename)
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return errors.New("nodedatabase file not exist")
		}
	}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	srv.nodeList = make([]string, 0)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if io.EOF == err && strings.Compare(line, "") != 0 {
			} else {
				break
			}
		}
		tmp := strings.Split(line, ":")
		if len(tmp) != 2 {
			continue
		}
		ip := net.ParseIP(tmp[0])
		tcpPort, err := strconv.Atoi(strings.TrimSpace(tmp[1]))
		if ip == nil || err != nil || tcpPort < 1 || tcpPort > 65535 {
			continue
		}

		srv.nodeList = append(srv.nodeList, strings.TrimSpace(line))
	}
	return nil
}

// 启动TCP监听
func (srv *Server) startListening() error {
	listener, err := net.Listen("tcp", srv.ListenAddr)
	if err != nil {
		return err
	}
	srv.listener = listener
	go srv.listenLoop()
	return nil
}

//
func (srv *Server) run() {
	srv.loopWG.Add(1)
	defer srv.loopWG.Done()

	peers := make(map[NodeID]*Peer) // 记录所有的节点连接
	go srv.runDialLoop()            // 启动主动连接调度
	for {
		select {
		case p := <-srv.addPeerCh:
			if _, ok := peers[p.nodeId]; ok {
				p.Close()
				break
			}
			peers[p.nodeId] = p
			go srv.runPeer(p)
			if srv.PeerEvent != nil { // 通知外界收到新的节点
				if err := srv.PeerEvent(p, AddPeerFlag); err != nil {
					p.Close()
				}
			}
		case p := <-srv.delPeerCh:
			delete(peers, p.nodeId)
			if srv.PeerEvent != nil { // 通知外界节点drop
				if err := srv.PeerEvent(p, DropPeerFlag); err != nil {
					log.Error("peer event error", "err", err)
				}
			}
			srv.needConnectNodeCh <- p.rw.fd.RemoteAddr().String() // 断线重连 todo
		case <-srv.quit:
			return
		}
	}
}

// 接受TCP请求
func (srv *Server) listenLoop() {
	for {
		select {
		case <-srv.quit:
			return
		default:
		}

		fd, err := srv.listener.Accept()
		if err != nil {
			log.Debug("TCP Accept error", "err", err)
			continue
		}
		if srv.NetRestrict != nil { // 黑名单处理
			if tcp, ok := fd.RemoteAddr().(*net.TCPAddr); ok && !srv.NetRestrict.Contains(tcp.IP) {
				log.Debug("Rejected conn (in NetRestrict)", "addr", fd.RemoteAddr())
				fd.Close()
				continue
			}
		}
		go srv.HandleConn(fd, true)
	}
}

// 处理接收到的连接 服务端客户端均走此函数
// dialDest == nil ? server : client
func (srv *Server) HandleConn(fd net.Conn, isSelfServer bool) error {
	if !srv.running {
		return errServerStopped
	}
	peer := srv.newTransport(fd)
	err := peer.doHandshake(srv.PrivateKey, isSelfServer)
	if err != nil {
		return err
	}
	srv.addPeerCh <- peer.(*Peer)
	return nil
}

func (srv *Server) runPeer(p *Peer) {
	srv.loopWG.Add(1)
	defer srv.loopWG.Done()

	p.run() // 正常情况下会阻塞 除非节点drop
	srv.delPeerCh <- p
	log.Info(fmt.Sprintf("peer:%s droped", common.ToHex(p.nodeId[:8])))
}

// 启动主动连接调度
func (srv *Server) runDialLoop() {
	srv.loopWG.Add(1)
	defer srv.loopWG.Done()

	failedNodes := make(map[string]struct{}, 0)
	for _, node := range srv.nodeList {
		dialTask := newDialTask(node, srv)
		if err := dialTask.Run(); err != nil {
			failedNodes[node] = struct{}{}
		}
	}
	retryTimer := time.NewTimer(0)
	<-retryTimer.C
	defer retryTimer.Stop()
	for {
		select {
		case <-srv.quit:
			return
		case <-retryTimer.C:
			if len(failedNodes) > 0 {
				for node, _ := range failedNodes {
					dialTask := newDialTask(node, srv)
					if err := dialTask.Run(); err == nil {
						delete(failedNodes, node)
					}
				}
			}
			retryTimer.Reset(retryConnTimeout)
		case node := <-srv.needConnectNodeCh:
			dialTask := newDialTask(node, srv)
			if err := dialTask.Run(); err != nil {
				failedNodes[node] = struct{}{}
			}
		}
	}
}

func (srv *Server) AddStaticPeer(node string) {
	srv.needConnectNodeCh <- node
}
