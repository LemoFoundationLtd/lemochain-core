package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
)

// ReadOnlyManager
func ReadOnlyManager(blockHash common.Hash, db protocol.ChainDB) *Manager {
	manager := &Manager{
		db:            db,
		baseBlockHash: blockHash,
	}
	manager.loadBaseBlock()
	manager.acctDb = db.GetActDatabase(blockHash)
	manager.processor = &LogProcessor{
		accountLoader: manager,
	}
	return manager
}
