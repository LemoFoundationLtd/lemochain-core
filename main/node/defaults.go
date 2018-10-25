package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"path/filepath"
	"runtime"
)

const (
	DefaultHTTPHost = "localhost" // Default host interface for the HTTP RPC server
	DefaultHTTPPort = 8001        // Default TCP port for the HTTP RPC server
	DefaultWSHost   = "localhost" // Default host interface for the websocket RPC server
	DefaultWSPort   = 8002        // Default TCP port for the websocket RPC server
)

var DefaultNodeConfig = NodeConfig{
	DataDir: DefaultDataDir(),

	HTTPPort:         DefaultHTTPPort,
	HTTPVirtualHosts: []string{"localhost"},
	WSPort:           DefaultWSPort,
	P2P: p2p.Config{
		Port:       60001,
		MaxPeerNum: 1000,
	},
}

func DefaultDataDir() string {
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "LemoChain")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "LemoChain")
		} else {
		}
		return filepath.Join(home, ".lemochain")
	}
	return ""
}
