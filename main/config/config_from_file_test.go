package config

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

const testDataPath = "../../testdata"

func delConfigFile(dir string) {
	filePath := filepath.Join(dir, JsonFileName)
	os.Remove(filePath)
}

func getTestConfig() *ConfigFromFile {
	return &ConfigFromFile{
		ChainID:         100,
		DeputyCount:     5,
		SleepTime:       3000,
		Timeout:         30000,
		TermDuration:    50,
		InterimDuration: 50,
		ConnectionLimit: 50,
		AlarmUrl:        "https://lemochain.com",
	}
}

func TestReadConfigFile(t *testing.T) {
	defer delConfigFile(testDataPath)

	cfg := getTestConfig()

	err := WriteConfigFile(testDataPath, cfg)
	assert.NoError(t, err)
	configFromFile, err := ReadConfigFile(testDataPath)
	assert.NoError(t, err)
	assert.NotNil(t, configFromFile)

	assert.Equal(t, cfg.ChainID, configFromFile.ChainID)
	assert.Equal(t, cfg.DeputyCount, configFromFile.DeputyCount)
	assert.Equal(t, cfg.SleepTime, configFromFile.SleepTime)
	assert.Equal(t, cfg.Timeout, configFromFile.Timeout)
	assert.Equal(t, cfg.TermDuration, configFromFile.TermDuration)
	assert.Equal(t, cfg.InterimDuration, configFromFile.InterimDuration)
	assert.Equal(t, cfg.ConnectionLimit, configFromFile.ConnectionLimit)
	assert.Equal(t, cfg.AlarmUrl, configFromFile.AlarmUrl)
}

func TestReadConfigFile_Check_Error(t *testing.T) {
	cfg := getTestConfig()
	cfg.ChainID = 0
	assert.PanicsWithValue(t, ErrChainIDInConfig, func() {
		cfg.Check()
	})

	cfg = getTestConfig()
	cfg.ChainID = 65536
	assert.PanicsWithValue(t, ErrChainIDInConfig, func() {
		cfg.Check()
	})

	cfg = getTestConfig()
	cfg.SleepTime = 500
	cfg.Timeout = 1000
	assert.PanicsWithValue(t, ErrTimeoutInConfig, func() {
		cfg.Check()
	})

	cfg = getTestConfig()
	cfg.SleepTime = 50000
	assert.PanicsWithValue(t, ErrSleepTimeInConfig, func() {
		cfg.Check()
	})
}

func TestReadConfigFile_Check_DefaultValue(t *testing.T) {
	cfg := getTestConfig()
	cfg.DeputyCount = 0
	cfg.Check()
	assert.Equal(t, uint64(17), cfg.DeputyCount)

	cfg = getTestConfig()
	cfg.SleepTime = 0
	cfg.Check()
	assert.Equal(t, uint64(3000), cfg.SleepTime)

	cfg = getTestConfig()
	cfg.Timeout = 0
	cfg.Check()
	assert.Equal(t, uint64(30000), cfg.Timeout)

	assert.Equal(t, uint32(cfg.TermDuration), params.TermDuration)
	assert.Equal(t, uint32(cfg.InterimDuration), params.InterimDuration)

	cfg = getTestConfig()
	cfg.ConnectionLimit = 0
	cfg.Check()
	assert.Equal(t, uint64(50), cfg.ConnectionLimit)

	assert.Equal(t, cfg.AlarmUrl, metrics.AlarmUrl)
}
