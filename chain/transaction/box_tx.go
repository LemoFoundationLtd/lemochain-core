package transaction

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math/big"
	"time"
)

const (
	applyBoxTxsTimeout = 4 // 执行箱子交易超时时间4s
)

var (
	ErrVerifyBoxTxs       = errors.New("not container box tx")
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
	box := &types.Box{}
	err := json.Unmarshal(data, box)
	if err != nil {
		return nil, err
	}
	// 验证箱子中的交易
	for _, tx := range box.SubTxList {
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
	txsHashMap := make(types.BoxTxsMap)
	now := time.Now() // 设置执行箱子中的交易时间限制
	for _, tx := range box.SubTxList {
		if time.Since(now).Seconds() > applyBoxTxsTimeout {
			log.Errorf("Box txs runtime: %fs", time.Since(now).Seconds())
			return 0, ErrApplyBoxTxsTimeout
		}
		gas, err := b.p.applyTx(gp, header, tx, txIndex, header.Hash())
		if err != nil {
			return 0, err
		}
		gasUsed += gas
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		totalGasFee.Add(totalGasFee, fee)
		txsHashMap[tx.Hash()] = tx
	}
	// 为矿工执行箱子交易中的交易发放奖励
	b.p.chargeForGas(totalGasFee, header.MinerAddress)
	// 保存箱子中的交易到from账户中
	from := boxTx.From()
	fromAcc := b.p.am.GetAccount(from)
	storeKey := boxTx.Hash()
	storeValue, err := json.Marshal(txsHashMap)
	if err != nil {
		log.Errorf("Json marshal box txs error: %s", err)
		return 0, err
	}

	err = fromAcc.SetStorageState(storeKey, storeValue) // key为箱子本身交易的交易hash, value为箱子中txs的hashMap
	if err != nil {
		log.Errorf("SetStorageState to box transaction from account error: %s", err)
		return 0, err
	}

	return gasUsed, nil
}
