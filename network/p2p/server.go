// Package p2p implements the Lemochain p2p network protocols.
package p2p

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/subscribe"
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

	heartbeatInterval = 10 * time.Second
	frameReadTimeout  = 30 * time.Second
	retryConnTimeout  = 30 * time.Second
)

// Config holds Server options.
type Config struct {
	// private key
	PrivateKey *ecdsa.PrivateKey

	// max accept connection count
	MaxPeerNum int

	// 最大连接中的节点数
	MaxPendingPeerNum int // reserve

	// server's Name
	Name string

	// 黑名单
	NetRestrict *Netlist

	// 节点数据库路径
	NodeDatabase string

	// listen port
	Port int
}

func (config *Config) ListenAddr() string {
	return fmt.Sprintf(":%d", config.Port)
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

	nodeList []string         // nodedatabase配置的节点列表
	peers    map[string]*Peer // 记录所有的节点连接
	peersMux sync.Mutex

	quitCh    chan struct{}
	addPeerCh chan *Peer
	delPeerCh chan *Peer

	loopWG sync.WaitGroup

	// needConnectNodeCh chan string // 需要立即拨号通道

	newTransport func(net.Conn) transport // 目前只有Peer使用到

	// for discover
	discover    *DiscoverManager
	dialManager *DialManager

	// PeerEvent PeerEventFn // 外界注册使用
}

func NewServer(config Config /*, peerEvent PeerEventFn*/, discover *DiscoverManager) *Server {
	srv := &Server{
		Config:   config,
		discover: discover,
		// PeerEvent: peerEvent,
	}
	srv.dialManager = NewDialManager(srv.HandleConn, srv.discover)
	return srv
}

type transport interface {
	doHandshake(prv *ecdsa.PrivateKey, nodeID *NodeID) error
	Close()
	NodeID() *NodeID
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
	if err := srv.startListening(); err != nil {
		return err
	}
	if srv.addPeerCh == nil {
		srv.addPeerCh = make(chan *Peer, 5)
	}
	if srv.peers == nil {
		srv.peers = make(map[string]*Peer)
	}
	if srv.delPeerCh == nil {
		srv.delPeerCh = make(chan *Peer, 5)
	}
	if srv.newTransport == nil {
		srv.newTransport = newPeer
	}
	if srv.quitCh == nil {
		srv.quitCh = make(chan struct{})
	}
	if srv.NodeDatabase != "" {
		if err := srv.readNodeDatabaseFile(); err != nil {
			log.Error(err.Error())
		}
	}
	// discover
	if srv.discover != nil { // todo
		nodes := deputynode.Instance().GetLatestDeputies()
		srv.discover.SetDeputyNodes(nodes)
		srv.discover.Start()
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
	if srv.peers != nil {
		for _, p := range srv.peers {
			p.Close()
		}
		srv.peers = nil
	}
	// stop discover
	if srv.discover != nil {
		if err := srv.discover.Stop(); err != nil {
			log.Errorf("stop discover error: %v", err)
		} else {
			// srv.discover = nil
			log.Debug("stop discover ok.")
		}
	}
	close(srv.quitCh)
	srv.loopWG.Wait()
	log.Debug("server stop success")
}

// 从本地读取节点列表
func (srv *Server) readNodeDatabaseFile() error {
	exePath := os.Args[0]
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
	listener, err := net.Listen("tcp", srv.ListenAddr())
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
	defer func() {
		srv.loopWG.Done()
		log.Debug("server.run stop")
	}()

	// peers := make(map[NodeID]*Peer) // 记录所有的节点连接
	go srv.dialManager.Start() // 启动主动连接调度
	for {
		select {
		case p := <-srv.addPeerCh:
			log.Debugf("receive srv.addPeerCh. node id: %s", common.ToHex(p.nodeID[:8]))
			// 判断此节点是否在peers中
			if _, ok := srv.peers[p.nodeID.String()]; ok {
				log.Warnf("Connection has already exist. Remote node id: %s", common.ToHex(p.nodeID[:8]))
				p.Close()
				srv.discover.SetConnectResult(p.NodeID(), false) // todo
				break
			}
			srv.peersMux.Lock()
			srv.peers[p.nodeID.String()] = p
			srv.peersMux.Unlock()
			go srv.runPeer(p)
			subscribe.Send(subscribe.AddNewPeer, p)
		case p := <-srv.delPeerCh:
			log.Debug("server: recv delete peer event")
			srv.peersMux.Lock()
			delete(srv.peers, p.nodeID.String())
			srv.peersMux.Unlock()
			subscribe.Send(subscribe.DeletePeer, p)
			if p.NeedReConnect() {
				srv.discover.SetReconnect(p.NodeID())
			}
		case <-srv.quitCh:
			return
		}
	}
}

// 接受TCP请求
func (srv *Server) listenLoop() {
	for {
		fd, err := srv.listener.Accept()
		if err != nil {
			if srv.running == false {
				return
			}
			log.Debug("TCP Accept error", "err", err)
			continue
		}
		if srv.NetRestrict != nil { // 黑名单处理
			if tcp, ok := fd.RemoteAddr().(*net.TCPAddr); ok && srv.NetRestrict.Contains(tcp.IP) {
				log.Debug("Rejected conn (in NetRestrict)", "addr", fd.RemoteAddr())
				fd.Close()
				continue
			}
		}
		go srv.HandleConn(fd, nil)
	}
}

// 处理接收到的连接 服务端客户端均走此函数
// isSelfServer == true ? server : client
func (srv *Server) HandleConn(fd net.Conn, nodeID *NodeID) error {
	if !srv.running {
		return errServerStopped
	}
	peer := srv.newTransport(fd)
	err := peer.doHandshake(srv.PrivateKey, nodeID)
	if err != nil {
		fd.Close()

		// for discover
		srv.discover.SetConnectResult(peer.NodeID(), false)

		return err
	}
	p := peer.(*Peer)
	if bytes.Compare(p.nodeID[:], deputynode.GetSelfNodeID()) == 0 {
		fd.Close()

		// for discover
		srv.discover.SetConnectResult(peer.NodeID(), false)

		return ErrConnectSelf
	}
	if nodeID == nil {
		log.Debugf("Receive new connect, IP: %s. ID: %s ", p.rw.fd.RemoteAddr().String(), common.ToHex(p.nodeID[:8]))
	} else {
		log.Debugf("Connect to server: %s. id: %s", p.rw.fd.RemoteAddr(), common.ToHex(p.nodeID[:8]))
	}
	srv.addPeerCh <- p
	log.Debug("transfer new peer to srv.addPeerCh")
	return nil
}

func (srv *Server) runPeer(p *Peer) {
	log.Debugf("start run peer. node id: %s", common.ToHex(p.nodeID[:8]))
	p.run() // 正常情况下会阻塞 除非节点drop
	if srv.running == true {
		srv.delPeerCh <- p
	}
	log.Debugf("peer: %s stopped", p.rw.fd.RemoteAddr().String())
}

func (srv *Server) Connect(node string) string {
	log.Infof("start add static peer: %s", node)
	srv.discover.AddNewList([]string{node})
	if res := srv.dialManager.runDialTask(node); res < 0 {
		return "connect failed"
	}
	return ""
}

//go:generate gencodec -type PeerConnInfo -out gen_peer_conn_info_json.go

type PeerConnInfo struct {
	LocalAddr  string `json:"localAddress"`
	RemoteAddr string `json:"remoteAddress"`
	NodeID     string `json:"nodeID"`
}

func (srv *Server) Connections() []PeerConnInfo {
	srv.peersMux.Lock()
	defer srv.peersMux.Unlock()
	result := make([]PeerConnInfo, 0, len(srv.peers))
	for _, v := range srv.peers {
		info := PeerConnInfo{v.rw.fd.LocalAddr().String(), v.rw.fd.RemoteAddr().String(), v.nodeID.String()}
		result = append(result, info)
	}
	return result
}

func (srv *Server) Disconnect(node string) bool {
	for id, v := range srv.peers {
		if strings.Compare(node, v.rw.fd.RemoteAddr().String()) == 0 {
			v.needReConnect = false
			v.Close()
			srv.peersMux.Lock()
			delete(srv.peers, id)
			srv.peersMux.Unlock()
			// if srv.PeerEvent != nil { // 通知外界节点drop
			// 	if err := srv.PeerEvent(v, DropPeerFlag); err != nil {
			// 		log.Error("peer event error", "err", err)
			// 		return false
			// 	}
			// }
			subscribe.Send(subscribe.DeletePeer, v)
			return true
		}
	}
	return false
}
