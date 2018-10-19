package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/flock"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/LemoFoundationLtd/lemochain-go/network/rpc"
	"github.com/LemoFoundationLtd/lemochain-go/network/synchronise"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Node struct {
	config      *NodeConfig
	lemoConfig  *LemoConfig
	chainConfig *params.ChainConfig

	db       protocol.ChainDB
	accMan   *account.Manager
	txPool   *chain.TxPool
	chain    *chain.BlockChain
	pm       *synchronise.ProtocolManager
	miner    *miner.Miner
	gasPrice *big.Int

	lemoBase common.Address

	networkId uint64

	instanceDirLock flock.Releaser

	serverConfig *p2p.Config
	server       *p2p.Server

	rpcAPIs       []rpc.API
	inprocHandler *rpc.Server

	ipcEndpoint string
	ipcListener net.Listener
	ipcHandler  *rpc.Server

	httpEndpoint  string
	httpWhitelist []string
	httpListener  net.Listener
	httpHander    *rpc.Server

	wsEndpoint string
	wsListener net.Listener
	wsHandler  *rpc.Server

	genesisBlock *types.Block

	newTxsCh chan types.Transactions
	// newMinedBlockCh chan *types.Block
	recvBlockCh chan *types.Block

	stop chan struct{}
	lock sync.RWMutex
}

func New(lemoConf *LemoConfig, conf *NodeConfig, flags map[string]string) (*Node, error) {
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		absDataDir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = absDataDir
	}
	deputynode.SetSelfNodeKey(conf.NodeKey())
	log.Infof("NodeID: %s", common.ToHex(deputynode.GetSelfNodeID()))
	if strings.ContainsAny(conf.Name, `/\`) {
		return nil, errors.New(`Config.Name must not contain '/' or '\'`)
	}
	if strings.HasSuffix(conf.Name, ".ipc") {
		return nil, errors.New(`Config.Name must not end in ".ipc"`)
	}
	dir := filepath.Join(conf.DataDir, "chaindata")
	db, err := store.NewCacheChain(dir)
	if err != nil {
		return nil, fmt.Errorf("new db failed: %v", err)
	}
	path := filepath.Join(conf.DataDir, "config.json")
	genesisConfig, err := readConfigFile(path)
	if err != nil {
		log.Errorf("Can't read config file: %v", err)
		return nil, err
	}
	deputynode.Instance().Add(0, genesisConfig.DeputyNodes)
	log.Debugf("genesis deputy node length: %d. self node id: %s", len(genesisConfig.DeputyNodes), common.ToHex(deputynode.GetSelfNodeID()))
	_, err = db.GetBlockByHeight(0)
	if err == store.ErrNotExist {
		genesis := chain.DefaultGenesisBlock()
		chain.SetupGenesisBlock(db, genesis)
	} else if err == nil {
		// normal
	} else {
		return nil, err
	}
	engine := chain.NewDpovp(int64(genesisConfig.Timeout), int64(genesisConfig.SleepTime))
	recvBlockCh := make(chan *types.Block)
	blockChain, err := chain.NewBlockChain(genesisConfig.ChainID, engine, db, recvBlockCh, flags)
	if err != nil {
		return nil, err
	}
	engine.SetBlockChain(blockChain)
	newTxsCh := make(chan types.Transactions)
	accMan := blockChain.AccountManager()
	txPool := chain.NewTxPool(accMan, newTxsCh)
	newMinedBlockCh := make(chan *types.Block)
	pm := synchronise.NewProtocolManager(lemoConf.NetworkId, deputynode.GetSelfNodeID(), blockChain, txPool, newMinedBlockCh, newTxsCh)
	n := &Node{
		config:       conf,
		ipcEndpoint:  conf.IPCEndpoint(),
		httpEndpoint: conf.HTTPEndpoint(),
		wsEndpoint:   conf.WSEndpoint(),
		db:           db,
		accMan:       accMan,
		chain:        blockChain,
		txPool:       txPool,
		newTxsCh:     newTxsCh,
		pm:           pm,
	}
	n.genesisBlock, _ = db.GetBlockByHeight(0)
	n.config.P2P.PrivateKey = deputynode.GetSelfNodeKey()
	miner := miner.New(int64(genesisConfig.SleepTime), int64(genesisConfig.Timeout), blockChain, txPool, n.config.NodeKey(), newMinedBlockCh, recvBlockCh, engine)
	n.miner = miner
	d_n := deputynode.Instance().GetNodeByNodeID(blockChain.CurrentBlock().Height()+1, deputynode.GetSelfNodeID())
	if d_n != nil {
		miner.SetLemoBase(d_n.LemoBase)
	}
	deputynode.Instance().Init()
	return n, nil
}

func readConfigFile(path string) (*ChainConfigFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var config ChainConfigFile
	if err = json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	msg := ""
	if config.ChainID > 65535 {
		msg += "chainID must be in [1, 65535]\r\n"
	}
	// todo
	if msg != "" {
		panic("config.json error: " + msg)
	}
	return &config, nil
}

func (n *Node) DataDir() string {
	return n.config.DataDir
}

func (n *Node) Db() protocol.ChainDB {
	return n.db
}

func (n *Node) AccountManager() *account.Manager {
	return n.accMan
}

func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.server != nil {
		return errors.New("node already running")
	}
	if err := n.openDataDir(); err != nil {
		return err
	}
	// n.serverConfig = n.config.P2P
	server := &p2p.Server{Config: n.config.P2P}
	server.PeerEvent = n.pm.PeerEvent
	if err := server.Start(); err != nil {
		log.Errorf("start p2p server failed: %v", err)
		return err
	}
	n.pm.Start()
	n.server = server
	n.stop = make(chan struct{})

	if err := n.startRPC(); err != nil {
		log.Errorf("start rpc failed: %v", err)
		return err
	}
	return nil
}

func (n *Node) startRPC() error {
	apis := n.apis()

	if err := n.startInProc(apis); err != nil {
		return err
	}
	if err := n.startIPC(apis); err != nil {
		return err
	}
	if err := n.startHTTP(apis); err != nil {
		return err
	}
	// if err := n.startWS(apis); err != nil {
	// 	return err
	// }
	n.rpcAPIs = apis
	return nil
}

func (n *Node) startInProc(apis []rpc.API) error {
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		log.Infof("InProc registered. service: %v. namespace: %s", api.Service, api.Namespace)
	}
	n.inprocHandler = handler
	return nil
}

func (n *Node) stopInProc() {
	if n.inprocHandler != nil {
		n.inprocHandler.Stop()
		n.inprocHandler = nil
	}
}

func (n *Node) startIPC(apis []rpc.API) error {
	if n.ipcEndpoint == "" {
		return nil
	}
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		log.Infof("InProc registered. service: %v. namespace: %s", api.Service, api.Namespace)
	}
	var (
		listener net.Listener
		err      error
	)
	if listener, err = rpc.CreateIPCListener(n.ipcEndpoint); err != nil {
		return err
	}
	go func() {
		log.Infof("IPC endpoint opened. url: %v", n.ipcEndpoint)
		for {
			conn, err := listener.Accept()
			if err != nil {
				n.lock.RLock()
				closed := n.ipcListener == nil
				n.lock.RUnlock()
				if closed {
					return
				}
				log.Errorf("IPC accept failed: % v", err)
			}
			go handler.ServeCodec(rpc.NewJSONCodec(conn), rpc.OptionMethodInvocation|rpc.OptionSubscriptions)
		}
	}()
	n.ipcListener = listener
	n.ipcHandler = handler
	return nil
}

func (n *Node) stopIPC() {
	if n.ipcListener != nil {
		n.ipcListener.Close()
		n.ipcListener = nil
		log.Infof("IPC endpoint closed. endpoint: %v", n.ipcEndpoint)
	}
	if n.ipcHandler != nil {
		n.ipcHandler.Stop()
		n.ipcHandler = nil
	}
}

func (n *Node) startHTTP(apis []rpc.API) error {
	return nil
}

func (n *Node) stopHTTP() {

}

func (n *Node) startWS(apis []rpc.API) error {
	return nil
}

func (n *Node) stopWS() {

}

func (n *Node) stopRPC() {

}

// Stop
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	log.Debug("start stopping node...")
	if n.server == nil {
		log.Warn("p2p server not started")
	} else {
		n.server.Stop()
	}
	if err := n.accMan.Stop(true); err != nil {
		log.Errorf("stop account manager failed: %v", err)
		return err
	}
	log.Debug("stop account manager ok...")
	n.stopRPC()
	if n.instanceDirLock != nil {
		if err := n.instanceDirLock.Release(); err != nil {
			log.Errorf("Can't release datadir lock: %v", err)
		}
		n.instanceDirLock = nil
	}
	if err := n.stopChain(); err != nil {
		log.Errorf("stop chain failed: %v", err)
		return err
	}
	close(n.stop)
	log.Info("stop command execute success.")
	return nil
}

// stopChain stop chain module
func (n *Node) stopChain() error {
	n.chain.Stop()
	n.pm.Stop()
	// n.txPool.Stop()
	n.miner.Close()
	if err := n.db.Close(); err != nil {
		return err
	}
	log.Debug("stop chain ok...")
	return nil
}

// Wait wait for stop
func (n *Node) Wait() {
	n.lock.RLock()
	if n.server == nil {
		n.lock.RUnlock()
		return
	}
	stop := n.stop
	n.lock.RUnlock()
	<-stop
}

func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}
	return nil
}

func (n *Node) openDataDir() error {
	if n.config.DataDir == "" {
		return nil
	}
	if err := os.MkdirAll(n.config.DataDir, 0700); err != nil {
		return err
	}
	release, _, err := flock.New(filepath.Join(n.config.DataDir, "LOCK"))
	if err != nil {
		return err
	}
	n.instanceDirLock = release
	return nil
}

func (n *Node) StartMining() error {
	n.miner.Start()
	return nil
}

func (n *Node) Attach() (*rpc.Client, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.server == nil {
		return nil, errors.New("node not started")
	}
	return rpc.DialInProc(n.inprocHandler), nil
}

func (n *Node) RPCHandler() (*rpc.Server, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	if n.inprocHandler == nil {
		return nil, errors.New("node not started")
	}
	return n.inprocHandler, nil
}

func (n *Node) Server() *p2p.Server {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.server
}

func (n *Node) IPCEndpoint() string {
	return n.ipcEndpoint
}

func (n *Node) HTTPEndpoint() string {
	return n.httpEndpoint
}

func (n *Node) WSEndpoint() string {
	return n.wsEndpoint
}

func (n *Node) apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "chain",
			Version:   "1.0",
			Service:   NewChainAPI(n.chain),
		},
		{
			Namespace: "mine",
			Version:   "1.0",
			Service:   NewMineAPI(n.miner),
		},
		{
			Namespace: "account",
			Version:   "1.0",
			Service:   NewAccountAPI(n.accMan),
		},
		{
			Namespace: "net",
			Version:   "1.0",
			Service:   NewNetAPI(n.server),
		},
		{
			Namespace: "tx",
			Version:   "1.0",
			Service:   NewTxAPI(n.txPool),
		},
	}
}
