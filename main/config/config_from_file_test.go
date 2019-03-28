package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadConfigFile(t *testing.T) {
	defer DelConfigFile("./")

	cfg := &ConfigFromFile{
		ChainID:         100,
		SleepTime:       10000,
		Timeout:         3000,
		DbUri:           "127.0.0.1:8080",
		DbDriver:        "mysql",
		TermDuration:    50,
		InterimDuration: 50,
		ConnectionLimit: 50,
	}

	WriteConfigFile("./", cfg)
	configFromFile, err := ReadConfigFile("./")
	assert.NoError(t, err)
	assert.NotNil(t, configFromFile)

	assert.Equal(t, configFromFile.ChainID, uint64(100))
	assert.Equal(t, configFromFile.SleepTime, uint64(10000))
	assert.Equal(t, configFromFile.Timeout, uint64(3000))
}

func TestReadConfigFile_Check1(t *testing.T) {
	configFromFile := &ConfigFromFile{
		ChainID:   uint64(99),
		SleepTime: uint64(50000),
		Timeout:   uint64(40000),
	}

	assert.PanicsWithValue(t, "config.json content error: sleepTime can't be larger than timeout", func() {
		configFromFile.Check()
	})
}

func TestReadConfigFile_Check2(t *testing.T) {
	configFromFile := &ConfigFromFile{
		ChainID:   uint64(65536),
		SleepTime: uint64(40000),
		Timeout:   uint64(50000),
	}

	assert.PanicsWithValue(t, "config.json content error: chainID must be in [1, 65535]", func() {
		configFromFile.Check()
	})
}

func TestReadConfigFile_Check3(t *testing.T) {
	configFromFile := &ConfigFromFile{
		ChainID:   uint64(0),
		SleepTime: uint64(40000),
		Timeout:   uint64(50000),
	}

	assert.PanicsWithValue(t, "config.json content error: chainID must be in [1, 65535]", func() {
		configFromFile.Check()
	})
}
