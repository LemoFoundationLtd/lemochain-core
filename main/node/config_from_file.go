package node

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"os"
)

//go:generate gencodec -type ConfigFromFile -field-override ConfigFromFileMarshaling -out gen_config_from_file_json.go

type ConfigFromFile struct {
	ChainID       uint64 `json:"chainID"        gencodec:"required"`
	SleepTime     uint64 `json:"sleepTime"`
	Timeout       uint64 `json:"timeout"`
	DbUri         string `json:"dbUri"          gencodec:"required"` // sample: root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
	DbDriver      string `json:"dbDriver"       gencodec:"required"` // sample: "mysql"
	SnapshotBlock uint64 `json:"snapshotBlock"`
	PeriodBlock   uint64 `json:"periodBlock"`
}

// driver = "mysql"
// dns = root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
type ConfigFromFileMarshaling struct {
	ChainID       hexutil.Uint64
	SleepTime     hexutil.Uint64
	Timeout       hexutil.Uint64
	SnapshotBlock hexutil.Uint64
	PeriodBlock   hexutil.Uint64
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
	if c.ChainID == 0 {
		c.ChainID = 1
	}
	if c.SleepTime == 0 {
		c.SleepTime = 3000
	}
	if c.Timeout == 0 {
		c.Timeout = 10000
	}
	if c.SnapshotBlock > 0 {
		params.SnapshotBlock = uint32(c.SnapshotBlock)
	}
	if c.PeriodBlock > 0 {
		params.PeriodBlock = uint32(c.PeriodBlock)
	}
}
