package main

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

type genesisTestData struct {
	CaseName           string
	Err                error
	GenesisFileContent string
}

func init() {
	// log.Setup(log.LevelInfo, false, true)
}

func writeGenesisToFile(content, file string) {
	ioutil.WriteFile(file, []byte(content), 777)
}

func clearTmpFiles(configFile, datadir string) {
	os.Remove(configFile)
	os.RemoveAll(datadir)
}

// test no test_genesis.json file
func Test_setupGenesisBlock_no_file(t *testing.T) {
	assert.PanicsWithValue(t, ErrFileReadFailed, func() {
		setupGenesisBlock("test_genesis_not_exist.json", "lemo_test_genesis_not_exist")
	})
}

func customContent(fieldName, fieldContent string) string {
	type kv struct {
		K, V string
	}
	deputyNodesStr := `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": "65535",
			"rank": "0",
			"votes": "17"
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"nodeID": "0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
			"ip": "127.0.0.1",
			"port": "7002",
			"rank": "1",
			"votes": "16"
		}
	]`
	baseContents := []kv{
		{"founder", "\"Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG\""},
		{"extraData", "\"\""},
		{"parentHash", "\"0x0000000000000000000000000000000000000000000000000000000000000000\""},
		{"gasLimit", "105000000"},
		{"timestamp", "1539051657"},
		{"deputyNodes", deputyNodesStr},
	}
	lines := make([]string, 0, len(baseContents))
	for _, item := range baseContents {
		var line string
		if item.K == fieldName {
			if fieldContent == "" {
				continue
			}
			line = fmt.Sprintf("\"%s\": %s", item.K, fieldContent)
		} else {
			line = fmt.Sprintf("\"%s\": %s", item.K, item.V)
		}
		lines = append(lines, line)
	}
	return "{" + strings.Join(lines, ",") + "}"
}

func getTestCases() []genesisTestData {
	return []genesisTestData{
		{"empty_genesis_file", ErrInvalidGenesisFile, ""},
		{"invalid_json_format", ErrInvalidGenesisFile, customContent("deputyNodes", `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": "65535",
			"rank": "0",
			"votes": "17"
		},
	]`)},
		{"no_deputy", chain.ErrNoDeputyNodes, customContent("deputyNodes", "[]")},
		{"lack_necessary_field", ErrInvalidGenesisFile, customContent("gasLimit", "")},
		{"invalid_founder", ErrInvalidGenesisFile, customContent("founder", "Lemo84GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")}, // correct: Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG
		{"invalid_extraData", ErrInvalidGenesisFile, customContent("extraData", "0x123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")},
		{"invalid_timestamp", ErrInvalidGenesisFile, customContent("timestamp", "1539051657aaa")},
		{"invalid_deputy_minerAddress", ErrInvalidGenesisFile, customContent("deputyNodes", `[
		{
			"minerAddress": "Lemo84GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "432.0.0.1",
			"port": "7001",
			"rank": "0",
			"votes": "17"
		}
	]`)},
		{"invalid_deputy_nodeID", chain.ErrInvalidDeputyNodes, customContent("deputyNodes", `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e360073cd9a3c02",
			"ip": "127.0.0.1",
			"port": "7001",
			"rank": "0",
			"votes": "17"
		}
	]`)},
		{"invalid_deputy_ip", ErrInvalidGenesisFile, customContent("deputyNodes", `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "432.0.0.1",
			"port": "7001",
			"rank": "0",
			"votes": "17"
		}
	]`)},
		{"invalid_deputy_port", chain.ErrInvalidDeputyNodes, customContent("deputyNodes", `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": "65536",
			"rank": "0",
			"votes": "17"
		}
	]`)},
		{"invalid_deputy_rank", chain.ErrInvalidDeputyNodes, customContent("deputyNodes", `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": "7001",
			"rank": 65536,
			"votes": "17"
		}
	]`)},
		{"invalid_deputy_votes", ErrInvalidGenesisFile, customContent("deputyNodes", `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"ip": "127.0.0.1",
			"port": "7001",
			"rank": "0",
			"votes": 12
		}
	]`)},
	}
}

// test valid file content
func Test_setupGenesisBlock_valid(t *testing.T) {
	fileName := "test_correct_genesis.json"
	datadir := "lemo_data_test_correct"
	writeGenesisToFile(customContent("", ""), fileName)
	defer clearTmpFiles(fileName, datadir)

	hash := setupGenesisBlock(fileName, datadir)
	assert.Equal(t, common.HexToHash("0x2a25c706ef4b7fbe478d38d50a634f70d01257d535f74c5241c4a5cdaf791e90"), hash)
}

// test invalid file content
func Test_setupGenesisBlock_invalid(t *testing.T) {
	for _, tc := range getTestCases() {
		tc := tc // capture range variable
		t.Run(tc.CaseName, func(t *testing.T) {
			t.Parallel()

			fileName := "test_" + tc.CaseName + "_genesis.json"
			datadir := "lemo_data_test_" + tc.CaseName
			writeGenesisToFile(tc.GenesisFileContent, fileName)
			defer clearTmpFiles(fileName, datadir)

			assert.PanicsWithValue(t, tc.Err, func() {
				setupGenesisBlock(fileName, datadir)
			})
		})
	}
}
