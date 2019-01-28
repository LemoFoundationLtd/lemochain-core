package account

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSafeAccount_Interface(t *testing.T) {
	var _ types.AccountAccessor = (*SafeAccount)(nil)
}

func loadSafeAccount(address common.Address) *SafeAccount {
	store.ClearData()
	db := newDB()
	data := db.GetActDatabase(newestBlock.Hash()).Find(address[:])
	return NewSafeAccount(NewManager(newestBlock.Hash(), db).processor, NewAccount(db, address, data))
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

	account = loadSafeAccount(defaultAccounts[0].Address)
	assert.Equal(t, false, account.IsDirty())
	assert.Equal(t, true, account.IsDirty())
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
	assert.Equal(t, big.NewInt(100), account.processor.changeLogs[0].OldVal.(*types.AccountData).Balance)
}

func TestSafeAccount_MarshalJSON_UnmarshalJSON(t *testing.T) {
	account := loadSafeAccount(defaultAccounts[0].Address)
	data, err := json.Marshal(account)
	assert.NoError(t, err)
	assert.Equal(t, `{"address":"Lemo8888888888888888888888888888883CPHBJ","balance":"100","codeHash":"0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e","root":"0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed","records":{"1":{"version":"100","height":"1"},"3":{"version":"101","height":"2"}}}`, string(data))
	var parsedAccount *Account
	err = json.Unmarshal(data, &parsedAccount)
	assert.NoError(t, err)
	assert.Equal(t, account.GetAddress(), parsedAccount.GetAddress())
	assert.Equal(t, account.GetBalance(), parsedAccount.GetBalance())
	assert.Equal(t, account.GetBaseVersion(BalanceLog), parsedAccount.GetBaseVersion(BalanceLog))
	assert.Equal(t, account.GetCodeHash(), parsedAccount.GetCodeHash())
	assert.Equal(t, account.GetStorageRoot(), parsedAccount.GetStorageRoot())
	// assert.Equal(t, account.processor, parsedAccount.processor)
}
