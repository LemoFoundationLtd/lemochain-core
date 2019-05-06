package consensus

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"time"
)

// Validator verify block
type Validator struct {
	timeoutTime uint64
	db          protocol.ChainDB
	dm          *deputynode.Manager
	canLoader   CandidateLoader
}

func NewValidator(timeout uint64, db protocol.ChainDB, dm *deputynode.Manager, canLoader CandidateLoader) *Validator {
	return &Validator{
		timeoutTime: timeout,
		db:          db,
		dm:          dm,
		canLoader:   canLoader,
	}
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
func verifyDeputy(block *types.Block, canLoader CandidateLoader) error {
	if deputynode.IsSnapshotBlock(block.Height()) {
		deputies := canLoader.LoadTopCandidates(block.ParentHash())
		// Make sure the DeputyRoot is derived from the deputy nodes in block body
		hash := block.DeputyNodes.MerkleRootSha()
		if bytes.Compare(hash[:], block.DeputyRoot()) != 0 {
			log.Error("Consensus verify fail: deputyRoot is incorrect", "deputyRoot", common.ToHex(block.DeputyRoot()), "expected", hash.Hex())
			return ErrVerifyBlockFailed
		}
		// Make sure the DeputyRoot is match with local deputy nodes data
		hash = deputies.MerkleRootSha()
		if bytes.Compare(hash[:], block.DeputyRoot()) != 0 {
			log.Error("Consensus verify fail: deputyNodes is incorrect", "deputyRoot", common.ToHex(block.DeputyRoot()), "expected", hash.Hex())
			log.Errorf("nodes in body: %s\nnodes in local: %s", block.DeputyNodes, deputies)
			return ErrVerifyBlockFailed
		}
	}
	return nil
}

// verifyChangeLog verify the LogRoot and ChangeLogs in block body
func verifyChangeLog(block *types.Block, computedLogs types.ChangeLogSlice) error {
	// The block may contains change logs from some protocol
	if len(block.ChangeLogs) > 0 {
		// Make sure the LogRoot is derived from the change logs in block body
		hash := block.ChangeLogs.MerkleRootSha()
		if hash != block.LogRoot() {
			log.Error("Consensus verify fail: logRoot is incorrect", "logRoot", block.LogRoot().Hex(), "expected", hash.Hex())
			return ErrVerifyBlockFailed
		}
	}
	// Make sure the LogRoot is match with local change logs data
	hash := computedLogs.MerkleRootSha()
	if hash != block.LogRoot() {
		log.Error("Consensus verify fail: changeLogs is incorrect", "logRoot", block.LogRoot().Hex(), "expected", hash.Hex())
		log.Errorf("Local logs: %s", computedLogs)
		return ErrVerifyBlockFailed
	}
	return nil
}

// verifyExtraData verify extra data in block header
func verifyExtraData(block *types.Block) error {
	if len(block.Extra()) > params.MaxExtraDataLen {
		log.Error("Consensus verify fail: extra data is too long", "current", len(block.Extra()), "max", params.MaxExtraDataLen)
		return ErrVerifyHeaderFailed
	}
	return nil
}

// verifyMineSlot verify the miner slot of deputy node
func verifyMineSlot(block *types.Block, parent *types.Block, timeoutTime uint64, dm *deputynode.Manager) error {
	if block.Height() == 1 {
		// first block, no need to verify slot
		return nil
	}

	slot, err := GetMinerDistance(block.Height(), parent.MinerAddress(), block.MinerAddress(), dm)
	if err != nil {
		log.Error("Consensus verify fail: can't calculate slot", "block.Height", block.Height(), "parent.MinerAddress", parent.MinerAddress(), "block.MinerAddress", block.MinerAddress(), "err", err)
		return ErrVerifyHeaderFailed
	}

	// The time interval between the current block and the parent block. unit：ms
	totalTimeSpan := uint64(block.Time()-parent.Header.Time) * 1000
	nodeCount := dm.GetDeputiesCount(block.Height())
	oneLoopTime := uint64(nodeCount) * timeoutTime // All timeout times for a round of deputy nodes

	timeSpan := totalTimeSpan % oneLoopTime
	if slot == 0 { // The last block was made by itself
		if timeSpan < oneLoopTime-timeoutTime {
			log.Error("Consensus verify fail: mined twice in one loop", "slot", slot, "totalTimeSpan", totalTimeSpan, "timeSpan", timeSpan, "nodeCount", nodeCount, "oneLoopTime", oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	} else if slot == 1 {
		if timeSpan >= timeoutTime {
			log.Error("Consensus verify fail: time out", "slot", slot, "totalTimeSpan", totalTimeSpan, "timeSpan", timeSpan, "nodeCount", nodeCount, "oneLoopTime", oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	} else {
		if timeSpan/timeoutTime != uint64(slot-1) {
			log.Error("Consensus verify fail: not in turn", "slot", slot, "totalTimeSpan", totalTimeSpan, "timeSpan", timeSpan, "nodeCount", nodeCount, "oneLoopTime", oneLoopTime)
			return ErrVerifyHeaderFailed
		}
	}
	return nil
}

// VerifyConfirmPacket verify the confirm data in block body, return valid new confirms and last confirm verification error
func (v *Validator) VerifyConfirmPacket(height uint32, blockHash common.Hash, sigList []types.SignData) ([]types.SignData, error) {
	block, err := v.db.GetBlockByHash(blockHash)
	if err != nil {
		return nil, ErrBlockNotExist
	}
	if block.Height() != height {
		log.Warn("Unmatched confirm height and hash", "height", height, "hash", blockHash.Hex())
		return nil, ErrInvalidSignedConfirmInfo
	}
	return v.VerifyNewConfirms(block, sigList, v.dm)
}

// verifyConfirm verify the confirm data, return valid new confirms and last confirm verification error
func (v *Validator) VerifyNewConfirms(block *types.Block, sigList []types.SignData, dm *deputynode.Manager) ([]types.SignData, error) {
	hash := block.Hash()
	validConfirms := make([]types.SignData, 0, len(sigList))
	var lastErr error = nil

	for _, sig := range sigList {
		nodeID, err := sig.RecoverNodeID(hash)
		if err != nil {
			log.Warn("Invalid confirm signature", "hash", hash.Hex(), "sig", common.ToHex(sig[:]), "err", err)
			lastErr = ErrInvalidSignedConfirmInfo
			continue
		}
		// find the signer in deputy nodes
		if deputy := dm.GetDeputyByNodeID(block.Height(), nodeID); deputy == nil {
			log.Warn("Invalid confirm signer", "nodeID", common.ToHex(nodeID))
			lastErr = ErrInvalidConfirmSigner
			continue
		}
		if block.IsConfirmExist(sig) {
			log.Warn("Duplicate confirm", "hash", hash.Hex(), "signer", common.ToHex(nodeID[:8]))
			lastErr = ErrInvalidConfirmSigner
			continue
		}
		validConfirms = append(validConfirms, sig)
	}
	return validConfirms, lastErr
}

// VerifyBeforeTxProcess verify the block data which has no relationship with the transaction processing result
func (v *Validator) VerifyBeforeTxProcess(block *types.Block) error {
	// cache parent block
	parent, err := verifyParentHash(block, v.db)
	if err != nil {
		return err
	}
	// verify miner address and signData
	if err := verifySigner(block, v.dm); err != nil {
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
	if err := verifyDeputy(block, v.canLoader); err != nil {
		return err
	}
	if err := verifyExtraData(block); err != nil {
		return err
	}
	if err := verifyMineSlot(block, parent, v.timeoutTime, v.dm); err != nil {
		return err
	}
	return nil
}

// VerifyAfterTxProcess verify the block data which computed from transactions
func (v *Validator) VerifyAfterTxProcess(block, computedBlock *types.Block) error {
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

// JudgeDeputy check if the deputy node is evil by his new block
func (v *Validator) JudgeDeputy(newBlock *types.Block) bool {
	newBlockNodeID, err := newBlock.SignerNodeID()
	if err != nil {
		log.Error("no NodeID, can't judge the deputy", "err", err)
		return false
	}

	isEvil := false
	// check if the deputy mine two blocks at same height
	v.db.IterateUnConfirms(func(node *types.Block) {
		// same height but different block
		if node.Height() == newBlock.Height() && node.Hash() != newBlock.Hash() {
			nodeID, err := node.SignerNodeID()
			if err != nil {
				log.Error("no NodeID, can't judge the deputy", "err", err)
				return
			}

			// same miner
			if bytes.Compare(newBlockNodeID, nodeID) == 0 {
				log.Warnf("The deputy %x is evil !!! It mined block %s and %s at same height %d", nodeID, newBlock.Hash().Prefix(), node.Hash().Prefix(), newBlock.Height())
				isEvil = true
			}
		}
	})
	return isEvil
}
