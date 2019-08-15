package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

// txPoolForValidator is a txPool for test. It only contains a bool which will be returned by VerifyTxInBlock
type txPoolForValidator struct {
	blockIsValid bool
}

func (txPoolForValidator) Get(time uint32, size int) []*types.Transaction {
	panic("implement me")
}

func (txPoolForValidator) DelInvalidTxs(txs []*types.Transaction) {
	panic("implement me")
}

func (tp txPoolForValidator) VerifyTxInBlock(block *types.Block) bool {
	return tp.blockIsValid
}

func (txPoolForValidator) RecvBlock(block *types.Block) {
	panic("implement me")
}

func (txPoolForValidator) PruneBlock(block *types.Block) {
	panic("implement me")
}

func TestNewValidator(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

	fm := NewValidator(1000, testBlockLoader{}, dm, txPoolForValidator{}, testCandidateLoader{})
	assert.Equal(t, uint64(1000), fm.timeoutTime)
}

func Test_verifyParentHash(t *testing.T) {
	// no parent
	loader := createUnconfirmBlockLoader([]int{})
	parent, err := verifyParentHash(testBlocks[0], loader)
	assert.Equal(t, ErrVerifyBlockFailed, err)

	// exist parent
	loader = createUnconfirmBlockLoader([]int{0, 1})
	parent, err = verifyParentHash(testBlocks[1], loader)
	assert.NoError(t, err)
	assert.Equal(t, testBlocks[0], parent)
}
