package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"os"
	"path/filepath"
)

const (
	JsonFileName   = "config.json"
	ConfigGuideUrl = "Please visit https://github.com/LemoFoundationLtd/lemochain-core#configuration-file for more detail"
)

var (
	ErrConfigFormat      = fmt.Errorf(`file "%s" format error. %s`, JsonFileName, ConfigGuideUrl)
	ErrSleepTimeInConfig = fmt.Errorf(`file "%s" error: sleepTime can't be larger than timeout`, JsonFileName)
	ErrTimeoutInConfig   = fmt.Errorf(`file "%s" error: timeout must be larger than 3000ms`, JsonFileName)
	ErrChainIDInConfig   = fmt.Errorf(`file "%s" error: chainID must be in [1, 65535]`, JsonFileName)
)

//go:generate gencodec -type ConfigFromFile -field-override ConfigFromFileMarshaling -out gen_config_from_file_json.go

type ConfigFromFile struct {
	ChainID         uint64 `json:"chainID"        gencodec:"required"`
	DeputyCount     uint64 `json:"deputyCount"`
	SleepTime       uint64 `json:"sleepTime"`
	Timeout         uint64 `json:"timeout"`
	TermDuration    uint64 `json:"termDuration"`
	InterimDuration uint64 `json:"interimDuration"`
	ConnectionLimit uint64 `json:"connectionLimit"`
	AlarmUrl        string `json:"alarmUrl"`
}

type ConfigFromFileMarshaling struct {
	ChainID         hexutil.Uint64
	DeputyCount     hexutil.Uint64
	SleepTime       hexutil.Uint64
	Timeout         hexutil.Uint64
	TermDuration    hexutil.Uint64
	InterimDuration hexutil.Uint64
	ConnectionLimit hexutil.Uint64
}

func WriteConfigFile(dir string, cfg *ConfigFromFile) error {
	result, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, JsonFileName)
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
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(result)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func ReadConfigFile(dir string) (*ConfigFromFile, error) {
	filePath := filepath.Join(dir, JsonFileName)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.New(err.Error() + "\r\n" + ConfigGuideUrl)
	}
	defer file.Close()
	var config ConfigFromFile
	if err = json.NewDecoder(file).Decode(&config); err != nil {
		log.Errorf("Decode config fail %v", err)
		return nil, ErrConfigFormat
	}
	return &config, nil
}

func (c *ConfigFromFile) Check() {
	if c.ChainID > 65535 || c.ChainID < 1 {
		panic(ErrChainIDInConfig)
	}
	if c.DeputyCount == 0 {
		c.DeputyCount = 17
	}
	if c.SleepTime == 0 {
		c.SleepTime = 3000
	}
	if c.Timeout == 0 {
		c.Timeout = 30000
	}
	if c.Timeout < 3000 {
		panic(ErrTimeoutInConfig)
	}
	if c.SleepTime >= c.Timeout {
		panic(ErrSleepTimeInConfig)
	}
	if c.TermDuration > 0 {
		params.TermDuration = uint32(c.TermDuration)
	}
	if c.InterimDuration > 0 {
		params.InterimDuration = uint32(c.InterimDuration)
	}
	if c.ConnectionLimit == 0 {
		c.ConnectionLimit = 50
	}
	if len(c.AlarmUrl) > 0 { // if configured, then start metrics and alarm system client
		metrics.AlarmUrl = c.AlarmUrl
	}
}
