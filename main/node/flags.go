package node

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"github.com/inconshreveable/log15"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"strings"
)

func NewApp(usage string) *cli.App {
	app := &cli.App{
		Name:    filepath.Base(os.Args[0]),
		Version: params.Version,
		Usage:   usage,
	}
	return app
}

var (
	DataDirFlag = cli.StringFlag{
		Name:  common.DataDir,
		Usage: "Data directory for the databases",
		Value: DefaultDataDir(),
	}
	NetworkIdFlag = cli.Uint64Flag{
		Name:  common.NetworkID,
		Usage: "Network identifier",
		Value: DefaultConfig.NetworkId,
	}
	MaxPeersFlag = cli.IntFlag{
		Name:  common.MaxPeers,
		Usage: "Maximum number of network peers",
		Value: DefaultConfig.MaxPeers,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  common.ListenPort,
		Usage: "Network listening port",
		Value: DefaultConfig.Port,
	}
	ExtraDataFlag = cli.StringFlag{
		Name:  common.ExtraData,
		Usage: "Block extra data set by the miner (default = client version)",
	}
	NodeKeyFileFlag = cli.StringFlag{
		Name:  common.NodeKeyFile,
		Usage: "node's private key for sign and handshake",
	}
	MiningEnabledFlag = cli.BoolFlag{
		Name:  common.MiningEnabled,
		Usage: "Enable mining",
	}

	RPCEnabledFlag = cli.BoolFlag{
		Name:  common.RPCEnabled,
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  common.RPCListenAddr,
		Usage: "HTTP-RPC server listening interface",
		Value: DefaultHTTPHost,
	}
	RPCPortFlag = cli.IntFlag{
		Name:  common.RPCPort,
		Usage: "HTTP-RPC server listening port",
		Value: DefaultHTTPPort,
	}
	RPCCORSDomainFlag = cli.StringFlag{
		Name:  common.RPCCORSDomain,
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	RPCVirtualHostsFlag = cli.StringFlag{
		Name:  common.RPCVirtualHosts,
		Usage: "Comma separated list of virtual hostnames from which to accept requests(server enforced). Accepts '*' wildcard",
		Value: strings.Join(DefaultNodeConfig.HTTPVirtualHosts, ","),
	}
	IPCDisabledFlag = cli.BoolFlag{
		Name:  common.IPCDisabled,
		Usage: "Disable the IPC-RPC server",
	}
	IPCPathFlag = cli.StringFlag{
		Name:  common.IPCPath,
		Usage: "Filename for IPC socket/pipe within the datadir",
	}
	WSEnabledFlag = cli.BoolFlag{
		Name:  common.WSEnabled,
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  common.WSListenAddr,
		Usage: "WS-RPC server listening interface",
		Value: DefaultWSHost,
	}
	WSPortFlag = cli.IntFlag{
		Name:  common.WSPort,
		Usage: "WS-RPC server listening port",
		Value: DefaultWSPort,
	}
	WSAllowedOriginsFlag = cli.StringFlag{
		Name:  common.WSAllowedOrigins,
		Usage: "Origins from which to accept websockets request.",
	}
	DebugFlag = cli.BoolFlag{
		Name:  common.Debug,
		Usage: "Debug for runtime",
	}
	// from eth
	JSpathFlag = cli.StringFlag{
		Name:  common.JSpath,
		Usage: "JavaScript root path for `loadScript`",
		Value: ".",
	}
	LogLevelFlag = cli.IntFlag{
		Name:  common.LogLevel,
		Usage: "output log level",
		Value: 4,
	}
)

// setNodeKey 设置NodeKey私钥
func setNodeKey(flags flag.CmdFlags, cfg *p2p.Config) {
	if flags.IsSet(NodeKeyFileFlag.Name) {
		nodeKeyFile := flags.String(NodeKeyFileFlag.Name)
		if nodeKeyFile == "" {
			return
		}
		key, err := crypto.LoadECDSA(nodeKeyFile)
		if err != nil {
			Fatalf("Option %q: %v", NodeKeyFileFlag.Name, err)
		}
		cfg.PrivateKey = key
	}
}

// setListenAddress 设置监听端口
func setListenAddress(flags flag.CmdFlags, cfg *p2p.Config) {
	cfg.ListenAddr = fmt.Sprintf(":%d", flags.Int(ListenPortFlag.Name))
}

// setMaxPeers 设置最大连接数
func setMaxPeers(flags flag.CmdFlags, cfg *p2p.Config) {
	cfg.MaxPeerNum = flags.Int(MaxPeersFlag.Name)
}

// setP2PConfig 设置P2P
func setP2PConfig(flags flag.CmdFlags, cfg *p2p.Config) {
	setNodeKey(flags, cfg)
	setListenAddress(flags, cfg)
	setMaxPeers(flags, cfg)
}

// setHttp 设置http-rpc
func setHttp(flags flag.CmdFlags, cfg *NodeConfig) {
	if flags.Bool(RPCEnabledFlag.Name) && cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
		if flags.IsSet(RPCListenAddrFlag.Name) {
			cfg.HTTPHost = flags.String(RPCListenAddrFlag.Name)
		}
	}
	cfg.HTTPPort = flags.Int(RPCPortFlag.Name)
	cfg.HTTPCors = splitAndTrim(flags.String(RPCCORSDomainFlag.Name))
	cfg.HTTPVirtualHosts = splitAndTrim(flags.String(RPCVirtualHostsFlag.Name))
}

func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// setIPC 设置IPC
func setIPC(flags flag.CmdFlags, cfg *NodeConfig) {
	flags.CheckExclusive(IPCDisabledFlag, IPCPathFlag)
	if flags.Bool(IPCDisabledFlag.Name) {
		cfg.IPCPath = ""
	} else if flags.IsSet(IPCPathFlag.Name) {
		cfg.IPCPath = flags.String(IPCPathFlag.Name)
	} else {
		cfg.IPCPath = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") + ".ipc"
	}
}

// setWS 设置websocket
func setWS(flags flag.CmdFlags, cfg *NodeConfig) {
	if flags.Bool(WSEnabledFlag.Name) && cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
		if flags.IsSet(WSListenAddrFlag.Name) {
			cfg.WSHost = flags.String(WSListenAddrFlag.Name)
		}
		cfg.WSPort = flags.Int(WSPortFlag.Name)
		cfg.WSOrigins = splitAndTrim(flags.String(WSAllowedOriginsFlag.Name))
	}
}

func SetNodeConfig(flags flag.CmdFlags, cfg *NodeConfig) {
	cfg.DataDir = flags.String(DataDirFlag.Name)
	logLevel := flags.Int(LogLevelFlag.Name)
	logLevel -= 1
	if logLevel < 0 || logLevel > 4 {
		logLevel = 2
	}
	log.Setup(log15.Lvl(logLevel), true, true) // log init

	setP2PConfig(flags, &cfg.P2P)
	setIPC(flags, cfg)
	setHttp(flags, cfg)
	setWS(flags, cfg)
}

func SetLemoConfig(flags flag.CmdFlags, cfg *LemoConfig) {
	cfg.NetworkId = flags.Uint64(NetworkIdFlag.Name)
	if flags.IsSet(ExtraDataFlag.Name) {
		cfg.ExtraData = []byte(flags.String(ExtraDataFlag.Name))
	}
}
