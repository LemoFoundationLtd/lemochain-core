package consensus

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math/big"
	"time"
)

// Assembler seal block
type BlockAssembler struct {
	am          *account.Manager
	dm          *deputynode.Manager
	txProcessor *transaction.TxProcessor
	canLoader   CandidateLoader
}

func NewBlockAssembler(am *account.Manager, dm *deputynode.Manager, txProcessor *transaction.TxProcessor, canLoader CandidateLoader) *BlockAssembler {
	return &BlockAssembler{
		am:          am,
		dm:          dm,
		txProcessor: txProcessor,
		canLoader:   canLoader,
	}
}

// Seal packages all products into a block
func (ba *BlockAssembler) RunBlock(block *types.Block) (*types.Block, error) {
	// execute tx
	gasUsed, err := ba.txProcessor.Process(block.Header, block.Txs)
	if err != nil {
		log.Errorf("processor internal error: %v", err)
		return nil, err
	}
	// Finalize accounts
	if err = ba.Finalize(block.Header.Height); err != nil {
		log.Errorf("Finalize accounts error: %v", err)
		return nil, err
	}
	// seal a new block
	newBlock := ba.Seal(block.Header, ba.am.GetTxsProduct(block.Txs, gasUsed), block.Confirms)
	return newBlock, nil
}

// MineBlock packages all products into a block
func (ba *BlockAssembler) MineBlock(header *types.Header, txs types.Transactions, applyTxTimeout int64) (*types.Block, types.Transactions, error) {
	// execute tx
	packagedTxs, invalidTxs, gasUsed := ba.txProcessor.ApplyTxs(header, txs, applyTxTimeout)
	log.Debug("ApplyTxs ok")
	// Finalize accounts
	if err := ba.Finalize(header.Height); err != nil {
		log.Errorf("Finalize accounts error: %v", err)
		return nil, invalidTxs, err
	}
	// seal block
	newBlock := ba.Seal(header, ba.am.GetTxsProduct(packagedTxs, gasUsed), nil)
	// sign block
	signData, err := SignBlock(newBlock.Hash())
	if err != nil {
		log.Errorf("Sign for block failed! block hash:%s", newBlock.Hash().Hex())
		return nil, invalidTxs, err
	}
	newBlock.Header.SignData = signData

	return newBlock, invalidTxs, nil
}

func (ba *BlockAssembler) PrepareHeader(parentHeader *types.Header, extra string) (*types.Header, error) {
	minerAddress, ok := ba.dm.GetMyMinerAddress(parentHeader.Height + 1)
	if !ok {
		log.Errorf("Not a deputy at height %d. can't mine", parentHeader.Height+1)
		return nil, ErrNotDeputy
	}
	h := &types.Header{
		ParentHash:   parentHeader.Hash(),
		MinerAddress: minerAddress,
		Height:       parentHeader.Height + 1,
		GasLimit:     calcGasLimit(parentHeader),
		Extra:        extra,
	}

	// allow 1 second time error
	// but next block's time can't be small than parent block
	parTime := parentHeader.Time
	blockTime := uint32(time.Now().Unix())
	if parTime > blockTime {
		blockTime = parTime
	}
	h.Time = blockTime
	return h, nil
}

// calcGasLimit computes the gas limit of the next block after parent.
// This is miner strategy, not consensus protocol.
func calcGasLimit(parentHeader *types.Header) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parentHeader.GasUsed + parentHeader.GasUsed/2) / params.GasLimitBoundDivisor

	// decay = parentGasLimit / 1024 -1
	decay := parentHeader.GasLimit/params.GasLimitBoundDivisor - 1

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	limit := parentHeader.GasLimit - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if limit < params.TargetGasLimit {
		limit = parentHeader.GasLimit + decay
		if limit > params.TargetGasLimit {
			limit = params.TargetGasLimit
		}
	}
	return limit
}

// Seal packages all products into a block
func (ba *BlockAssembler) Seal(header *types.Header, txProduct *account.TxsProduct, confirms []types.SignData) *types.Block {
	newHeader := header.Copy()
	newHeader.VersionRoot = txProduct.VersionRoot
	newHeader.LogRoot = txProduct.ChangeLogs.MerkleRootSha()
	newHeader.TxRoot = txProduct.Txs.MerkleRootSha()
	newHeader.GasUsed = txProduct.GasUsed

	block := types.NewBlock(newHeader, txProduct.Txs, txProduct.ChangeLogs)
	block.SetConfirms(confirms)
	if deputynode.IsSnapshotBlock(header.Height) {
		deputies := ba.canLoader.LoadTopCandidates(header.ParentHash)
		block.SetDeputyNodes(deputies)
		root := deputies.MerkleRootSha()
		newHeader.DeputyRoot = root[:]
		log.Debug("snapshot new term", "deputies", log.Lazy{Fn: func() string {
			return deputies.String()
		}})
	}

	return block
}

// checkTermReward 在设定的区块高度检查本届是否设置了换届奖励并进行事件推送，返回值表示是否已正确设置
func (ba *BlockAssembler) checkTermReward(height uint32) bool {
	// 在奖励块前第100000个区块进行校验
	if deputynode.IsRewardBlock(height + params.RewardCheckHeight) {
		termIndex := deputynode.GetSignerTermIndexByHeight(height)
		termRewards, err := getTermRewards(ba.am, termIndex)
		if err == nil && termRewards.Cmp(big.NewInt(0)) == 0 { // 本届还未设置换届奖励，事件推送通知
			log.Eventf(log.TxEvent, "There was no consensus node award in the [%d] term. The current block height is %d.", termIndex, height)
			return false
		}
	}
	return true
}

// refundCandidateDeposit 退还取消候选节点的质押押金
func (ba *BlockAssembler) refundCandidateDeposit(am *account.Manager, height uint32) error {
	// 判断是否到了发放换届奖励的区块高度
	if !deputynode.IsRewardBlock(height) {
		return nil
	}

	// the address list of candidates who need to refund
	addrList, err := ba.canLoader.LoadRefundCandidates(height)
	if err != nil {
		return err
	}
	for _, candidateAddress := range addrList {
		// 退押金操作
		transaction.Refund(candidateAddress, ba.am)
	}
	return nil
}

// issueTermReward 发放换届奖励
func (ba *BlockAssembler) issueTermReward(am *account.Manager, height uint32) error {
	// 判断是否到了发放换届奖励的区块高度
	if !deputynode.IsRewardBlock(height) {
		return nil
	}

	// 奖励块是新一届开始的第一个块，这里应该发放前一届的奖励
	term, err := ba.dm.GetTermByHeight(height-1, true)
	if err != nil {
		log.Warnf("load term information failed: %v", err)
		return err
	}
	totalRewards, err := getTermRewards(am, term.TermIndex)
	if err != nil {
		log.Warnf("load term rewards failed: %v", err)
		return err
	}

	log.Debugf("the reward of term %d is %s", term.TermIndex, totalRewards.String())
	// issue reward if reward greater than 0
	if totalRewards.Cmp(big.NewInt(0)) > 0 {
		rewards := DivideSalary(totalRewards, am, term)
		for _, item := range rewards {
			acc := am.GetAccount(item.Address)
			oldBalance := acc.GetBalance()
			newBalance := new(big.Int).Add(oldBalance, item.Salary)
			acc.SetBalance(newBalance)
		}
	}

	return nil
}

// Finalize increases miners' balance and fix all account changes
func (ba *BlockAssembler) Finalize(height uint32) error {
	// 在设定的区块高度检查本届是否设置了换届奖励，如果未设置则进行事件通知
	ba.checkTermReward(height)

	// 发放换届奖励
	if err := ba.issueTermReward(ba.am, height); err != nil {
		log.Warnf("issue term reward failed: %v", err)
		return err
	}
	// 退还取消候选节点的质押金额
	if err := ba.refundCandidateDeposit(ba.am, height); err != nil {
		log.Warnf("refund deposit failed: %v", err)
		return err
	}

	// 设置执行区块之后余额变化造成的候选节点的票数变化
	transaction.ChangeVotesByBalance(ba.am)
	// merge
	ba.am.MergeChangeLogs()

	// finalize accounts
	err := ba.am.Finalise()
	if err != nil {
		log.Warnf("finalise manager failed: %v", err)
		return err
	}
	return nil
}

func DivideSalary(totalSalary *big.Int, am *account.Manager, t *deputynode.TermRecord) []*deputynode.DeputySalary {
	salaries := make([]*deputynode.DeputySalary, len(t.Nodes))
	totalVotes := t.GetTotalVotes()
	for i, node := range t.Nodes {
		salaries[i] = &deputynode.DeputySalary{
			Address: getDeputyIncomeAddress(am, node),
			Salary:  calculateSalary(totalSalary, node.Votes, totalVotes, params.MinRewardPrecision, len(t.Nodes)),
		}
	}
	return salaries
}

func calculateSalary(totalSalary, deputyVotes, totalVotes, precision *big.Int, nodesNum int) *big.Int {
	r := new(big.Int)
	if totalVotes.Cmp(big.NewInt(0)) == 0 {
		r.Div(totalSalary, big.NewInt(int64(nodesNum)))
	} else {
		// totalSalary * deputyVotes / totalVotes
		r.Mul(totalSalary, deputyVotes)
		r.Div(r, totalVotes)
	}
	// r - ( r % precision )
	mod := new(big.Int).Mod(r, precision)
	r.Sub(r, mod)
	return r
}

// getDeputyIncomeAddress
func getDeputyIncomeAddress(am *account.Manager, node *types.DeputyNode) common.Address {
	minerAcc := am.GetAccount(node.MinerAddress)
	strIncomeAddress := minerAcc.GetCandidateState(types.CandidateKeyIncomeAddress)
	if strIncomeAddress == "" {
		log.Errorf("Not exist income address. the salary will be awarded to minerAddress %s", node.MinerAddress.String())
		return node.MinerAddress
	}
	incomeAddress, err := common.StringToAddress(strIncomeAddress)
	if err != nil {
		log.Errorf("Income address invalid. the salary will be awarded to minerAddress. Candidate income address = %s", strIncomeAddress)
		return node.MinerAddress
	}
	return incomeAddress
}

// getTermRewards load the rewards of miners at the end of a term
func getTermRewards(am *account.Manager, term uint32) (*big.Int, error) {
	// Precompile the contract address
	address := params.TermRewardContract
	acc := am.GetAccount(address)
	// key for db
	key := address.Hash()
	value, err := acc.GetStorageState(key)
	if err != nil {
		return nil, err
	}
	// return 0 if the reward not exist
	if len(value) == 0 {
		return big.NewInt(0), nil
	}

	rewardMap := make(params.RewardsMap)
	err = json.Unmarshal(value, &rewardMap)
	if err != nil {
		return nil, err
	}
	if reward, ok := rewardMap[term]; ok {
		return reward.Value, nil
	} else {
		return big.NewInt(0), nil
	}
}
