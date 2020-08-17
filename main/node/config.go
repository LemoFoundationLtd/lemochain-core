package node

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain"
	"github.com/LemoFoundationLtd/lemochain-core/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DefaultHTTPPort = 8001 // Default TCP port for the HTTP RPC server
	DefaultWSPort   = 8002 // Default TCP port for the websocket RPC server
	DefaultP2PPort  = 60001

	datadirPrivateKey   = "nodekey"
	datadirStaticNodes  = "static-nodes.json"
	datadirTrustedNodes = "trusted-nodes.json"
)

var DefaultHTTPVirtualHosts = []string{"localhost"}

type Config struct {
	Name    string `toml:"-"`
	Version string `toml:"-"`

	DataDir string
	P2P     p2p.Config
	Chain   chain.Config
	Miner   miner.MineConfig

	IPCPath          string   `toml:",omitempty"`
	HTTPPort         int      `toml:",omitempty"`
	HTTPCors         []string `toml:",omitempty"`
	HTTPVirtualHosts []string `toml:",omitempty"`
	WSPort           int      `toml:",omitempty"`
	WSOrigins        []string `toml:",omitempty"`
}

// IPCEndpoint
func (c *Config) IPCEndpoint() string {
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + c.IPCPath
	}
	// Resolve names into the data directory full paths otherwise
	return filepath.Join(c.DataDir, c.IPCPath)
}

func (c *Config) HTTPEndpoint() string {
	if c.HTTPPort == 0 {
		return ""
	}
	return fmt.Sprintf("0.0.0.0:%d", c.HTTPPort)
}

func (c *Config) WSEndpoint() string {
	if c.WSPort == 0 {
		return ""
	}
	return fmt.Sprintf("0.0.0.0:%d", c.WSPort)
}

func (c *Config) NodeName() string {
	if c.Name == "" {
		return "Lemo"
	}
	return c.Name
}

func (c *Config) NodeKey() *ecdsa.PrivateKey {
	if c.P2P.PrivateKey != nil {
		return c.P2P.PrivateKey
	}

	keyFile := filepath.Join(c.DataDir, datadirPrivateKey)
	if key, err := crypto.LoadECDSA(keyFile); err == nil {
		return key
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		log.Critf("Failed to generate node key: %v", err)
	}
	instanceDir, _ := filepath.Abs(c.DataDir)
	if err := os.MkdirAll(instanceDir, 0700); err != nil {
		log.Errorf("Failed to persist node key: %v", err)
		return key
	}
	keyFile = filepath.Join(instanceDir, datadirPrivateKey)
	if err := crypto.SaveECDSA(keyFile, key); err != nil {
		log.Errorf("Failed to persist node key: %v", err)
	}
	return key
}

func parseNodes(path string) []string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Can't read file. file name: %s, err: %v", path, err)
	}
	text := string(content)
	lines := strings.Split(text, "\n")
	res := make([]string, 0, len(lines))
	for _, line := range lines {
		tmp := strings.TrimSpace(line)
		if tmp != "" {
			res = append(res, tmp)
		}
	}
	return res
}

func (c *Config) TrustedNodes() []string {
	return parseNodes(filepath.Join(c.DataDir, datadirTrustedNodes))
}

func (c *Config) StaticNodes() []string {
	return parseNodes(filepath.Join(c.DataDir, datadirStaticNodes))
}

func DefaultDataDir() string {
	return filepath.Dir(os.Args[0])
}
