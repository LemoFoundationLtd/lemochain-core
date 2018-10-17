package chain

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"time"
)

type Engine interface {
	VerifyHeader(block *types.Block) error

	Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error)

	Finalize(header *types.Header)
}

type Dpovp struct {
	timeoutTime   int64
	blockInternal int64
	bc            *BlockChain
}

func NewDpovp(timeout, blockInternal int64) *Dpovp {
	dpovp := &Dpovp{
		timeoutTime:   timeout,
		blockInternal: blockInternal,
	}
	return dpovp
}

func (d *Dpovp) SetBlockChain(bc *BlockChain) {
	d.bc = bc
}

// verifyHeaderTime verify that the block timestamp is less than the current time
func verifyHeaderTime(block *types.Block) error {
	header := block.Header
	b_t := int64(header.Time.Uint64())
	t_n := time.Now().Unix()
	if b_t-t_n > 1 { // Prevent validation failure due to time error
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
	node := deputynode.Instance().GetNodeByAddress(block.Height(), header.LemoBase)
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

	// Verify that the node is out of the block ?
	header := block.Header
	parent := d.bc.GetBlock(header.ParentHash, header.Height-1)
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
		log.Debug(fmt.Sprintf("verifyHeader: timeSpan:%d is smaller than blockInternal:%d", timeSpan, d.blockInternal))
		return fmt.Errorf("verifyHeader: block is not enough newer than it's parent")
	}
	nodeCount := deputynode.Instance().GetDeputyNodesCount() // The total number of nodes
	slot := deputynode.Instance().GetSlot(header.Height, parent.Header.LemoBase, header.LemoBase)
	oneLoopTime := int64(nodeCount) * d.timeoutTime // All timeout times for a round of deputy nodes
	log.Debug(fmt.Sprintf("verifyHeader: timeSpan:%d nodeCount:%d slot:%d oneLoopTime:%d", timeSpan, nodeCount, slot, oneLoopTime))
	// There's only one out block node
	if nodeCount == 1 {
		// if timeSpan < d.blockInternal { // The time interval between blocks should be at least blockInternal
		// 	log.Debug("verifyHeader: Only one node, but not sleep enough time -1")
		// 	return fmt.Errorf("verifyHeader: Only one node, but not sleep enough time -1")
		// }
		log.Debug("verifyHeader: nodeCount == 1")
		return nil
	}

	if slot == 0 { // The last block was made for itself
		timeSpan = timeSpan % oneLoopTime
		log.Debug(fmt.Sprintf("verifyHeader: slot:0 timeSpan:%d", timeSpan))
		if timeSpan >= oneLoopTime-d.timeoutTime {
			// normal situation
		} else {
			log.Debug(fmt.Sprintf("verifyHeader: slot:0 verify failed"))
			return fmt.Errorf("verifyHeader: Not turn to produce block -2")
		}
		return nil
	} else if slot == 1 {
		if timeSpan < oneLoopTime { // The interval is less than one cycle
			if timeSpan >= d.blockInternal && timeSpan < d.timeoutTime {
				// normal situation
			} else {
				log.Debug(fmt.Sprintf("verifyHeader: slot:1, timeSpan<oneLoopTime, verify failed"))
				return fmt.Errorf("verifyHeader: Not turn to produce block -3")
			}
		} else { // The interval is more than one cycle
			timeSpan = timeSpan % oneLoopTime
			if timeSpan < d.timeoutTime {
				// normal situation
			} else {
				log.Debug(fmt.Sprintf("verifyHeader: slot:1,timeSpan>=oneLoopTime, verify failed"))
				return fmt.Errorf("verifyHeader: Not turn to produce block -4")
			}
		}
	} else {
		timeSpan = timeSpan % oneLoopTime
		log.Debug(fmt.Sprintf("verifyHeader: slot:%d timeSpan:%d", slot, timeSpan))
		if timeSpan/d.timeoutTime == int64(slot-1) {
			// normal situation
		} else {
			log.Debug(fmt.Sprintf("verifyHeader: slot>1, verify failed"))
			return fmt.Errorf("verifyHeader: Not turn to produce block -5")
		}
	}
	return nil
}

// Seal packaged into a block
func (d *Dpovp) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog, events []*types.Event) (*types.Block, error) {
	block := types.NewBlock(header, txs, changeLog, events, nil)
	return block, nil
}

// Finalize
func (d *Dpovp) Finalize(header *types.Header) {
	d.handOutRewards(header.Height)
}

// handOutRewards issue rewards
func (d *Dpovp) handOutRewards(height uint32) {
	if deputynode.Instance().TimeToHandOutRewards(height) {
		rewards := deputynode.CalcReward(height)
		for _, item := range rewards {
			account := d.bc.AccountManager().GetAccount(item.Address)
			balance := account.GetBalance()
			balance.Add(balance, item.Reward)
			account.SetBalance(balance)
		}
	}
}
