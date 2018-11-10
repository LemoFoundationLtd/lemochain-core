package chain

import (
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// GetBlockByHash returns the hash corresponding to their hash.
	GetBlockByHash(hash common.Hash) *types.Block
}

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(tx *types.Transaction, header *types.Header, txIndex uint, txHash common.Hash, blockHash common.Hash, chain ChainContext) vm.Context {
	if (header.MinerAddress == common.Address{}) {
		panic("NewEVMContext is called without author")
	}
	from, _ := tx.From()
	return vm.Context{
		CanTransfer:  CanTransfer,
		Transfer:     Transfer,
		GetHash:      GetHashFn(header, chain),
		TxIndex:      txIndex,
		TxHash:       txHash,
		BlockHash:    blockHash,
		Origin:       from,
		MinerAddress: header.MinerAddress,
		BlockHeight:  header.Height,
		Time:         header.Time,
		GasLimit:     header.GasLimit,
		GasPrice:     new(big.Int).Set(tx.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(ref *types.Header, chain ChainContext) vm.GetHashFunc {
	var cache map[uint32]common.Hash

	return func(n uint32) common.Hash {
		// If there's no hash cache yet, make one
		if cache == nil {
			cache = map[uint32]common.Hash{
				ref.Height - 1: ref.ParentHash,
			}
		}
		// Try to fulfill the request from the cache
		if hash, ok := cache[n]; ok {
			return hash
		}
		// Not cached, iterate the blocks and cache the hashes
		for block := chain.GetBlockByHash(ref.ParentHash); block != nil; block = chain.GetBlockByHash(block.Header.ParentHash) {
			cache[block.Header.Height-1] = block.Header.ParentHash
			if n == block.Header.Height-1 {
				return block.Header.ParentHash
			}
		}
		return common.Hash{}
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(am vm.AccountManager, addr common.Address, amount *big.Int) bool {
	return am.GetAccount(addr).GetBalance().Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(am vm.AccountManager, sender, recipient common.Address, amount *big.Int) {
	senderAccount := am.GetAccount(sender)
	recipientAccount := am.GetAccount(recipient)
	senderAccount.SetBalance(new(big.Int).Sub(senderAccount.GetBalance(), amount))
	recipientAccount.SetBalance(new(big.Int).Add(recipientAccount.GetBalance(), amount))
}
