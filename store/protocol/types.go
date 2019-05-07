package protocol

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
)

type ChainDB interface {
	SizeOfValue(hash common.Hash) (int, error)

	SetBlock(hash common.Hash, block *types.Block) error
	GetBlockByHeight(height uint32) (*types.Block, error)
	GetBlockByHash(hash common.Hash) (*types.Block, error)
	IsExistByHash(hash common.Hash) (bool, error)
	GetUnConfirmByHeight(height uint32, leafBlockHash common.Hash) (*types.Block, error)
	IterateUnConfirms(fn func(*types.Block))

	GetConfirms(hash common.Hash) ([]types.SignData, error)
	SetConfirm(hash common.Hash, signData types.SignData) error
	SetConfirms(hash common.Hash, pack []types.SignData) error

	LoadLatestBlock() (*types.Block, error)
	SetStableBlock(hash common.Hash) ([]*types.Block, error)

	GetAccount(addr common.Address) (*types.AccountData, error)

	GetTrieDatabase() *store.TrieDatabase
	GetActDatabase(hash common.Hash) (*store.AccountTrieDB, error)

	GetContractCode(hash common.Hash) (types.Code, error)
	SetContractCode(hash common.Hash, code types.Code) error

	CandidatesRanking(hash common.Hash)
	GetCandidatesTop(hash common.Hash) []*store.Candidate

	GetAssetID(id common.Hash) (common.Address, error)
	GetAssetCode(code common.Hash) (common.Address, error)

	SerializeForks(currentHash common.Hash) string

	Close() error
}
