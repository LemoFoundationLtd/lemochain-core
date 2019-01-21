package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
)

// onlyReadManager
func ReadOnlyManager(blockHash common.Hash, db protocol.ChainDB) *Manager {
	return NewManager(blockHash, db)
}