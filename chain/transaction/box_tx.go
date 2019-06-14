package transaction

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/math"
	"math/big"
	"time"
)

var (
	ErrNestedBoxTx        = errors.New("box transaction cannot be a sub transaction")
	ErrApplyBoxTxsTimeout = errors.New("apply box txs timeout")
)

type BoxTxEnv struct {
	p *TxProcessor
}

func NewBoxTxEnv(p *TxProcessor) *BoxTxEnv {
	return &BoxTxEnv{p}
}

// unmarshalBoxTxs
func (b *BoxTxEnv) unmarshalBoxTxs(data []byte) (*types.Box, error) {
	box, err := types.GetBox(data)
	if err != nil {
		return nil, err
	}
	return box, nil
}

// RunBoxTx 执行箱子中的子交易
func (b *BoxTxEnv) RunBoxTxs(gp *types.GasPool, boxTx *types.Transaction, header *types.Header, txIndex uint, restApplyTime int64) (uint64, error) {
	box, err := b.unmarshalBoxTxs(boxTx.Data())
	if err != nil {
		return 0, err
	}

	var (
		gasUsed     = uint64(0)
		totalGasFee = new(big.Int)
	)
	now := time.Now() // 设置执行箱子中的交易时间限制
	for _, tx := range box.SubTxList {
		if int64(time.Since(now)/time.Millisecond) > restApplyTime {
			log.Errorf("Box txs runtime: %fs", time.Since(now).Seconds())
			return 0, ErrApplyBoxTxsTimeout
		}
		gas, err := b.p.applyTx(gp, header, tx, txIndex, header.Hash(), math.MaxInt64)
		if err != nil {
			return 0, err
		}
		gasUsed += gas
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		totalGasFee.Add(totalGasFee, fee)
	}
	// 为矿工执行箱子交易中的交易发放奖励
	b.p.chargeForGas(totalGasFee, header.MinerAddress)

	return gasUsed, nil
}
