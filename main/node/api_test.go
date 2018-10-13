package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/testify/assert"
	"math/big"
	"testing"
)

//

// 发送交易api的测试用例
func TestTxAPI_SendTx(t *testing.T) {
	testTx := types.NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 200, big.NewInt(1544584596), "aa", []byte{34})
	txCh := make(chan types.Transactions, 100)
	pool := chain.NewTxPool(nil, txCh)
	txAPI := NewTxAPI(pool)
	encodeTx, err := rlp.EncodeToBytes(testTx)
	if err != nil {
		t.Error(t, err)
	}
	byteTx, err := txAPI.SendTx(encodeTx)
	if err != nil {
		t.Error(t, err)
	}

	assert.Equal(t, testTx.Hash(), byteTx)
}
