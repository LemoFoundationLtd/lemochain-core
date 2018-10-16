package account

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSafeAccount_Interface(t *testing.T) {
	var _ types.AccountAccessor = (*SafeAccount)(nil)
}

func loadSafeAccount(address common.Address) *SafeAccount {
	db := newDB()
	data, _ := db.GetAccount(newestBlock.Hash(), address)
	return NewSafeAccount(NewManager(newestBlock.Hash(), db).processor, NewAccount(db, address, data, 10))
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
	account.AppendTx(th(1))
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
	assert.Equal(t, `{"address":"0x0000000000000000000000000000000000010000","balance":"0x64","codeHash":"0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e","root":"0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed","records":{"1":{"Version":100,"Height":1},"3":{"Version":101,"Height":2}},"TxHashList":["0x0000000000000000000000000000000000000000000000000000000000000011","0x0000000000000000000000000000000000000000000000000000000000000022"]}`, string(data))
	var parsedAccount *Account
	err = json.Unmarshal(data, &parsedAccount)
	assert.NoError(t, err)
	assert.Equal(t, account.GetAddress(), parsedAccount.GetAddress())
	assert.Equal(t, account.GetBalance(), parsedAccount.GetBalance())
	assert.Equal(t, account.GetVersion(BalanceLog), parsedAccount.GetVersion(BalanceLog))
	assert.Equal(t, account.GetCodeHash(), parsedAccount.GetCodeHash())
	assert.Equal(t, account.GetStorageRoot(), parsedAccount.GetStorageRoot())
	// assert.Equal(t, account.processor, parsedAccount.processor)
}
