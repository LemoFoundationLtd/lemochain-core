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
	"sync/atomic"
	"time"
)

const (
	heartbeatInterval = 5 * time.Second
	frameReadTimeout  = 30 * time.Second
	frameWriteTimeout = 20 * time.Second
)

// Config holds Server options.
type Config struct {
	Name        string            // server's Name
	PrivateKey  *ecdsa.PrivateKey // private key
	MaxPeerNum  int               // max accept connection count
	Port        int               // listen port
	NetRestrict *Netlist          // black list. revert
}

func (config *Config) listenAddr() string {
	return fmt.Sprintf(":%d", config.Port)
}

// Server manages all peer connections
type Server struct {
	Config // configuration

	running  int32        // flag for is server running
	listener net.Listener // TCP listener

	connectedNodes map[NodeID]IPeer
	peersMux       sync.Mutex

	quitCh    chan struct{}
	addPeerCh chan IPeer
	delPeerCh chan IPeer

	newIPeer func(net.Conn) IPeer

	discover    *DiscoverManager // node discovery
	dialManager *DialManager     // node dial
	wg          sync.WaitGroup
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

// Start
func (srv *Server) Start() error {
	if !atomic.CompareAndSwapInt32(&srv.running, 0, 1) {
		return ErrAlreadyRunning
	}

	if srv.PrivateKey == nil {
		panic("node key is empty")
	}
	if err := srv.startListening(); err != nil {
		panic("start server's listen failed")
	}

	nodes := deputynode.Instance().GetLatestDeputies()
	srv.discover.SetDeputyNodes(nodes)
	if err := srv.discover.Start(); err != nil {
		log.Warnf("discover.start: %v", err)
	}

	go srv.run()
	return nil
}

// Stop
func (srv *Server) Stop() {
	if !atomic.CompareAndSwapInt32(&srv.running, 1, 0) {
		log.Debug("server not start, but exec stop command")
		return
	}

	srv.listener.Close()
	close(srv.quitCh)

	for _, p := range srv.connectedNodes {
		p.Close()
	}

	if err := srv.discover.Stop(); err != nil {
		log.Errorf("discover stop failed: %v", err)
	}

	srv.wg.Wait()
	log.Debug("server stop success")
}

// run
func (srv *Server) run() {
	srv.wg.Add(1)
	defer srv.wg.Done()

	// start dial task
	go srv.dialManager.Start()

	for {
		select {
		case p := <-srv.addPeerCh:
			log.Debugf("receive receive add peer event. nodeID: %s", common.ToHex(p.RNodeID()[:8]))
			if _, ok := srv.connectedNodes[*p.RNodeID()]; ok {
				log.Warnf("receive receive add peer event. But connection has already exist. nodeID: %s", common.ToHex(p.RNodeID()[:8]))
				p.Close()
				srv.discover.SetConnectResult(p.RNodeID(), false)
				break
			}
			srv.peersMux.Lock()
			srv.connectedNodes[*p.RNodeID()] = p
			srv.peersMux.Unlock()
			go srv.runPeer(p)
			subscribe.Send(subscribe.AddNewPeer, p)
		case p := <-srv.delPeerCh:
			log.Debug("receive delete peer event. nodeID: %s", common.ToHex(p.RNodeID()[:8]))
			srv.peersMux.Lock()
			delete(srv.connectedNodes, *p.RNodeID())
			srv.peersMux.Unlock()
			subscribe.Send(subscribe.DeletePeer, p)
		case <-srv.quitCh:
			log.Debug("receive server stop signal")
			return
		}
	}
}

// startListening start tcp listening
func (srv *Server) startListening() error {
	listener, err := net.Listen("tcp", srv.listenAddr())
	if err != nil {
		return err
	}
	srv.listener = listener
	go srv.listenLoop()
	return nil
}

// listenLoop accept net connection
func (srv *Server) listenLoop() {
	srv.wg.Add(1)
	defer srv.wg.Done()

	for {
		fd, err := srv.listener.Accept()
		if err != nil {
			if atomic.LoadInt32(&srv.running) == 0 {
				log.Debug("listenLoop finished")
				return
			}
			log.Debug("TCP Accept error", "err", err)
			continue
		}
		go srv.HandleConn(fd, nil)
	}
}

// HandleConn handle net connection
func (srv *Server) HandleConn(fd net.Conn, nodeID *NodeID) error {
	if atomic.LoadInt32(&srv.running) == 0 {
		return ErrSrvHasStopped
	}

	peer := srv.newIPeer(fd)
	err := peer.doHandshake(srv.PrivateKey, nodeID)
	if err != nil {
		log.Debugf("peer handshake failed: %v", err)
		fd.Close()
		srv.discover.SetConnectResult(peer.RNodeID(), false)
		return err
	}

	if bytes.Compare(peer.RNodeID()[:], deputynode.GetSelfNodeID()) == 0 {
		fd.Close()
		srv.discover.SetConnectResult(peer.RNodeID(), false)
		return ErrConnectSelf
	}

	if nodeID == nil {
		log.Debugf("Receive new connect, IP: %s. ID: %s ", peer.RAddress(), common.ToHex(peer.RNodeID()[:8]))
	} else {
		log.Debugf("Connect to server: %s. id: %s", peer.RAddress(), common.ToHex(peer.RNodeID()[:8]))
	}

	srv.addPeerCh <- peer
	return nil
}

// runPeer run peer
func (srv *Server) runPeer(p IPeer) {
	log.Debugf("peer start running: %s", common.ToHex(p.RNodeID()[:8]))
	p.run() // block this

	if atomic.LoadInt32(&srv.running) == 1 {
		srv.delPeerCh <- p
	}
	log.Debugf("peer run finished: %s", common.ToHex(p.RNodeID()[:8]))
}

//go:generate gencodec -type PeerConnInfo -out gen_peer_conn_info_json.go

type PeerConnInfo struct {
	LocalAddr  string `json:"localAddress"`
	RemoteAddr string `json:"remoteAddress"`
	NodeID     string `json:"remoteNodeID"`
}

// Connections get total connections for api
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

// Connect add new connection for api
// format: "NodeID@ip:port"
func (srv *Server) Connect(node string) string {
	log.Infof("start add static peer: %s", node)
	srv.discover.AddNewList([]string{node})
	if res := srv.dialManager.runDialTask(node); res < 0 {
		return "connect node failed: %s" + node
	}
	return ""
}

// Disconnect disconnect a connection for api
func (srv *Server) Disconnect(rAddr string) bool {
	for k, v := range srv.connectedNodes {
		if strings.Compare(rAddr, v.RAddress()) == 0 {
			v.Close()
			srv.peersMux.Lock()
			delete(srv.connectedNodes, k)
			srv.peersMux.Unlock()
			subscribe.Send(subscribe.DeletePeer, v)
			return true
		}
	}
	return false
}
