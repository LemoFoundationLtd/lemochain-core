package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"strconv"
	"time"
)

//go:generate gencodec -type VTransaction -out gen_vTransaction_info_json.go
//go:generate gencodec -type VTransactionDetail -out gen_vTransactionDetail_info_json.go

type VTransaction struct {
	Tx *types.Transaction `json:"tx"  		gencodec:"required"`
	St int64              `json:"time"  		gencodec:"required"`
}

type VTransactionDetail struct {
	BlockHash common.Hash        `json:"blockHash"  	gencodec:"required"`
	Height    uint32             `json:"height"  	gencodec:"required"`
	Tx        *types.Transaction `json:"tx"  		gencodec:"required"`
	St        int64              `json:"time"  		gencodec:"required"`
}

type BizDb interface {
	GetTxByHash(hash common.Hash) (*VTransactionDetail, error)

	GetTxByAddr(src common.Address, start int64, size int) ([]*VTransaction, int64, error)
}

type Reader interface {
	GetBlockByHash(hash common.Hash) (*types.Block, error)
}

type BizDatabase struct {
	Reader   Reader
	Database *MySqlDB
}

func NewBizDatabase(reader Reader, database *MySqlDB) *BizDatabase {
	return &BizDatabase{
		Reader:   reader,
		Database: database,
	}
}

func (db *BizDatabase) GetTxByHash(hash common.Hash) (*VTransactionDetail, error) {
	blockHash, val, st, err := db.Database.TxGetByHash(hash.Hex())
	if err != nil {
		return nil, err
	}

	block, err := db.Reader.GetBlockByHash(common.HexToHash(blockHash))
	if err == ErrNotExist {
		return nil, ErrNotExist
	}

	if err != nil {
		return nil, err
	}

	var tx types.Transaction
	err = rlp.DecodeBytes(val, &tx)
	if err != nil {
		return nil, err
	}

	return &VTransactionDetail{
		BlockHash: block.Hash(),
		Height:    block.Height(),
		Tx:        &tx,
		St:        st,
	}, nil
}

func (db *BizDatabase) GetTxByAddr(src common.Address, start int64, size int) ([]*VTransaction, int64, error) {
	_, vals, sts, ver, err := db.Database.TxGetByAddr(src.Hex(), start, size)
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
		panic("unknown flag.flag = " + strconv.Itoa(int(flag)))
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
		if to == nil {
			to = &params.FeeReceiveAddress
		}

		hashStr := hash.Hex()
		fromStr := from.Hex()
		toStr := to.Hex()

		val, err := rlp.EncodeToBytes(tx)
		if err != nil {
			return err
		}

		err = db.Database.TxSet(hashStr, common.ToHex(key), fromStr, toStr, val, ver+int64(index), int64(block.Header.Time))
		if err != nil {
			return err
		}
	}
	return nil
}
