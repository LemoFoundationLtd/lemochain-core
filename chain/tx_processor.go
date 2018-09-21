package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
)

type TxProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain
	engine *Dpovp
}

type ApplyTxsResult struct {
	Txs     types.Transactions // The transactions executed indeed. These transactions will be packaged in a block
	Events  []*types.Event     // events
	Bloom   types.Bloom
	GasUsed uint64
}

func NewTxProcessor(config *params.ChainConfig, bc *BlockChain, engine *Dpovp) *TxProcessor {
	return &TxProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

func (p *TxProcessor) Process(block *types.Block) (*ApplyTxsResult, error) {
	// todo

	return nil, nil
}

// ApplyTxs 执行交易 打包区块使用
func ApplyTxs(bc *BlockChain, header *types.Header, txs types.Transactions) *ApplyTxsResult {
	result := &ApplyTxsResult{}
	// todo
	return result
}
