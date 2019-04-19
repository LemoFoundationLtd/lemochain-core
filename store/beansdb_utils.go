package store

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
)

func UtilsGetBlockByHash(db *BeansDB, hash common.Hash) (*types.Block, error) {
	val, err := db.Get(leveldb.ItemFlagBlock, hash.Bytes())
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	var block types.Block
	err = rlp.DecodeBytes(val, &block)
	if err != nil {
		return nil, err
	} else {
		return &block, nil
	}
}

func UtilsGetBlockByHeight(db *BeansDB, height uint32) (*types.Block, error) {
	val, err := db.Get(leveldb.ItemFlagBlockHeight, leveldb.EncodeNumber(height))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	return UtilsGetBlockByHash(db, common.BytesToHash(val))
}

func UtilsHashBlock(db *BeansDB, hash common.Hash) (bool, error) {
	block, err := UtilsGetBlockByHash(db, hash)
	if err != nil {
		return false, err
	} else {
		return block != nil, nil
	}
}

func UtilsGetAssetCode(db *BeansDB, code common.Hash) (common.Address, error) {
	val, err := db.Get(leveldb.ItemFlagAssetCode, code.Bytes())
	if err != nil {
		return common.Address{}, err
	}

	if len(val) <= 0 {
		return common.Address{}, nil
	}

	return common.BytesToAddress(val), nil
}

func UtilsGetAssetId(db *BeansDB, id common.Hash) (common.Hash, error) {
	val, err := db.Get(leveldb.ItemFlagAssetId, id.Bytes())
	if err != nil {
		return common.Hash{}, err
	}

	if len(val) <= 0 {
		return common.Hash{}, nil
	}

	return common.BytesToHash(val), nil
}

func UtilsGetAccount(db *BeansDB, address common.Address) (*types.AccountData, error) {
	val, err := db.Get(leveldb.ItemFlagAct, address.Bytes())
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	var account types.AccountData
	err = rlp.DecodeBytes(val, &account)
	if err != nil {
		return nil, err
	} else {
		return &account, nil
	}
}

func UtilsSetAssetCode(db *BeansDB, code common.Hash, address common.Address) error {
	return db.Put(leveldb.ItemFlagAssetCode, code.Bytes(), address.Bytes())
}

func UtilsSetAssetId(db *BeansDB, id common.Hash, code common.Hash) error {
	return db.Put(leveldb.ItemFlagAssetId, id.Bytes(), code.Bytes())
}
