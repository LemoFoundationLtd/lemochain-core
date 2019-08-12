package chain

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

type EngineTestForChain struct{}

func (engine *EngineTestForChain) VerifyBeforeTxProcess(block *types.Block) error {
	return nil
}
func (engine *EngineTestForChain) VerifyAfterTxProcess(block, computedBlock *types.Block) error {
	return nil
}
func (engine *EngineTestForChain) Finalize(height uint32, am *account.Manager) error {
	return nil
}
func (engine *EngineTestForChain) Seal(header *types.Header, txProduct *account.TxsProduct, confirms []types.SignData, dNodes types.DeputyNodes) (*types.Block, error) {
	return types.NewBlock(header, txProduct.Txs, txProduct.ChangeLogs), nil
}
func (engine *EngineTestForChain) VerifyConfirmPacket(height uint32, blockHash common.Hash, sigList []types.SignData) error {
	return nil
}
func (engine *EngineTestForChain) TrySwitchFork(stable, oldCurrent *types.Block) *types.Block {
	return nil
}
func (engine *EngineTestForChain) ChooseNewFork() *types.Block {
	return nil
}
func (engine *EngineTestForChain) CanBeStable(height uint32, confirmCount int) bool {
	return true
}
