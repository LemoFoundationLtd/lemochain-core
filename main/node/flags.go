package node

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
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
	MaxPeersFlag = cli.IntFlag{
		Name:  common.MaxPeers,
		Usage: "Maximum number of network peers",
		Value: DefaultP2pMaxPeerNum,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  common.ListenPort,
		Usage: "Network listening port",
		Value: DefaultP2PPort,
	}
	ExtraDataFlag = cli.StringFlag{
		Name:  common.ExtraData,
		Usage: "Block extra data set by the miner (default = client version)",
	}
	AutoMineFlag = cli.BoolFlag{
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
		Value: strings.Join(DefaultHTTPVirtualHosts, ","),
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
	JSpathFlag = cli.StringFlag{
		Name:  common.JSpath,
		Usage: "JavaScript root path for `loadScript`",
		Value: ".",
	}
	LogLevelFlag = cli.IntFlag{
		Name:  common.LogLevel,
		Usage: "Output log level",
		Value: 4,
	}
)

// setListenPort set listen port
func setListenPort(flags flag.CmdFlags, cfg *p2p.Config) {
	cfg.Port = flags.Int(ListenPortFlag.Name)
}

// setMaxPeers set max connection number
func setMaxPeers(flags flag.CmdFlags, cfg *p2p.Config) {
	cfg.MaxPeerNum = flags.Int(MaxPeersFlag.Name)
}

// setP2PConfig set p2p config
func setP2PConfig(flags flag.CmdFlags, cfg *p2p.Config) {
	setListenPort(flags, cfg)
	setMaxPeers(flags, cfg)
}

// setHttp set http-rpc
func setHttp(flags flag.CmdFlags, cfg *Config) {
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

// setIPC set ipc
func setIPC(flags flag.CmdFlags, cfg *Config) {
	flags.CheckExclusive(IPCDisabledFlag, IPCPathFlag)
	if flags.Bool(IPCDisabledFlag.Name) {
		cfg.IPCPath = ""
	} else if flags.IsSet(IPCPathFlag.Name) {
		cfg.IPCPath = flags.String(IPCPathFlag.Name)
	} else {
		cfg.IPCPath = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") + ".ipc"
	}
}

// setWS set web socket
func setWS(flags flag.CmdFlags, cfg *Config) {
	if flags.Bool(WSEnabledFlag.Name) && cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
		if flags.IsSet(WSListenAddrFlag.Name) {
			cfg.WSHost = flags.String(WSListenAddrFlag.Name)
		}
		cfg.WSPort = flags.Int(WSPortFlag.Name)
		cfg.WSOrigins = splitAndTrim(flags.String(WSAllowedOriginsFlag.Name))
	}
}

func getNodeConfig(flags flag.CmdFlags) *Config {
	cfg := new(Config)
	cfg.DataDir = flags.String(DataDirFlag.Name)
	if cfg.DataDir != "" {
		absDataDir, err := filepath.Abs(cfg.DataDir)
		if err == nil {
			cfg.DataDir = absDataDir
		}
	}
	setP2PConfig(flags, &cfg.P2P)
	setIPC(flags, cfg)
	setHttp(flags, cfg)
	setWS(flags, cfg)
	// set node version
	cfg.Version = params.Version
	return cfg
}
