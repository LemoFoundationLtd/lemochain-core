package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestManager_Interface(t *testing.T) {
	var _ vm.AccountManager = (*Manager)(nil)
	var _ types.ChangeLogProcessor = (*LogProcessor)(nil)
}

func TestNewManager_withoutDB(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, "account.NewManager is called without a database", err)
	}()
	NewManager(common.Hash{}, nil)
}

func TestNewManager(t *testing.T) {
	NewManager(common.Hash{}, newDB())
}

func TestManager_GetAccount(t *testing.T) {
	db := newDB()

	// exist in db
	manager := NewManager(newestBlock.Hash(), db)
	account := manager.GetAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetBaseVersion(BalanceLog))
	assert.Equal(t, false, account.IsEmpty())
	// not exist in db
	account = manager.GetAccount(common.HexToAddress("0xaaa"))
	assert.Equal(t, common.HexToAddress("0xaaa"), account.GetAddress())
	assert.Equal(t, true, account.IsEmpty())

	// load from older block
	manager = NewManager(defaultBlockInfos[0].hash, db)
	account = manager.GetAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetBaseVersion(BalanceLog))
	assert.Equal(t, false, account.IsEmpty())

	// load from genesis' parent block
	manager = NewManager(common.Hash{}, db)
	account = manager.GetAccount(common.Address{})
	assert.Equal(t, common.Address{}, account.GetAddress())
	assert.Equal(t, account, manager.accountCache[common.Address{}])

	// get twice
	account2 := manager.GetAccount(common.Address{})
	account.SetBalance(big.NewInt(200))
	assert.Equal(t, big.NewInt(200), account.GetBalance())
	account2.SetBalance(big.NewInt(100))
	assert.Equal(t, big.NewInt(100), account2.GetBalance())
}

func TestManager_GetCanonicalAccount(t *testing.T) {
	db := newDB()

	// exist in db
	manager := NewManager(newestBlock.Hash(), db)
	account := manager.GetCanonicalAccount(defaultAccounts[0].Address)
	assert.Equal(t, uint32(100), account.GetBaseVersion(BalanceLog))
	// not exist in db
	account = manager.GetCanonicalAccount(common.HexToAddress("0xaaa"))
	assert.Equal(t, common.HexToAddress("0xaaa"), account.GetAddress())

	// load from genesis' parent block
	manager = NewManager(common.Hash{}, db)
	account = manager.GetCanonicalAccount(common.Address{})
	assert.Equal(t, common.Address{}, account.GetAddress())
	assert.Empty(t, manager.accountCache[common.Address{}])
}

func TestManager_GetChangeLogs(t *testing.T) {
	db := newDB()
	manager := NewManager(newestBlock.Hash(), db)

	logs := manager.GetChangeLogs()
	assert.Equal(t, 0, len(logs))

	// create a new change log
	manager.processor.changeLogs = append(manager.processor.changeLogs, &types.ChangeLog{NewVal: 123})
	logs = manager.GetChangeLogs()
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, 123, logs[0].NewVal.(int))
}

func TestManager_AddEvent(t *testing.T) {
	db := newDB()
	manager := NewManager(newestBlock.Hash(), db)

	event1 := &types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 11}
	event2 := &types.Event{Address: common.HexToAddress("0x1"), TxHash: th(1), BlockHeight: 22}
	manager.AddEvent(event1)
	assert.Equal(t, uint(0), event1.Index)
	manager.AddEvent(event2)
	assert.Equal(t, uint(1), event2.Index)
	events := manager.GetEvents()
	assert.Equal(t, 2, len(events))
	assert.Equal(t, uint32(11), events[0].BlockHeight)
	logs := manager.GetChangeLogs()
	assert.Equal(t, 2, len(logs))
	assert.Equal(t, AddEventLog, logs[0].LogType)
	assert.Equal(t, event1, logs[0].NewVal.(*types.Event))
}

func TestManager_GetVersionRoot(t *testing.T) {
	db := newDB()

	// empty version trie
	manager := NewManager(common.Hash{}, db)
	root := manager.GetVersionRoot()
	assert.Equal(t, emptyTrieRoot, root)

	// not empty version trie
	manager = NewManager(newestBlock.Hash(), db)
	root = manager.GetVersionRoot()
	assert.Equal(t, newestBlock.VersionRoot(), root)
	// read version from trie
	key := append(defaultAccounts[0].Address.Bytes(), big.NewInt(int64(BalanceLog)).Bytes()...)
	value, err := ReadTrie(db, root, key)
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), manager.GetAccount(defaultAccounts[0].Address).GetBaseVersion(BalanceLog))
	assert.Equal(t, big.NewInt(100), new(big.Int).SetBytes(value))
}

func TestManager_Reset(t *testing.T) {
	db := newDB()
	manager := NewManager(common.Hash{}, db)

	account := manager.GetAccount(common.HexToAddress("0x1"))
	account.SetBalance(big.NewInt(2))
	manager.GetVersionRoot()
	assert.NotEmpty(t, manager.accountCache)
	assert.NotEmpty(t, manager.processor.changeLogs)
	assert.NotEmpty(t, manager.versionTrie)
	manager.Reset(newestBlock.Hash())
	assert.Equal(t, newestBlock.Hash(), manager.baseBlockHash)
	assert.Empty(t, manager.accountCache)
	assert.Empty(t, manager.processor.changeLogs)
	assert.Empty(t, manager.processor.events)
	assert.Empty(t, manager.versionTrie)
}

// saving blocks after the newest block
func TestManager_Finalise_Save(t *testing.T) {
	db := newDB()
	manager := NewManager(newestBlock.Hash(), db)

	// nothing to finalise
	account := manager.GetAccount(common.HexToAddress("0x1"))
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, false, account.(*SafeAccount).IsDirty())
	err := manager.Finalise()
	assert.NoError(t, err)
	// save
	err = manager.Save(b(1))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))

	// finalise dirty
	account = manager.GetAccount(defaultAccounts[0].Address)
	account.SetBalance(big.NewInt(2))
	err = account.SetStorageState(k(1), []byte{100})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, true, account.(*SafeAccount).IsDirty())
	assert.Equal(t, 2, len(manager.processor.changeLogs))
	root := manager.GetVersionRoot()
	assert.Equal(t, newestBlock.VersionRoot(), root)
	err = manager.Finalise()
	assert.NoError(t, err)
	root = manager.GetVersionRoot()
	assert.NotEqual(t, newestBlock.VersionRoot(), root)
	// save
	block := &types.Block{}
	block.SetHeader(&types.Header{VersionRoot: root})
	err = db.SetBlock(b(2), block)
	assert.NoError(t, err)
	err = manager.Save(b(2))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))
	assert.Equal(t, 0, len(manager.processor.changeLogs))
	manager = NewManager(b(2), db)
	assert.Equal(t, root, manager.GetVersionRoot())
}

// saving for genesis block and first block
func TestManager_Finalise_Save2(t *testing.T) {
	db := newDB()
	// load from genesis' parent block
	manager := NewManager(common.Hash{}, db)

	// nothing to finalise
	account := manager.GetAccount(common.HexToAddress("0x1"))
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, false, account.(*SafeAccount).IsDirty())
	err := manager.Finalise()
	assert.NoError(t, err)
	// save
	err = manager.Save(b(11))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))

	// finalise dirty
	account = manager.GetAccount(common.HexToAddress("0x2"))
	account.SetBalance(big.NewInt(2))
	err = account.SetStorageState(k(1), []byte{100})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(manager.accountCache))
	assert.Equal(t, true, account.(*SafeAccount).IsDirty())
	assert.Equal(t, 2, len(manager.processor.changeLogs))
	root := manager.GetVersionRoot()
	assert.Equal(t, emptyTrieRoot, root)
	err = manager.Finalise()
	assert.NoError(t, err)
	root = manager.GetVersionRoot()
	assert.NotEqual(t, emptyTrieRoot, root)
	// save
	block := &types.Block{}
	block.SetHeader(&types.Header{VersionRoot: root})
	err = db.SetBlock(b(12), block)
	assert.NoError(t, err)
	err = manager.Save(b(12))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.accountCache))
	assert.Equal(t, 0, len(manager.processor.changeLogs))
	manager = NewManager(b(12), db)
	assert.Equal(t, root, manager.GetVersionRoot())
}

func TestManager_Save_Reset(t *testing.T) {
	db := newDB()
	// load from genesis' parent block
	manager := NewManager(common.Hash{}, db)

	// save balance to 1 in block1
	account := manager.GetAccount(common.HexToAddress("0x1"))
	account.SetBalance(big.NewInt(1))
	assert.Equal(t, uint32(0), account.GetBaseVersion(BalanceLog))
	account.(*SafeAccount).AppendTx(th(12))
	assert.Equal(t, 1, len(account.GetTxHashList()))
	err := manager.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), account.GetBaseVersion(BalanceLog))
	assert.Equal(t, uint32(0), account.(*SafeAccount).rawAccount.data.NewestRecords[BalanceLog].Height)
	block := &types.Block{}
	block.SetHeader(&types.Header{Height: 0, VersionRoot: manager.GetVersionRoot()})
	err = db.SetBlock(block.Hash(), block)
	assert.NoError(t, err)
	err = manager.Save(block.Hash())
	assert.NoError(t, err)

	// save balance to 2 in block2
	block1Hash := block.Hash()
	manager.Reset(block1Hash)
	account = manager.GetAccount(common.HexToAddress("0x1"))
	account.SetBalance(big.NewInt(2))
	assert.Equal(t, uint32(1), account.GetBaseVersion(BalanceLog))
	err = manager.Finalise()
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), account.GetBaseVersion(BalanceLog))
	assert.Equal(t, uint32(1), account.(*SafeAccount).rawAccount.data.NewestRecords[BalanceLog].Height)
	block = &types.Block{}
	block.SetHeader(&types.Header{Height: 1, ParentHash: block1Hash, VersionRoot: manager.GetVersionRoot()})
	err = db.SetBlock(block.Hash(), block)
	assert.NoError(t, err)
	err = manager.Save(block.Hash())
	assert.NoError(t, err)

	// load state from block1
	manager.Reset(block1Hash)
	account = manager.GetAccount(common.HexToAddress("0x1"))
	assert.Equal(t, big.NewInt(1), account.GetBalance())
	assert.Equal(t, uint32(1), account.GetBaseVersion(BalanceLog))
	assert.Equal(t, 1, len(account.GetTxHashList()))
	assert.Equal(t, th(12), account.GetTxHashList()[0])
}

func TestManager_SaveTxInAccount(t *testing.T) {
	db := newDB()
	manager := NewManager(common.Hash{}, db)

	account1 := manager.GetAccount(defaultAccounts[0].Address)
	account2 := manager.GetAccount(common.HexToAddress("0x1"))
	assert.Equal(t, 0, len(account1.GetTxHashList()))
	assert.Equal(t, 0, len(account2.GetTxHashList()))
	manager.SaveTxInAccount(account1.GetAddress(), account2.GetAddress(), common.HexToHash("0x111"))
	assert.Equal(t, 1, len(account1.GetTxHashList()))
	assert.Equal(t, common.HexToHash("0x111"), account1.GetTxHashList()[0])
	assert.Equal(t, 1, len(account2.GetTxHashList()))

	// from is to
	manager.SaveTxInAccount(account1.GetAddress(), account1.GetAddress(), common.HexToHash("0x222"))
	assert.Equal(t, 2, len(account1.GetTxHashList()))
}
