// Package p2p implements the Lemochain p2p network protocols.
package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/subscribe"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	heartbeatInterval = 5 * time.Second
	frameReadTimeout  = 30 * time.Second
	frameWriteTimeout = 20 * time.Second
)

// Config holds Server options.
type Config struct {
	// private key
	PrivateKey *ecdsa.PrivateKey

	// max accept connection count
	MaxPeerNum int

	// max light peer limit
	MaxLightPeerNum int

	// server's Name
	Name string

	// black list
	NetRestrict *Netlist // revert

	// listen port
	Port int
}

func (config *Config) ListenAddr() string {
	return fmt.Sprintf(":%d", config.Port)
}

// Server manages all peer connections.
type Server struct {
	Config // server的一些基本配置

	lock    sync.Mutex // running 读写保护
	running bool       // 标识server是否在运行

	listener net.Listener // TCP监听

	connectedNodes map[NodeID]IPeer

	peersMux sync.Mutex

	quitCh    chan struct{}
	addPeerCh chan IPeer
	delPeerCh chan IPeer

	loopWG sync.WaitGroup

	newIPeer func(net.Conn) IPeer

	// for discover
	discover    *DiscoverManager
	dialManager *DialManager
}

func NewServer(config Config, discover *DiscoverManager) *Server {
	srv := &Server{
		Config:   config,
		discover: discover,

		newIPeer: newPeer,

		addPeerCh: make(chan IPeer, 1),
		delPeerCh: make(chan IPeer, 1),

		connectedNodes: make(map[NodeID]IPeer),
		quitCh:         make(chan struct{}),
	}
	srv.dialManager = NewDialManager(srv.HandleConn, srv.discover)
	return srv
}

// 启动服务器
func (srv *Server) Start() error {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if srv.running {
		return ErrAlreadyRunning
	}
	if srv.PrivateKey == nil {
		return ErrNilPrvKey
	}
	if err := srv.startListening(); err != nil {
		return err
	}

	// discover
	nodes := deputynode.Instance().GetLatestDeputies()
	srv.discover.SetDeputyNodes(nodes)
	srv.discover.Start()

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

	srv.listener.Close()

	for _, p := range srv.connectedNodes {
		p.Close()
	}

	// stop discover
	if err := srv.discover.Stop(); err != nil {
		log.Errorf("stop discover error: %v", err)
	} else {
		log.Debug("stop discover ok.")
	}

	close(srv.quitCh)
	srv.loopWG.Wait()
	log.Debug("server stop success")
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

	// peers := make(map[RNodeID]*Peer) // 记录所有的节点连接
	go srv.dialManager.Start() // 启动主动连接调度
	for {
		select {
		case p := <-srv.addPeerCh:
			log.Debugf("receive srv.addPeerCh. node id: %s", common.ToHex(p.RNodeID()[:8]))
			// 判断此节点是否在peers中
			if _, ok := srv.connectedNodes[*p.RNodeID()]; ok {
				log.Warnf("Connection has already exist. Remote node id: %s", common.ToHex(p.RNodeID()[:8]))
				p.Close()
				srv.discover.SetConnectResult(p.RNodeID(), false) // todo
				break
			}
			srv.peersMux.Lock()
			srv.connectedNodes[*p.RNodeID()] = p
			srv.peersMux.Unlock()
			go srv.runPeer(p)
			subscribe.Send(subscribe.AddNewPeer, p)
		case p := <-srv.delPeerCh:
			log.Debug("server: recv delete peer event")
			srv.peersMux.Lock()
			delete(srv.connectedNodes, *p.RNodeID())
			srv.peersMux.Unlock()
			subscribe.Send(subscribe.DeletePeer, p)
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
		return ErrSrvHasStopped
	}
	peer := srv.newIPeer(fd)
	err := peer.doHandshake(srv.PrivateKey, nodeID)
	if err != nil {
		fd.Close()

		// for discover
		srv.discover.SetConnectResult(peer.RNodeID(), false)

		return err
	}
	// p := peer.(*Peer)
	if bytes.Compare(peer.RNodeID()[:], deputynode.GetSelfNodeID()) == 0 {
		fd.Close()

		// for discover
		srv.discover.SetConnectResult(peer.RNodeID(), false)

		return ErrConnectSelf
	}
	if nodeID == nil {
		log.Debugf("Receive new connect, IP: %s. ID: %s ", peer.RAddress(), common.ToHex(peer.RNodeID()[:8]))
	} else {
		log.Debugf("Connect to server: %s. id: %s", peer.RAddress(), common.ToHex(peer.RNodeID()[:8]))
	}
	srv.addPeerCh <- peer
	log.Debug("transfer new peer to srv.addPeerCh")
	return nil
}

func (srv *Server) runPeer(p IPeer) {
	log.Debugf("start run peer. node id: %s", common.ToHex(p.RNodeID()[:8]))
	p.run() // block this
	if srv.running == true {
		srv.delPeerCh <- p
	}
	log.Debugf("peer: %s stopped", p.RAddress())
}

//go:generate gencodec -type PeerConnInfo -out gen_peer_conn_info_json.go

type PeerConnInfo struct {
	LocalAddr  string `json:"localAddress"`
	RemoteAddr string `json:"remoteAddress"`
	NodeID     string `json:"remoteNodeID"`
}

func (srv *Server) Connections() []PeerConnInfo {
	srv.peersMux.Lock()
	defer srv.peersMux.Unlock()
	result := make([]PeerConnInfo, 0, len(srv.connectedNodes))
	for _, v := range srv.connectedNodes {
		info := PeerConnInfo{v.LAddress(), v.RAddress(), v.RNodeID().String()}
		result = append(result, info)
	}
	return result
}

func (srv *Server) Connect(node string) string {
	log.Infof("start add static peer: %s", node)
	srv.discover.AddNewList([]string{node})
	if res := srv.dialManager.runDialTask(node); res < 0 {
		return "connect failed"
	}
	return ""
}

func (srv *Server) Disconnect(node string) bool {
	for id, v := range srv.connectedNodes {
		if strings.Compare(node, v.RAddress()) == 0 {
			v.Close()
			srv.peersMux.Lock()
			delete(srv.connectedNodes, id)
			srv.peersMux.Unlock()
			subscribe.Send(subscribe.DeletePeer, v)
			return true
		}
	}
	return false
}
