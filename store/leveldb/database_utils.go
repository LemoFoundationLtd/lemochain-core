package leveldb

import (
	"encoding/binary"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
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
	hashPrefix = []byte("B")
	hashSuffix = []byte("b")

	heightPrefix = []byte("H")
	heightSuffix = []byte("h") // // headerPrefix + height (uint64 big endian) + heightSuffix -> hash

	accountPrefix = []byte("A")
	accountSuffix = []byte("a")

	assetCodePrefix = []byte("C")
	assetCodeSuffix = []byte("c")

	assetIdPrefix = []byte("I")
	assetIdSuffix = []byte("i")
)

type Position struct {
	Flag   uint32
	Route  []byte
	Offset uint32
}

func encodeBlockNumber(height uint32) []byte {
	enc := make([]byte, 4)
	binary.BigEndian.PutUint32(enc, height)
	return enc
}

func GetCanonicalKey(height uint32) []byte {
	return append(append(heightPrefix, encodeBlockNumber(height)...), heightSuffix...)
}

func GetCanonicalHash(db DatabaseReader, height uint32) (common.Hash, error) {
	data, err := db.Get(GetCanonicalKey(height))
	if err != nil {
		return common.Hash{}, err
	}

	if len(data) <= 0 {
		return common.Hash{}, nil
	} else {
		return common.BytesToHash(data), nil
	}
}

func SetCanonicalHash(db DatabasePutter, height uint32, hash common.Hash) error {
	return db.Put(GetCanonicalKey(height), hash.Bytes())
}

func GetBlockHashKey(hash common.Hash) []byte {
	return append(append(hashPrefix, hash.Bytes()...), hashSuffix...)
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

func GetBlockHash(db DatabaseReader, hash common.Hash) (*Position, error) {
	val, err := db.Get(hash[:])
	if err != nil {
		return nil, err
	} else {
		return toPosition(val)
	}
}

func SetBlockHash(db DatabasePutter, hash common.Hash, position *Position) error {
	val, err := rlp.EncodeToBytes(position)
	if err != nil {
		return err
	} else {
		return db.Put(hash[:], val)
	}
}

func GetAddressKey(addr common.Address) []byte {
	return append(append(accountPrefix, addr[:]...), accountSuffix...)
}

func GetAddress(db DatabaseReader, addr common.Address) (*Position, error) {
	val, err := db.Get(GetAddressKey(addr))
	if err != nil {
		return nil, err
	} else {
		return toPosition(val)
	}
}

func SetAddress(db DatabasePutter, addr common.Address, position *Position) error {
	val, err := rlp.EncodeToBytes(position)
	if err != nil {
		return err
	} else {
		return db.Put(GetAddressKey(addr), val)
	}
}

func GetAssetIdKey(id common.Hash) []byte {
	return append(append(assetIdPrefix, id[:]...), assetIdSuffix...)
}

func GetAssetID(db DatabaseReader, id common.Hash) (common.Address, error) {
	val, err := db.Get(GetAssetIdKey(id))
	if err != nil {
		return common.Address{}, err
	}

	if len(val) <= 0 {
		return common.Address{}, nil
	}

	return common.BytesToAddress(val), nil
}

func SetAssetID(db DatabasePutter, id common.Hash, addr common.Address) error {
	return db.Put(GetAssetIdKey(id), addr.Bytes())
}

func GetAssetCodeKey(code common.Hash) []byte {
	return append(append(assetCodePrefix, code.Bytes()...), assetCodeSuffix...)
}

func GetAssetCode(db DatabaseReader, code common.Hash) (common.Address, error) {
	val, err := db.Get(GetAssetCodeKey(code))
	if err != nil {
		return common.Address{}, err
	}

	if len(val) <= 0 {
		return common.Address{}, nil
	}

	return common.BytesToAddress(val), nil
}

func SetAssetCode(db DatabasePutter, code common.Hash, addr common.Address) error {
	return db.Put(GetAssetCodeKey(code), addr.Bytes())
}

func Set(db DatabasePutter, key []byte, val []byte) error {
	return db.Put(key, val)
}

func Get(db DatabaseReader, key []byte) ([]byte, error) {
	return db.Get(key)
}

func SetPos(db DatabasePutter, key []byte, position *Position) error {
	val, err := rlp.EncodeToBytes(position)
	if err != nil {
		return err
	} else {
		return db.Put(key, val)
	}
}

func GetPos(db DatabaseReader, key []byte) (*Position, error) {
	val, err := db.Get(key)
	if err != nil {
		return nil, err
	} else {
		return toPosition(val)
	}
}
