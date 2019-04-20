package consensus

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
	"math/big"
)

var (
	minPrecision = common.Lemo2Mo("1") // 1 LEMO
)

const MaxExtraDataLen = 256

type Engine interface {
	VerifyBeforeTxProcess(block *types.Block) error
	VerifyAfterTxProcess(block, computedBlock *types.Block) error
	VerifyConfirmPacket(height uint32, blockHash common.Hash, sigList []types.SignData) error
	TrySwitchFork(stable, oldCurrent *types.Block) *types.Block
	ChooseNewFork() *types.Block
	CanBeStable(height uint32, confirmsCount int) bool
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

// TrySwitchFork switch fork if its length reached to a multiple of "deputy nodes count * 2/3"
func (d *Dpovp) TrySwitchFork(stable, current *types.Block) *types.Block {
	maxHeightBlock := d.ChooseNewFork()
	// make sure the fork is the first one reaching the height
	if maxHeightBlock != nil && maxHeightBlock.Height() > current.Height() {
		judgeLength := d.dm.TwoThirdDeputyCount(maxHeightBlock.Height())
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
	// nodeCount < 3 means two deputy nodes scene: One node mined a block and broadcasted it. Then it means two confirms after the receiver one's verification
	if d.dm.GetDeputiesCount(height) < 3 {
		return true
	}

	// +1 for the miner
	return uint32(confirmsCount+1) >= d.dm.TwoThirdDeputyCount(height)
}
