package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
)

// Confirmer process the confirm logic
type Confirmer struct {
	db      protocol.ChainDB
	dm      *deputynode.Manager
	lastSig blockSig
}

type blockSig struct {
	Height uint32
	Hash   common.Hash
}

func NewConfirmer(dm *deputynode.Manager, db protocol.ChainDB, stable *types.Block) *Confirmer {
	confirmer := &Confirmer{
		db: db,
		dm: dm,
	}
	confirmer.lastSig.Height = stable.Height()
	confirmer.lastSig.Hash = stable.Hash()
	return confirmer
}

// TryConfirm try to sign and save a confirm into a received block
func (c *Confirmer) TryConfirm(block *types.Block) (types.SignData, bool) {
	if !c.needConfirm(block) {
		return types.SignData{}, false
	}

	hash := block.Hash()

	sig, err := c.signBlock(hash)
	if err != nil {
		log.Error("sign for confirm data error", "err", err)
		return types.SignData{}, false
	}

	if block.IsConfirmExist(sig) {
		return types.SignData{}, false
	}

	block.Confirms = append(block.Confirms, sig)
	c.lastSig.Height = block.Height()
	c.lastSig.Hash = hash

	return sig, true
}

func (c *Confirmer) needConfirm(block *types.Block) bool {
	// test if we are deputy node
	if !c.dm.IsSelfDeputyNode(block.Height()) {
		return false
	}
	// test if it contains enough confirms
	if IsConfirmEnough(block, c.dm) {
		return false
	}
	// It's not necessary to test if the block has mined or been confirmed by myself. Because confirmed blocks must be in database. So they will be dropped by network module at the beginning

	// the block is at same fork with last signed block
	if block.ParentHash() == c.lastSig.Hash {
		return true
	}
	// the block is deputyCount*2/3 far from signed block
	signDistance := c.dm.TwoThirdDeputyCount(block.Height())
	// not ">=" so that we would never need to confirm a new block after switch fork
	if block.Height() > c.lastSig.Height+signDistance {
		return true
	}

	log.Debug("can't confirm the block", "lastSig", c.lastSig.Height, "height", block.Height(), "minDistance", signDistance)
	return false
}

// BatchConfirmStable confirm and broadcast unsigned stable blocks one by one
func (c *Confirmer) BatchConfirmStable(startHeight, endHeight uint32) []*network.BlockConfirmData {
	if endHeight < startHeight {
		return nil
	}

	result := make([]*network.BlockConfirmData, 0, endHeight-startHeight+1)
	for i := startHeight; i <= endHeight; i++ {
		block, err := c.db.GetBlockByHeight(i)
		if err != nil {
			log.Error("Load block fail, can't confirm it", "height", i)
			continue
		}
		if sig := c.tryConfirmStable(block); sig != nil {
			result = append(result, &network.BlockConfirmData{
				Hash:     block.Hash(),
				Height:   block.Height(),
				SignInfo: *sig,
			})
		}
		c.lastSig.Height = block.Height()
		c.lastSig.Hash = block.Hash()
	}

	return result
}

// SetLastSig
func (c *Confirmer) SetLastSig(block *types.Block) {
	c.lastSig.Height = block.Height()
	c.lastSig.Hash = block.Hash()
}

// TryConfirmStable try to sign and save a confirm into a stable block
func (c *Confirmer) tryConfirmStable(block *types.Block) *types.SignData {
	// test if we are deputy node
	if !c.dm.IsSelfDeputyNode(block.Height()) {
		return nil
	}
	// test if it contains enough confirms
	if IsConfirmEnough(block, c.dm) {
		return nil
	}

	hash := block.Hash()
	sig, err := c.signBlock(hash)
	if err != nil {
		log.Error("sign for confirm data error", "err", err)
		return nil
	}

	if block.IsConfirmExist(sig) {
		return nil
	}

	_, _ = c.SaveConfirm(block, []types.SignData{sig})
	return &sig
}

// SaveConfirm save a confirm to store, then return a new block
func (c *Confirmer) SaveConfirm(block *types.Block, sigList []types.SignData) (*types.Block, error) {
	newBlock, err := c.db.SetConfirms(block.Hash(), sigList)
	if err != nil {
		log.Errorf("SetConfirm failed: %v", err)
		return nil, err
	}
	return newBlock, nil
}

// signBlock sign a block and return signData
func (c *Confirmer) signBlock(hash common.Hash) (types.SignData, error) {
	// TODO make a cache. share with assembler
	// sign
	privateKey := deputynode.GetSelfNodeKey()
	sig, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		return types.SignData{}, err
	}
	var signData types.SignData
	copy(signData[:], sig)
	return signData, nil
}
