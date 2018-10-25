package chain

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"time"
)

const MaxExtraDataLen = 256

type Engine interface {
	VerifyHeader(block *types.Block) error

	Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error)

	Finalize(header *types.Header, am *account.Manager)
}

type Dpovp struct {
	timeoutTime   int64
	blockInternal int64
	db            protocol.ChainDB
}

func NewDpovp(timeout int64, db protocol.ChainDB) *Dpovp {
	dpovp := &Dpovp{
		timeoutTime:   timeout,
		blockInternal: 3000, // todo
		db:            db,
	}
	return dpovp
}

// verifyHeaderTime verify that the block timestamp is less than the current time
func verifyHeaderTime(block *types.Block) error {
	header := block.Header
	blockTime := int64(header.Time.Uint64())
	timeNow := time.Now().Unix()
	if blockTime-timeNow > 1 { // Prevent validation failure due to time error
		return errors.New("block in the future")
	}
	return nil
}

// verifyHeaderSignData verify the block signature data
func verifyHeaderSignData(block *types.Block) error {
	header := block.Header
	hash := block.Hash()
	pubKey, err := crypto.Ecrecover(hash[:], header.SignData)
	if err != nil {
		return err
	}
	node := deputynode.Instance().GetDeputyByAddress(block.Height(), header.LemoBase)
	if node == nil || bytes.Compare(pubKey[1:], node.NodeID) != 0 {
		return fmt.Errorf("illegal block. height:%d, hash:%s", header.Height, header.Hash().Hex())
	}
	return nil
}

// VerifyHeader verify block header
func (d *Dpovp) VerifyHeader(block *types.Block) error {
	// Verify that the block timestamp is less than the current time
	if err := verifyHeaderTime(block); err != nil {
		return err
	}
	// Verify the block signature data
	if err := verifyHeaderSignData(block); err != nil {
		return err
	}

	// verify extra data
	if len(block.Header.Extra) > MaxExtraDataLen {
		return fmt.Errorf("extra data's max len is %d bytes, current length is %d", MaxExtraDataLen, len(block.Header.Extra))
	}

	// Verify that the node is out of the block ?
	header := block.Header
	parent, _ := d.db.GetBlock(header.ParentHash, header.Height-1)
	if parent == nil {
		return fmt.Errorf("can't get parent block. height:%d, hash:%s", header.Height-1, header.ParentHash)
	}
	if parent.Header.Height == 0 { // parentBlock is genesisBlock
		log.Debug("verifyHeader: parent is genesis block")
		return nil
	}
	// The time interval between the current block and the parent block. unit：ms
	timeSpan := int64(header.Time.Uint64()-parent.Header.Time.Uint64()) * 1000
	// The time interval between the current block and the parent block should be at least greater than the block internal
	if timeSpan < d.blockInternal {
		log.Debugf("verifyHeader: verify failed. timeSpan:%d is smaller than blockInternal:%d -1", timeSpan, d.blockInternal)
		return fmt.Errorf("verifyHeader: block internal need to large than %d millsecond -1", d.blockInternal)
	}
	nodeCount := deputynode.Instance().GetDeputiesCount() // The total number of nodes
	slot := deputynode.Instance().GetSlot(header.Height, parent.Header.LemoBase, header.LemoBase)
	oneLoopTime := int64(nodeCount) * d.timeoutTime // All timeout times for a round of deputy nodes
	oldTimeSpan := timeSpan
	// There's only one out block node
	if nodeCount == 1 {
		// if timeSpan < d.blockInternal { // The time interval between blocks should be at least blockInternal
		// 	log.Debug("verifyHeader: Only one node, but not sleep enough time -1")
		// 	return fmt.Errorf("verifyHeader: Only one node, but not sleep enough time -1")
		// }
		// log.Debug("verifyHeader: nodeCount == 1")
		return nil
	}

	if slot == 0 { // The last block was made for itself
		timeSpan = timeSpan % oneLoopTime
		if timeSpan >= oneLoopTime-d.timeoutTime {
			// normal situation
		} else {
			log.Debugf("verifyHeader: verify failed. oldTimeSpan: %d timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d -2", oldTimeSpan, timeSpan, nodeCount, slot, oneLoopTime)
			return fmt.Errorf("verifyHeader: Not turn to produce block -2")
		}
		return nil
	} else if slot == 1 {
		if timeSpan < oneLoopTime { // The interval is less than one cycle
			if timeSpan >= d.blockInternal && timeSpan < d.timeoutTime {
				// normal situation
			} else {
				log.Debugf("verifyHeader: verify failed.timeSpan< oneLoopTime. timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d -3", timeSpan, nodeCount, slot, oneLoopTime)
				return fmt.Errorf("verifyHeader: Not turn to produce block -3")
			}
		} else { // The interval is more than one cycle
			timeSpan = timeSpan % oneLoopTime
			if timeSpan < d.timeoutTime {
				// normal situation
			} else {
				log.Debugf("verifyHeader: verify failed. timeSpan>=oneLoopTime. oldTimeSpan: %d timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d -4", oldTimeSpan, timeSpan, nodeCount, slot, oneLoopTime)
				return fmt.Errorf("verifyHeader: Not turn to produce block -4")
			}
		}
	} else {
		timeSpan = timeSpan % oneLoopTime
		if timeSpan/d.timeoutTime == int64(slot-1) {
			// normal situation
		} else {
			log.Debugf("verifyHeader: verify failed. oldTimeSpan: %d timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d -5", oldTimeSpan, timeSpan, nodeCount, slot, oneLoopTime)
			return fmt.Errorf("verifyHeader: Not turn to produce block -5")
		}
	}
	return nil
}

// Seal packaged into a block
func (d *Dpovp) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error) {
	block := types.NewBlock(header, txs, changeLog, events, nil, nil)
	return block, nil
}

// Finalize
func (d *Dpovp) Finalize(header *types.Header, am *account.Manager) {
	// handout rewards
	if deputynode.Instance().TimeToHandOutRewards(header.Height) {
		rewards := deputynode.CalcReward(header.Height)
		for _, item := range rewards {
			account := am.GetAccount(item.Address)
			balance := account.GetBalance()
			balance.Add(balance, item.Reward)
			account.SetBalance(balance)
		}
	}
}
