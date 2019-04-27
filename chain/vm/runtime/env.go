package runtime

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/consensus"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

func NewEnv(cfg *Config) *vm.EVM {
	context := vm.Context{
		CanTransfer: consensus.CanTransfer,
		Transfer:    consensus.Transfer,
		GetHash:     func(uint32) common.Hash { return common.Hash{} },

		Origin:       cfg.Origin,
		MinerAddress: cfg.MinerAddress,
		BlockHeight:  cfg.BlockHeight,
		Time:         cfg.Time,
		GasLimit:     cfg.GasLimit,
		GasPrice:     cfg.GasPrice,
	}

	return vm.NewEVM(context, cfg.AccountManager, cfg.EVMConfig)
}
