package txpool

import (
	"crypto/ecdsa"
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
	return makeTx(to, int64(time.Now().Unix()-2*int64(TransactionExpiration)))
}

func makeTransaction(fromPrivate *ecdsa.PrivateKey, to common.Address, txType uint16, amount, gasPrice *big.Int, expiration uint64, gasLimit uint64) *types.Transaction {
	tx := types.NewTransaction(to, amount, gasLimit, gasPrice, []byte{}, txType, chainID, expiration, "", "")
	return signTransaction(tx, fromPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := testSigner.SignTx(tx, private)
	if err != nil {
		panic(err)
	}
	return tx
}
