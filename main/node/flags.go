package node

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
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
		Name:  "datadir",
		Usage: "Data directory for the databases",
		Value: DefaultDataDir(),
	}
	NetworkIdFlag = cli.Uint64Flag{
		Name:  "networkid",
		Usage: "Network identifier",
		Value: DefaultConfig.NetworkId,
	}
	MaxPeersFlag = cli.IntFlag{
		Name:  "maxpeers",
		Usage: "Maximum number of network peers",
		Value: DefaultConfig.MaxPeers,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: DefaultConfig.Port,
	}
	ExtraDataFlag = cli.StringFlag{
		Name:  "extradata",
		Usage: "Block extra data set by the miner (default = client version)",
	}
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "node's private key for sign and handshake",
	}
	MiningEnabledFlag = cli.BoolFlag{
		Name:  "mine",
		Usage: "Enable mining",
	}

	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: DefaultHTTPHost,
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: DefaultHTTPPort,
	}
	RPCCORSDomainFlag = cli.StringFlag{
		Name:  "rpccorsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	RPCVirtualHostsFlag = cli.StringFlag{
		Name:  "rpcvhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests(server enforced). Accepts '*' wildcard",
		Value: strings.Join(DefaultNodeConfig.HTTPVirtualHosts, ","),
	}
	RPCApiFlag = cli.StringFlag{
		Name:  "rpcapi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
	}
	IPCDisabledFlag = cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCPathFlag = cli.StringFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir",
	}
	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  "wsaddr",
		Usage: "WS-RPC server listening interface",
		Value: DefaultWSHost,
	}
	WSPortFlag = cli.IntFlag{
		Name:  "wsport",
		Usage: "WS-RPC server listening port",
		Value: DefaultWSPort,
	}
	WSApiFlag = cli.StringFlag{
		Name:  "wsapi",
		Usage: "API's offered over the WS-RPC interface",
		Value: "",
	}
	WSAllowedOriginsFlag = cli.StringFlag{
		Name:  "wsorigins",
		Usage: "Origins from which to accept websockets request.",
	}

	// from eth
	JSpathFlag = cli.StringFlag{
		Name:  "jspath",
		Usage: "JavaScript root path for `loadScript`",
		Value: ".",
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
	switch {
	case ctx.GlobalBool(IPCDisabledFlag.Name):
		cfg.IPCPath = ""
	case ctx.GlobalIsSet(IPCPathFlag.Name):
		cfg.IPCPath = ctx.GlobalString(IPCPathFlag.Name)
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
