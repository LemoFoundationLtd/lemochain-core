package main

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func writeContentToFile(content, file string) {
	ioutil.WriteFile(file, []byte(content), 744)
}

func deleteTmpFile(file string) {
	os.Remove(file)
}

func deleteDir(dir string) {
	os.RemoveAll(dir)
}

func init() {
	// log.Setup(log.LevelInfo, false, true)
}

// test correct condition max port: 65535
func Test_setupGenesisBlock_correct(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 65535,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash("0x8aec56fbe87e6a9faabb62acefdcecc609f17fb08538a7fdf0751ac1a1c7cae9"), nil}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	hash, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	assert.Equal(t, test.Hash, hash)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test no test_genesis.json file
func Test_setupGenesisBlock_no_file(t *testing.T) {
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, ErrFileReadFailed, err)
	deleteDir(datadir)
}

// test empty file content
func Test_setupGenesisBlock_empty(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		``, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test error format
func Test_setupGenesisBlock_error_format(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 65535,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	],
}`, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test not set deputy nodes
func Test_setupGenesisBlock_no_deputy(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[]
}`, common.HexToHash(""), ErrEmptyDeputyNodes}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	assert.PanicsWithValue(t, "default deputy nodes can't be empty", func() {
		setupGenesisBlock(fileName, datadir)
	})
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test lack required field: {"gasLimit": 105000000}
func Test_setupGenesisBlock_lack_required(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test error field: {"timestamp": 1539051657aaa}
func Test_setupGenesisBlock_error_field(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657aaa,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test error minerAddress: {"minerAddress": "Lemo84GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG"} ==> correct: Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG
func Test_setupGenesisBlock_error_minerAddress(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo84GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test error ip: {"ip": "432.0.0.1"}
func Test_setupGenesisBlock_error_ip(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "432.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test port==65536
func Test_setupGenesisBlock_error_port(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 65536,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), nil}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	assert.PanicsWithValue(t, "genesis deputy nodes check error", func() {
		setupGenesisBlock(fileName, datadir)
	})
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test error NodeID
func Test_setupGenesisBlock_error_NodeID(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e360073cd9a3c02",
			"ip": "127.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 17
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), nil}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	assert.PanicsWithValue(t, "genesis deputy nodes check error", func() {
		setupGenesisBlock(fileName, datadir)
	})
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test error votes
func Test_setupGenesisBlock_error_votes(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 1234567890123456
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	_, err := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, test.Output, err)
	deleteTmpFile(fileName)
	deleteDir(datadir)
}

// test error extraData: len(extraData)==300
func Test_setupGenesisBlock_error_extraData(t *testing.T) {
	test := struct {
		Content string
		Hash    common.Hash
		Output  error
	}{
		`{
  "minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
  "extraData": "0x123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": 105000000,
  "timestamp": 1539051657,
  "deputyNodes":[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": 7001,
			"rank": 0,
			"votes": 123456789
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": 7002,
			"rank": 1,
			"votes": 16
		}
	]
}`, common.HexToHash(""), ErrInvalidGenesisFile}
	fileName := "test_genesis.json"
	datadir := "lemo-test"
	writeContentToFile(test.Content, fileName)
	assert.PanicsWithValue(t, "genesis block's extraData length larger than 256", func() {
		setupGenesisBlock(fileName, datadir)
	})
	deleteTmpFile(fileName)
	deleteDir(datadir)
}
