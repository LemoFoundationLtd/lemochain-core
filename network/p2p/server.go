// Package p2p implements the Lemochain p2p network protocols.
package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
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
	Name       string            // server's Name
	PrivateKey *ecdsa.PrivateKey // private key
	MaxPeerNum int               // max accept connection count
	Port       int               // listen port
}

// listenAddr fetch listen address
func (config *Config) listenAddr() string {
	return fmt.Sprintf(":%d", config.Port)
}

// Server manages all peer connections
type Server struct {
	Config // configuration

	running  int32        // flag for is server running
	listener net.Listener // TCP listener

	connectedNodes map[NodeID]IPeer // connect pool
	peersMux       sync.Mutex

	quitCh    chan struct{}
	addPeerCh chan IPeer
	delPeerCh chan IPeer

	newPeer func(net.Conn) IPeer

	discover    *DiscoverManager // node discovery
	dialManager IDialManager     // node dial
	wg          sync.WaitGroup
}

func NewServer(config Config, discover *DiscoverManager) *Server {
	srv := &Server{
		Config:   config,
		discover: discover,

		newPeer: NewPeer,

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
	// start listen
	if err := srv.startListening(); err != nil {
		panic("start server's listen failed")
	}
	if err := srv.discover.Start(); err != nil {
		log.Warnf("discover.start: %v", err)
	}

	// start dial task
	go func() {
		if err := srv.dialManager.Start(); err != nil {
			log.Errorf("start dialManager failed: %v", err)
		}
	}()
	// Run receive logic code
	go srv.run()
	return nil
}

// Stop
func (srv *Server) Stop() {
	if !atomic.CompareAndSwapInt32(&srv.running, 1, 0) {
		log.Debug("server not start, but exec stop command")
		return
	}
	// close listener
	if err := srv.listener.Close(); err != nil {
		log.Infof("stop listener failed: %v", err)
	}
	close(srv.quitCh)
	// close connected nodes
	for _, p := range srv.connectedNodes {
		p.Close()
	}
	// stop discover
	if err := srv.discover.Stop(); err != nil {
		log.Errorf("discover stop failed: %v", err)
	}
	// wait for stop
	srv.wg.Wait()
	log.Debug("server stop success")
}

// Run
func (srv *Server) run() {
	srv.wg.Add(1)
	defer srv.wg.Done()

	for {
		select {
		case p := <-srv.addPeerCh:
			// is already exist
			if _, ok := srv.connectedNodes[*p.RNodeID()]; ok {
				log.Debugf("Add peer event. But connection has already exist. nodeID: %s", p.RNodeID().String()[:16])
				p.Close()
				if err := srv.discover.SetConnectResult(p.RNodeID(), true); err != nil {
					log.Infof("SetConnectResult failed: %v", err)
				}
				break
			} else {
				log.Debugf("Add peer event. nodeID: %s", p.RNodeID().String()[:16])
			}
			// record
			srv.peersMux.Lock()
			srv.connectedNodes[*p.RNodeID()] = p
			srv.peersMux.Unlock()
			// Run peer
			go srv.runPeer(p)
			// notice
			subscribe.Send(subscribe.AddNewPeer, p)
		case p := <-srv.delPeerCh:
			log.Infof("Remove peer event. nodeID: %s", p.RNodeID().String()[:16])
			// remove
			srv.peersMux.Lock()
			delete(srv.connectedNodes, *p.RNodeID())
			srv.peersMux.Unlock()
			// notice
			subscribe.Send(subscribe.DeletePeer, p)
		case <-srv.quitCh:
			log.Debug("receive server stop signal")
			return
		}
	}
}

// startListening start tcp listening
func (srv *Server) startListening() error {
	if srv.Config.Port < 1024 {
		return errors.New("p2p listening port can't be less than 1024")
	}
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
			// server has stopped
			if atomic.LoadInt32(&srv.running) == 0 {
				log.Debug("listenLoop finished")
				return
			}
			// server not stopped, but has something else error
			log.Debug("TCP Accept error", "err", err)
			continue
		}
		go func() {
			if err := srv.HandleConn(fd, nil); err != nil {
				log.Errorf("HandleConn failed: %v", err)
			}
		}()
	}
}

// HandleConn handle net connection
func (srv *Server) HandleConn(fd net.Conn, nodeID *NodeID) error {
	if atomic.LoadInt32(&srv.running) == 0 {
		return ErrSrvHasStopped
	}
	// handshake
	peer := srv.newPeer(fd)
	err := peer.DoHandshake(srv.PrivateKey, nodeID)
	if err != nil {
		log.Debugf("peer handshake failed: %v", err)
		if err = fd.Close(); err != nil {
			log.Errorf("close connections failed", err)
		}
		if err = srv.discover.SetConnectResult(peer.RNodeID(), false); err != nil {
			log.Errorf("SetConnectResult failed: %v", err)
		}
		return err
	}
	// is itself
	if bytes.Compare(peer.RNodeID()[:], deputynode.GetSelfNodeID()) == 0 {
		if err = fd.Close(); err != nil {
			log.Errorf("close connections failed", err)
		}
		if err = srv.discover.SetConnectResult(peer.RNodeID(), false); err != nil {
			log.Errorf("SetConnectResult failed: %v", err)
		}
		return ErrConnectSelf
	}
	// output log
	if nodeID == nil {
		log.Debugf("First handshake as server ok, IP: %s. ID: %s ", peer.RAddress(), common.ToHex(peer.RNodeID()[:8]))
	} else {
		log.Debugf("First handshake as client ok: %s. id: %s", peer.RAddress(), common.ToHex(peer.RNodeID()[:8]))
	}
	// notice other goroutine
	srv.addPeerCh <- peer
	return nil
}

// runPeer Run peer
func (srv *Server) runPeer(p IPeer) {
	log.Debugf("peer(nodeID: %s) start running", common.ToHex(p.RNodeID()[:8]))
	if err := p.Run(); err != nil { // block this
		log.Debugf("runPeer error: %v", err)
	}

	// peer has stopped
	if atomic.LoadInt32(&srv.running) == 1 {
		srv.delPeerCh <- p
	}
	log.Debugf("peer Run finished: %s", common.ToHex(p.RNodeID()[:8]))
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
// format must be: "NodeID@ip:port"
func (srv *Server) Connect(node string) string {
	log.Infof("start add static peer: %s", node)
	srv.discover.AddNewList([]string{node})
	if res := srv.dialManager.runDialTask(node); res < 0 {
		return "connect node failed: %s" + node
	}
	return ""
}

// Disconnect disconnect a connection for api
// only support address
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
