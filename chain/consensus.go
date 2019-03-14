package chain

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"time"
)

const MaxExtraDataLen = 256

type Engine interface {
	VerifyHeader(block *types.Block) error

	Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog) (*types.Block, error)

	Finalize(header *types.Header, am *account.Manager)
}

type Dpovp struct {
	timeoutTime int64
	db          protocol.ChainDB
}

func NewDpovp(timeout int64, db protocol.ChainDB) *Dpovp {
	dpovp := &Dpovp{
		timeoutTime: timeout,
		db:          db,
	}
	return dpovp
}

// verifyHeaderTime verify that the block timestamp is less than the current time
func verifyHeaderTime(block *types.Block) error {
	header := block.Header
	blockTime := header.Time
	timeNow := time.Now().Unix()
	if int64(blockTime)-timeNow > 1 { // Prevent validation failure due to time error
		log.Error("verifyHeader: block in the future")
		return ErrVerifyHeaderFailed
	}
	return nil
}

// verifyHeaderSignData verify the block signature data
func verifyHeaderSignData(block *types.Block) error {
	header := block.Header
	hash := block.Hash()
	pubKey, err := crypto.Ecrecover(hash[:], header.SignData)
	if err != nil {
		log.Errorf("verifyHeaderSignData: illegal signData. %s", err)
		return ErrVerifyHeaderFailed
	}
	node := deputynode.Instance().GetDeputyByAddress(header.Height, header.MinerAddress)
	if node == nil {
		nodes := deputynode.Instance().GetDeputiesByHeight(block.Height(), false)
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
		hash := types.DeriveDeputyRootSha(block.DeputyNodes)
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
	nodeCount := deputynode.Instance().GetDeputiesCount(block.Height()) // The total number of nodes
	// There's only one out block node
	if nodeCount == 1 {
		return nil
	}
	// Verify that the block timestamp is less than the current time
	if err := verifyHeaderTime(block); err != nil {
		return err
	}
	// Verify the block signature data
	if err := verifyHeaderSignData(block); err != nil {
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
	var slot int
	if (header.Height > params.InterimDuration+1) && (header.Height-params.InterimDuration-1)%params.TermDuration == 0 {
		rank := deputynode.Instance().GetNodeRankByAddress(header.Height, block.MinerAddress())
		if rank == -1 {
			return ErrVerifyHeaderFailed
		}
		slot = rank + 1
		log.Debugf("rank: %d", rank)
	} else {
		slot = deputynode.Instance().GetSlot(header.Height, parent.Header.MinerAddress, header.MinerAddress)
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

// Seal packaged into a block
func (d *Dpovp) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog) (*types.Block, error) {
	block := types.NewBlock(header, txs, changeLog, nil)
	return block, nil
}

// Finalize
func (d *Dpovp) Finalize(header *types.Header, am *account.Manager) {
	// handout rewards
	if deputynode.Instance().TimeToHandOutRewards(header.Height) {
		term := (header.Height - params.InterimDuration) / params.TermDuration
		termRewards, err := getTermRewardValue(am, term)
		if err != nil {
			log.Warnf("Rewards failed: %v", err)
			return
		}
		rewards := deputynode.CalcSalary(header.Height, termRewards)
		for _, item := range rewards {
			acc := am.GetAccount(item.Address)
			balance := acc.GetBalance()
			balance.Add(balance, item.Salary)
			acc.SetBalance(balance)
		}
	}
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
