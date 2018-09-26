package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func loadSafeAccount(address common.Address) *SafeAccount {
	db := newDB()
	data, _ := db.GetAccount(defaultBlock.hash, address)
	return NewSafeAccount(NewManager(defaultBlock.hash, db).processor, NewAccount(db, address, data))
}

func TestSafeAccount_SetBalance_IsDirty(t *testing.T) {
	account := loadSafeAccount(defaultAccounts[0].Address)

	assert.Equal(t, false, account.IsDirty())
	account.SetBalance(big.NewInt(200))
	assert.Equal(t, true, account.IsDirty())
	assert.Equal(t, big.NewInt(200), account.GetBalance())
	assert.Equal(t, 1, len(account.processor.changeLogs))
	assert.Equal(t, BalanceLog, account.processor.changeLogs[0].LogType)
	assert.Equal(t, *big.NewInt(200), account.processor.changeLogs[0].NewVal.(big.Int))
}

func TestSafeAccount_SetCode_IsDirty(t *testing.T) {
	account := loadSafeAccount(defaultAccounts[0].Address)

	account.SetCode(types.Code{0x12})
	assert.Equal(t, true, account.IsDirty())
	assert.Equal(t, 1, len(account.processor.changeLogs))
	assert.Equal(t, CodeLog, account.processor.changeLogs[0].LogType)
	assert.Equal(t, types.Code{0x12}, account.processor.changeLogs[0].NewVal.(types.Code))
}

func TestSafeAccount_SetStorageState_IsDirty(t *testing.T) {
	account := loadSafeAccount(defaultAccounts[0].Address)

	err := account.SetStorageState(k(1), []byte{11})
	assert.NoError(t, err)
	assert.Equal(t, true, account.IsDirty())
	assert.Equal(t, 1, len(account.processor.changeLogs))
	assert.Equal(t, StorageLog, account.processor.changeLogs[0].LogType)
	assert.Equal(t, []byte{11}, account.processor.changeLogs[0].NewVal.([]byte))
}

func TestSafeAccount_SetSuicide_GetSuicide(t *testing.T) {
	account := loadSafeAccount(defaultAccounts[0].Address)
	assert.Equal(t, false, account.GetSuicide())

	account.SetSuicide(true)
	assert.Equal(t, true, account.GetSuicide())
	assert.Equal(t, 1, len(account.processor.changeLogs))
	assert.Equal(t, SuicideLog, account.processor.changeLogs[0].LogType)
	assert.Equal(t, *big.NewInt(100), account.processor.changeLogs[0].OldVal.(big.Int))
}
