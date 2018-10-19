package node

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
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
	RPCApiFlag = cli.StringFlag{
		Name:  common.RPCApi,
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
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
	WSApiFlag = cli.StringFlag{
		Name:  common.WSApi,
		Usage: "API's offered over the WS-RPC interface",
		Value: "",
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
func setNodeKey(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.GlobalIsSet(NodeKeyFileFlag.Name) {
		nodeKeyFile := ctx.GlobalString(NodeKeyFileFlag.Name)
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
func setListenAddress(ctx *cli.Context, cfg *p2p.Config) {
	cfg.ListenAddr = fmt.Sprintf(":%d", ctx.GlobalInt(ListenPortFlag.Name))
}

// setMaxPeers 设置最大连接数
func setMaxPeers(ctx *cli.Context, cfg *p2p.Config) {
	cfg.MaxPeerNum = ctx.Int(MaxPeersFlag.Name)
}

// setP2PConfig 设置P2P
func setP2PConfig(ctx *cli.Context, cfg *p2p.Config) {
	setNodeKey(ctx, cfg)
	setListenAddress(ctx, cfg)
	setMaxPeers(ctx, cfg)
}

// setHttp 设置http-rpc
func setHttp(ctx *cli.Context, cfg *NodeConfig) {
	if ctx.GlobalBool(RPCEnabledFlag.Name) && cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
		if ctx.GlobalIsSet(RPCListenAddrFlag.Name) {
			cfg.HTTPHost = ctx.GlobalString(RPCListenAddrFlag.Name)
		}
	}
	cfg.HTTPPort = ctx.GlobalInt(RPCPortFlag.Name)
	cfg.HTTPCors = splitAndTrim(ctx.GlobalString(RPCCORSDomainFlag.Name))
	cfg.HTTPModules = splitAndTrim(ctx.GlobalString(RPCApiFlag.Name))
	cfg.HTTPVirtualHosts = splitAndTrim(ctx.GlobalString(RPCVirtualHostsFlag.Name))
}

func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// setIPC 设置IPC
func setIPC(ctx *cli.Context, cfg *NodeConfig) {
	checkExclusive(ctx, IPCDisabledFlag, IPCPathFlag)
	if ctx.GlobalBool(IPCDisabledFlag.Name) {
		cfg.IPCPath = ""
	} else if ctx.GlobalIsSet(IPCPathFlag.Name) {
		cfg.IPCPath = ctx.GlobalString(IPCPathFlag.Name)
	} else {
		cfg.IPCPath = DefaultIPCPath()
	}
}

// setWS 设置websocket
func setWS(ctx *cli.Context, cfg *NodeConfig) {
	if ctx.GlobalBool(WSEnabledFlag.Name) && cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
		if ctx.GlobalIsSet(WSListenAddrFlag.Name) {
			cfg.WSHost = ctx.GlobalString(WSListenAddrFlag.Name)
		}
		cfg.WSPort = ctx.GlobalInt(WSPortFlag.Name)
		cfg.WSOrigins = splitAndTrim(ctx.GlobalString(WSAllowedOriginsFlag.Name))
		cfg.WSModules = splitAndTrim(ctx.GlobalString(WSApiFlag.Name))
	}
}

func checkExclusive(ctx *cli.Context, args ...interface{}) {
	set := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		flag, ok := args[i].(cli.Flag)
		if !ok {
			panic(fmt.Sprintf("invalid argument, not cli.Flag type: %T", args[i]))
		}
		name := flag.GetName()
		if i+1 < len(args) {
			switch option := args[i+1].(type) {
			case string:
				if ctx.String(name) == option {
					name += "=" + option
				}
				i++
			case cli.Flag:
			default:
				panic(fmt.Sprintf("invalid arguments, not cli.Flag or string extension: %T", args[i+1]))
			}
		}
		if ctx.IsSet(name) {
			set = append(set, "--"+name)
		}
	}
	if len(set) > 1 {
		Fatalf("Flags %v can't be used at the same time.", strings.Join(set, ", "))
	}
}

func SetNodeConfig(ctx *cli.Context, cfg *NodeConfig) {
	cfg.DataDir = ctx.GlobalString(DataDirFlag.Name)
	logLevel := ctx.GlobalInt(LogLevelFlag.Name)
	logLevel -= 1
	if logLevel < 0 || logLevel > 4 {
		logLevel = 2
	}
	log.Setup(log15.Lvl(logLevel), false, true) // log init

	setP2PConfig(ctx, &cfg.P2P)
	setHttp(ctx, cfg)
	setIPC(ctx, cfg)
	setWS(ctx, cfg)
}

func SetLemoConfig(ctx *cli.Context, cfg *LemoConfig) {
	cfg.NetworkId = ctx.GlobalUint64(NetworkIdFlag.Name)
	if ctx.GlobalIsSet(ExtraDataFlag.Name) {
		cfg.ExtraData = []byte(ctx.GlobalString(ExtraDataFlag.Name))
	}
}

func MigrateFlags(action func(ctx *cli.Context) error) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		return action(ctx)
	}
}
