package config

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"os"
	"path/filepath"
)

const ConfigGuideUrl = "Please visit https://github.com/LemoFoundationLtd/lemochain-core#configuration-file for detail"

var (
	ErrConfig = errors.New(`file "config.json" format error.` + ConfigGuideUrl)
)

//go:generate gencodec -type ConfigFromFile -field-override ConfigFromFileMarshaling -out gen_config_from_file_json.go

type ConfigFromFile struct {
	ChainID         uint64 `json:"chainID"        gencodec:"required"`
	SleepTime       uint64 `json:"sleepTime"`
	Timeout         uint64 `json:"timeout"`
	DbUri           string `json:"dbUri"          gencodec:"required"` // sample: root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
	DbDriver        string `json:"dbDriver"       gencodec:"required"` // sample: "mysql"
	TermDuration    uint64 `json:"termDuration"`
	InterimDuration uint64 `json:"interimDuration"`
	ConnectionLimit uint64 `json:"connectionLimit"`
}

// driver = "mysql"
// dns = root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
type ConfigFromFileMarshaling struct {
	ChainID         hexutil.Uint64
	SleepTime       hexutil.Uint64
	Timeout         hexutil.Uint64
	TermDuration    hexutil.Uint64
	InterimDuration hexutil.Uint64
	ConnectionLimit hexutil.Uint64
}

func DelConfigFile(dir string) error {
	filePath := filepath.Join(dir, "config.json")
	return os.Remove(filePath)
}

func WriteConfigFile(dir string, cfg *ConfigFromFile) error {
	result, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, "config.json")
	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(filePath)
			if err != nil {
				return err
			} else {
				file.Close()
			}
		} else {
			return err
		}
	}

	file, err := os.OpenFile(filePath, os.O_APPEND, os.ModePerm)
	defer file.Close()

	if err != nil {
		return err
	}

	_, err = file.Write(result)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func ReadConfigFile(dir string) (*ConfigFromFile, error) {
	filePath := filepath.Join(dir, "config.json")
	file, err := os.Open(filePath)
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
	if c.TermDuration > 0 {
		params.TermDuration = uint32(c.TermDuration)
	}
	if c.InterimDuration > 0 {
		params.InterimDuration = uint32(c.InterimDuration)
	}
}
