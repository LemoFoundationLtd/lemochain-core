package consensus

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"time"
)

// Assembler seal block
type BlockAssembler struct {
	db          protocol.ChainDB
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
	if err = ba.Finalize(block.Header.Height, ba.am); err != nil {
		log.Errorf("Finalize accounts error: %v", err)
		return nil, err
	}
	// seal a new block
	newBlock := ba.Seal(block.Header, ba.am.GetTxsProduct(block.Txs, gasUsed), block.Confirms)
	return newBlock, nil
}

// MineBlock packages all products into a block
func (ba *BlockAssembler) MineBlock(parent *types.Block, extra []byte, txPool TxPool, timeLimitSeconds int64) (*types.Block, error) {
	minerAddress, ok := ba.dm.GetMyMinerAddress(parent.Height() + 1)
	if !ok {
		log.Errorf("Not a deputy at height %d. can't mine", parent.Height()+1)
		return nil, ErrNotDeputy
	}
	// create header
	header := ba.sealHeader(parent, minerAddress, extra)
	// execute tx
	txs := txPool.Get(header.Time, 10000)
	log.Debugf("pick %d txs from txPool", len(txs))
	packagedTxs, invalidTxs, gasUsed := ba.txProcessor.ApplyTxs(header, txs, timeLimitSeconds)
	log.Debug("ApplyTxs ok")
	// remove invalid txs from pool
	txPool.DelInvalidTxs(invalidTxs)
	// Finalize accounts
	if err := ba.Finalize(header.Height, ba.am); err != nil {
		log.Errorf("Finalize accounts error: %v", err)
		return nil, err
	}
	// seal block
	newBlock := ba.Seal(header, ba.am.GetTxsProduct(packagedTxs, gasUsed), nil)
	if err := ba.signBlock(newBlock); err != nil {
		log.Errorf("Sign for block failed! block hash:%s", newBlock.Hash().Hex())
		return nil, err
	}

	return newBlock, nil
}

func (ba *BlockAssembler) sealHeader(parent *types.Block, minerAddress common.Address, extra []byte) *types.Header {
	height := parent.Height() + 1
	h := &types.Header{
		ParentHash:   parent.Hash(),
		MinerAddress: minerAddress,
		Height:       height,
		GasLimit:     calcGasLimit(parent),
		Extra:        extra,
	}

	// allowable 1 second time error
	// but next block's time can't be small than parent block
	parTime := parent.Time()
	blockTime := uint32(time.Now().Unix())
	if parTime > blockTime {
		blockTime = parTime
	}
	h.Time = blockTime
	return h
}

// calcGasLimit computes the gas limit of the next block after parent.
// This is miner strategy, not consensus protocol.
func calcGasLimit(parent *types.Block) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor

	// decay = parentGasLimit / 1024 -1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if limit < params.TargetGasLimit {
		limit = parent.GasLimit() + decay
		if limit > params.TargetGasLimit {
			limit = params.TargetGasLimit
		}
	}
	return limit
}

// signBlock signed the block and fill in header
func (ba *BlockAssembler) signBlock(block *types.Block) (err error) {
	hash := block.Hash()
	signData, err := crypto.Sign(hash[:], deputynode.GetSelfNodeKey())
	if err == nil {
		block.Header.SignData = signData
	}
	return
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

// Finalize increases miners' balance and fix all account changes
func (ba *BlockAssembler) Finalize(height uint32, am *account.Manager) error {
	// Pay miners at the end of their tenure
	if deputynode.IsRewardBlock(height) {
		term := (height-params.InterimDuration)/params.TermDuration - 1
		termRewards, err := getTermRewardValue(am, term)
		if err != nil {
			log.Warnf("load term rewards failed: %v", err)
			return err
		}
		log.Debugf("the reward of term %d is %s ", term, termRewards.String())
		// issue reward if reward greater than 0
		if termRewards.Cmp(big.NewInt(0)) > 0 {
			lastTermRecord, err := ba.dm.GetTermByHeight(height - 1)
			if err != nil {
				log.Warnf("load deputy nodes failed: %v", err)
				return err
			}
			rewards := DivideSalary(termRewards, am, lastTermRecord)
			for _, item := range rewards {
				acc := am.GetAccount(item.Address)
				oldBalance := acc.GetBalance()
				newBalance := new(big.Int).Add(oldBalance, item.Salary)
				acc.SetBalance(newBalance)
			}

			// 退还取消候选节点的质押金额
			allCandidateAddressList, err := ba.db.GetAllCandidates()
			if err != nil {
				panic(err)
			}

			for _, candidateAddress := range allCandidateAddressList {
				// 判断addr的candidate信息
				candidateAcc := am.GetAccount(candidateAddress)
				pledgAmountString := candidateAcc.GetCandidateState(types.CandidateKeyPledgeAmount)
				if candidateAcc.GetCandidateState(types.CandidateKeyIsCandidate) == params.NotCandidateNode && pledgAmountString != "" { // 满足退还押金的条件
					if pledgeAmount, success := new(big.Int).SetString(pledgAmountString, 10); !success {
						panic("Big.Int SetString function failed")
					} else {
						// 退还押金
						candidatePledgePoolAcc := am.GetAccount(params.CandidateDepositAddress)
						if candidatePledgePoolAcc.GetBalance().Cmp(pledgeAmount) < 0 { // 判断押金池中的押金是否足够，如果不足直接panic
							panic("candidate pledge pool account balance insufficient.")
						}
						// 减少押金池中的余额
						candidatePledgePoolAcc.SetBalance(new(big.Int).Sub(candidatePledgePoolAcc.GetBalance(), pledgeAmount))
						// 退还押金到取消的候选节点账户
						candidateAcc.SetBalance(new(big.Int).Add(candidateAcc.GetBalance(), pledgeAmount))
						// 设置候选节点info中的押金余额为nil
						candidateAcc.SetCandidateState(types.CandidateKeyPledgeAmount, "")
					}

				}
			}
		}
	}

	// 设置执行此block之后余额变化造成的候选节点的票数变化
	transaction.SetCandidateVotesByChangeBalance(am)
	// merge
	am.MergeChangeLogs()

	// finalize accounts
	err := am.Finalise()
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
			Address: getIncomeAddressFromDeputyNode(am, node),
			Salary:  calculateSalary(totalSalary, node.Votes, totalVotes, params.MinRewardPrecision),
		}
	}
	return salaries
}

func calculateSalary(totalSalary, deputyVotes, totalVotes, precision *big.Int) *big.Int {
	r := new(big.Int)
	// totalSalary * deputyVotes / totalVotes
	r.Mul(totalSalary, deputyVotes)
	r.Div(r, totalVotes)
	// r - ( r % precision )
	mod := new(big.Int).Mod(r, precision)
	r.Sub(r, mod)
	return r
}

// getIncomeAddressFromDeputyNode
func getIncomeAddressFromDeputyNode(am *account.Manager, node *deputynode.DeputyNode) common.Address {
	minerAcc := am.GetAccount(node.MinerAddress)
	profile := minerAcc.GetCandidate()
	strIncomeAddress, ok := profile[types.CandidateKeyIncomeAddress]
	if !ok {
		log.Errorf("not exist income address. the salary will be awarded to minerAddress. miner address = %s", node.MinerAddress.String())
		return node.MinerAddress
	}
	incomeAddress, err := common.StringToAddress(strIncomeAddress)
	if err != nil {
		log.Errorf("income address invalid. the salary will be awarded to minerAddress. incomeAddress = %s", strIncomeAddress)
		return node.MinerAddress
	}
	return incomeAddress
}

// getTermRewardValue reward value of miners at the change of term
func getTermRewardValue(am *account.Manager, term uint32) (*big.Int, error) {
	// Precompile the contract address
	address := params.TermRewardContract
	acc := am.GetAccount(address)
	// key of db
	key := address.Hash()
	value, err := acc.GetStorageState(key)
	if err != nil {
		return nil, err
	}
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
