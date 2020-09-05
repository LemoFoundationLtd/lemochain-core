package main

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
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

func writeGenesisToFile(dataDir, content string) {
	configPath := filepath.Join(dataDir, genesisConfigName)
	err := os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(configPath, []byte(content), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func clearTmpFiles(dataDir string) {
	os.RemoveAll(dataDir)
}

// test no test_genesis.json file
func Test_setupGenesisBlock_no_file(t *testing.T) {
	assert.PanicsWithValue(t, ErrFileReadFailed, func() {
		setupGenesisBlock("lemo_test_genesis_not_exist")
	})
}

func editDefaultTestContent(fieldName, fieldContent string) string {
	type kv struct {
		K, V string
	}
	deputyNodesStr := `[
		{
			"minerAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"incomeAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"host": "127.0.0.1",
			"port": "65534",
			"introduction": "genesis"
		},
		{
			"minerAddress": "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
			"incomeAddress": "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
			"nodeID": "0x5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
			"host": "127.0.0.2",
			"port": "65535",
			"introduction": "genesis"
		}
	]`
	baseContents := []kv{
		{"founder", "\"Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG\""},
		{"extraData", "\"\""},
		{"gasLimit", "105000000"},
		{"timestamp", "1539051657"},
		{"deputyNodesInfo", deputyNodesStr},
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
		{"invalid_founder", ErrInvalidGenesisFile, editDefaultTestContent("founder", "\"Lemo84GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG\"")}, // correct: Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG
		{"invalid_extraData", ErrInvalidGenesisFile, editDefaultTestContent("extraData", "0x123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")},
		{"invalid_timestamp", ErrInvalidGenesisFile, editDefaultTestContent("timestamp", "1539051657aaa")},
	}
}

// test valid file content
func Test_setupGenesisBlock_valid(t *testing.T) {
	dataDir := "lemo_data_test_correct"
	writeGenesisToFile(dataDir, editDefaultTestContent("", ""))
	defer clearTmpFiles(dataDir)

	block := setupGenesisBlock(dataDir)
	assert.Equal(t, uint32(0), block.Height())
	assert.Equal(t, common.HexToHash("0x2d9cd33d77e199c6ae7a657a9758ec58003ee2f82c811155152bf863de870251"), block.Hash())
}

// test invalid file content
func Test_setupGenesisBlock_invalid(t *testing.T) {
	for _, tc := range getTestCases() {
		tc := tc // capture range variable
		t.Run(tc.CaseName, func(t *testing.T) {
			t.Parallel()

			dataDir := filepath.Join("testdata", "int_cmd_"+tc.CaseName)
			writeGenesisToFile(dataDir, tc.GenesisFileContent)
			defer clearTmpFiles(dataDir)

			assert.PanicsWithValue(t, tc.Err, func() {
				setupGenesisBlock(dataDir)
			})
		})
	}
}
