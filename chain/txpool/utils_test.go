package txpool

import (
	"crypto/ecdsa"
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"math/big"
	"time"
)

var (
	testPrivate, _        = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb")
	testSigner            = types.DefaultSigner{}
	chainID        uint16 = 200
)

func makeBoxTx(amount int64, nowOffset int64, txs ...*types.Transaction) *types.Transaction {
	box := &types.Box{SubTxList: txs}
	data, err := json.Marshal(box)
	if err != nil {
		panic(err)
	}

	from := crypto.PubkeyToAddress(testPrivate.PublicKey)
	tx := types.NoReceiverTransaction(from, big.NewInt(amount), 20000, big.NewInt(30000), data, params.BoxTx, chainID, uint64(time.Now().Unix()+nowOffset), "", "")
	return signTransaction(tx, testPrivate)
}

func makeTx(amount int64, nowOffset int64) *types.Transaction {
	return makeTransaction(
		params.OrdinaryTx,
		new(big.Int).SetInt64(amount),
		uint64(time.Now().Unix()+nowOffset),
	)
}

func makeTransaction(txType uint16, amount *big.Int, expiration uint64) *types.Transaction {
	pubKey := testPrivate.PublicKey
	from := crypto.PubkeyToAddress(pubKey)
	tx := types.NewTransaction(from, common.HexToAddress("12AB"), amount, 1000000, common.Big1, []byte{}, txType, chainID, expiration, "", "")
	return signTransaction(tx, testPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := testSigner.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}

func makeBlock(height, timestamp uint32, txs ...*types.Transaction) *types.Block {
	txsTmp := (types.Transactions)(txs)
	return &types.Block{
		Header: &types.Header{
			TxRoot: txsTmp.MerkleRootSha(),
			Time:   timestamp,
			Height: height,
		},
		Txs: txs,
	}
}

// generateBlocks generate block forks like this: (the number in bracket is transaction name)
//          ┌─2(c)
// 0───1(a)─┼─3(b)───6(c)
//          ├─4────┬─7───9(bc)
//          │      └─8
//          └─5(box[cd])
// the blocks' time bucket belonging like this: (the number in bracket is index of time bucket)
//             ┌─2(1)
// 0(0)───1(1)─┼─3(2)───6(3)
//             ├─4(1)─┬─7(4)───9(4)
//             │      └─8(3)
//             └─5(2)
func generateBlocks() []*types.Block {
	cur := uint32(time.Now().Unix())
	blocks := make([]*types.Block, 0)
	txa := makeTx(0xa, 100)
	txb := makeTx(0xb, 100)
	txc := makeTx(0xc, 100)
	txd := makeTx(0xd, 100)

	appendBlock := func(index int, height, time uint32, parentIndex int, txs ...*types.Transaction) {
		block := makeBlock(height, time, txs...)
		block.Header.Extra = []byte{byte(index)}
		if parentIndex >= 0 {
			block.Header.ParentHash = blocks[parentIndex].Hash()
		}
		blocks = append(blocks, block)
	}
	appendBlock(0, 100, cur, -1)
	appendBlock(1, 101, cur+BucketDuration, 0, txa)
	appendBlock(2, 102, cur+BucketDuration, 1, txc)
	appendBlock(3, 102, cur+BucketDuration*2, 1, txb)
	appendBlock(4, 102, cur+BucketDuration, 1)
	appendBlock(5, 102, cur+BucketDuration*2, 1, makeBoxTx(0x105, 100, txc, txd))
	appendBlock(6, 103, cur+BucketDuration*3, 3, txc)
	appendBlock(7, 103, cur+BucketDuration*4, 4)
	appendBlock(8, 103, cur+BucketDuration*3, 4)
	appendBlock(9, 104, cur+BucketDuration*4, 7, txb, txc)
	return blocks
}
