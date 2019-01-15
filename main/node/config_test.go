package node

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func initTestConfig() *Config {
	return &Config{
		Name:             "TEST_CONFIG",
		Version:          "v1.0",
		ExtraData:        nil,
		DataDir:          "D://GoWorks//src//github.com//LemoFoundationLtd//lmstore",
		IPCPath:          "./ipc",
		HTTPHost:         "www.baidu.com",
		HTTPPort:         3541,
		HTTPVirtualHosts: nil,
		WSHost:           "www.qq.com",
		WSPort:           3542,
		WSExposeAll:      true,
	}
}

func TestConfig_HTTPEndpoint(t *testing.T) {
	config := initTestConfig()
	assert.Equal(t, "TEST_CONFIG", config.Name)
}
