package transaction

import (
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(tx *types.Transaction, header *types.Header, txIndex uint, blockHash common.Hash, chain ParentBlockLoader) vm.Context {
	if (header.MinerAddress == common.Address{}) {
		panic("NewEVMContext is called without author")
	}
	from, _ := tx.From()
	return vm.Context{
		CanTransfer:  CanTransfer,
		Transfer:     Transfer,
		GetHash:      GetHashFn(header, chain),
		TxIndex:      txIndex,
		TxHash:       tx.Hash(),
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
func GetHashFn(ref *types.Header, chain ParentBlockLoader) vm.GetHashFunc {
	var cache map[uint32]common.Hash

	return func(n uint32) common.Hash {
		// If there's no hash cache yet, make one
		if cache == nil {
			// ref references the block we are building now. So we should not use its hash
			cache = map[uint32]common.Hash{
				ref.Height - 1: ref.ParentHash,
			}
		}
		// Try to fulfill the request from the cache
		if hash, ok := cache[n]; ok {
			return hash
		}
		// Not cached, find the block and cache the hash
		if block := chain.GetParentByHeight(n, ref.ParentHash); block != nil {
			hash := block.Hash()
			cache[block.Header.Height] = hash
			return hash
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
