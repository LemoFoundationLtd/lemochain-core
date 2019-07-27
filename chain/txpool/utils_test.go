package txpool

import (
	"crypto/ecdsa"
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"math/big"
	"strconv"
	"time"
)

var (
	testPrivate, _        = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb")
	testSigner            = types.DefaultSigner{}
	chainID        uint16 = 200
)

func createBoxTxRandom(from common.Address, childCnt int, expiration uint64) *types.Transaction {
	box := &types.Box{SubTxList: make(types.Transactions, 0)}

	for index := 0; index < childCnt; index++ {
		tmp := makeTx(common.HexToAddress(strconv.Itoa(index)), int64(expiration))
		box.SubTxList = append(box.SubTxList, tmp)
	}

	data, err := json.Marshal(box)
	if err != nil {
		panic(err)
	}

	return makeBoxTransaction(from, data, expiration)
}

func createDoubleBoxTxRandom(from common.Address, childCnt int, expiration uint64) (*types.Transaction, *types.Transaction) {
	box1 := &types.Box{SubTxList: make(types.Transactions, 0)}
	box2 := &types.Box{SubTxList: make(types.Transactions, 0)}

	box := &types.Box{SubTxList: make(types.Transactions, 0)}
	for index := 0; index < childCnt; index++ {
		tmp := makeTx(common.HexToAddress(strconv.Itoa(index)), int64(expiration))
		if index%4 == 0 {
			box1.SubTxList = append(box.SubTxList, tmp)
			box2.SubTxList = append(box.SubTxList, tmp)
		} else {
			if index%2 == 0 {
				box1.SubTxList = append(box.SubTxList, tmp)
			} else {
				box2.SubTxList = append(box.SubTxList, tmp)
			}
		}
	}

	data1, err := json.Marshal(box1)
	if err != nil {
		panic(err)
	}

	data2, err := json.Marshal(box2)
	if err != nil {
		panic(err)
	}

	return makeBoxTransaction(from, data1, expiration+1), makeBoxTransaction(from, data2, expiration+2)
}

func makeTxRandom(to common.Address) *types.Transaction {
	return makeTx(to, int64(time.Now().Unix()+300))
}

func makeTx(to common.Address, expiration int64) *types.Transaction {
	return makeTransaction(testPrivate,
		to,
		params.OrdinaryTx,
		new(big.Int).SetInt64(100),
		common.Big1,
		uint64(expiration),
		1000000)
}

func makeExpirationTx(to common.Address) *types.Transaction {
	return makeTx(to, int64(time.Now().Unix()-2*int64(params.TransactionExpiration)))
}

func makeBoxTransaction(from common.Address, data []byte, expiration uint64) *types.Transaction {
	return types.NoReceiverTransaction(from, new(big.Int).SetInt64(10000), 20000, new(big.Int).SetInt64(30000), data, params.BoxTx, chainID, expiration, "", "")
}

func makeTransaction(fromPrivate *ecdsa.PrivateKey, to common.Address, txType uint16, amount, gasPrice *big.Int, expiration uint64, gasLimit uint64) *types.Transaction {
	pubKey := fromPrivate.PublicKey
	from := crypto.PubkeyToAddress(pubKey)
	tx := types.NewTransaction(from, to, amount, gasLimit, gasPrice, []byte{}, txType, chainID, expiration, "", "")
	return signTransaction(tx, fromPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := testSigner.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}
