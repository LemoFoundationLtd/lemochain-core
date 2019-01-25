package protocol

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
)

type ChainDB interface {
	SizeOfValue(hash common.Hash) (int, error)

	SetBlock(hash common.Hash, block *types.Block) error
	GetBlock(hash common.Hash, height uint32) (*types.Block, error)
	GetBlockByHeight(height uint32) (*types.Block, error)
	GetBlockByHash(hash common.Hash) (*types.Block, error)
	IsExistByHash(hash common.Hash) (bool, error)

	GetConfirms(hash common.Hash) ([]types.SignData, error)
	SetConfirm(hash common.Hash, signData types.SignData) error
	SetConfirms(hash common.Hash, pack []types.SignData) error

	LoadLatestBlock() (*types.Block, error)
	SetStableBlock(hash common.Hash) error

	GetAccount(addr common.Address) (*types.AccountData, error)

	GetTrieDatabase() *store.TrieDatabase
	GetActDatabase(hash common.Hash) *store.PatriciaTrie
	GetBizDatabase() store.BizDb

	GetContractCode(hash common.Hash) (types.Code, error)
	SetContractCode(hash common.Hash, code types.Code) error

	CandidatesRanking(hash common.Hash)
	GetCandidatesTop(hash common.Hash) []*store.Candidate

	Close() error
}
