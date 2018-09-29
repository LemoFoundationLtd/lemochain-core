package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"strconv"
)

type TxProcessor struct {
	bc  *BlockChain
	cfg *vm.Config // configuration of vm
}

type ApplyTxsResult struct {
	Txs     types.Transactions // The transactions executed indeed. These transactions will be packaged in a block
	Events  []*types.Event     // contract events
	Bloom   types.Bloom        // used to search contract events
	GasUsed uint64             // gas used by all transactions
}

func NewTxProcessor(bc *BlockChain) *TxProcessor {
	debug, _ := strconv.ParseBool(bc.Flags()[common.Debug])
	cfg := &vm.Config{
		Debug: debug,
	}
	return &TxProcessor{
		bc:  bc,
		cfg: cfg,
	}
}

// Process processes all transactions in a block. Change accounts' data and execute contract codes.
func (p *TxProcessor) Process(block *types.Block) (*ApplyTxsResult, error) {
	var (
		gasUsed = uint64(0)
		header  = block.Header
		gp      = new(types.GasPool).AddGas(block.GasLimit())
		txs     = block.Txs
	)
	// Iterate over and process the individual transactions
	for i, tx := range txs {
		gas, err := p.ApplyTx(gp, header, tx, uint(i), block.Hash())
		if err != nil {
			return nil, err
		}
		gasUsed = gasUsed + gas
	}

	events := p.bc.am.GetEvents()
	bloom := types.CreateBloom(events)

	return &ApplyTxsResult{
		Txs:     txs,
		Events:  events,
		Bloom:   bloom,
		GasUsed: gasUsed,
	}, nil
}

// ApplyTxs picks and processes transactions from miner's tx pool.
func (p *TxProcessor) ApplyTxs(header *types.Header, txs types.Transactions) (*ApplyTxsResult, error) {
	gp := new(types.GasPool).AddGas(header.GasLimit)
	gasUsed := uint64(0)
	selectedTxs := make(types.Transactions, 0)

	for _, tx := range txs {
		// If we don't have enough gas for any further transactions then we're done
		if gp.Gas() < params.TxGas {
			log.Info("Not enough gas for further transactions", "gp", gp)
			break
		}
		// Start executing the transaction
		snap := p.bc.am.Snapshot()

		gas, err := p.ApplyTx(gp, header, tx, uint(len(selectedTxs)), common.Hash{})
		if err != nil {
			p.bc.am.RevertToSnapshot(snap)
			return nil, err
		}
		selectedTxs = append(selectedTxs, tx)

		if err == types.ErrGasLimitReached {
			// Error may be ignored here. The error has already been checked
			// during transaction acceptance is the transaction pool.
			from, _ := tx.From()
			log.Info("Gas limit exceeded for current block", "sender", from)
		} else if err != nil {
			// Strange error, discard the transaction and get the next in line.
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
		}
		gasUsed = gasUsed + gas
	}

	events := p.bc.am.GetEvents()
	bloom := types.CreateBloom(events)

	return &ApplyTxsResult{
		Txs:     txs,
		Events:  events,
		Bloom:   bloom,
		GasUsed: gasUsed,
	}, nil
}

// ApplyTx processes transaction. Change accounts' data and execute contract codes.
func (p *TxProcessor) ApplyTx(gp *types.GasPool, header *types.Header, tx *types.Transaction, txIndex uint, blockHash common.Hash) (uint64, error) {
	// msg, err := tx.AsMessage(types.MakeSigner())
	// if err != nil {
	// 	return 0, err
	// }
	// // Create a new context to be used in the EVM environment
	// context := NewEVMContext(msg, header, txIndex, tx.Hash(), blockHash, p.bc)
	// // Create a new environment which holds all relevant information
	// // about the transaction and calling mechanisms.
	// vmEnv := vm.NewEVM(context, p.bc.am, *p.cfg)
	// // Apply the transaction to the current state (included in the env)
	// _, gas, _, err := ApplyMessage(vmEnv, msg, gp)
	// if err != nil {
	// 	return 0, err
	// }
	// // Update the state with pending changes
	// p.bc.am.Finalise()
	//
	// return gas, nil
	return 0, nil
}
