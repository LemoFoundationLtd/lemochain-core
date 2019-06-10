package transaction

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"math/big"
	"testing"
	"time"
)

var (
	godFrom, _ = common.StringToAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	godPriv, _ = crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
)

// Test_unmarshalBoxTxs
func Test_unmarshalBoxTxs(t *testing.T) {

}

// getBoxTxs
func getBoxTxs(length int) types.Transactions {
	txs := make(types.Transactions, 0)
	for i := 0; i < length; i++ {
		randPriv, _ := crypto.GenerateKey()
		by := crypto.FromECDSA(randPriv)
		to := common.BytesToAddress(by)

		tx := types.NewTransaction(godFrom, to, big.NewInt(100), uint64(21000), common.Big1, nil, params.OrdinaryTx, chainID, uint64(time.Now().Unix()+60*30), "", "")
		signTx, err := types.MakeSigner().SignTx(tx, godPriv)
		if err != nil {
			panic(err)
		}
		txs = append(txs, signTx)
	}
	return txs
}
