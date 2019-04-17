package account

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestAccount_Interface(t *testing.T) {
	var _ types.AccountAccessor = (*Account)(nil)
}

func loadAccount(db protocol.ChainDB, address common.Address) *Account {
	acctDb, _ := db.GetActDatabase(newestBlock.Hash())
	data, _ := acctDb.Get(address)
	return NewAccount(db, address, data)
}

func TestAccount_GetAddress(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	// load default account
	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetVersion(BalanceLog))

	// load not exist account
	account = loadAccount(db, common.HexToAddress("0xaaa"))

	assert.Equal(t, common.HexToAddress("0xaaa"), account.GetAddress())

	// load from genesis' parent block
	db.GetActDatabase(common.Hash{})
	_, err := db.GetAccount(common.HexToAddress("0xaaa"))
	assert.Equal(t, store.ErrNotExist, err)

}

func TestAccount_SetBalance_GetBalance(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, big.NewInt(100), account.GetBalance())

	account.SetBalance(big.NewInt(200))
	assert.Equal(t, big.NewInt(200), account.GetBalance())
}

func TestAccount_SetVersion_GetVersion(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetVersion(BalanceLog))
	assert.Equal(t, defaultAccounts[0].NewestRecords[BalanceLog].Height, account.data.NewestRecords[BalanceLog].Height)

	account.SetVersion(BalanceLog, 200, 3)
	assert.Equal(t, uint32(200), account.GetVersion(BalanceLog))
	assert.Equal(t, uint32(3), account.data.NewestRecords[BalanceLog].Height)
}

func TestAccount_SetSuicide_GetSuicide(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, false, account.GetSuicide())

	account.SetSuicide(true)
	assert.Equal(t, true, account.GetSuicide())
	assert.Equal(t, big.NewInt(0), account.GetBalance())
	assert.Equal(t, common.Hash{}, account.GetCodeHash())
	assert.Equal(t, common.Hash{}, account.GetStorageRoot())
}

func TestAccount_GetVoteFor(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, common.Address{}, account.GetVoteFor())

	account.SetVoteFor(common.HexToAddress("0x5000"))
	assert.Equal(t, common.HexToAddress("0x5000"), account.GetVoteFor())
}

func TestAccount_GetCandidate(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.NotNil(t, account.GetCandidate())
	assert.Equal(t, 0, len(account.GetCandidate()))

	profiles := account.GetCandidate()
	profiles[types.CandidateKeyIsCandidate] = "true"
	account.SetCandidate(profiles)
	assert.Equal(t, 1, len(account.GetCandidate()))

	profiles[types.CandidateKeyNodeID] = "NodeId"
	assert.Equal(t, 1, len(account.GetCandidate()))

	profiles = account.GetCandidate()
	profiles[types.CandidateKeyNodeID] = "NodeId"
	assert.Equal(t, 1, len(account.GetCandidate()))
}

func TestAccount_GetVotes(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, new(big.Int).SetInt64(0), account.GetVotes())

	votes := new(big.Int).SetInt64(100)
	account.SetVotes(votes)
	assert.Equal(t, votes, account.GetVotes())

	votes.SetInt64(200)
	assert.NotEqual(t, votes, account.GetVotes())
}

func TestAccount_SetCodeHash_GetCodeHash(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, defaultCodes[0].hash, account.GetCodeHash())

	account.code = types.Code{0x12}
	account.SetCodeHash(c(2))
	assert.Equal(t, c(2), account.GetCodeHash())
	assert.Empty(t, account.code)

	// set to empty
	account.SetCodeHash(common.Hash{})
	assert.Equal(t, common.Hash{}, account.GetCodeHash())
}

func TestAccount_SetCode_GetCode(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	readCode, err := account.GetCode()
	assert.NoError(t, err)
	assert.Equal(t, types.Code{12, 34}, readCode)

	account.SetCode(types.Code{0x12})
	readCode, err = account.GetCode()
	assert.NoError(t, err)
	assert.Equal(t, types.Code{0x12}, readCode)
	assert.Equal(t, common.HexToHash("0x5fa2358263196dbbf23d1ca7a509451f7a2f64c15837bfbb81298b1e3e24e4fa"), account.GetCodeHash())
	assert.Equal(t, true, account.codeIsDirty)

	// clear code
	account.codeIsDirty = false
	account.SetCode(nil)
	readCode, err = account.GetCode()
	assert.NoError(t, err)
	assert.Empty(t, readCode)
	assert.Equal(t, common.Sha3Nil, account.GetCodeHash())
	assert.Equal(t, true, account.codeIsDirty)

	// set nil to new account
	account = loadAccount(db, common.HexToAddress("0xaaa"))
	account.SetCode(nil)
	readCode, err = account.GetCode()
	assert.NoError(t, err)
	assert.Empty(t, readCode)
	assert.Equal(t, common.Hash{}, account.GetCodeHash())
	assert.Equal(t, false, account.codeIsDirty)
}

func TestAccount_SetStorageRoot_GetStorageRoot(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	assert.Equal(t, defaultAccounts[0].StorageRoot, account.GetStorageRoot())

	account.storage.dirty[k(1)] = []byte{12}
	account.SetStorageRoot(h(200))
	assert.Equal(t, h(200), account.GetStorageRoot())
	assert.Empty(t, account.storage.dirty)
}

func TestAccount_SetStorageState_GetStorageState(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)

	// exist in db
	readValue, err := account.GetStorageState(defaultStorage[0].key)
	assert.NoError(t, err)
	assert.Equal(t, defaultStorage[0].value, readValue)

	// exist in cache
	key1 := k(1)
	value1 := []byte{11}
	account.storage.cached[key1] = value1
	readValue, err = account.GetStorageState(key1)
	assert.NoError(t, err)
	assert.Equal(t, value1, readValue)

	// not exist value
	readValue, err = account.GetStorageState(k(2))
	assert.NoError(t, err)
	assert.Empty(t, readValue) // []byte(nil)

	// set
	key3 := k(3)
	value3 := []byte{22}
	account.SetStorageState(key3, value3)
	assert.Equal(t, value3, account.storage.cached[key3])
	assert.Equal(t, value3, account.storage.dirty[key3])

	// set empty
	key4 := k(4)
	value4 := []byte{}
	account.SetStorageState(key4, value4)
	readValue, err = account.GetStorageState(key4)
	assert.NoError(t, err)
	assert.Equal(t, value4, readValue)
	// set nil
	account.SetStorageState(key4, nil)
	readValue, err = account.GetStorageState(key4)
	assert.NoError(t, err)
	assert.Empty(t, readValue) // []byte(nil)

	// set with empty key
	key5 := common.Hash{}
	value5 := []byte{55}
	account.SetStorageState(key5, value5)
	readValue, err = account.GetStorageState(key5)
	assert.NoError(t, err)
	assert.Equal(t, value5, readValue)

	// invalid root
	account.SetStorageRoot(h(1))
	readValue, err = account.GetStorageState(k(6))
	assert.Equal(t, ErrTrieFail, err)
	assert.Empty(t, readValue) // []byte(nil)
}

func TestAccount_IsEmpty(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, common.HexToAddress("0x1"))
	assert.Equal(t, true, account.IsEmpty())
	account.SetVersion(BalanceLog, 100, 3)
	assert.Equal(t, false, account.IsEmpty())
}

func TestAccount_MarshalJSON_UnmarshalJSON(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)
	data, err := json.Marshal(account)
	assert.NoError(t, err)
	assert.Equal(t, `{"address":"Lemo8888888888888888888888888888883CPHBJ","balance":"100","codeHash":"0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e","root":"0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed","assetCodeRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","assetIdRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","equityRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","voteFor":"Lemo888888888888888888888888888888888888","candidate":{"votes":"0","profile":{}},"records":{"1":{"version":"100","height":"1"},"14":{"version":"101","height":"2"}}}`, string(data))
	var parsedAccount *Account
	err = json.Unmarshal(data, &parsedAccount)
	assert.NoError(t, err)
	assert.Equal(t, account.GetAddress(), parsedAccount.GetAddress())
	assert.Equal(t, account.GetBalance(), parsedAccount.GetBalance())
	assert.Equal(t, account.GetNextVersion(BalanceLog), parsedAccount.GetNextVersion(BalanceLog))
	assert.Equal(t, account.GetCodeHash(), parsedAccount.GetCodeHash())
	assert.Equal(t, account.GetStorageRoot(), parsedAccount.GetStorageRoot())
	// assert.Equal(t, account.db, parsedAccount.db)
}

func TestAccount_Finalise_Save(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()

	account := loadAccount(db, defaultAccounts[0].Address)

	// nothing to finalise
	value, err := account.GetStorageState(defaultStorage[0].key)
	assert.NoError(t, err)
	assert.Equal(t, defaultStorage[0].value, value)
	assert.Equal(t, 1, len(account.storage.cached))
	assert.Equal(t, 0, len(account.storage.dirty))
	assert.Equal(t, 2, len(account.data.NewestRecords))
	err = account.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, defaultAccounts[0].StorageRoot, account.GetStorageRoot())
	// save
	err = account.Save()
	assert.NoError(t, err)

	// finalise dirty storage
	key := k(1)
	value = []byte{11, 22, 33}
	err = account.SetStorageState(key, value)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(account.storage.cached))
	assert.Equal(t, 1, len(account.storage.dirty))
	assert.Equal(t, value, account.storage.dirty[key])
	account.SetVersion(StorageLog, 10, 3)
	err = account.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, "0xfb4fbcae2c19f15b34c53b059a4af53d8d793607bd8ca5868eeb9c817c4e5bc7", account.GetStorageRoot().Hex())
	assert.Equal(t, 3, len(account.data.NewestRecords))
	assert.Equal(t, uint32(3), account.data.NewestRecords[StorageLog].Height)
	assert.Equal(t, uint32(10), account.data.NewestRecords[StorageLog].Version)
	assert.Equal(t, 0, len(account.storage.dirty))
	// save
	err = account.Save()
	assert.NoError(t, err)
	account2 := loadAccount(db, defaultAccounts[0].Address)
	account2.SetStorageRoot(account.GetStorageRoot())
	readValue, err := account2.GetStorageState(key)
	assert.NoError(t, err)
	assert.Equal(t, value, readValue)

	// finalise after modify value
	value = []byte{44, 55}
	err = account.SetStorageState(key, value)
	assert.NoError(t, err)
	assert.Equal(t, value, account.storage.dirty[key])
	err = account.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, "0x0adade766035e43ef12b9ac1a84db5eae1c9a3501d81510cdc8cbd0fb3a4b922", account.GetStorageRoot().Hex())
	// save
	err = account.Save()
	assert.NoError(t, err)
	account2 = loadAccount(db, defaultAccounts[0].Address)
	account2.SetStorageRoot(account.GetStorageRoot())
	readValue, err = account2.GetStorageState(key)
	assert.NoError(t, err)
	assert.Equal(t, value, readValue)

	// finalise after remove value
	err = account.SetStorageState(key, nil)
	assert.NoError(t, err)
	assert.Empty(t, account.storage.dirty[key])
	err = account.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, defaultAccounts[0].StorageRoot, account.GetStorageRoot())
	// save
	err = account.Save()
	assert.NoError(t, err)
	account2 = loadAccount(db, defaultAccounts[0].Address)
	account2.SetStorageRoot(account.GetStorageRoot())
	readValue, err = account2.GetStorageState(key)
	assert.NoError(t, err)
	assert.Empty(t, readValue)

	// finalise after remove value with empty []byte
	value = []byte{}
	err = account.SetStorageState(key, value)
	assert.Equal(t, value, account.storage.dirty[key])
	assert.NoError(t, err)
	err = account.Finalise()
	assert.Equal(t, defaultAccounts[0].StorageRoot, account.GetStorageRoot())
	assert.NoError(t, err)
	// save
	err = account.Save()
	assert.NoError(t, err)
	account2 = loadAccount(db, defaultAccounts[0].Address)
	account2.SetStorageRoot(account.GetStorageRoot())
	readValue, err = account2.GetStorageState(key)
	assert.NoError(t, err)
	assert.Empty(t, readValue)

	// dirty code
	account.SetCode(types.Code{0x12})
	err = account.Save()
	assert.NoError(t, err)
	assert.Equal(t, false, account.codeIsDirty)
	account2 = loadAccount(db, defaultAccounts[0].Address)
	account2.SetCodeHash(common.HexToHash("0x5fa2358263196dbbf23d1ca7a509451f7a2f64c15837bfbb81298b1e3e24e4fa"))
	readCode, err := account2.GetCode()
	assert.NoError(t, err)
	assert.Equal(t, types.Code{0x12}, readCode)

	// root changed after finalise
	key = k(2)
	value = []byte{44, 55}
	err = account.SetStorageState(key, value)
	assert.NoError(t, err)
	err = account.Finalise()
	assert.NoError(t, err)
	account.data.StorageRoot = defaultAccounts[0].StorageRoot
	err = account.Save()
	assert.Equal(t, ErrTrieChanged, err)

	// invalid root
	account.SetStorageRoot(h(1))
	value = []byte{11}
	err = account.SetStorageState(key, value)
	assert.NoError(t, err)
	err = account.Finalise()
	assert.Equal(t, ErrTrieFail, err)
}

func TestAccount_LoadChangeLogs(t *testing.T) {
	// db := newDB()
	// TODO
}
