package runtime

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

func NewEnv(cfg *Config) *vm.EVM {
	context := vm.Context{
		CanTransfer: chain.CanTransfer,
		Transfer:    chain.Transfer,
		GetHash:     func(uint32) common.Hash { return common.Hash{} },

		Origin:      cfg.Origin,
		LemoBase:    cfg.Coinbase,
		BlockHeight: cfg.BlockHeight,
		Time:        cfg.Time,
		GasLimit:    cfg.GasLimit,
		GasPrice:    cfg.GasPrice,
	}

	return vm.NewEVM(context, cfg.AccountManager, cfg.EVMConfig)
}
