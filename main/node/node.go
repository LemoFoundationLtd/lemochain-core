package node

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/common/flock"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/LemoFoundationLtd/lemochain-go/network/rpc"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

const ConfigGuideUrl = "Please visit https://github.com/LemoFoundationLtd/lemochain-go#configuration-file for detail"

var (
	ErrConfig = errors.New(`file "config.json" format error.` + ConfigGuideUrl)
)

type Node struct {
	config  *Config
	chainID uint16

	db       protocol.ChainDB
	accMan   *account.Manager
	txPool   *chain.TxPool
	chain    *chain.BlockChain
	pm       *network.ProtocolManager
	miner    *miner.Miner
	gasPrice *big.Int

	minerAddress common.Address

	instanceDirLock flock.Releaser

	server *p2p.Server

	rpcAPIs       []rpc.API
	inprocHandler *rpc.Server

	ipcEndpoint string
	ipcListener net.Listener
	ipcHandler  *rpc.Server

	httpEndpoint  string
	httpWhitelist []string
	httpListener  net.Listener
	httpHandler   *rpc.Server

	wsEndpoint string
	wsListener net.Listener
	wsHandler  *rpc.Server

	genesisBlock *types.Block

	// newTxsCh chan types.Transactions
	// newMinedBlockCh chan *types.Block
	recvBlockCh chan *types.Block

	stop     chan struct{}
	stopping uint32
	lock     sync.RWMutex
}

func initConfig(flags flag.CmdFlags) (*Config, *ConfigFromFile, *miner.MineConfig) {
	cfg := getNodeConfig(flags)
	deputynode.SetSelfNodeKey(cfg.NodeKey())
	cfg.P2P.PrivateKey = deputynode.GetSelfNodeKey()
	log.Infof("Local nodeID: %s", common.ToHex(deputynode.GetSelfNodeID()))

	filePath := filepath.Join(cfg.DataDir, "config.json")
	configFromFile, err := readConfigFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("read config.json error: %v", err))
	}
	configFromFile.Check()

	mineCfg := &miner.MineConfig{
		SleepTime: int64(configFromFile.SleepTime),
		Timeout:   int64(configFromFile.Timeout),
	}
	return cfg, configFromFile, mineCfg
}

func initDb(dataDir string, driver string, dns string) protocol.ChainDB {
	dir := filepath.Join(dataDir, "chaindata")
	return store.NewChainDataBase(dir, driver, dns)
}

func getGenesis(db protocol.ChainDB) *types.Block {
	block, err := db.GetBlockByHeight(0)
	if err == store.ErrNotExist {
		genesis := chain.DefaultGenesisBlock()
		if _, err = chain.SetupGenesisBlock(db, genesis); err != nil {
			panic("SetupGenesisBlock Failed")
		}
		block, _ = db.GetBlockByHeight(0)
	} else if err == nil {
		// normal
	} else {
		panic(fmt.Sprintf("can't get genesis block. err: %v", err))
	}
	return block
}

func (n *Node) setMinerAddress() {
	nextHeight := n.chain.CurrentBlock().Height() + 1
	deputyNode := deputynode.Instance().GetDeputyByNodeID(nextHeight, deputynode.GetSelfNodeID())
	if deputyNode != nil {
		n.miner.SetMinerAddress(deputyNode.MinerAddress)
	}
}

// initDeputyNodes init deputy nodes information
func initDeputyNodes(db protocol.ChainDB) {
	block, _ := db.GetBlockByHeight(0)
	var err error
	for block != nil {
		if block.DeputyNodes == nil || len(block.DeputyNodes) == 0 {
			log.Warnf("initDeputyNodes: can't get deputy nodes in snapshot block")
			return
		}
		deputynode.Instance().Add(block.Height(), block.DeputyNodes)
		block, err = db.GetBlockByHeight(block.Height() + deputynode.SnapshotBlockInterval)
		if err == store.ErrNotExist {
			break
		} else if err == nil {
			// normal
		} else {
			panic(fmt.Sprintf("block get error: %v", err))
		}
	}
}

func New(flags flag.CmdFlags) *Node {
	cfg, configFromFile, mineCfg := initConfig(flags)
	db := initDb(cfg.DataDir, configFromFile.DbDriver, configFromFile.DbUri)
	// read genesis block
	genesisBlock := getGenesis(db)
	if genesisBlock == nil {
		panic("can't get genesis block")
	}
	// read all deputy nodes from snapshot block
	initDeputyNodes(db)
	// new dpovp consensus engine
	engine := chain.NewDpovp(int64(configFromFile.Timeout), db)
	blockChain, err := chain.NewBlockChain(uint16(configFromFile.ChainID), engine, db, flags)
	if err != nil {
		panic("new block chain failed!!!")
	}
	// account manager
	accMan := blockChain.AccountManager()
	// tx pool
	txPool := chain.NewTxPool(uint16(configFromFile.ChainID))
	// discover manager
	discover := p2p.NewDiscoverManager(cfg.DataDir)
	selfNodeID := p2p.NodeID{}
	copy(selfNodeID[:], deputynode.GetSelfNodeID())
	// protocol manager
	pm := network.NewProtocolManager(uint16(configFromFile.ChainID), selfNodeID, blockChain, txPool, discover, params.VersionUint())
	// p2p server
	server := p2p.NewServer(cfg.P2P, discover)
	n := &Node{
		config:       cfg,
		chainID:      uint16(configFromFile.ChainID),
		ipcEndpoint:  cfg.IPCEndpoint(),
		httpEndpoint: cfg.HTTPEndpoint(),
		wsEndpoint:   cfg.WSEndpoint(),
		db:           db,
		accMan:       accMan,
		chain:        blockChain,
		txPool:       txPool,
		miner:        miner.New(mineCfg, blockChain, txPool, engine),
		pm:           pm,
		server:       server,
		genesisBlock: genesisBlock,
	}
	// set Founder for next block
	n.setMinerAddress()
	return n
}

func (n *Node) DataDir() string {
	return n.config.DataDir
}

func (n *Node) Db() protocol.ChainDB {
	return n.db
}

func (n *Node) ChainID() uint16 {
	return n.chainID
}

func (n *Node) AccountManager() *account.Manager {
	return n.accMan
}

func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	// if n.server != nil {
	// 	return ErrAlreadyRunning
	// }
	if err := n.openDataDir(); err != nil {
		log.Errorf("%v", err)
		return ErrOpenFileFailed
	}
	if err := n.server.Start(); err != nil {
		log.Errorf("%v", err)
		return ErrServerStartFailed
	}
	n.pm.Start()
	n.stop = make(chan struct{})

	if err := n.startRPC(); err != nil {
		log.Errorf("%v", err)
		return ErrRpcStartFailed
	}
	return nil
}

func (n *Node) startRPC() error {
	apis := n.apis()

	if err := n.startInProc(apis); err != nil {
		return err
	}
	if err := n.startIPC(apis); err != nil {
		n.stopInProc()
		return err
	}
	if err := n.startHTTP(n.httpEndpoint, apis, n.config.HTTPCors, n.config.HTTPVirtualHosts); err != nil {
		n.stopIPC()
		n.stopInProc()
		return err
	}
	if err := n.startWS(apis); err != nil {
		n.stopHTTP()
		n.stopIPC()
		n.stopInProc()
		return err
	}
	n.rpcAPIs = apis
	return nil
}

func (n *Node) startInProc(apis []rpc.API) error {
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		// log.Debug("InProc registered", "namespace", api.Namespace)
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
	if n.config.IPCPath == "" || n.ipcEndpoint == "" {
		return nil
	}
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		// log.Debug("IPC registered", "namespace", api.Namespace)
	}
	var (
		listener net.Listener
		err      error
	)
	if listener, err = rpc.CreateIPCListener(n.ipcEndpoint); err != nil {
		log.Error("IPC listen failed.")
		return err
	}
	go func() {
		log.Info("IPC endpoint opened", "url", n.ipcEndpoint)
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
			go handler.ServeCodec(rpc.NewJSONCodec(conn))
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
		log.Info("IPC endpoint closed", "endpoint", n.ipcEndpoint)
	}
	if n.ipcHandler != nil {
		n.ipcHandler.Stop()
		n.ipcHandler = nil
	}
}

func (n *Node) startHTTP(endpoint string, apis []rpc.API, cors []string, vhosts []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if api.Public {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
			// log.Debug("HTTP registered", "namespace", api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpc.NewHTTPServer(cors, vhosts, handler).Serve(listener)
	log.Info("HTTP endpoint opened", "url", fmt.Sprintf("h ttp://%s", endpoint), "cors", strings.Join(cors, ","), "vhosts", strings.Join(vhosts, ","))
	// All listeners booted successfully
	n.httpEndpoint = endpoint
	n.httpListener = listener
	n.httpHandler = handler

	return nil
}

func (n *Node) stopHTTP() {
	if n.httpListener != nil {
		n.httpListener.Close()
		n.httpListener = nil

		log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%s", n.httpEndpoint))
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

func (n *Node) startWS(apis []rpc.API) error {
	return nil
}

func (n *Node) stopWS() {

}

func (n *Node) stopRPC() {
	// Terminate the API, services and the p2p server.
	n.stopWS()
	n.stopHTTP()
	n.stopIPC()
	n.stopInProc()
	n.rpcAPIs = nil
}

// Stop
func (n *Node) Stop() error {
	if !atomic.CompareAndSwapUint32(&n.stopping, 0, 1) {
		return nil
	}
	n.lock.Lock()
	defer n.lock.Unlock()
	log.Debug("start stopping node...")
	n.stopRPC()
	if n.server == nil {
		log.Warn("p2p server not started")
	} else {
		n.server.Stop()
		n.server = nil
	}
	if err := n.accMan.Stop(true); err != nil {
		log.Errorf("stop account manager failed: %v", err)
		return err
	}
	log.Debug("stop account manager ok...")
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

// func (n *Node) Restart() error {
// 	if err := n.Stop(); err != nil {
// 		return err
// 	}
// 	if err := n.Start(); err != nil {
// 		return err
// 	}
// 	return nil
// }

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

func (n *Node) apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "chain",
			Version:   "1.0",
			Service:   NewPublicChainAPI(n.chain),
			Public:    true,
		},
		{
			Namespace: "mine",
			Version:   "1.0",
			Service:   NewPublicMineAPI(n.miner),
			Public:    true,
		},
		{
			Namespace: "mine",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(n.miner),
			Public:    false,
		},
		{
			Namespace: "account",
			Version:   "1.0",
			Service:   NewPublicAccountAPI(n.accMan),
			Public:    true,
		},
		{
			Namespace: "account",
			Version:   "1.0",
			Service:   NewPrivateAccountAPI(n.accMan),
			Public:    false,
		},
		{
			Namespace: "net",
			Version:   "1.0",
			Service:   NewPublicNetAPI(n),
			Public:    true,
		},
		{
			Namespace: "net",
			Version:   "1.0",
			Service:   NewPrivateNetAPI(n),
			Public:    false,
		},
		{
			Namespace: "tx",
			Version:   "1.0",
			Service:   NewPublicTxAPI(n),
			Public:    true,
		},
	}
}
