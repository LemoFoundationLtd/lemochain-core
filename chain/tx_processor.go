package chain

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"math"
	"math/big"
)

var (
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
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
	return &TxProcessor{
		bc:  bc,
		cfg: &vm.Config{},
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
	p.bc.am.Reset(block.Hash())
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

	p.bc.am.Reset(header.ParentHash)
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
	senderAddr, err := tx.From()
	if err != nil {
		return 0, err
	}
	var (
		// Create a new context to be used in the EVM environment
		context = NewEVMContext(tx, header, txIndex, tx.Hash(), blockHash, p.bc)
		// Create a new environment which holds all relevant information
		// about the transaction and calling mechanisms.
		vmEnv            = vm.NewEVM(context, p.bc.am, *p.cfg)
		sender           = p.bc.am.GetAccount(senderAddr)
		contractCreation = tx.To() == nil
		restGas          = tx.GasLimit()
	)
	err = p.buyGas(gp, tx)
	if err != nil {
		return 0, err
	}

	restGas, err = p.payIntrinsicGas(tx, restGas)
	if err != nil {
		return 0, err
	}

	// vm errors do not effect consensus and are therefor
	// not assigned to err, except for insufficient balance
	// error.
	var vmerr error
	if contractCreation {
		_, _, restGas, vmerr = vmEnv.Create(sender, tx.Data(), restGas, tx.Value())
	} else {
		_, restGas, vmerr = vmEnv.Call(sender, *tx.To(), tx.Data(), restGas, tx.Value())
	}
	if vmerr != nil {
		log.Debug("VM returned with error", "err", vmerr)
		// The only possible consensus-error would be if there wasn't
		// sufficient balance to make the transfer happen. The first
		// balance transfer may never fail.
		if vmerr == vm.ErrInsufficientBalance {
			return 0, vmerr
		}
	}
	p.refundGas(gp, tx, restGas, vmEnv.Lemobase)

	// Update the state with pending changes
	p.bc.am.Finalise()

	return tx.GasLimit() - restGas, nil
}

func (p *TxProcessor) buyGas(gp *types.GasPool, tx *types.Transaction) error {
	// ignore the error because it is checked in ApplyTx
	senderAddr, _ := tx.From()
	sender := p.bc.am.GetAccount(senderAddr)

	maxFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit()), tx.GasPrice())
	if sender.GetBalance().Cmp(maxFee) < 0 {
		return errInsufficientBalanceForGas
	}
	if err := gp.SubGas(tx.GasLimit()); err != nil {
		return err
	}
	sender.SetBalance(new(big.Int).Sub(sender.GetBalance(), maxFee))
	return nil
}

func (p *TxProcessor) payIntrinsicGas(tx *types.Transaction, restGas uint64) (uint64, error) {
	gas, err := IntrinsicGas(tx.Data(), tx.To() == nil)
	if err != nil {
		return restGas, err
	}
	if restGas < gas {
		return restGas, vm.ErrOutOfGas
	}
	return restGas - gas, nil
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, contractCreation bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if contractCreation {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		if (math.MaxUint64-gas)/params.TxDataNonZeroGas < nz {
			return 0, vm.ErrOutOfGas
		}
		gas += nz * params.TxDataNonZeroGas

		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, vm.ErrOutOfGas
		}
		gas += z * params.TxDataZeroGas
	}
	return gas, nil
}

func (p *TxProcessor) refundGas(gp *types.GasPool, tx *types.Transaction, restGas uint64, minerAddress common.Address) {
	// ignore the error because it is checked in ApplyTx
	senderAddr, _ := tx.From()
	sender := p.bc.am.GetAccount(senderAddr)

	// Return ETH for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(restGas), tx.GasPrice())
	sender.SetBalance(new(big.Int).Add(sender.GetBalance(), remaining))

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	gp.AddGas(restGas)

	usedFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit()-restGas), tx.GasPrice())
	miner := p.bc.am.GetAccount(minerAddress)
	miner.SetBalance(new(big.Int).Add(miner.GetBalance(), usedFee))
}
