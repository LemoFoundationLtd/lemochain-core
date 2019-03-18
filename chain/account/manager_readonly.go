package account

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
)

// ReadOnlyManager
func ReadOnlyManager(blockHash common.Hash, db protocol.ChainDB) *Manager {
	manager := &Manager{
		db:            db,
		baseBlockHash: blockHash,
	}
	manager.loadBaseBlock()
	manager.acctDb, _ = db.GetActDatabase(blockHash)
	manager.processor = &LogProcessor{
		accountLoader: manager,
	}
	return manager
}
