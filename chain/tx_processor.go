package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/math"
	"math/big"
	"sync"
	"time"
)

const (
	defaultGasPrice = 1e9
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

type ApplyTxsResult struct {
	Txs     types.Transactions // The transactions executed indeed. These transactions will be packaged in a block
	Events  []*types.Event     // contract events
	Bloom   types.Bloom        // used to search contract events
	GasUsed uint64             // gas used by all transactions
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
func (p *TxProcessor) Process(block *types.Block) (*types.Header, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	var (
		gp          = new(types.GasPool).AddGas(block.GasLimit())
		gasUsed     = uint64(0)
		totalGasFee = new(big.Int)
		header      = block.Header
		txs         = block.Txs
	)
	p.am.Reset(header.ParentHash)
	// genesis
	if header.Height == 0 {
		log.Warn("It is not necessary to process genesis block.")
		return header, nil
	}
	// Iterate over and process the individual transactions
	for i, tx := range txs {
		gas, err := p.applyTx(gp, header, tx, uint(i), block.Hash())
		if err != nil {
			log.Info("Invalid transaction", "hash", tx.Hash(), "err", err)
			return nil, ErrInvalidTxInBlock
		}
		gasUsed = gasUsed + gas
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		totalGasFee.Add(totalGasFee, fee)
	}
	p.chargeForGas(totalGasFee, header.MinerAddress)

	return p.FillHeader(header.Copy(), txs, gasUsed)
}

// ApplyTxs picks and processes transactions from miner's tx pool.
func (p *TxProcessor) ApplyTxs(header *types.Header, txs types.Transactions) (*types.Header, types.Transactions, types.Transactions, error) {
	gp := new(types.GasPool).AddGas(header.GasLimit)
	gasUsed := uint64(0)
	totalGasFee := new(big.Int)
	selectedTxs := make(types.Transactions, 0)
	invalidTxs := make(types.Transactions, 0)

	p.am.Reset(header.ParentHash)

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

	newHeader, err := p.FillHeader(header.Copy(), selectedTxs, gasUsed)
	return newHeader, selectedTxs, invalidTxs, err
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
	fmt.Println("初始的Balance=", initialSenderBalance.String())
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
		vmErr         error
		recipientAddr common.Address
	)
	if contractCreation {
		_, recipientAddr, restGas, vmErr = vmEnv.Create(sender, tx.Data(), restGas, tx.Amount())
	} else {
		recipientAddr = *tx.To()
		recipientAccount := p.am.GetAccount(recipientAddr)
		initialRecipientBalance := recipientAccount.GetBalance()

		// 判断交易是普通转账交易、投票交易还是注册参加节点竞选的交易
		switch tx.Type() {
		case params.OrdinaryTx:
			_, restGas, vmErr = vmEnv.Call(sender, recipientAddr, tx.Data(), restGas, tx.Amount())

		case params.VoteTx: // 执行投票交易逻辑
			restGas, vmErr = vmEnv.CallVoteTx(senderAddr, recipientAddr, restGas, initialSenderBalance)

		case params.RegisterTx: // 执行注册参加代理节点选举交易逻辑
			// // 判断tx的接收者是否为"0x1001"地址,(目前只是通过TxType判断是注册交易的,交易的接受者自动变为"0x1001",这里判断不判断都不影响)
			// if *tx.To() != params.FeeReceiveAddress {
			// 	log.Error("RegisterTx recipient Address false")
			// 	return 0, errors.New("RegisterTx recipient Address false")
			// }

			// 解析交易data中申请候选节点的信息
			txData := tx.Data()
			candiNode := new(deputynode.CandidateNode)
			err = json.Unmarshal(txData, candiNode)
			if err != nil {
				log.Errorf("unmarshal Candidate node error: %s", err)
				return 0, err
			}
			restGas, vmErr = vmEnv.RegisterOrUpdateToCandidate(senderAddr, params.FeeReceiveAddress, candiNode, restGas, tx.Amount(), initialSenderBalance)

		default:
			log.Errorf("The type of transaction is not defined. txType = %d\n", tx.Type())
			// p.refundGas(gp, tx, restGas) // 交易不满足所定义的交易类型的交易视为攻击，则不返还剩下的gas
			// return 0, errors.New("the type of transaction error")
		}
		// 接收者对应的候选节点的票数变化
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
	p.am.SaveTxInAccount(senderAddr, recipientAddr, tx.Hash())

	// 发送者对应的候选节点票数变动
	endSenderBalance := sender.GetBalance()
	fmt.Println("一笔交易结束时的senderBalance:", endSenderBalance.String())

	senderBalanceChange := new(big.Int).Sub(endSenderBalance, initialSenderBalance)
	fmt.Printf("发送者减少的Balance = %s", senderBalanceChange.String())
	p.changeCandidateVotes(senderAddr, senderBalanceChange)

	// Merge change logs by transaction will save more transaction execution detail than by block
	p.am.MergeChangeLogs(mergeFrom)
	mergeFrom = len(p.am.GetChangeLogs())

	return tx.GasLimit() - restGas, nil
}

// changeCandidateVotes 账户Balance变化对应投给的候选节点的票数的变化
func (p *TxProcessor) changeCandidateVotes(accountAddress common.Address, changeBalance *big.Int) {
	if changeBalance.Sign() == 0 {
		return
	}
	acc := p.am.GetAccount(accountAddress)
	CandidataAddress := acc.GetVoteFor()

	if (CandidataAddress == common.Address{}) { // 不存在投票候选节点地址或者候选者已经取消候选者资格
		return
	}
	// 查看投给的候选者是否已经取消了候选资格
	CandidateAccount := p.am.GetAccount(CandidataAddress)
	profile := CandidateAccount.GetCandidateProfile()
	if profile[types.CandidateKeyIsCandidate] == "false" {
		return
	}
	// changeBalance为正数则加，为负数则减
	CandidateAccount.SetVotes(new(big.Int).Add(CandidateAccount.GetVotes(), changeBalance))
}

func (p *TxProcessor) buyGas(gp *types.GasPool, tx *types.Transaction) error {
	// ignore the error because it is checked in applyTx
	senderAddr, _ := tx.From()
	sender := p.am.GetAccount(senderAddr)

	maxFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasLimit()), tx.GasPrice())
	if sender.GetBalance().Cmp(maxFee) < 0 {
		return ErrInsufficientBalanceForGas
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

func (p *TxProcessor) refundGas(gp *types.GasPool, tx *types.Transaction, restGas uint64) {
	// ignore the error because it is checked in applyTx
	senderAddr, _ := tx.From()
	sender := p.am.GetAccount(senderAddr)

	// Return LEMO for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(restGas), tx.GasPrice())
	sender.SetBalance(new(big.Int).Add(sender.GetBalance(), remaining))

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	gp.AddGas(restGas)
}

// chargeForGas change the gas to miner
func (p *TxProcessor) chargeForGas(charge *big.Int, minerAddress common.Address) {
	if charge.Cmp(new(big.Int)) != 0 {
		miner := p.am.GetAccount(minerAddress)
		miner.SetBalance(new(big.Int).Add(miner.GetBalance(), charge))
		// 	矿工账户投给的候选节点的票数的变化
		p.changeCandidateVotes(minerAddress, charge)
	}
}

// FillHeader creates a new header then fills it with the result of transactions process
func (p *TxProcessor) FillHeader(header *types.Header, txs types.Transactions, gasUsed uint64) (*types.Header, error) {
	if len(txs) > 0 {
		log.Infof("process %d transactions", len(txs))
	}
	events := p.am.GetEvents()
	header.Bloom = types.CreateBloom(events)
	header.EventRoot = types.DeriveEventsSha(events)
	header.GasUsed = gasUsed
	header.TxRoot = types.DeriveTxsSha(txs)
	// Pay miners at the end of their tenure. This method increases miners' balance.
	p.chain.engine.Finalize(header, p.am)
	// Update version trie, storage trie.
	err := p.am.Finalise()
	if err != nil {
		// Access trie node fail.
		return nil, err
	}
	header.VersionRoot = p.am.GetVersionRoot()
	changeLogs := p.am.GetChangeLogs()
	header.LogRoot = types.DeriveChangeLogsSha(changeLogs)
	return header, nil
}

// CallTx pre-execute transactions and contracts.
func (p *TxProcessor) CallTx(ctx context.Context, header *types.Header, to *common.Address, data hexutil.Bytes, blockHash common.Hash, timeout time.Duration) ([]byte, uint64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	accM := account.ReadOnlyManager(header.Hash(), p.chain.db)
	accM.Reset(header.ParentHash)

	// A random address is found as our caller address.
	strAddress := "0x1002" // todo Consider letting users pass in their own addresses
	caller, err := common.StringToAddress(strAddress)
	if err != nil {
		return nil, 0, err
	}
	// enough gasLimit
	gasLimit := uint64(math.MaxUint64 / 2)
	gasPrice := new(big.Int).SetUint64(defaultGasPrice)

	var tx *types.Transaction
	if to == nil { // avoid null pointer references
		tx = types.NewContractCreation(big.NewInt(0), gasLimit, gasPrice, data, params.OrdinaryTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
	} else {
		tx = types.NewTransaction(*to, big.NewInt(0), gasLimit, gasPrice, data, params.OrdinaryTx, p.chain.chainID, uint64(time.Now().Unix()+30*60), "", "")
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
	if IsContractCreate {
		ret, _, restGas, err = Evm.Create(sender, tx.Data(), restGas, big.NewInt(0))
	} else {
		recipientAddr := *tx.To()
		ret, restGas, err = Evm.Call(sender, recipientAddr, tx.Data(), restGas, big.NewInt(0))
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
