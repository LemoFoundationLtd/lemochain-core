package chain

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"time"
)

var (
	minPrecision = new(big.Int).SetUint64(uint64(1000000000000000000)) // 1 LEMO
)

const MaxExtraDataLen = 256

type Engine interface {
	VerifyHeader(block *types.Block) error

	Finalize(height uint32, am *account.Manager) error

	Seal(header *types.Header, txProduct *account.TxsProduct, confirms []types.SignData, dNodes deputynode.DeputyNodes) (*types.Block, error)
}

type Dpovp struct {
	timeoutTime int64
	db          protocol.ChainDB
	dm          *deputynode.Manager
}

func NewDpovp(timeout int64, dm *deputynode.Manager, db protocol.ChainDB) *Dpovp {
	dpovp := &Dpovp{
		timeoutTime: timeout,
		db:          db,
		dm:          dm,
	}
	return dpovp
}

// verifyHeaderTime verify that the block timestamp is less than the current time
func verifyHeaderTime(block *types.Block) error {
	header := block.Header
	blockTime := header.Time
	timeNow := time.Now().Unix()
	if int64(blockTime)-timeNow > 1 { // Prevent validation failure due to time error
		log.Errorf("verifyHeader: block in the future. height:%d", block.Height())
		return ErrVerifyHeaderFailed
	}
	return nil
}

// verifyHeaderSignData verify the block signature data
func verifyHeaderSignData(dm *deputynode.Manager, block *types.Block) error {
	header := block.Header
	hash := block.Hash()
	pubKey, err := crypto.Ecrecover(hash[:], header.SignData)
	if err != nil {
		log.Errorf("verifyHeaderSignData: illegal signData. %s. height:%d", err, block.Height())
		return ErrVerifyHeaderFailed
	}
	node := dm.GetDeputyByAddress(header.Height, header.MinerAddress)
	if node == nil {
		nodes := dm.GetDeputiesByHeight(block.Height())
		log.Errorf("verifyHeaderSignData: can't get deputy node, height: %d, miner: %s, deputy nodes: %s", header.Height, header.MinerAddress.String(), nodes.String())
		return ErrVerifyHeaderFailed
	}
	if node == nil || bytes.Compare(pubKey[1:], node.NodeID) != 0 {
		log.Errorf("verifyHeaderSignData: illegal block. height:%d, hash:%s", header.Height, header.Hash().Hex())
		return ErrVerifyHeaderFailed
	}
	return nil
}

// VerifyDeputyRoot verify deputy root
func (d *Dpovp) VerifyDeputyRoot(block *types.Block) error {
	if block.Height()%params.TermDuration == 0 && block.Height() > 0 {
		hash := block.DeputyNodes.MerkleRootSha()
		root := block.Header.DeputyRoot
		if bytes.Compare(hash[:], root) != 0 {
			log.Errorf("verify block failed. deputyRoot not match. header's root: %s, check root: %s", common.ToHex(root), hash.String())
			return ErrVerifyBlockFailed
		}
	}
	return nil
}

// VerifyHeader verify block header
func (d *Dpovp) VerifyHeader(block *types.Block) error {
	nodeCount := d.dm.GetDeputiesCount(block.Height()) // The total number of nodes
	// There's only one out block node
	if nodeCount == 1 {
		return nil
	}
	// VerifyAndFill that the block timestamp is less than the current time
	if err := verifyHeaderTime(block); err != nil {
		return err
	}
	// VerifyAndFill the block signature data
	if err := verifyHeaderSignData(d.dm, block); err != nil {
		return err
	}
	// verify deputy node root when height is 100W*N
	if err := d.VerifyDeputyRoot(block); err != nil {
		return err
	}
	header := block.Header
	// verify extra data
	if len(header.Extra) > MaxExtraDataLen {
		log.Errorf("verifyHeader: extra data's max len is %d bytes, current length is %d", MaxExtraDataLen, len(block.Header.Extra))
		return ErrVerifyHeaderFailed
	}

	parent, _ := d.db.GetBlockByHash(header.ParentHash)
	if parent == nil {
		log.Errorf("verifyHeader: can't get parent block. height:%d, hash:%s", header.Height-1, header.ParentHash)
		return ErrVerifyHeaderFailed
	}
	if parent.Header.Height == 0 {
		log.Debug("verifyHeader: parent block is genesis block")
		return nil
	}
	var slot uint32
	if (header.Height > params.InterimDuration+1) && (header.Height-params.InterimDuration-1)%params.TermDuration == 0 {
		deputyNode := d.dm.GetDeputyByAddress(header.Height, block.MinerAddress())
		if deputyNode == nil {
			return ErrVerifyHeaderFailed
		}
		slot = deputyNode.Rank + 1
		log.Debugf("rank: %d", deputyNode.Rank)
	} else {
		slot, _ = d.dm.GetMinerDistance(header.Height, parent.Header.MinerAddress, header.MinerAddress)
	}

	// The time interval between the current block and the parent block. unit：ms
	timeSpan := int64(header.Time-parent.Header.Time) * 1000
	oldTimeSpan := timeSpan
	oneLoopTime := int64(nodeCount) * d.timeoutTime // All timeout times for a round of deputy nodes

	timeSpan %= oneLoopTime
	if slot == 0 { // The last block was made for itself
		if timeSpan < oneLoopTime-d.timeoutTime {
			log.Debugf("verifyHeader: verify failed. height:%d. oldTimeSpan: %d timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d -2", header.Height, oldTimeSpan, timeSpan, nodeCount, slot, oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	} else if slot == 1 {
		if timeSpan >= d.timeoutTime {
			log.Debugf("verifyHeader: height: %d. verify failed.timeSpan< oneLoopTime. timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d -3", block.Height(), timeSpan, nodeCount, slot, oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	} else {
		if timeSpan/d.timeoutTime != int64(slot-1) {
			log.Debugf("verifyHeader: verify failed. height: %d. oldTimeSpan: %d timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d -4", header.Height, oldTimeSpan, timeSpan, nodeCount, slot, oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	}
	return nil
}

// Seal packages all products into a block
func (d *Dpovp) Seal(header *types.Header, txProduct *account.TxsProduct, confirms []types.SignData, dNodes deputynode.DeputyNodes) (*types.Block, error) {
	log.Errorf("%d changlog.00000000002: %v", header.Height, txProduct.ChangeLogs)
	newHeader := header.Copy()
	newHeader.VersionRoot = txProduct.VersionRoot
	newHeader.LogRoot = txProduct.ChangeLogs.MerkleRootSha()
	newHeader.TxRoot = txProduct.Txs.MerkleRootSha()
	newHeader.GasUsed = txProduct.GasUsed

	block := types.NewBlock(newHeader, txProduct.Txs, txProduct.ChangeLogs)
	block.SetConfirms(confirms)
	if dNodes != nil {
		block.SetDeputyNodes(dNodes)
	}
	return block, nil
}

// Finalize increases miners' balance and fix all account changes
func (d *Dpovp) Finalize(height uint32, am *account.Manager) error {
	// Pay miners at the end of their tenure
	if deputynode.IsRewardBlock(height) {
		term := (height-params.InterimDuration)/params.TermDuration - 1
		termRewards, err := getTermRewardValue(am, term)
		log.Debugf("the %d term's reward value = %s ", term, termRewards.String())
		if err != nil {
			log.Warnf("load rewards failed: %v", err)
			return err
		}
		lastTermRecord, err := d.dm.GetTermByHeight(height - 1)
		if err != nil {
			log.Warnf("load deputy nodes failed: %v", err)
			return err
		}
		rewards := DivideSalary(termRewards, am, lastTermRecord)
		for _, item := range rewards {
			acc := am.GetAccount(item.Address)
			balance := acc.GetBalance()
			balance.Add(balance, item.Salary)
			acc.SetBalance(balance)
			// 	candidate node vote change corresponding to balance change
			candidateAddr := acc.GetVoteFor()
			if (candidateAddr == common.Address{}) {
				continue
			}
			candidateAcc := am.GetAccount(candidateAddr)
			profile := candidateAcc.GetCandidate()
			if profile[types.CandidateKeyIsCandidate] == params.NotCandidateNode {
				continue
			}
			// set votes
			candidateAcc.SetVotes(new(big.Int).Add(candidateAcc.GetVotes(), item.Salary))
		}
	}

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
			Salary:  calculateSalary(totalSalary, node.Votes, totalVotes, minPrecision),
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
		log.Errorf("not exist income address; miner address = %s", node.MinerAddress)
		return common.Address{}
	}
	incomeAddress, err := common.StringToAddress(strIncomeAddress)
	if err != nil {
		log.Errorf("income address unavailability; incomeAddress = %s", strIncomeAddress)
		return common.Address{}
	}
	return incomeAddress
}

// getTermRewardValue reward value of miners at the change of term
func getTermRewardValue(am *account.Manager, term uint32) (*big.Int, error) {
	// Precompile the contract address
	address := params.TermRewardPrecompiledContractAddress
	acc := am.GetAccount(address)
	// key of db
	key := address.Hash()
	value, err := acc.GetStorageState(key)
	if err != nil {
		return nil, err
	}

	rewardMap := make(params.RewardsMap)
	err = json.Unmarshal(value, &rewardMap)
	if err != nil {
		return nil, err
	}
	if reward, ok := rewardMap[term]; ok {
		return reward.Value, nil
	} else {
		return nil, errors.New("reward value does not exit. ")
	}
}
