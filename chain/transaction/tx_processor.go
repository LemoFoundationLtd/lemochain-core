package transaction

import (
	"context"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/math"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"sync"
	"time"
)

const (
	defaultGasPrice                  = 1e9
	NodeIDFieldLength                = 130
	MaxProfileFieldLength            = 128
	MaxIntroductionLength            = 1024
	MaxMarshalCandidateProfileLength = 1200 // candidate 中profile marshal之后得到的byte数组的最大长度
	StandardNodeIdLength             = 64
	SignerWeightThreshold            = 100
	MaxSignersNumber                 = 100
)

var (
	invalidTxMeter = metrics.NewMeter(metrics.InvalidTx_meterName) // 执行失败的交易的频率

	ErrInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
	ErrInvalidTxInBlock          = errors.New("block contains invalid transaction")
	ErrTxGasUsedNotEqual         = errors.New("tx gas used not equal")
	ErrInvalidGenesis            = errors.New("can't process genesis block")
	ErrInvalidProfile            = errors.New("the length of candidate profile field in transaction is out of max length limit")
	ErrInvalidPort               = errors.New("invalid port")
	ErrInvalidIntroduction       = errors.New("the length of introduction field in transaction is out of max length limit")
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
	blockLoader ParentBlockLoader
	am          *account.Manager
	dm          *deputynode.Manager
	db          protocol.ChainDB
	cfg         *vm.Config // configuration of vm

	lock sync.Mutex
}

func NewTxProcessor(issueRewardAddress common.Address, chainID uint16, blockLoader ParentBlockLoader, am *account.Manager, db protocol.ChainDB, dm *deputynode.Manager) *TxProcessor {
	cfg := &vm.Config{
		Debug:         false,
		RewardManager: issueRewardAddress,
	}
	return &TxProcessor{
		ChainID:     chainID,
		blockLoader: blockLoader,
		am:          am,
		dm:          dm,
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
		if tx.GasUsed() != gas {
			log.Errorf("Transaction gas used not equal.oldGasUsed: %d, newGasUsed: %d", tx.GasUsed(), gas)
			return gasUsed, ErrTxGasUsedNotEqual
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
		selectedTxs = make(types.Transactions, 0, len(txs))
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
				invalidTxMeter.Mark(1) // 标记一笔交易执行失败
				// Strange error, discard the transaction and get the next in line.
				log.Info("Skipped invalid transaction", "hash", tx.Hash(), "err", err)
				invalidTxs = append(invalidTxs, tx)
			}
			continue
		}
		tx.SetGasUsed(gas)
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

// VerifyAssetTx
func (p *TxProcessor) VerifyAssetTx(tx *types.Transaction) error {
	// 获取资产code
	assetCode := common.Hash{}
	switch tx.Type() {
	case params.IssueAssetTx:
		issueAsset, err := types.GetIssueAsset(tx.Data())
		if err != nil {
			return err
		}
		assetCode = issueAsset.AssetCode
	case params.ReplenishAssetTx:
		repl, err := types.GetReplenishAsset(tx.Data())
		if err != nil {
			return err
		}
		assetCode = repl.AssetCode
	case params.ModifyAssetTx:
		modifyInfo, err := types.GetModifyAssetInfo(tx.Data())
		if err != nil {
			return err
		}
		assetCode = modifyInfo.AssetCode
	case params.TransferAssetTx:
		TradingAssetInfo, err := types.GetTradingAsset(tx.Data())
		if err != nil {
			return err
		}
		assetId := TradingAssetInfo.AssetId
		// 查询是否存在此资产
		issueAcc := p.am.GetCanonicalAccount(tx.From())
		_, err = issueAcc.GetAssetIdState(assetId)
		return err
	default:
		return nil
	}

	if (assetCode != common.Hash{}) {
		// 在稳定块中查询此资产是否已经被创建上链
		issueAcc := p.am.GetCanonicalAccount(tx.From())
		_, err := issueAcc.GetAssetCode(assetCode)
		return err
	}
	return nil
}

// VerifyTxBeforeApply 执行交易之前的交易校验
func (p *TxProcessor) VerifyTxBeforeApply(tx *types.Transaction) error {
	// 验证资产交易依赖
	if err := p.VerifyAssetTx(tx); err != nil {
		return err
	}
	// 验证交易的签名
	if err := p.verifyTransactionSigs(tx); err != nil {
		return err
	}
	return nil
}

// applyTx processes transaction. Change accounts' data and execute contract codes.
func (p *TxProcessor) applyTx(gp *types.GasPool, header *types.Header, tx *types.Transaction, txIndex uint, blockHash common.Hash, restApplyTime int64) (uint64, error) {
	// 执行交易之前的交易校验
	err := p.VerifyTxBeforeApply(tx)
	if err != nil {
		return 0, err
	}

	var (
		senderAddr = tx.From()
		sender     = p.am.GetAccount(senderAddr)
		// initialSenderBalance参数代表的是sender执行交易之前的balance值，为投票交易中计算初始票数使用
		initialSenderBalance = sender.GetBalance()
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

// handleTx 执行交易,返回消耗之后剩余的gas、evm中执行的error和交易执行不成功的error.
// 注：initialSenderBalance参数代表的是sender执行交易之前的balance值，为投票交易中计算初始票数使用
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
		candidateVoteEnv := NewCandidateVoteEnv(p.am, p.dm)
		err = candidateVoteEnv.CallVoteTx(senderAddr, recipientAddr, initialSenderBalance)

	case params.RegisterTx:
		candidateVoteEnv := NewCandidateVoteEnv(p.am, p.dm)
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
	// 如果为箱子交易，则不计算箱子交易的data。因为如果是进行验证区块中的箱子交易操作，箱子交易已经被执行过了，其data会改变(子交易的gasUsed被赋值).
	// 而且箱子中的子交易本身自己会扣除相应的gas，所以箱子本身就没有必要再次扣除data字段的gas了
	if txType == params.BoxTx {
		data = []byte{}
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

// ReadContract pre-execute transactions and contracts.
func (p *TxProcessor) ReadContract(accM *account.ReadOnlyManager, header *types.Header, to common.Address, data hexutil.Bytes, timeout time.Duration) ([]byte, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	accM.Reset(header.Hash())

	// A random address is found as our caller address.
	// todo Consider let users pass in their own address
	caller := common.HexToAddress("0x20190306")

	// enough gasLimit
	gasLimit := uint64(math.MaxUint64 / 2)
	tx := types.NewTransaction(caller, to, big.NewInt(0), gasLimit, big.NewInt(defaultGasPrice), data, params.OrdinaryTx, p.ChainID, uint64(time.Now().Unix())+uint64(params.TransactionExpiration), "", "")

	var (
		ctx    context.Context
		cancel context.CancelFunc
		sender = accM.GetAccount(caller)
		vmEvn  = getEVM(tx, header, 0, common.Hash{}, p.blockLoader, *p.cfg, accM)
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// listen timeout
	go func() {
		<-ctx.Done()
		vmEvn.Cancel()
	}()

	ret, _, err := vmEvn.Call(sender, to, tx.Data(), gasLimit, big.NewInt(0))
	return ret, err
}

// getEVM
func getEVM(tx *types.Transaction, header *types.Header, txIndex uint, blockHash common.Hash, chain ParentBlockLoader, cfg vm.Config, accM vm.AccountManager) *vm.EVM {
	evmContext := NewEVMContext(tx, header, txIndex, blockHash, chain)
	vmEnv := vm.NewEVM(evmContext, accM, cfg)
	return vmEnv
}

type votesChange map[common.Address]*big.Int // 记录账户balance变化之后换算出的票数变化
type balanceLogAddressMap map[common.Address]*types.ChangeLog

// ChangeVotesByBalance 设置余额变化导致的候选节点票数的变化
func ChangeVotesByBalance(am *account.Manager) {
	changes := votesChangeByBalanceLog(am)
	for addr, changeVotes := range changes {
		changeCandidateVotes(am, addr, changeVotes)
	}
}

// votesChangeByBalanceLog 通过changelog获取账户的余额变化,并进行计算出票数的变化
func votesChangeByBalanceLog(am *account.Manager) votesChange {
	// 获取所有的changelog
	logs := am.GetChangeLogs()
	// 筛选出同一个账户的balanceLog并merge
	balanceLogsByAddress := filterLogs(logs, account.BalanceLog)

	// 根据balance变化得到vote变化
	return getVotesChangesByLogs(balanceLogsByAddress)
}

// getVotesChangesByLogs
func getVotesChangesByLogs(balanceLogs balanceLogAddressMap) votesChange {
	votesChange := make(votesChange, len(balanceLogs))
	for addr, newLog := range balanceLogs {
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

// filterLogs
func filterLogs(logs types.ChangeLogSlice, logType types.ChangeLogType) balanceLogAddressMap {
	logsByAddress := make(balanceLogAddressMap, len(logs))
	for _, log := range logs {
		if log.LogType == logType {
			// merge
			if _, ok := logsByAddress[log.Address]; !ok {
				logsByAddress[log.Address] = log.Copy()
			} else {
				logsByAddress[log.Address].NewVal = log.NewVal
			}
		}
	}
	return logsByAddress
}

// changeCandidateVotes candidate node vote change corresponding to votes change
func changeCandidateVotes(am *account.Manager, accountAddress common.Address, changeVotes *big.Int) {
	if changeVotes.Sign() == 0 {
		return
	}
	acc := am.GetAccount(accountAddress)
	candidateAddress := acc.GetVoteFor()

	if (candidateAddress == common.Address{}) {
		return
	}
	candidateAccount := am.GetAccount(candidateAddress)
	// 判断是否为候选节点
	if candidateAccount.GetCandidateState(types.CandidateKeyIsCandidate) == types.IsCandidateNode {
		// set votes
		candidateAccount.SetVotes(new(big.Int).Add(candidateAccount.GetVotes(), changeVotes))
	}
}
