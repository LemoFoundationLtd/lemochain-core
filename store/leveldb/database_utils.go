package leveldb

import (
	"bytes"
	"encoding/binary"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"strconv"
)

type DatabasePutter interface {
	Put(key []byte, value []byte) error
}

type DatabaseReader interface {
	Get(key []byte) (value []byte, err error)
}

type DatabaseDeleter interface {
	Delete(key []byte) error
}

var (
	ItemFlagStart       = uint32(0)
	ItemFlagBlock       = uint32(1)
	ItemFlagBlockHeight = uint32(2)
	ItemFlagTrie        = uint32(3)
	ItemFlagAct         = uint32(4)
	ItemFlagTxIndex     = uint32(5)
	ItemFlagCode        = uint32(6)
	ItemFlagKV          = uint32(7)
	ItemFlagAssetCode   = uint32(8)
	ItemFlagAssetId     = uint32(9)
	ItemFlagStop        = uint32(10)
)

var (
	BlockPrefix = []byte("B")
	BlockSuffix = []byte("b")

	BlockHeightPrefix = []byte("BH")
	BlockHeightSuffix = []byte("bh") // // headerPrefix + height (uint64 big endian) + heightSuffix -> hash

	AccountPrefix = []byte("A")
	AccountSuffix = []byte("a")

	TxPrefix = []byte("TX")
	TxSuffix = []byte("tx")

	AssetCodePrefix = []byte("AC")
	AssetCodeSuffix = []byte("ac")

	AssetIdPrefix = []byte("AI")
	AssetIdSuffix = []byte("ai")

	TrieNodePrefix = []byte("TN")
	TrieNodeSuffix = []byte("tn")

	CodePrefix = []byte("CC")
	CodeSuffix = []byte("cc")

	KVPrefix = []byte("KV")
	KVSuffix = []byte("kv")

	BitCaskCurrentOffsetPrefix = []byte("OFFSET")
	BitCaskCurrentOffsetSuffix = []byte("offset")

	StableBlockKey = []byte("LEMO-CURRENT-BLOCK")
)

func CheckItemFlag(flg uint32) bool {
	if (flg <= ItemFlagStart) || (flg >= ItemFlagStop) {
		return false
	} else {
		return true
	}
}

type Position struct {
	Flag   uint32
	Offset uint32
}

func Key(flag uint32, key []byte) []byte {
	if len(key) <= 0 {
		return nil
	}

	switch flag {
	case ItemFlagBlock:
		return append(append(BlockPrefix, key...), BlockSuffix...)
	case ItemFlagBlockHeight:
		return append(append(BlockHeightPrefix, key...), BlockHeightSuffix...)
	case ItemFlagTrie:
		return append(append(TrieNodePrefix, key...), TrieNodeSuffix...)
	case ItemFlagAct:
		return append(append(AccountPrefix, key...), AccountSuffix...)
	case ItemFlagTxIndex:
		return append(append(TxPrefix, key...), TxSuffix...)
	case ItemFlagCode:
		return append(append(CodePrefix, key...), CodeSuffix...)
	case ItemFlagKV:
		return append(append(KVPrefix, key...), KVSuffix...)
	case ItemFlagAssetCode:
		return append(append(AssetCodePrefix, key...), AssetCodeSuffix...)
	case ItemFlagAssetId:
		return append(append(AssetIdPrefix, key...), AssetIdSuffix...)
	default:
		return key
	}
}

func toPosition(val []byte) (*Position, error) {
	if len(val) <= 0 {
		return nil, nil
	} else {
		var position Position
		err := rlp.DecodeBytes(val, &position)
		if err != nil {
			return nil, err
		} else {
			return &position, nil
		}
	}
}

func SetPos(db DatabasePutter, flg uint32, key []byte, position *Position) error {
	val, err := rlp.EncodeToBytes(position)
	if err != nil {
		return err
	} else {
		tmp := Key(flg, key)
		return db.Put(tmp, val)
	}
}

func GetPos(db DatabaseReader, flg uint32, key []byte) (*Position, error) {
	tmp := Key(flg, key)
	val, err := db.Get(tmp)
	if err != nil {
		return nil, err
	} else {
		return toPosition(val)
	}
}

func DelPos(db DatabaseDeleter, flg uint32, key []byte) error {
	tmp := Key(flg, key)
	return db.Delete(tmp)
}

func EncodeNumber(height uint32) []byte {
	enc := make([]byte, 4)
	binary.BigEndian.PutUint32(enc, height)
	return enc
}

func GetCurrentPos(db DatabaseReader, index int) (uint32, error) {
	key := append(append(BitCaskCurrentOffsetPrefix, []byte(strconv.Itoa(index))...), BitCaskCurrentOffsetSuffix...)
	data, err := db.Get(key)
	if err != nil {
		return 0, err
	}

	if len(data) <= 0 {
		return 0, nil
	}

	var pos uint32
	err = binary.Read(bytes.NewBuffer(data), binary.BigEndian, &pos)
	if err != nil {
		return 0, nil
	}

	return pos, nil
}

func SetCurrentPos(db DatabasePutter, index int, pos uint32) error {
	key := append(append(BitCaskCurrentOffsetPrefix, []byte(strconv.Itoa(index))...), BitCaskCurrentOffsetSuffix...)
	return db.Put(key, EncodeNumber(pos))
}

func GetCurrentBlock(db DatabaseReader) (common.Hash, error) {
	val, err := db.Get(StableBlockKey)
	if err != nil {
		return common.Hash{}, err
	}

	if len(val) <= 0 {
		return common.Hash{}, nil
	}

	return common.BytesToHash(val), nil
}

func SetCurrentBlock(db DatabasePutter, hash common.Hash) error {
	return db.Put(StableBlockKey, hash.Bytes())
}

func Set(db DatabasePutter, key []byte, val []byte) error {
	return db.Put(key, val)
}

func Get(db DatabaseReader, key []byte) ([]byte, error) {
	return db.Get(key)
}
