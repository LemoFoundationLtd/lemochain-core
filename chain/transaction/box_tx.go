package transaction

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"math/big"
	"time"
)

var (
	ErrVerifyBoxTxs = errors.New("not container box tx")
)

//go:generate gencodec -type Box -out gen_box_json.go
type Box struct {
	Txs types.Transactions `json:"txs"  gencodec:"required"`
}

type BoxTxEnv struct {
	p *TxProcessor
}

type BoxTxsMap map[common.Hash]*types.Transaction

func NewBoxTxEnv(p *TxProcessor) *BoxTxEnv {
	return &BoxTxEnv{p}
}

// unmarshalBoxTxs 解析并校验箱子中的交易
func (b *BoxTxEnv) unmarshalBoxTxs(data []byte) (*Box, error) {
	box := &Box{}
	err := json.Unmarshal(data, box)
	if err != nil {
		return nil, err
	}
	// 验证箱子中的交易
	for _, tx := range box.Txs {
		if tx.Type() == params.BoxTx {
			return nil, ErrVerifyBoxTxs
		}
		err = tx.VerifyTx(b.p.ChainID, uint64(time.Now().Unix()))
		if err != nil {
			return nil, err
		}
	}
	return box, nil
}

// RunBoxTx 执行箱子交易
func (b *BoxTxEnv) RunBoxTxs(gp *types.GasPool, boxTx *types.Transaction, header *types.Header, txIndex uint) (uint64, error) {
	box, err := b.unmarshalBoxTxs(boxTx.Data())
	if err != nil {
		return 0, err
	}

	var (
		gasUsed     = uint64(0)
		totalGasFee = new(big.Int)
	)
	mm := make(BoxTxsMap)
	for _, tx := range box.Txs {
		gas, err := b.p.applyTx(gp, header, tx, txIndex, header.Hash())
		if err != nil {
			return 0, err
		}
		gasUsed += gas
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		totalGasFee.Add(totalGasFee, fee)
		mm[tx.Hash()] = tx
	}
	// 为矿工执行箱子交易中的交易发放奖励
	b.p.chargeForGas(totalGasFee, header.MinerAddress)
	return gasUsed, nil
}
