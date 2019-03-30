package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/math"
	"math/big"
	"sync"
	"time"
)

const (
	defaultGasPrice      = 1e9
	MaxDeputyHostLength  = 128
	StandardNodeIdLength = 128
)

var (
	ErrInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
	ErrInvalidTxInBlock          = errors.New("block contains invalid transaction")
)

type TxProcessor struct {
	chain *BlockChain
	am    *account.Manager
	cfg   *vm.Config // configuration of vm

	lock sync.Mutex
}

func NewTxProcessor(bc *BlockChain) *TxProcessor {
	debug := bc.Flags().Bool(common.Debug)
	cfg := &vm.Config{
		Debug: debug,
	}
	return &TxProcessor{
		chain: bc,
		am:    bc.am,
		cfg:   cfg,
	}
}

// Process processes all transactions in a block. Change accounts' data and execute contract codes.
func (p *TxProcessor) Process(header *types.Header, txs types.Transactions) (uint64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	var (
		gp          = new(types.GasPool).AddGas(header.GasLimit)
		gasUsed     = uint64(0)
		totalGasFee = new(big.Int)
	)

	p.am.Reset(header.ParentHash)

	// genesis
	if header.Height == 0 {
		log.Warn("It is not necessary to process genesis block.")
		return gasUsed, nil
	}
	// Iterate over and process the individual transactions
	for i, tx := range txs {
		gas, err := p.applyTx(gp, header, tx, uint(i), header.Hash())
		if err != nil {
			log.Info("Invalid transaction", "hash", tx.Hash(), "err", err)
			return gasUsed, ErrInvalidTxInBlock
		}
		gasUsed = gasUsed + gas
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		totalGasFee.Add(totalGasFee, fee)
	}
	p.chargeForGas(totalGasFee, header.MinerAddress)

	if len(txs) > 0 {
		log.Infof("process %d transactions", len(txs))
	}
	return gasUsed, nil
}

// ApplyTxs picks and processes transactions from miner's tx pool.
func (p *TxProcessor) ApplyTxs(header *types.Header, txs types.Transactions) (types.Transactions, types.Transactions, uint64) {
	var (
		gp          = new(types.GasPool).AddGas(header.GasLimit)
		gasUsed     = uint64(0)
		totalGasFee = new(big.Int)
		selectedTxs = make(types.Transactions, 0)
		invalidTxs  = make(types.Transactions, 0)
	)

	p.am.Reset(header.ParentHash)

	// Iterate over and process the individual transactions
	for _, tx := range txs {
		// If we don't have enough gas for any further transactions then we're done
		if gp.Gas() < params.TxGas {
			log.Info("Not enough gas for further transactions", "gp", gp)
			break
		}
		// Start executing the transaction
		snap := p.am.Snapshot()

		gas, err := p.applyTx(gp, header, tx, uint(len(selectedTxs)), common.Hash{})
		if err != nil {
			p.am.RevertToSnapshot(snap)
			if err == types.ErrGasLimitReached {
				// block is full
				log.Info("Not enough gas for further transactions", "gp", gp, "lastTxGasLimit", tx.GasLimit())
			} else {
				// Strange error, discard the transaction and get the next in line.
				log.Info("Skipped invalid transaction", "hash", tx.Hash(), "err", err)
				invalidTxs = append(invalidTxs, tx)
			}
			continue
		}
		selectedTxs = append(selectedTxs, tx)

		gasUsed = gasUsed + gas
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		totalGasFee.Add(totalGasFee, fee)
	}
	p.chargeForGas(totalGasFee, header.MinerAddress)

	if len(selectedTxs) > 0 {
		log.Infof("process %d transactions", len(selectedTxs))
	}
	return selectedTxs, invalidTxs, gasUsed
}

// applyTx processes transaction. Change accounts' data and execute contract codes.
func (p *TxProcessor) applyTx(gp *types.GasPool, header *types.Header, tx *types.Transaction, txIndex uint, blockHash common.Hash) (uint64, error) {
	senderAddr, err := tx.From()
	if err != nil {
		return 0, err
	}
	var (
		// Create a new context to be used in the EVM environment
		context = NewEVMContext(tx, header, txIndex, tx.Hash(), blockHash, p.chain)
		// Create a new environment which holds all relevant information
		// about the transaction and calling mechanisms.
		vmEnv                = vm.NewEVM(context, p.am, *p.cfg)
		sender               = p.am.GetAccount(senderAddr)
		initialSenderBalance = sender.GetBalance()
		contractCreation     = tx.To() == nil
		restGas              = tx.GasLimit()
		mergeFrom            = len(p.am.GetChangeLogs())
	)
	err = p.buyGas(gp, tx)
	if err != nil {
		return 0, err
	}
	restGas, err = p.payIntrinsicGas(tx, restGas)
	if err != nil {
		return 0, err
	}

	// vm errors do not effect consensus and are therefor not assigned to err,
	// except for insufficient balance error.
	var (
		vmErr                   error
		recipientAddr           common.Address
		initialRecipientBalance *big.Int
		recipientAccount        types.AccountAccessor
	)
	if !contractCreation {
		recipientAddr = *tx.To()
		recipientAccount = p.am.GetAccount(recipientAddr)
		initialRecipientBalance = recipientAccount.GetBalance()
	}
	// Judge the type of transaction
	switch tx.Type() {
	case params.OrdinaryTx:
		if contractCreation {
			_, recipientAddr, restGas, vmErr = vmEnv.Create(sender, tx.Data(), restGas, tx.Amount())
		} else {
			_, restGas, vmErr = vmEnv.Call(sender, recipientAddr, tx.Data(), restGas, tx.Amount())
		}
	case params.VoteTx:
		restGas, vmErr = vmEnv.CallVoteTx(senderAddr, recipientAddr, restGas, initialSenderBalance)

	case params.RegisterTx:
		// Unmarshal tx data
		txData := tx.Data()
		profile := make(types.Profile)
		err = json.Unmarshal(txData, &profile)
		if err != nil {
			log.Errorf("unmarshal Candidate node error: %s", err)
			return 0, err
		}

		if nodeId, ok := profile[types.CandidateKeyNodeID]; ok {
			nodeIdLength := len(nodeId)
			if nodeIdLength != StandardNodeIdLength {
				nodeIdErr := fmt.Errorf("the nodeId length [%d] is not equal the standard length [%d] ", nodeIdLength, StandardNodeIdLength)
				return 0, nodeIdErr
			}
		}
		if host, ok := profile[types.CandidateKeyHost]; ok {
			hostLength := len(host)
			if hostLength > MaxDeputyHostLength {
				hostErr := fmt.Errorf("the length of host field in transaction is out of max length limit. host length = %d. max length limit = %d. ", hostLength, MaxDeputyHostLength)
				return 0, hostErr
			}
		}

		restGas, vmErr = vmEnv.RegisterOrUpdateToCandidate(senderAddr, params.FeeReceiveAddress, profile, restGas, initialSenderBalance)

	case params.CreateAssetTx:
		vmErr = vmEnv.CreateAssetTx(senderAddr, tx.Data(), tx.Hash())
	case params.IssueAssetTx:
		vmErr = vmEnv.IssueAssetTx(senderAddr, recipientAddr, tx.Hash(), tx.Data())
	case params.ReplenishAssetTx:
		vmErr = vmEnv.ReplenishAssetTx(senderAddr, recipientAddr, tx.Data())
	case params.ModifyAssetTx:
		vmErr = vmEnv.ModifyAssetProfileTx(senderAddr, tx.Data())
	case params.TradingAssetTx:
		tradingAsset := &types.TradingAsset{}
		err = json.Unmarshal(tx.Data(), tradingAsset)
		if err != nil {
			log.Errorf("unmarshal trading asset data err: %s", err)
			return 0, err
		}
		_, restGas, vmErr = vmEnv.TradingAssetTx(sender, recipientAddr, restGas, tradingAsset.AssetId, tradingAsset.Value, tradingAsset.Input, p.chain.db)
	default:
		log.Errorf("The type of transaction is not defined. txType = %d\n", tx.Type())
	}
	// Candidate node votes change
	if !contractCreation {
		endRecipientBalance := recipientAccount.GetBalance()
		if initialRecipientBalance == nil {
			p.changeCandidateVotes(recipientAddr, endRecipientBalance)
		} else {
			recipientBalanceChange := new(big.Int).Sub(endRecipientBalance, initialRecipientBalance)
			p.changeCandidateVotes(recipientAddr, recipientBalanceChange)
		}
	}

	if vmErr != nil {
		log.Info("VM returned with error", "err", vmErr)
		// The only possible consensus-error would be if there wasn't
		// sufficient balance to make the transfer happen. The first
		// balance transfer may never fail.
		if vmErr == vm.ErrInsufficientBalance {
			return 0, vmErr
		}
	}
	p.refundGas(gp, tx, restGas)

	// The number of votes of the candidate nodes corresponding to the sender.
	endSenderBalance := sender.GetBalance()
	senderBalanceChange := new(big.Int).Sub(endSenderBalance, initialSenderBalance)
	p.changeCandidateVotes(senderAddr, senderBalanceChange)

	// reimbursement transaction
	if len(tx.GasPayerSig()) != 0 {
		payer, _ := tx.GasPayer()
		// balance decrease the amount
		reduceBalance := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit()-restGas), tx.GasPrice())
		negativeChangeBalance := new(big.Int).Neg(reduceBalance)
		p.changeCandidateVotes(payer, negativeChangeBalance)
	}

	// Merge change logs by transaction will save more transaction execution detail than by block
	p.am.MergeChangeLogs(mergeFrom)

	return tx.GasLimit() - restGas, nil
}

// changeCandidateVotes candidate node vote change corresponding to balance change
func (p *TxProcessor) changeCandidateVotes(accountAddress common.Address, changeBalance *big.Int) {
	if changeBalance.Sign() == 0 {
		return
	}
	acc := p.am.GetAccount(accountAddress)
	CandidataAddress := acc.GetVoteFor()

	if (CandidataAddress == common.Address{}) {
		return
	}
	CandidateAccount := p.am.GetAccount(CandidataAddress)
	profile := CandidateAccount.GetCandidate()
	if profile[types.CandidateKeyIsCandidate] == params.NotCandidateNode {
		return
	}
	// set votes
	CandidateAccount.SetVotes(new(big.Int).Add(CandidateAccount.GetVotes(), changeBalance))
}

func (p *TxProcessor) buyGas(gp *types.GasPool, tx *types.Transaction) error {
	payerAddr, err := tx.GasPayer()
	log.Infof("tx gas payer address: %s", payerAddr.String())
	if err != nil {
		return err
	}
	payer := p.am.GetAccount(payerAddr)

	maxFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit()), tx.GasPrice())
	if payer.GetBalance().Cmp(maxFee) < 0 {
		return ErrInsufficientBalanceForGas
	}
	if err = gp.SubGas(tx.GasLimit()); err != nil {
		return err
	}
	payer.SetBalance(new(big.Int).Sub(payer.GetBalance(), maxFee))
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

func (p *TxProcessor) refundGas(gp *types.GasPool, tx *types.Transaction, restGas uint64) {
	// ignore the error because it is checked in buyGas
	payerAddr, _ := tx.GasPayer()
	payer := p.am.GetAccount(payerAddr)

	// Return LEMO for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(restGas), tx.GasPrice())
	payer.SetBalance(new(big.Int).Add(payer.GetBalance(), remaining))

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	gp.AddGas(restGas)
}

// chargeForGas change the gas to miner
func (p *TxProcessor) chargeForGas(charge *big.Int, minerAddress common.Address) {
	if charge.Cmp(new(big.Int)) != 0 {
		miner := p.am.GetAccount(minerAddress)
		miner.SetBalance(new(big.Int).Add(miner.GetBalance(), charge))
		// change in the number of votes cast by the miner's account to the candidate node
		p.changeCandidateVotes(minerAddress, charge)
	}
}

// CallTx pre-execute transactions and contracts.
func (p *TxProcessor) CallTx(ctx context.Context, header *types.Header, to *common.Address, txType uint16, data hexutil.Bytes, blockHash common.Hash, timeout time.Duration) ([]byte, uint64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	accM := account.ReadOnlyManager(header.Hash(), p.chain.db)
	accM.Reset(header.ParentHash)

	// A random address is found as our caller address.
	strAddress := "0x20190306" // todo Consider letting users pass in their own addresses
	caller, err := common.StringToAddress(strAddress)
	if err != nil {
		return nil, 0, err
	}
	// enough gasLimit
	gasLimit := uint64(math.MaxUint64 / 2)
	gasPrice := new(big.Int).SetUint64(defaultGasPrice)

	var tx *types.Transaction
	switch txType {
	case params.OrdinaryTx:
		if to == nil { // avoid null pointer references
			tx = types.NewContractCreation(big.NewInt(0), gasLimit, gasPrice, data, params.OrdinaryTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
		} else {
			tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.OrdinaryTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
		}
	case params.VoteTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.VoteTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.RegisterTx:
		tx = types.NewContractCreation(big.NewInt(0), gasLimit, gasPrice, data, params.RegisterTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.CreateAssetTx:
		tx = types.NoReceiverTransaction(big.NewInt(0), gasLimit, gasPrice, data, params.CreateAssetTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.IssueAssetTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.IssueAssetTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.ReplenishAssetTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.ReplenishAssetTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.ModifyAssetTx:
		tx = types.NoReceiverTransaction(big.NewInt(0), gasLimit, gasPrice, data, params.ModifyAssetTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.TradingAssetTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.TradingAssetTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	default:
		err = errors.New("tx type error")
		return nil, 0, err
	}

	// Timeout limit
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	Evm, vmError, sender := getEVM(ctx, caller, tx, header, 0, tx.Hash(), blockHash, p.chain, *p.cfg, accM)

	go func() {
		<-ctx.Done()
		Evm.Cancel()
	}()

	restGas := gasLimit
	// Fixed cost
	restGas, err = p.payIntrinsicGas(tx, restGas)
	if err != nil {
		return nil, 0, err
	}
	IsContractCreate := tx.To() == nil
	var ret []byte
	switch tx.Type() {
	case params.OrdinaryTx:
		if IsContractCreate {
			ret, _, restGas, err = Evm.Create(sender, tx.Data(), restGas, big.NewInt(0))
		} else {
			recipientAddr := *tx.To()
			ret, restGas, err = Evm.Call(sender, recipientAddr, tx.Data(), restGas, big.NewInt(0))
		}
	case params.VoteTx:
		recipientAddr := *tx.To()
		restGas, err = Evm.CallVoteTx(sender.GetAddress(), recipientAddr, restGas, sender.GetBalance())
	case params.RegisterTx:
		// Unmarshal tx data
		txData := tx.Data()
		profile := make(types.Profile)
		err = json.Unmarshal(txData, &profile)
		if err != nil {
			log.Errorf("unmarshal Candidate node error: %s", err)
			return nil, 0, err
		}

		if nodeId, ok := profile[types.CandidateKeyNodeID]; ok {
			nodeIdLength := len(nodeId)
			if nodeIdLength != StandardNodeIdLength {
				nodeIdErr := fmt.Errorf("the nodeId length [%d] is not equal the standard length [%d] ", nodeIdLength, StandardNodeIdLength)
				return nil, 0, nodeIdErr
			}
		}
		if host, ok := profile[types.CandidateKeyHost]; ok {
			hostLength := len(host)
			if hostLength > MaxDeputyHostLength {
				hostErr := fmt.Errorf("the length of host field in transaction is out of max length limit. host length = %d. max length limit = %d. ", hostLength, MaxDeputyHostLength)
				return nil, 0, hostErr
			}
		}
		sender = accM.GetAccount(caller)
		sender.SetBalance(params.RegisterCandidateNodeFees)
		restGas, err = Evm.RegisterOrUpdateToCandidate(sender.GetAddress(), params.FeeReceiveAddress, profile, restGas, sender.GetBalance())
	case params.CreateAssetTx, params.IssueAssetTx, params.ReplenishAssetTx, params.ModifyAssetTx, params.TradingAssetTx:

	}

	if err := vmError(); err != nil {
		return nil, 0, err
	}

	return ret, gasLimit - restGas, err
}

// getEVM
func getEVM(ctx context.Context, caller common.Address, tx *types.Transaction, header *types.Header, txIndex uint, txHash common.Hash, blockHash common.Hash, chain ChainContext, cfg vm.Config, accM *account.Manager) (*vm.EVM, func() error, types.AccountAccessor) {
	sender := accM.GetCanonicalAccount(caller)
	vmError := func() error { return nil }
	evmContext := NewEVMContext(tx, header, txIndex, txHash, blockHash, chain)
	vmEnv := vm.NewEVM(evmContext, accM, cfg)
	return vmEnv, vmError, sender
}
