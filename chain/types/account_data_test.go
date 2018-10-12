package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestAccountData_EncodeRLP_DecodeRLP(t *testing.T) {
	logType1 := ChangeLogType(1)
	logType2 := ChangeLogType(2)
	account := &AccountData{
		Address:     common.HexToAddress("0x10000"),
		Balance:     big.NewInt(100),
		Versions:    map[ChangeLogType]uint32{logType1: 100, logType2: 101},
		CodeHash:    common.HexToHash("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e"),
		StorageRoot: common.HexToHash("0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed"),
		NewestRecords: map[ChangeLogType]VersionRecord{
			logType1: {Version: 100, Height: 1},
			logType2: {Version: 101, Height: 2},
		},
	}
	data, err := rlp.EncodeToBytes(account)
	assert.NoError(t, err)
	decoded := new(AccountData)
	err = rlp.DecodeBytes(data, decoded)
	assert.NoError(t, err)
	assert.Equal(t, decoded, account)
}
