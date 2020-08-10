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
		return nil, ErrBlockNotExist
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
		return nil, ErrBlockNotExist
	}

	return UtilsGetBlockByHash(db, common.BytesToHash(val))
}

func UtilsHasBlock(db *BeansDB, hash common.Hash) (bool, error) {
	_, err := UtilsGetBlockByHash(db, hash)

	if err == ErrBlockNotExist {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func UtilsGetIssurer(db *BeansDB, assetCode common.Hash) (common.Address, error) {
	val, err := db.Get(leveldb.ItemFlagAssetCode, assetCode.Bytes())
	if err != nil {
		return common.Address{}, err
	}

	if len(val) <= 0 {
		return common.Address{}, types.ErrAssetNotExist
	}

	return common.BytesToAddress(val), nil
}

func UtilsGetAccount(db *BeansDB, address common.Address) (*types.AccountData, error) {
	val, err := db.Get(leveldb.ItemFlagAct, address.Bytes())
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, ErrAccountNotExist
	}

	var account types.AccountData
	err = rlp.DecodeBytes(val, &account)
	if err != nil {
		return nil, err
	} else {
		return &account, nil
	}
}

func UtilsBindIssurerAndAssetCode(db *BeansDB, code common.Hash, address common.Address) error {
	return db.Put(leveldb.ItemFlagAssetCode, code.Bytes(), address.Bytes())
}
