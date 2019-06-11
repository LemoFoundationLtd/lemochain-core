package transaction

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

var (
	godFrom, _ = common.StringToAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	godPriv, _ = crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
)

// getBoxTx length == 箱子中交易数量
func getBoxTx(length int, containBoxTx bool) *types.Transaction {
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
	if containBoxTx {
		boxTx := types.NoReceiverTransaction(godFrom, big.NewInt(100), uint64(21000), common.Big1, nil, params.BoxTx, chainID, uint64(time.Now().Unix()+60*30), "", "")
		txs = append(txs, boxTx)
	}

	box := &Box{
		Txs: txs,
	}
	data, err := json.Marshal(box)
	if err != nil {
		panic(err)
	}
	boxTx := types.NewContractCreation(godFrom, nil, uint64(900000), common.Big1, data, params.BoxTx, chainID, uint64(time.Now().Unix()+60*30), "box txs", "")
	signBoxTx, err := types.MakeSigner().SignTx(boxTx, godPriv)
	if err != nil {
		panic(err)
	}
	return signBoxTx
}

// Test_unmarshalBoxTxs
func Test_unmarshalBoxTxs(t *testing.T) {
	normalBoxTx := getBoxTx(5, false)
	p := &TxProcessor{
		ChainID: chainID,
	}
	b := NewBoxTxEnv(p)
	box, err := b.unmarshalBoxTxs(normalBoxTx.Data())
	if err != nil {
		panic(err)
	}
	data, err := json.Marshal(box)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, normalBoxTx.Data(), data)
	// box中含有boxTx
	errBoxTx := getBoxTx(5, true)
	_, err = b.unmarshalBoxTxs(errBoxTx.Data())
	assert.Equal(t, ErrVerifyBoxTxs, err)
}

// TestBoxTxEnv_RunBoxTxs
func TestBoxTxEnv_RunBoxTxs(t *testing.T) {
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	bc := newTestChain(db)
	p := NewTxProcessor(godFrom, chainID, bc, am, db)
	b := NewBoxTxEnv(p)

	total, _ := new(big.Int).SetString("1600000000000000000000000000", 10)
	am.GetAccount(godFrom).SetBalance(total) // 给godFrom  16亿lemo
	minerAddr := common.HexToAddress("0x12321")
	am.GetAccount(minerAddr).SetCandidateState(types.CandidateKeyIncomeAddress, minerAddr.String()) // 设置矿工的income地址
	header := &types.Header{
		MinerAddress: minerAddr,
		Height:       1,
		GasLimit:     uint64(500000000),
		Time:         uint32(time.Now().Unix()),
	}
	gp := new(types.GasPool).AddGas(header.GasLimit)
	txNum := 5
	boxTx := getBoxTx(txNum, false)
	gasUsed, err := b.RunBoxTxs(gp, boxTx, header, 1)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, uint64(txNum)*params.OrdinaryTxGas, gasUsed)                                                      // 测试盒子中的交易花费的gas
	assert.Equal(t, new(big.Int).Mul(big.NewInt(int64(gasUsed)), common.Big1), am.GetAccount(minerAddr).GetBalance()) // 测试盒子交易执行完之后给矿工的交易打包费用
}
