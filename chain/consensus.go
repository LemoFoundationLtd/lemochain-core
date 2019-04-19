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
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math"
	"math/big"
	"time"
)

var (
	minPrecision = new(big.Int).SetUint64(uint64(1000000000000000000)) // 1 LEMO
)

const MaxExtraDataLen = 256

type Engine interface {
	VerifyBeforeTxProcess(block *types.Block) error
	VerifyAfterTxProcess(block, computedBlock *types.Block) error
	Finalize(height uint32, am *account.Manager) error
	Seal(header *types.Header, txProduct *account.TxsProduct, confirms []types.SignData, dNodes deputynode.DeputyNodes) (*types.Block, error)
	VerifyConfirmPacket(height uint32, blockHash common.Hash, sigList []types.SignData) error
	TrySwitchFork(stable, oldCurrent *types.Block) *types.Block
	ChooseNewFork() *types.Block
	CanBeStable(height uint32, confirmsCount int) bool
}

type DeputySnapshoter interface {
	SnapshotDeputyNodes() deputynode.DeputyNodes
}

// Dpovp seal and verify block
type Dpovp struct {
	timeoutTime int64
	db          protocol.ChainDB
	dm          *deputynode.Manager
	snapshoter  DeputySnapshoter
}

func NewDpovp(timeout int64, dm *deputynode.Manager, db protocol.ChainDB) *Dpovp {
	dpovp := &Dpovp{
		timeoutTime: timeout,
		db:          db,
		dm:          dm,
	}
	return dpovp
}

func (d *Dpovp) SetSnapshoter(snapshoter DeputySnapshoter) {
	d.snapshoter = snapshoter
}

// verifyParentHash verify the parent block hash in header
func verifyParentHash(block *types.Block, db protocol.ChainDB) (*types.Block, error) {
	parent, err := db.GetBlockByHash(block.ParentHash())
	if err != nil {
		log.Error("Consensus verify fail: can't load parent block", "ParentHash", block.ParentHash(), "err", err)
		return nil, ErrVerifyBlockFailed
	}
	return parent, nil
}

// verifyTxRoot verify the TxRoot is derived from the deputy nodes in block body
func verifyTxRoot(block *types.Block) error {
	hash := block.Txs.MerkleRootSha()
	if hash != block.TxRoot() {
		log.Error("Consensus verify fail: txRoot is incorrect", "txRoot", block.TxRoot().Hex(), "expected", hash.Hex())
		return ErrVerifyBlockFailed
	}
	return nil
}

// verifyHeight verify the hash of parent block
func verifyHeight(block *types.Block, parent *types.Block) error {
	if parent.Height()+1 != block.Height() {
		log.Error("Consensus verify fail: height is incorrect", "expect:%d", parent.Height()+1)
		return ErrVerifyHeaderFailed
	}
	return nil
}

// verifyTime verify that the block timestamp is less than the current time
func verifyTime(block *types.Block) error {
	timeNow := time.Now().Unix()
	if int64(block.Time())-timeNow > 1 { // Prevent validation failure due to time error
		log.Error("Consensus verify fail: block is in the future", "time", block.Time(), "now", timeNow)
		return ErrVerifyHeaderFailed
	}
	return nil
}

// verifySigner verify miner address and block signature data
func verifySigner(block *types.Block, dm *deputynode.Manager) error {
	nodeID, err := block.SignerNodeID()
	if err != nil {
		log.Error("Consensus verify fail: signData is incorrect", "err", err)
		return ErrVerifyHeaderFailed
	}

	// find the deputy node information of the miner
	deputy := dm.GetDeputyByNodeID(block.Height(), nodeID)
	if deputy == nil {
		log.Errorf("Consensus verify fail: can't find deputy node, nodeID: %s, deputy nodes: %s", common.ToHex(nodeID), dm.GetDeputiesByHeight(block.Height()).String())
		return ErrVerifyHeaderFailed
	}

	// verify miner address
	if deputy.MinerAddress != block.MinerAddress() {
		log.Error("Consensus verify fail: minerAddress is incorrect", "MinerAddress", block.MinerAddress(), "expected", deputy.MinerAddress)
		return ErrVerifyHeaderFailed
	}
	return nil
}

// verifyDeputy verify the DeputyRoot and DeputyNodes in block body
func verifyDeputy(block *types.Block, snapshoter DeputySnapshoter) error {
	if deputynode.IsSnapshotBlock(block.Height()) {
		// Make sure the DeputyRoot is derived from the deputy nodes in block body
		hash := block.DeputyNodes.MerkleRootSha()
		if bytes.Compare(hash[:], block.DeputyRoot()) != 0 {
			log.Error("Consensus verify fail: deputyRoot is incorrect", "deputyRoot", common.ToHex(block.DeputyRoot()), "expected", hash.Hex())
			return ErrVerifyBlockFailed
		}
		if snapshoter == nil {
			log.Error("Snapshoter is nil. Please call \"SetSnapshoter\" first")
			return ErrSnapshoterIsNil
		}
		// Make sure the DeputyRoot is match with local deputy nodes data
		deputySnapshot := snapshoter.SnapshotDeputyNodes()
		hash = deputySnapshot.MerkleRootSha()
		if bytes.Compare(hash[:], block.DeputyRoot()) != 0 {
			log.Error("Consensus verify fail: deputyNodes is incorrect", "deputyRoot", common.ToHex(block.DeputyRoot()), "expected", hash.Hex())
			log.Errorf("nodes in body: %s\nnodes in local: %s", block.DeputyNodes, deputySnapshot)
			return ErrVerifyBlockFailed
		}
	}
	return nil
}

// verifyChangeLog verify the LogRoot and ChangeLogs in block body
func verifyChangeLog(block *types.Block, computedLogs types.ChangeLogSlice) error {
	// Make sure the LogRoot is derived from the change logs in block body
	hash := block.ChangeLogs.MerkleRootSha()
	if hash != block.LogRoot() {
		log.Error("Consensus verify fail: logRoot is incorrect", "logRoot", block.LogRoot().Hex(), "expected", hash.Hex())
		return ErrVerifyBlockFailed
	}
	// Make sure the LogRoot is match with local change logs data
	hash = computedLogs.MerkleRootSha()
	if hash != block.LogRoot() {
		log.Error("Consensus verify fail: changeLogs is incorrect", "logRoot", block.LogRoot().Hex(), "expected", hash.Hex())
		log.Errorf("Logs in body: %s\nlogs in local: %s", block.ChangeLogs, computedLogs)
		return ErrVerifyBlockFailed
	}
	return nil
}

// verifyExtraData verify extra data in block header
func verifyExtraData(block *types.Block) error {
	if len(block.Extra()) > MaxExtraDataLen {
		log.Error("Consensus verify fail: extra data is too long", "current", len(block.Extra()), "max", MaxExtraDataLen)
		return ErrVerifyHeaderFailed
	}
	return nil
}

// verifyMineSlot verify the miner slot of deputy node
func verifyMineSlot(block *types.Block, parent *types.Block, timeoutTime int64, dm *deputynode.Manager) error {
	if block.Height() == 1 {
		// first block, no need to verify slot
		return nil
	}

	slot, err := GetMinerDistance(block.Height(), parent.MinerAddress(), block.MinerAddress(), dm)
	if err != nil {
		log.Error("Consensus verify fail: can't calculate slot", "block.Height", block.Height(), "parent.MinerAddress", parent.MinerAddress(), "block.MinerAddress", block.MinerAddress())
		return ErrVerifyHeaderFailed
	}

	// The time interval between the current block and the parent block. unit：ms
	timeSpan := int64(block.Time()-parent.Header.Time) * 1000
	oldTimeSpan := timeSpan
	oneLoopTime := int64(dm.GetDeputiesCount(block.Height())) * timeoutTime // All timeout times for a round of deputy nodes

	timeSpan %= oneLoopTime
	if slot == 0 { // The last block was made by itself
		if timeSpan < oneLoopTime-timeoutTime {
			log.Error("Consensus verify fail: mined twice in one loop", "slot", slot, "oldTimeSpan", oldTimeSpan, "timeSpan", timeSpan, "nodeCount", "oneLoopTime", oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	} else if slot == 1 {
		if timeSpan >= timeoutTime {
			log.Error("Consensus verify fail: time out", "slot", slot, "oldTimeSpan", oldTimeSpan, "timeSpan", timeSpan, "nodeCount", "oneLoopTime", oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	} else {
		if timeSpan/timeoutTime != int64(slot-1) {
			log.Error("Consensus verify fail: not in turn", "slot", slot, "oldTimeSpan", oldTimeSpan, "timeSpan", timeSpan, "nodeCount", "oneLoopTime", oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	}
	return nil
}

// verifyConfirms verify the confirm data in block body
func (d *Dpovp) VerifyConfirmPacket(height uint32, blockHash common.Hash, sigList []types.SignData) error {
	block, err := d.db.GetBlockByHash(blockHash)
	if err != nil {
		return ErrBlockNotExist
	}
	if block.Height() != height {
		log.Warn("Unmatched confirm height and hash", "height", height, "hash", blockHash.Hex())
		return ErrInvalidSignedConfirmInfo
	}
	return verifyConfirms(block, sigList, d.dm)
}

// verifyConfirm verify the confirm data
func verifyConfirms(block *types.Block, sigList []types.SignData, dm *deputynode.Manager) error {
	hash := block.Hash()
	for _, sig := range sigList {
		nodeID, err := sig.RecoverNodeID(hash)
		if err != nil {
			log.Warn("Invalid confirm signature", "hash", hash.Hex(), "sig", common.ToHex(sig[:]))
			return ErrInvalidSignedConfirmInfo
		}
		if bytes.Compare(sig[:], block.SignData()) == 0 {
			log.Warn("Invalid confirm from miner", "hash", hash.Hex(), "sig", common.ToHex(sig[:]))
			return ErrInvalidConfirmSigner
		}
		// find the signer
		if err := dm.GetDeputyByNodeID(block.Height(), nodeID); err != nil {
			log.Warn("Invalid confirm signer", "nodeID", common.ToHex(nodeID))
			return ErrInvalidConfirmSigner
		}
		// TODO every node can sign only once
	}
	return nil
}

// VerifyBeforeTxProcess verify the block data which has no relationship with the transaction processing result
func (d *Dpovp) VerifyBeforeTxProcess(block *types.Block) error {
	// cache parent block
	parent, err := verifyParentHash(block, d.db)
	if err != nil {
		return err
	}
	// verify miner address and signData
	if err := verifySigner(block, d.dm); err != nil {
		return err
	}
	if err := verifyTxRoot(block); err != nil {
		return err
	}
	if err := verifyHeight(block, parent); err != nil {
		return err
	}
	if err := verifyTime(block); err != nil {
		return err
	}
	if err := verifyDeputy(block, d.snapshoter); err != nil {
		return err
	}
	if err := verifyExtraData(block); err != nil {
		return err
	}
	if err := verifyMineSlot(block, parent, d.timeoutTime, d.dm); err != nil {
		return err
	}
	if err := verifyConfirms(block, block.Confirms, d.dm); err != nil {
		return err
	}
	return nil
}

// VerifyAfterTxProcess verify the block data which computed from transactions
func (d *Dpovp) VerifyAfterTxProcess(block, computedBlock *types.Block) error {
	// verify block hash. It also verify the rest fields in header: VersionRoot, LogRoot, GasLimit, GasUsed
	if computedBlock.Hash() != block.Hash() {
		// it contains
		log.Errorf("verify block error! oldHeader: %v, newHeader:%v", block.Header, computedBlock.Header)
		return ErrVerifyBlockFailed
	}
	// verify changeLog
	if err := verifyChangeLog(block, computedBlock.ChangeLogs); err != nil {
		return err
	}
	return nil
}

func findDeputyByAddress(deputies []*deputynode.DeputyNode, addr common.Address) *deputynode.DeputyNode {
	for _, node := range deputies {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// GetMinerDistance get miner index distance in same term
func GetMinerDistance(targetHeight uint32, lastBlockMiner, targetMiner common.Address, dm *deputynode.Manager) (uint32, error) {
	if targetHeight == 0 {
		return 0, ErrMineGenesis
	}
	deputies := dm.GetDeputiesByHeight(targetHeight)

	// find target block miner deputy
	targetDeputy := findDeputyByAddress(deputies, targetMiner)
	if targetDeputy == nil {
		return 0, ErrNotDeputy
	}

	// only one deputy
	nodeCount := uint32(len(deputies))
	if nodeCount == 1 {
		return 1, nil
	}

	// Genesis block is pre-set, not belong to any deputy node. So only blocks start with height 1 is mined by deputies
	// The reward block changes deputy nodes, so we need recompute the slot
	if targetHeight == 1 || deputynode.IsRewardBlock(targetHeight) {
		return targetDeputy.Rank + 1, nil
	}

	// find last block miner deputy
	lastDeputy := findDeputyByAddress(deputies, lastBlockMiner)
	if lastDeputy == nil {
		return 0, ErrNotDeputy
	}

	return (targetDeputy.Rank - lastDeputy.Rank + nodeCount) % nodeCount, nil
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
		if err != nil {
			log.Warnf("load rewards failed: %v", err)
			return err
		}
		log.Debugf("the %d term's reward value = %s ", term, termRewards.String())
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

// twoThird calculate num * 2 / 3
func twoThird(num int) uint32 {
	return uint32(math.Ceil(float64(num) * 2.0 / 3.0))
}

// TrySwitchFork switch fork if its length reached to a multiple of "deputy nodes count * 2/3"
func (d *Dpovp) TrySwitchFork(stable, current *types.Block) *types.Block {
	maxHeightBlock := d.ChooseNewFork()
	// make sure the fork is the first one reaching the height
	if maxHeightBlock != nil && maxHeightBlock.Height() > current.Height() {
		nodeCount := d.dm.GetDeputiesCount(maxHeightBlock.Height())
		judgeLength := twoThird(nodeCount)
		if (maxHeightBlock.Height()-stable.Height())%judgeLength == 0 {
			return maxHeightBlock
		}
	}
	return current
}

// ChooseNewFork choose a fork and return the last block on the fork. It would return nil if there is no unstable block
func (d *Dpovp) ChooseNewFork() *types.Block {
	var max *types.Block
	d.db.IterateUnConfirms(func(node *types.Block) {
		if max == nil || node.Height() > max.Height() {
			// 1. Choose the longest fork
			max = node
		} else if node.Height() == max.Height() {
			// 2. Choose the one which has smaller hash in dictionary order
			nodeHash := node.Hash()
			maxHash := max.Hash()
			if bytes.Compare(nodeHash[:], maxHash[:]) < 0 {
				max = node
			}
		}
	})
	return max
}

func (d *Dpovp) CanBeStable(height uint32, confirmsCount int) bool {
	nodeCount := d.dm.GetDeputiesCount(height)

	// nodeCount < 3 means two deputy nodes scene: One node mined a block and broadcasted it. Then it means two confirms after the receiver one's verification
	// +1 for the miner
	return nodeCount < 3 || uint32(confirmsCount+1) >= twoThird(nodeCount)
}
