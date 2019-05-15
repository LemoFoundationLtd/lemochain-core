package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/math"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
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
	ErrInvalidGenesis            = errors.New("can't process genesis block")
	ErrInvalidHost               = errors.New("the length of host field in transaction is out of max length limit")
	ErrInvalidAddress            = errors.New("invalid address")
	ErrInvalidNodeId             = errors.New("invalid nodeId")
)

type TxProcessor struct {
	ChainID     uint16
	blockLoader BlockLoader
	am          *account.Manager
	db          protocol.ChainDB
	cfg         *vm.Config // configuration of vm

	lock sync.Mutex
}

func NewTxProcessor(issueRewardAddress common.Address, chainID uint16, blockLoader BlockLoader, am *account.Manager, db protocol.ChainDB) *TxProcessor {
	cfg := &vm.Config{
		Debug:         false,
		RewardManager: issueRewardAddress,
	}
	return &TxProcessor{
		ChainID:     chainID,
		blockLoader: blockLoader,
		am:          am,
		db:          db,
		cfg:         cfg,
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

	// Process genesis block. It's a develop error
	if header.Height == 0 {
		log.Warn("It is not necessary to process genesis block.")
		panic(ErrInvalidGenesis)
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

	p.am.MergeChangeLogs()

	if len(txs) > 0 {
		log.Infof("Process %d transactions", len(txs))
	}
	return gasUsed, nil
}

// ApplyTxs picks and processes transactions from miner's tx pool.
func (p *TxProcessor) ApplyTxs(header *types.Header, txs types.Transactions, timeLimitSecond int64) (types.Transactions, types.Transactions, uint64) {
	var (
		gp          = new(types.GasPool).AddGas(header.GasLimit)
		gasUsed     = uint64(0)
		totalGasFee = new(big.Int)
		selectedTxs = make(types.Transactions, 0)
		invalidTxs  = make(types.Transactions, 0)
	)

	p.am.Reset(header.ParentHash)

	// limit the time to execute txs
	applyTxsInterval := time.Duration(timeLimitSecond) * time.Millisecond
	applyTimer := time.NewTimer(applyTxsInterval)
	// Iterate over and process the individual transactions
label:
	for _, tx := range txs {
		// timer
		select {
		case <-applyTimer.C:
			break label
		default:
		}
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
	p.am.MergeChangeLogs()

	if len(selectedTxs) > 0 {
		log.Infof("Process %d transactions", len(selectedTxs))
	}
	return selectedTxs, invalidTxs, gasUsed
}

// buyAndPayIntrinsicGas
func (p *TxProcessor) buyAndPayIntrinsicGas(gp *types.GasPool, tx *types.Transaction, gasLimit uint64) (uint64, error) {
	err := p.buyGas(gp, tx)
	if err != nil {
		return 0, err
	}
	restGas, err := p.payIntrinsicGas(tx, gasLimit)
	if err != nil {
		return 0, err
	}
	return restGas, nil
}

// applyTx processes transaction. Change accounts' data and execute contract codes.
func (p *TxProcessor) applyTx(gp *types.GasPool, header *types.Header, tx *types.Transaction, txIndex uint, blockHash common.Hash) (uint64, error) {
	senderAddr, err := tx.From()
	if err != nil {
		return 0, err
	}
	var (
		sender               = p.am.GetAccount(senderAddr)
		initialSenderBalance = sender.GetBalance()
		restGas              = tx.GasLimit()
		vmErr, execErr       error
	)

	restGas, err = p.buyAndPayIntrinsicGas(gp, tx, restGas)
	if err != nil {
		return 0, err
	}
	// 执行交易
	restGas, vmErr, execErr = p.handleTx(tx, header, txIndex, blockHash, initialSenderBalance, restGas)
	if execErr != nil {
		log.Errorf("Apply transaction failure. error:%s, transaction: %s.", execErr.Error(), tx.String())
		return 0, execErr
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

	gasUsed := tx.GasLimit() - restGas
	// 余额变化造成的候选节点的票数变化
	p.setCandidateVotesByChangeBalance()

	return gasUsed, nil
}

// setCandidateVotesByChangeBalance 设置余额变化导致的候选节点票数的变化
func (p *TxProcessor) setCandidateVotesByChangeBalance() {
	changes := p.getBalanceChangeBychangelog()
	for addr, changeBalance := range changes {
		p.changeCandidateVotes(addr, changeBalance)
	}
}

type balanceChange map[common.Address]*big.Int

// 通过changelog获取账户的余额变化
func (p *TxProcessor) getBalanceChangeBychangelog() balanceChange {
	copyLogs := make(types.ChangeLogSlice, 0)
	// 获取所以的changelog
	logs := p.am.GetChangeLogs()
	// copy
	for _, log := range logs {
		copyLogs = append(copyLogs, log.Copy())
	}
	// 筛选出同一个账户的balanceLog
	balanceLogsByAddress := make(map[common.Address]types.ChangeLogSlice)
	for _, log := range copyLogs {
		// BalanceLog
		if log.LogType == account.BalanceLog {
			balanceLogsByAddress[log.Address] = append(balanceLogsByAddress[log.Address], log)
		}
	}
	// merge BalanceLogs
	newBalanceLogByAddr := make(map[common.Address]*types.ChangeLog)
	for addr, logs := range balanceLogsByAddress {
		if len(logs) == 1 { // 不用merge
			newBalanceLogByAddr[addr] = logs[0]
		} else {
			newLogs := mergeBalanceLogs(logs)
			newBalanceLogByAddr[addr] = newLogs[0]
		}
	}
	// 获取balance change
	balanceChange := make(balanceChange)
	for addr, newLog := range newBalanceLogByAddr {
		newValue := newLog.NewVal.(big.Int)
		oldValue := newLog.OldVal.(big.Int)
		change := new(big.Int).Sub(&newValue, &oldValue)
		balanceChange[addr] = change
	}
	return balanceChange
}

// merge balanceLog
func mergeBalanceLogs(logs types.ChangeLogSlice) types.ChangeLogSlice {
	newLogs := make(types.ChangeLogSlice, 0, 1)
	for _, balanceLog := range logs {
		if len(newLogs) == 0 {
			newLogs = append(newLogs, balanceLog)
		} else {
			newLogs[0].NewVal = balanceLog.NewVal
		}
	}
	return newLogs
}

// handleTx 执行交易,返回消耗之后剩余的gas、evm中执行的error和交易执行不成功的error
func (p *TxProcessor) handleTx(tx *types.Transaction, header *types.Header, txIndex uint, blockHash common.Hash, initialSenderBalance *big.Int, restGas uint64) (gas uint64, vmErr, err error) {
	senderAddr, err := tx.From()
	if err != nil {
		return 0, nil, err
	}
	var (
		recipientAddr common.Address
		sender        = p.am.GetAccount(senderAddr)
		nullRecipient = tx.To() == nil
	)
	if !nullRecipient {
		recipientAddr = *tx.To()
	}

	// Judge the type of transaction
	switch tx.Type() {
	case params.OrdinaryTx:
		newContext := NewEVMContext(tx, header, txIndex, blockHash, p.blockLoader)
		vmEnv := vm.NewEVM(newContext, p.am, *p.cfg)
		if nullRecipient {
			_, recipientAddr, restGas, vmErr = vmEnv.Create(sender, tx.Data(), restGas, tx.Amount())
		} else {
			_, restGas, vmErr = vmEnv.Call(sender, recipientAddr, tx.Data(), restGas, tx.Amount())
		}

	case params.VoteTx:
		candidateVoteEnv := NewCandidateVoteEnv(p.am)
		err = candidateVoteEnv.CallVoteTx(senderAddr, recipientAddr, initialSenderBalance)

	case params.RegisterTx:
		candidateVoteEnv := NewCandidateVoteEnv(p.am)
		err = candidateVoteEnv.RegisterOrUpdateToCandidate(tx, initialSenderBalance)

	case params.CreateAssetTx:
		assetEnv := NewRunAssetEnv(p.am)
		err = assetEnv.CreateAssetTx(senderAddr, tx.Data(), tx.Hash())

	case params.IssueAssetTx:
		assetEnv := NewRunAssetEnv(p.am)
		err = assetEnv.IssueAssetTx(senderAddr, recipientAddr, tx.Hash(), tx.Data())

	case params.ReplenishAssetTx:
		assetEnv := NewRunAssetEnv(p.am)
		err = assetEnv.ReplenishAssetTx(senderAddr, recipientAddr, tx.Data())

	case params.ModifyAssetTx:
		assetEnv := NewRunAssetEnv(p.am)
		err = assetEnv.ModifyAssetProfileTx(senderAddr, tx.Data())

	case params.TransferAssetTx:
		newContext := NewEVMContext(tx, header, txIndex, blockHash, p.blockLoader)
		vmEnv := vm.NewEVM(newContext, p.am, *p.cfg)
		_, restGas, err, vmErr = vmEnv.TransferAssetTx(sender, recipientAddr, restGas, tx.Data(), p.db)

	default:
		log.Errorf("The type of transaction is not defined. ErrType = %d\n", tx.Type())
		return restGas, vmErr, types.ErrTxType
	}
	return restGas, vmErr, err
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
	log.Infof("Tx's gas payer address: %s", payerAddr.String())
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
		// find income address
		miner := p.am.GetAccount(minerAddress)
		profile := miner.GetCandidate()
		strIncomeAddress, ok := profile[types.CandidateKeyIncomeAddress]
		if !ok {
			log.Errorf("IncomeAddress is null when charge gas for minerAddress. minerAddress = %s", minerAddress.String())
			return
		}
		incomeAddress, err := common.StringToAddress(strIncomeAddress)
		if err != nil {
			log.Errorf("Get incomeAddress error by strIncomeAddress or incomeAddress invalid; strIncomeAddress = %s,minerAddress = %s", strIncomeAddress, minerAddress)
			return
		}
		// get income account
		incomeAcc := p.am.GetAccount(incomeAddress)
		// get charge
		incomeAcc.SetBalance(new(big.Int).Add(incomeAcc.GetBalance(), charge))
		// change in the number of votes cast by the miner's account to the candidate node
		p.changeCandidateVotes(incomeAddress, charge)
	}
}

// PreExecutionTransaction pre-execute transactions and contracts.
func (p *TxProcessor) PreExecutionTransaction(ctx context.Context, header *types.Header, to *common.Address, txType uint16, data hexutil.Bytes, blockHash common.Hash, timeout time.Duration) ([]byte, uint64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	accM := account.NewReadOnlyManager(p.db)
	accM.Reset(header.Hash())

	// A random address is found as our caller address.
	strAddress := "0x20190306" // todo Consider letting users pass in their own addresses
	caller, err := common.StringToAddress(strAddress)
	if err != nil {
		return nil, 0, err
	}

	tx, err := newTx(to, txType, data, p.ChainID)
	if err != nil {
		return nil, 0, err
	}
	// Timeout limit
	var (
		cancel context.CancelFunc
		vmEvn  *vm.EVM
		sender types.AccountAccessor
	)
	sender = accM.GetAccount(caller)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// load different Env
	switch tx.Type() {
	case params.OrdinaryTx, params.TransferAssetTx: // need use evm environment
		vmEvn = getEVM(tx, header, 0, tx.Hash(), blockHash, p.blockLoader, *p.cfg, accM)

	case params.RegisterTx, params.VoteTx, params.CreateAssetTx, params.IssueAssetTx, params.ReplenishAssetTx, params.ModifyAssetTx:
	// case params.ModifyAssetTx, params.ReplenishAssetTx, params.IssueAssetTx, params.CreateAssetTx: // use asset tx environment
	// 	assetEnv = NewRunAssetEnv(p.am)
	default:
		log.Errorf("The type of transaction is not defined. ErrType = %d\n", tx.Type())
	}

	// listen timeout
	go func() {
		<-ctx.Done()
		vmEvn.Cancel()
	}()

	gasLimit := tx.GasLimit()
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
			ret, _, restGas, err = vmEvn.Create(sender, tx.Data(), restGas, big.NewInt(0))
		} else {
			recipientAddr := *tx.To()
			ret, restGas, err = vmEvn.Call(sender, recipientAddr, tx.Data(), restGas, big.NewInt(0))
		}
	case params.TransferAssetTx:
		tradingAsset := &types.TradingAsset{}
		err := json.Unmarshal(tx.Data(), tradingAsset)
		if err != nil {
			log.Errorf("Unmarshal transfer asset data err: %s", err)
			return nil, 0, err
		}
		input := tradingAsset.Input
		if input == nil || bytes.Compare(input, []byte{}) == 0 {
			break
		}
		ret, restGas, err = vmEvn.CallCode(sender, *tx.To(), input, restGas, big.NewInt(0))

	case params.RegisterTx, params.VoteTx, params.CreateAssetTx, params.IssueAssetTx, params.ReplenishAssetTx, params.ModifyAssetTx:
	}
	return ret, gasLimit - restGas, err
}

// newTx return created transaction
func newTx(to *common.Address, txType uint16, data []byte, chainID uint16) (*types.Transaction, error) {
	// enough gasLimit
	gasLimit := uint64(math.MaxUint64 / 2)
	gasPrice := new(big.Int).SetUint64(defaultGasPrice)

	var tx *types.Transaction
	switch txType {
	case params.OrdinaryTx:
		if to == nil { // avoid null pointer references
			tx = types.NewContractCreation(big.NewInt(0), gasLimit, gasPrice, data, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
		} else {
			tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
		}
	case params.VoteTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.VoteTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.RegisterTx:
		tx = types.NewContractCreation(big.NewInt(0), gasLimit, gasPrice, data, params.RegisterTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.CreateAssetTx:
		tx = types.NoReceiverTransaction(big.NewInt(0), gasLimit, gasPrice, data, params.CreateAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.IssueAssetTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.IssueAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.ReplenishAssetTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.ReplenishAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.ModifyAssetTx:
		tx = types.NoReceiverTransaction(big.NewInt(0), gasLimit, gasPrice, data, params.ModifyAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
	case params.TransferAssetTx:
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.TransferAssetTx, chainID, uint64(time.Now().Unix()+30*60), "", "")
	default:
		err := errors.New("tx type error")
		return nil, err
	}
	return tx, nil
}

// getEVM
func getEVM(tx *types.Transaction, header *types.Header, txIndex uint, txHash common.Hash, blockHash common.Hash, chain BlockLoader, cfg vm.Config, accM *account.Manager) *vm.EVM {
	evmContext := NewEVMContext(tx, header, txIndex, blockHash, chain)
	vmEnv := vm.NewEVM(evmContext, accM, cfg)
	return vmEnv
}
