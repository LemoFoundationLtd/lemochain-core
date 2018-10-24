package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/network/synchronise"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"sync"
)

type Lemochain struct {
	chainConfig *params.ChainConfig

	txPool *chain.TxPool
	chain  *chain.BlockChain

	pm     *synchronise.ProtocolManager
	db     protocol.ChainDB
	engine *chain.Dpovp

	am *account.Manager

	miner    *miner.Miner
	gasPrice *big.Int

	lemoBase common.Address

	// networkId uint64

	lock sync.RWMutex
}
