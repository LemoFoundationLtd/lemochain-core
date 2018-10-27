package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto/secp256k1"
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewTxProcessor(t *testing.T) {
	chain := newChain()
	p := NewTxProcessor(chain)
	assert.NotEqual(t, (*vm.Config)(nil), p.cfg)
	assert.Equal(t, false, p.cfg.Debug)

	flags := flag.CmdFlags{}
	flags.Set(common.Debug, "1")
	chain, _ = NewBlockChain(chainID, NewDpovp(10*1000, chain.db), chain.db, chain.newBlockCh, flags)
	p = NewTxProcessor(chain)
}

// test valid block processing
func TestTxProcessor_Process(t *testing.T) {
	clearDB()
	p := NewTxProcessor(newChain())

	// last not stable block
	block := defaultBlocks[2]
	newHeader, err := p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())
	sender := p.am.GetAccount(testAddr)
	assert.Equal(t, 3, len(sender.GetTxHashList()))
	assert.Equal(t, block.Txs[0].Hash(), sender.GetTxHashList()[2])

	// block not in db
	block = defaultBlocks[3]
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())
	sender = p.am.GetAccount(testAddr)
	assert.Equal(t, 5, len(sender.GetTxHashList()))
	assert.Equal(t, block.Txs[0].Hash(), sender.GetTxHashList()[3])

	// genesis block
	block = defaultBlocks[0]
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())

	// block on fork branch
	block = createNewBlock()
	newHeader, err = p.Process(block)
	assert.NoError(t, err)
	assert.Equal(t, block.Header.Bloom, newHeader.Bloom)
	assert.Equal(t, block.Header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, block.Header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, block.Header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, block.Header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, block.Header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, block.Hash(), newHeader.Hash())
}

// test invalid block processing
func TestTxProcessor_Process2(t *testing.T) {
	clearDB()
	p := NewTxProcessor(newChain())

	// tamper with amount
	block := createNewBlock()
	rawTx, _ := rlp.EncodeToBytes(block.Txs[0])
	rawTx[25]++ // amount++
	cpy := new(types.Transaction)
	err := rlp.DecodeBytes(rawTx, cpy)
	assert.NoError(t, err)
	assert.Equal(t, new(big.Int).Add(block.Txs[0].Amount(), big.NewInt(1)), cpy.Amount())
	block.Txs[0] = cpy
	_, err = p.Process(block)
	// recover to another from address
	assert.Equal(t, ErrInsufficientBalanceForGas, err)

	// invalid signature
	block = createNewBlock()
	rawTx, _ = rlp.EncodeToBytes(block.Txs[0])
	rawTx[43] = 0 // invalid S
	cpy = new(types.Transaction)
	err = rlp.DecodeBytes(rawTx, cpy)
	assert.NoError(t, err)
	block.Txs[0] = cpy
	_, err = p.Process(block)
	// sometimes the err is this one
	// assert.Equal(t, ErrInsufficientBalanceForGas, err)
	assert.Equal(t, secp256k1.ErrRecoverFailed, err)

	// not enough gas (resign by another address)
	block = createNewBlock()
	private, _ := crypto.GenerateKey()
	origFrom, _ := block.Txs[0].From()
	block.Txs[0] = signTransaction(block.Txs[0], private)
	newFrom, _ := block.Txs[0].From()
	assert.NotEqual(t, origFrom, newFrom)
	block.Header.TxRoot = types.DeriveTxsSha(block.Txs)
	_, err = p.Process(block)
	assert.Equal(t, ErrInsufficientBalanceForGas, err)

	// reach block gas limit
	block = createNewBlock()
	block.Header.GasLimit = 1
	_, err = p.Process(block)
	assert.Equal(t, types.ErrGasLimitReached, err)

	// used gas reach limit
	block = createNewBlock()
	block.Txs[0] = makeTransaction(testPrivate, defaultAccounts[1], big.NewInt(100), common.Big1, 0, 1)
	block.Header.TxRoot = types.DeriveTxsSha(block.Txs)
	_, err = p.Process(block)
	assert.Equal(t, vm.ErrOutOfGas, err)

	// balance not enough
	block = createNewBlock()
	balance := p.am.GetAccount(testAddr).GetBalance()
	block.Txs[0] = makeTx(testPrivate, defaultAccounts[1], new(big.Int).Add(balance, big.NewInt(1)))
	block.Header.TxRoot = types.DeriveTxsSha(block.Txs)
	_, err = p.Process(block)
	assert.Equal(t, vm.ErrInsufficientBalance, err)

	// TODO test create or call contract fail
}

func createNewBlock() *types.Block {
	db, _ := store.NewCacheChain("../../db")
	return makeBlock(db, blockInfo{
		height:     2,
		parentHash: defaultBlocks[1].Hash(),
		author:     testAddr,
		txList: []*types.Transaction{
			makeTx(testPrivate, defaultAccounts[1], big.NewInt(100)),
		}}, false)
}

func TestTxProcessor_ApplyTxs(t *testing.T) {
	clearDB()
	p := NewTxProcessor(newChain())

	// 1 txs
	header := defaultBlocks[2].Header
	txs := defaultBlocks[2].Txs
	emptyHeader := &types.Header{
		ParentHash: header.ParentHash,
		LemoBase:   header.LemoBase,
		Height:     header.Height,
		GasLimit:   header.GasLimit,
		GasUsed:    header.GasUsed,
		Time:       header.Time,
	}
	newHeader, selectedTxs, invalidTxs, err := p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, header.Hash(), newHeader.Hash())
	assert.Equal(t, len(txs), len(selectedTxs))
	assert.Equal(t, 0, len(invalidTxs))

	// 2 txs
	header = defaultBlocks[3].Header
	txs = defaultBlocks[3].Txs
	emptyHeader = &types.Header{
		ParentHash: header.ParentHash,
		LemoBase:   header.LemoBase,
		Height:     header.Height,
		GasLimit:   header.GasLimit,
		GasUsed:    header.GasUsed,
		Time:       header.Time,
	}
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, header.Hash(), newHeader.Hash())
	assert.Equal(t, len(txs), len(selectedTxs))
	assert.Equal(t, 0, len(invalidTxs))

	// 0 txs
	header = defaultBlocks[3].Header
	emptyHeader = &types.Header{
		ParentHash: header.ParentHash,
		LemoBase:   header.LemoBase,
		Height:     header.Height,
		GasLimit:   header.GasLimit,
		GasUsed:    header.GasUsed,
		Time:       header.Time,
	}
	auther := p.am.GetAccount(header.LemoBase)
	origBalance := auther.GetBalance()
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, nil)
	assert.NoError(t, err)
	assert.Equal(t, types.Bloom{}, newHeader.Bloom)
	emptyTrieHash := common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	assert.Equal(t, emptyTrieHash, newHeader.EventRoot)
	assert.Equal(t, emptyTrieHash, newHeader.TxRoot)
	assert.Equal(t, emptyTrieHash, newHeader.LogsRoot)
	assert.Equal(t, 0, len(selectedTxs))
	assert.Equal(t, *origBalance, *auther.GetBalance())
	assert.Equal(t, 0, len(p.am.GetChangeLogs()))

	// too many txs
	header = defaultBlocks[3].Header
	txs = defaultBlocks[3].Txs
	emptyHeader = &types.Header{
		ParentHash: header.ParentHash,
		LemoBase:   header.LemoBase,
		Height:     header.Height,
		GasLimit:   45000, // Every transaction's gasLimit is 30000. So the block only contains one transaction.
		GasUsed:    header.GasUsed,
		Time:       header.Time,
	}
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.NotEqual(t, header.GasUsed, newHeader.GasUsed)
	assert.NotEqual(t, header.TxRoot, newHeader.TxRoot)
	assert.NotEqual(t, header.VersionRoot, newHeader.VersionRoot)
	assert.NotEqual(t, header.LogsRoot, newHeader.LogsRoot)
	assert.NotEqual(t, header.Hash(), newHeader.Hash())
	assert.NotEqual(t, len(txs), len(selectedTxs))
	assert.Equal(t, 0, len(invalidTxs))

	// balance not enough
	header = defaultBlocks[3].Header
	txs = defaultBlocks[3].Txs
	emptyHeader = &types.Header{
		ParentHash: header.ParentHash,
		LemoBase:   header.LemoBase,
		Height:     header.Height,
		GasLimit:   header.GasLimit,
		GasUsed:    header.GasUsed,
		Time:       header.Time,
	}
	balance := p.am.GetAccount(testAddr).GetBalance()
	txs = types.Transactions{
		txs[0],
		makeTx(testPrivate, defaultAccounts[1], new(big.Int).Add(balance, big.NewInt(1))),
		txs[1],
	}
	newHeader, selectedTxs, invalidTxs, err = p.ApplyTxs(emptyHeader, txs)
	assert.NoError(t, err)
	assert.Equal(t, header.Bloom, newHeader.Bloom)
	assert.Equal(t, header.EventRoot, newHeader.EventRoot)
	assert.Equal(t, header.GasUsed, newHeader.GasUsed)
	assert.Equal(t, header.TxRoot, newHeader.TxRoot)
	assert.Equal(t, header.VersionRoot, newHeader.VersionRoot)
	assert.Equal(t, header.LogsRoot, newHeader.LogsRoot)
	assert.Equal(t, header.Hash(), newHeader.Hash())
	assert.Equal(t, len(txs)-1, len(selectedTxs))
	assert.Equal(t, 1, len(invalidTxs))
}
