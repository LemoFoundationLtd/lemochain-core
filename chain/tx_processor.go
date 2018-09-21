package chain

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
)

type TxProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain
	engine *Dpovp
}

type ApplyTxsResult struct {
	Txs     types.Transactions // The transactions executed indeed. These transactions will be packaged in a block
	Events  []*types.Event     // contract events
	Bloom   types.Bloom        // used to search contract events
	GasUsed uint64             // gas used by all transactions
}

func NewTxProcessor(config *params.ChainConfig, bc *BlockChain, engine *Dpovp) *TxProcessor {
	return &TxProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes all transactions in a block. Change accounts' data and execute contract codes.
func (p *TxProcessor) Process(block *types.Block) (*ApplyTxsResult, error) {
	var (
		usedGas  = new(uint64)
		header   = block.Header()
		gp       = new(types.GasPool).AddGas(block.GasLimit())
		txs= block.Txs()
	)
	// Iterate over and process the individual transactions
	for i, tx := range txs {
		p.bc.am.Prepare(tx.Hash(), block.Hash(), i)
		receipt, _, err := ApplyTx(p.config, p.bc, nil, gp, p.bc.am, header, tx, usedGas, cfg)
		if err != nil {
			return nil, err
		}
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize( header)

	return &ApplyTxsResult{
		Txs: txs,
		Events:p.bc.am.GetEvents(),
		Bloom  : ,
		GasUsed:*usedGas ,
	}, nil
}

// ApplyTxs picks and processes transactions from miner's tx pool.
func ApplyTxs(bc *BlockChain, header *types.Header, txs types.Transactions) *ApplyTxsResult {
	gp := new(core.GasPool).AddGas(env.header.GasLimit)

	var coalescedLogs []*types.Log

	for {
		// If we don't have enough gas for any further transactions then we're done
		if gp.Gas() < params.TxGas {
			log.Info("Not enough gas for further transactions", "gp", gp)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		from, _ := types.Sender(env.signer, tx)
		// Start executing the transaction
		env.state.Prepare(tx.Hash(), common.Hash{}, env.tcount)

		snap := env.state.Snapshot()

		receipt, _, err := core.ApplyTransaction(env.config, bc, &coinbase, gp, env.state, env.header, tx, &env.header.GasUsed, vm.Config{})
		if err != nil {
			env.state.RevertToSnapshot(snap)
			return err, nil
		}
		env.txs = append(env.txs, tx)
		env.receipts = append(env.receipts, receipt)

		switch err {
		case core.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Info("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Info("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Info("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	if len(coalescedLogs) > 0 || env.tcount > 0 {
		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processeself.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		go func(logs []*types.Log, tcount int) {
			if len(logs) > 0 {
				mux.Post(core.PendingLogsEvent{Logs: logs})
			}
			if tcount > 0 {
				mux.Post(core.PendingStateEvent{})
			}
		}(cpy, env.tcount)
	}
}

// ApplyTx processes transaction. Change accounts' data and execute contract codes.
func ApplyTx(bc *BlockChain, header *types.Header, tx *types.Transaction) *ApplyTxsResult {
	msg, err := tx.AsMessage(types.MakeSigner(config))
	if err != nil {
		return nil, 0, err
	}
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, bc.am, config, cfg)
	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, 0, err
	}
	// Update the state with pending changes
	var root []byte
	bc.am.Finalise(true)
	*usedGas += gas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wlemo the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = gas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = bc.am.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return &ApplyTxsResult{
		Txs: txs,
		Events:p.bc.am.GetEvents(),
		Bloom  : ,
		GasUsed:*usedGas ,
	}
}
