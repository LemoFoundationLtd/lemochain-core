package node

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func initTestConfig() *Config {
	return &Config{
		Name:             "TEST_CONFIG",
		Version:          "v1.0",
		DataDir:          "D://GoWorks//src//github.com//LemoFoundationLtd//lmstore",
		IPCPath:          "./ipc",
		HTTPPort:         3541,
		HTTPVirtualHosts: nil,
		WSPort:           3542,
	}
}

func TestConfig_HTTPEndpoint(t *testing.T) {
	config := initTestConfig()
	assert.Equal(t, "TEST_CONFIG", config.Name)
}
