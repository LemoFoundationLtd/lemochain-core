package node

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/network/p2p"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DefaultHTTPHost      = "localhost" // Default host interface for the HTTP RPC server
	DefaultHTTPPort      = 8001        // Default TCP port for the HTTP RPC server
	DefaultWSHost        = "localhost" // Default host interface for the websocket RPC server
	DefaultWSPort        = 8002        // Default TCP port for the websocket RPC server
	DefaultP2PPort       = 60001
	DefaultP2pMaxPeerNum = 1000

	datadirPrivateKey   = "nodekey"
	datadirStaticNodes  = "static-nodes.json"
	datadirTrustedNodes = "trusted-nodes.json"
	datadirNodeDatabase = "nodes"
)

var DefaultHTTPVirtualHosts = []string{"localhost"}

type Config struct {
	Name    string `toml:"-"`
	Version string `toml:"-"`

	ExtraData []byte `toml:",omitempty"`

	DataDir string
	P2P     p2p.Config

	IPCPath          string   `toml:",omitempty"`
	HTTPHost         string   `toml:",omitempty"`
	HTTPPort         int      `toml:",omitempty"`
	HTTPCors         []string `toml:",omitempty"`
	HTTPVirtualHosts []string `toml:",omitempty"`
	WSHost           string   `toml:",omitempty"`
	WSPort           int      `toml:",omitempty"`
	WSOrigins        []string `toml:",omitempty"`
	WSExposeAll      bool     `toml:",omitempty"`
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
	if c.HTTPHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}

func (c *Config) WSEndpoint() string {
	if c.WSHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.WSHost, c.WSPort)
}

func (c *Config) name() string {
	if c.Name == "" {
		progname := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		if progname == "" {
			panic("empty executable name, set Config.Name")
		}
		return progname
	}
	return c.Name
}

func (c *Config) NodeName() string {
	name := c.name()
	if name == "glemo" || name == "glemo-testnet" {
		name = "Glemo"
	}
	if c.Version != "" {
		name += "/v" + c.Version
	}
	name += "/" + runtime.GOOS + "-" + runtime.GOARCH
	name += "/" + runtime.Version()
	return name
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
		log.Errorf("can't read file. file name: %s, err: %v", path, err)
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
