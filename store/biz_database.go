package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"strconv"
	"time"
)

type VTransaction struct {
	Tx *types.Transaction
	St int64
}

type BizDb interface {
	GetTx8Hash(hash common.Hash) (*VTransaction, error)

	GetTx8AddrPre(src common.Address, start int64, size int) ([]*VTransaction, int64, error)

	GetTx8AddrNext(src common.Address, start int64, size int) ([]*VTransaction, int64, error)
}

type BizDatabase struct {
	Reader   DatabaseReader
	Database *MySqlDB
}

func NewBizDatabase(reader DatabaseReader, database *MySqlDB) *BizDatabase {
	return &BizDatabase{
		Reader:   reader,
		Database: database,
	}
}

func (db *BizDatabase) GetTx8Hash(hash common.Hash) (*VTransaction, error) {
	val, st, err := db.Database.TxGet8Hash(hash.Hex())
	if err != nil {
		return nil, err
	}

	var tx types.Transaction
	err = rlp.DecodeBytes(val, &tx)
	if err != nil {
		return nil, err
	}

	return &VTransaction{
		Tx: &tx,
		St: st,
	}, nil
}

func (db *BizDatabase) GetTx8AddrPre(src common.Address, start int64, size int) ([]*VTransaction, int64, error) {
	vals, sts, ver, err := db.Database.TxGet8AddrPre(src.Hex(), start, size)
	if err != nil {
		return nil, -1, err
	}

	txs := make([]*VTransaction, len(vals))
	for index := 0; index < len(vals); index++ {
		var tx types.Transaction
		err = rlp.DecodeBytes(vals[index], &tx)
		if err != nil {
			return nil, -1, err
		}

		txs[index] = &VTransaction{
			Tx: &tx,
			St: sts[index],
		}
	}
	return txs, ver, nil
}

func (db *BizDatabase) GetTx8AddrNext(src common.Address, start int64, size int) ([]*VTransaction, int64, error) {
	vals, sts, ver, err := db.Database.TxGet8AddrNext(src.Hex(), start, size)
	if err != nil {
		return nil, -1, err
	}

	txs := make([]*VTransaction, len(vals))
	for index := 0; index < len(vals); index++ {
		var tx types.Transaction
		err = rlp.DecodeBytes(vals[index], &tx)
		if err != nil {
			return nil, -1, err
		}

		txs[index] = &VTransaction{
			Tx: &tx,
			St: sts[index],
		}
	}
	return txs, ver, nil
}

func (db *BizDatabase) AfterCommit(flag uint, key []byte, val []byte) error {
	if flag == CACHE_FLG_BLOCK {
		return db.afterBlock(key, val)
	} else if flag == CACHE_FLG_BLOCK_HEIGHT {
		return nil
	} else if flag == CACHE_FLG_TRIE {
		return nil
	} else if flag == CACHE_FLG_ACT {
		return nil
	} else if flag == CACHE_FLG_TX_INDEX {
		return nil
	} else if flag == CACHE_FLG_CODE {
		return nil
	} else if flag == CACHE_FLG_KV {
		return nil
	} else {
		panic("unknow flag.flag = " + strconv.Itoa(int(flag)))
	}
}

func (db *BizDatabase) afterBlock(key []byte, val []byte) error {
	var block types.Block
	err := rlp.DecodeBytes(val, &block)
	if err != nil {
		return err
	}

	txs := block.Txs
	if len(txs) <= 0 {
		return nil
	}

	ver := time.Now().UnixNano()
	for index := 0; index < len(txs); index++ {
		tx := txs[index]
		hash := tx.Hash()
		from, err := tx.From()
		if err != nil {
			return err
		}

		to := tx.To()

		hashStr := hash.Hex()
		fromStr := from.Hex()
		toStr := to.Hex()

		val, err := rlp.EncodeToBytes(tx)
		if err != nil {
			return err
		}

		err = db.Database.TxSet(hashStr, fromStr, toStr, val, ver+int64(index), int64(block.Header.Time))
		if err != nil {
			return err
		}
	}
	return nil
}
