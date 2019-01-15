package node

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"os"
)

//go:generate gencodec -type ConfigFromFile -field-override ConfigFromFileMarshaling -out gen_config_from_file_json.go

type ConfigFromFile struct {
	ChainID   uint64 `json:"chainID"     gencodec:"required"`
	SleepTime uint64 `json:"sleepTime"   gencodec:"required"`
	Timeout   uint64 `json:"timeout"     gencodec:"required"`
	DbDns     string `json:"DbDns"     	 gencodec:"required"`
	DbDriver  string `json:"DbDriver"    gencodec:"required"`
}

// driver = "mysql"
// dns = root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
type ConfigFromFileMarshaling struct {
	ChainID   hexutil.Uint64
	SleepTime hexutil.Uint64
	Timeout   hexutil.Uint64
	DbDns     string // dns = root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
	DbDriver  string // driver = "mysql"
}

func readConfigFile(path string) (*ConfigFromFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New(err.Error() + "\r\n" + ConfigGuideUrl)
	}
	var config ConfigFromFile
	if err = json.NewDecoder(file).Decode(&config); err != nil {
		return nil, ErrConfig
	}
	return &config, nil
}

func (c *ConfigFromFile) Check() {
	if c.SleepTime >= c.Timeout {
		panic("config.json content error: sleepTime can't be larger than timeout")
	}
	if c.Timeout < 3000 {
		panic("timeout must be larger than 3000ms")
	}
	if c.ChainID > 65535 || c.ChainID < 1 {
		panic("config.json content error: chainID must be in [1, 65535]")
	}
}
