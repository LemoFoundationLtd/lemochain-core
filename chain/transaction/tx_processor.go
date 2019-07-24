package transaction

import (
	"bytes"
	"context"
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
	defaultGasPrice       = 1e9
	MaxDeputyHostLength   = 128
	StandardNodeIdLength  = 128
	SignerWeightThreshold = 100
	MaxSignersNumber      = 100
)

var (
	ErrInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
	ErrInvalidTxInBlock          = errors.New("block contains invalid transaction")
	ErrInvalidGenesis            = errors.New("can't process genesis block")
	ErrInvalidHost               = errors.New("the length of host field in transaction is out of max length limit")
	ErrInvalidAddress            = errors.New("invalid address")
	ErrInvalidNodeId             = errors.New("invalid nodeId")
	ErrTxNotSign                 = errors.New("the transaction is not signed")
	ErrTotalWeight               = errors.New("insufficient total weight of signatories")
	ErrSignerAndFromUnequally    = errors.New("the signer and from of transaction are not equal")
	ErrGasPayer                  = errors.New("the gasPayer error")
	ErrSetMulisig                = errors.New("from and to must be equal")
	ErrAddressType               = errors.New("address type wrong")
	ErrTempAddress               = errors.New("the issuer part in temp address is incorrect")
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
		gas, err := p.applyTx(gp, header, tx, uint(i), header.Hash(), math.MaxInt64)
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

	now := time.Now() // 当前时间，用于计算箱子交易中执行子交易的限制时间
	// limit the time to execute txs
	applyTxsInterval := time.Duration(timeLimitSecond) * time.Millisecond // 单位: 纳秒
	// Iterate over and process the individual transactions
txsLoop:
	for _, tx := range txs {

		// 打包交易已用时间
		usedTime := time.Since(now) // 单位：纳秒
		// 计算还剩下多少时间来打包交易
		restApplyTime := int64(applyTxsInterval) - int64(usedTime)

		// 判断打包交易剩余时间
		if restApplyTime <= 0 {
			break txsLoop
		}

		// If we don't have enough gas for any further transactions then we're done
		if gp.Gas() < params.OrdinaryTxGas {
			log.Info("Not enough gas for further transactions", "gp", gp)
			break
		}
		// Start executing the transaction
		snap := p.am.Snapshot()

		gas, err := p.applyTx(gp, header, tx, uint(len(selectedTxs)), common.Hash{}, restApplyTime)
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

// checkSignersWeight 比较得到的签名者是否为预期的签名者
func (p *TxProcessor) checkSignersWeight(sender common.Address, tx *types.Transaction, interfaceSigner types.Signer) error {
	// 获取交易的签名者列表
	signers, err := interfaceSigner.GetSigners(tx)
	if err != nil {
		log.Errorf("Verification signature error")
		return err
	}
	if len(signers) == 0 {
		return ErrTxNotSign
	}
	// 获取账户的签名者列表
	accSigners := p.am.GetAccount(sender).GetSigners()
	length := len(accSigners)
	if length == 0 { // 非多签账户
		signer := signers[0]
		// 判断签名者是否为from
		if signer != sender {
			log.Errorf("The signer and from of transaction are not equal. Siger: %s. From: %s", signer.String(), sender.String())
			return ErrSignerAndFromUnequally
		}
	} else { // 多签账户
		signersMap := accSigners.ToSignerMap()
		// 计算签名者权重总和
		var totalWeight int64 = 0
		for _, addr := range signers {
			if w, ok := signersMap[addr]; ok {
				totalWeight = totalWeight + int64(w)
			}
		}
		// 比较签名权重总和大小
		if totalWeight < SignerWeightThreshold {
			return ErrTotalWeight
		}
	}
	return nil
}

// verifyTransactionSigs 验证交易签名
func (p *TxProcessor) verifyTransactionSigs(tx *types.Transaction) error {
	from := tx.From()
	gasPayerSigsLength := len(tx.GasPayerSigs())

	log.Infof("tx: %s", tx.String())
	// 验证gasPayer签名
	gasPayer := tx.GasPayer()
	if gasPayerSigsLength >= 1 {
		// 验证签名账户
		err := p.checkSignersWeight(gasPayer, tx, types.MakeGasPayerSigner())
		if err != nil {
			log.Errorf("gasPayer sigs error: %s", err)
			return err
		}
	} else {
		// 判断gasPayer字段是否为默认的from
		if gasPayer != from {
			log.Errorf("The default transaction gasPayer must equal the from. gasPayer: %s. from: %s", tx.GasPayer().String(), from.String())
			return ErrGasPayer
		}
	}

	// 验证from签名
	var fromSigner types.Signer
	if gasPayerSigsLength >= 1 {
		fromSigner = types.MakeReimbursementTxSigner()
	} else {
		fromSigner = types.MakeSigner()
	}

	err := p.checkSignersWeight(from, tx, fromSigner)
	if err != nil {
		log.Errorf("from sigs error: %s", err)
		return err
	}
	return nil
}

// applyTx processes transaction. Change accounts' data and execute contract codes.
func (p *TxProcessor) applyTx(gp *types.GasPool, header *types.Header, tx *types.Transaction, txIndex uint, blockHash common.Hash, restApplyTime int64) (uint64, error) {
	// 验证交易的签名
	err := p.verifyTransactionSigs(tx)
	if err != nil {
		return 0, err
	}

	var (
		senderAddr           = tx.From()
		sender               = p.am.GetAccount(senderAddr)
		initialSenderBalance = sender.GetBalance() // initialSenderBalance参数代表的是sender执行交易之前的balance值，为投票交易中计算初始票数使用
		restGas              = tx.GasLimit()
		vmErr, execErr       error
		gasUsed              uint64
	)

	restGas, err = p.buyAndPayIntrinsicGas(gp, tx, restGas)
	if err != nil {
		return 0, err
	}
	// 执行交易. 注：如果此交易为箱子交易，则返回的gasUsed为箱子中的子交易消耗gas与箱子交易本身消耗gas之和
	restGas, gasUsed, vmErr, execErr = p.handleTx(tx, header, txIndex, blockHash, initialSenderBalance, restGas, gp, restApplyTime)
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

	return gasUsed, nil
}

// handleTx 执行交易,返回消耗之后剩余的gas、evm中执行的error和交易执行不成功的error，注：initialSenderBalance参数代表的是sender执行交易之前的balance值，为投票交易中计算初始票数使用
func (p *TxProcessor) handleTx(tx *types.Transaction, header *types.Header, txIndex uint, blockHash common.Hash, initialSenderBalance *big.Int, restGas uint64, gp *types.GasPool, restApplyTime int64) (gas, gasUsed uint64, vmErr, err error) {
	senderAddr := tx.From()
	var (
		recipientAddr common.Address
		sender        = p.am.GetAccount(senderAddr)
		nullRecipient = tx.To() == nil
		gasLimit      = tx.GasLimit()
		subTxsGasUsed uint64 // 箱子中的交易gas used
	)
	if !nullRecipient {
		recipientAddr = *tx.To()
	}

	// Judge the type of transaction
	switch tx.Type() {
	case params.OrdinaryTx:
		newContext := NewEVMContext(tx, header, txIndex, blockHash, p.blockLoader)
		vmEnv := vm.NewEVM(newContext, p.am, *p.cfg)
		_, restGas, vmErr = vmEnv.Call(sender, recipientAddr, tx.Data(), restGas, tx.Amount())
	case params.CreateContractTx:
		newContext := NewEVMContext(tx, header, txIndex, blockHash, p.blockLoader)
		vmEnv := vm.NewEVM(newContext, p.am, *p.cfg)
		_, recipientAddr, restGas, vmErr = vmEnv.Create(sender, tx.Data(), restGas, tx.Amount())
	case params.VoteTx:
		candidateVoteEnv := NewCandidateVoteEnv(p.am)
		err = candidateVoteEnv.CallVoteTx(senderAddr, recipientAddr, initialSenderBalance)

	case params.RegisterTx:
		candidateVoteEnv := NewCandidateVoteEnv(p.am)
		err = candidateVoteEnv.RegisterOrUpdateToCandidate(tx)

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
	case params.ModifySignersTx:
		multisigEnv := NewSetMultisigAccountEnv(p.am)
		err = multisigEnv.ModifyMultisigTx(senderAddr, recipientAddr, tx.Data())
	case params.BoxTx:
		boxEnv := NewBoxTxEnv(p)
		// 返回箱子中子交易消耗的总gas
		subTxsGasUsed, err = boxEnv.RunBoxTxs(gp, tx, header, txIndex, restApplyTime)

	default:
		log.Errorf("The type of transaction is not defined. ErrType = %d\n", tx.Type())
		return 0, 0, nil, types.ErrTxType
	}
	// 只有交易类型为BoxTx时，subTxsGasUsed才有值
	gasUsed = gasLimit - restGas + subTxsGasUsed

	return restGas, gasUsed, vmErr, err
}

func (p *TxProcessor) buyGas(gp *types.GasPool, tx *types.Transaction) error {
	payerAddr := tx.GasPayer()
	log.Infof("Tx's gas payer address: %s", payerAddr.String())

	payer := p.am.GetAccount(payerAddr)

	maxFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit()), tx.GasPrice())
	if payer.GetBalance().Cmp(maxFee) < 0 {
		return ErrInsufficientBalanceForGas
	}
	if err := gp.SubGas(tx.GasLimit()); err != nil {
		return err
	}
	payer.SetBalance(new(big.Int).Sub(payer.GetBalance(), maxFee))
	return nil
}

func (p *TxProcessor) payIntrinsicGas(tx *types.Transaction, restGas uint64) (uint64, error) {
	gas, err := IntrinsicGas(tx.Type(), tx.Data(), tx.Message())
	if err != nil {
		return restGas, err
	}
	if restGas < gas {
		return restGas, vm.ErrOutOfGas
	}
	return restGas - gas, nil
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(txType uint16, data []byte, txMessage string) (uint64, error) {
	// Set the starting gas for the raw transaction
	gas, err := getTxBaseSpendGas(txType)
	if err != nil {
		return 0, err
	}
	// calculate txData spend gas and  add it and return
	return addTxDataSpendGas(data, txMessage, gas)
}

// getTxBaseSpendGas 获取不同类型的交易需要花费的基础固定gas
func getTxBaseSpendGas(txType uint16) (uint64, error) {
	var gas uint64
	switch txType {
	case params.OrdinaryTx:
		gas = params.OrdinaryTxGas
	case params.CreateContractTx:
		gas = params.TxGasContractCreation
	case params.VoteTx:
		gas = params.VoteTxGas
	case params.RegisterTx:
		gas = params.RegisterTxGas
	case params.CreateAssetTx:
		gas = params.CreateAssetTxGas
	case params.IssueAssetTx:
		gas = params.IssueAssetTxGas
	case params.ReplenishAssetTx:
		gas = params.ReplenishAssetTxGas
	case params.ModifyAssetTx:
		gas = params.ModifyAssetTxGas
	case params.TransferAssetTx:
		gas = params.TransferAssetTxGas
	case params.ModifySignersTx:
		gas = params.ModifySigsTxGas
	case params.BoxTx:
		gas = params.BoxTxGas
	default:
		log.Errorf("Transaction type is not exist. error type: %d", txType)
		return 0, types.ErrTxType
	}
	return gas, nil
}

// addTxDataSpendGas 加上交易data消耗的固定gas
func addTxDataSpendGas(data []byte, message string, gas uint64) (uint64, error) {
	// 计算tx中message消耗的gas
	messageLen := len(message)
	if messageLen > 0 {
		gas += uint64(messageLen) * params.TxMessageGas
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
	payerAddr := tx.GasPayer()
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
	}
}

// PreExecutionTransaction pre-execute transactions and contracts.
func (p *TxProcessor) PreExecutionTransaction(ctx context.Context, accM *account.ReadOnlyManager, header *types.Header, to *common.Address, txType uint16, data hexutil.Bytes, blockHash common.Hash, timeout time.Duration) ([]byte, uint64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	accM.Reset(header.Hash())

	// A random address is found as our caller address.
	strAddress := "0x20190306" // todo Consider letting users pass in their own addresses
	caller, err := common.StringToAddress(strAddress)
	if err != nil {
		return nil, 0, err
	}

	tx := newTx(caller, to, txType, data, p.ChainID)
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
	case params.OrdinaryTx, params.CreateContractTx, params.TransferAssetTx: // need use evm environment
		vmEvn = getEVM(tx, header, 0, tx.Hash(), blockHash, p.blockLoader, *p.cfg, accM)

	case params.RegisterTx, params.VoteTx, params.CreateAssetTx, params.IssueAssetTx, params.ReplenishAssetTx, params.ModifyAssetTx, params.ModifySignersTx, params.BoxTx:
		// 	todo 箱子交易的预估gas需要执行交易来预估，待做...

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

	var ret []byte
	switch tx.Type() {
	case params.OrdinaryTx:
		recipientAddr := *tx.To()
		ret, restGas, err = vmEvn.Call(sender, recipientAddr, tx.Data(), restGas, big.NewInt(0))
	case params.CreateContractTx:
		ret, _, restGas, err = vmEvn.Create(sender, tx.Data(), restGas, big.NewInt(0))
	case params.TransferAssetTx:
		tradingAsset, err := types.GetTradingAsset(tx.Data())
		if err != nil {
			log.Errorf("Unmarshal transfer asset data err: %s", err)
			return nil, 0, err
		}
		input := tradingAsset.Input
		if input == nil || bytes.Compare(input, []byte{}) == 0 {
			break
		}
		ret, restGas, err = vmEvn.CallCode(sender, *tx.To(), input, restGas, big.NewInt(0))

	case params.RegisterTx, params.VoteTx, params.CreateAssetTx, params.IssueAssetTx, params.ReplenishAssetTx, params.ModifyAssetTx, params.ModifySignersTx, params.BoxTx:

	default:
		log.Errorf("The type of transaction is not defined. ErrType = %d\n", tx.Type())
	}
	return ret, gasLimit - restGas, err
}

// newTx return created transaction
func newTx(from common.Address, to *common.Address, txType uint16, data []byte, chainID uint16) *types.Transaction {
	// enough gasLimit
	gasLimit := uint64(math.MaxUint64 / 2)
	gasPrice := new(big.Int).SetUint64(defaultGasPrice)
	var tx *types.Transaction
	if to == nil {
		tx = types.NoReceiverTransaction(from, big.NewInt(0), gasLimit, gasPrice, data, txType, chainID, uint64(time.Now().Unix())+uint64(params.TransactionExpiration), "", "")
	} else {
		tx = types.NewTransaction(from, *to, big.NewInt(0), gasLimit, gasPrice, data, txType, chainID, uint64(time.Now().Unix())+uint64(params.TransactionExpiration), "", "")
	}

	return tx
}

// getEVM
func getEVM(tx *types.Transaction, header *types.Header, txIndex uint, txHash common.Hash, blockHash common.Hash, chain BlockLoader, cfg vm.Config, accM vm.AccountManager) *vm.EVM {
	evmContext := NewEVMContext(tx, header, txIndex, blockHash, chain)
	vmEnv := vm.NewEVM(evmContext, accM, cfg)
	return vmEnv
}

// SetCandidateVotesByChangeBalance 设置余额变化导致的候选节点票数的变化
func SetCandidateVotesByChangeBalance(am *account.Manager) {
	changes := votesChangeByBalanceChangelog(am)
	for addr, changeVotes := range changes {
		changeCandidateVotes(am, addr, changeVotes)
	}
}

// changeCandidateVotes candidate node vote change corresponding to votes change
func changeCandidateVotes(am *account.Manager, accountAddress common.Address, changeVotes *big.Int) {
	if changeVotes.Sign() == 0 {
		return
	}
	acc := am.GetAccount(accountAddress)
	candidataAddress := acc.GetVoteFor()

	if (candidataAddress == common.Address{}) {
		return
	}
	candidateAccount := am.GetAccount(candidataAddress)
	// 判断是否为候选节点
	if candidateAccount.GetCandidateState(types.CandidateKeyIsCandidate) == params.IsCandidateNode {
		// set votes
		candidateAccount.SetVotes(new(big.Int).Add(candidateAccount.GetVotes(), changeVotes))
	}
}

type votesChange map[common.Address]*big.Int // 记录账户balance变化之后换算出的票数变化
// 通过changelog获取账户的余额变化,并进行计算出票数的变化
func votesChangeByBalanceChangelog(am *account.Manager) votesChange {
	// 获取所有的changelog
	logs := am.GetChangeLogs()

	// 筛选出同一个账户的balanceLog并merge
	balanceLogsByAddress := make(map[common.Address]*types.ChangeLog, len(logs))
	for _, log := range logs {
		if log.LogType == account.BalanceLog {
			// merge
			if _, ok := balanceLogsByAddress[log.Address]; !ok {
				balanceLogsByAddress[log.Address] = log.Copy()
			} else {
				balanceLogsByAddress[log.Address].NewVal = log.NewVal
			}
		}
	}
	// 根据balance变化得到vote变化
	votesChange := make(votesChange, len(balanceLogsByAddress))
	for addr, newLog := range balanceLogsByAddress {
		newValue := newLog.NewVal.(big.Int)
		oldValue := newLog.OldVal.(big.Int)
		oldNum := new(big.Int).Div(&oldValue, params.VoteExchangeRate) // oldBalance兑换出来的票数
		newNum := new(big.Int).Div(&newValue, params.VoteExchangeRate) // newBalance兑换出来的票数
		// 如果余额变化未能导致票数的变化则不进行修改票数的操作
		if newNum.Cmp(oldNum) == 0 {
			continue
		}
		changeNum := new(big.Int).Sub(newNum, oldNum)
		votesChange[addr] = changeNum
	}
	return votesChange
}
