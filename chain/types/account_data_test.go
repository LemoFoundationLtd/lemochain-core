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
	}

	// before update records
	_, err := rlp.EncodeToBytes(account)
	assert.Equal(t, ErrInvalidRecord, err)

	// after update records
	account.UpdateRecords(10)
	data, err := rlp.EncodeToBytes(account)
	assert.NoError(t, err)

	// decode correct data
	decoded := new(AccountData)
	err = rlp.DecodeBytes(data, decoded)
	assert.NoError(t, err)
	assert.Equal(t, account, decoded)
	assert.Equal(t, uint32(100), decoded.Versions[logType1])

	// decode incorrect data
	decoded = new(AccountData)
	err = rlp.DecodeBytes(common.Hex2Bytes("f86094000000000000000000000000000000000001000064a01d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500ea0cbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3edc20102c26465c10a"), decoded)
	assert.Equal(t, ErrInvalidRecord, err)
}

func TestAccountData_UpdateRecords(t *testing.T) {
	logType1 := ChangeLogType(1)
	logType2 := ChangeLogType(2)
	account := &AccountData{
		Address:  common.HexToAddress("0x10000"),
		Versions: map[ChangeLogType]uint32{},
	}

	// empty
	account.UpdateRecords(100)
	assert.Equal(t, 0, len(account.NewestRecords))

	// create record
	account.Versions[logType1] = 100
	account.Versions[logType2] = 101
	account.UpdateRecords(10)
	assert.Equal(t, 2, len(account.NewestRecords))
	assert.Equal(t, uint32(10), account.NewestRecords[logType1].Height)
	assert.Equal(t, uint32(100), account.NewestRecords[logType1].Version)

	// update record
	account.Versions[logType1] = 102
	account.UpdateRecords(11)
	assert.Equal(t, 2, len(account.NewestRecords))
	assert.Equal(t, uint32(11), account.NewestRecords[logType1].Height)
	assert.Equal(t, uint32(102), account.NewestRecords[logType1].Version)

	// not update
	account.UpdateRecords(12)
	assert.Equal(t, 2, len(account.NewestRecords))
	assert.Equal(t, uint32(11), account.NewestRecords[logType1].Height)
	assert.Equal(t, uint32(102), account.NewestRecords[logType1].Version)
}
