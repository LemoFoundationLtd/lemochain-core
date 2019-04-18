package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"os"
	"path/filepath"
	"sync"
)

type RecordHead struct {
	Flg       uint32
	Len       uint32
	TimeStamp uint64
	Crc       uint16
}

type RecordBody struct {
	Key []byte
	Val []byte
}

type Inject struct {
	Flg uint32
	Key []byte
	Val []byte
}

var RecordHeadLength = binary.Size(RecordHead{})

type BitCask struct {
	RW   sync.RWMutex
	Home string

	BitCaskIndex int
	CurIndex     int
	CurOffset    int64
	LevelDB      *leveldb.LevelDBDatabase
}

func (bitcask *BitCask) path(index int) string {
	dataPath := filepath.Join(bitcask.Home, "%03d.data")
	return fmt.Sprintf(dataPath, index)
}

func (bitcask *BitCask) homeIsNotExist(home string) error {
	err := os.MkdirAll(home, os.ModePerm)
	if err != nil {
		return err
	}

	bitcask.CurIndex = 0
	bitcask.CurOffset = 0

	path := bitcask.path(bitcask.CurIndex)
	return FileUtilsCreateFile(path)
}

func (bitcask *BitCask) homeIsExist(home string) error {
	pos, err := leveldb.GetCurrentPos(bitcask.LevelDB, bitcask.BitCaskIndex)
	if err != nil {
		return err
	}

	bitcask.CurIndex = int(pos & 0xFF)
	bitcask.CurOffset = int64(pos & 0xFFFFFF00)
	path := bitcask.path(bitcask.CurIndex)
	isExist, err := FileUtilsIsExist(path)
	if err != nil {
		return err
	}

	if isExist {
		return nil
	} else {
		return FileUtilsCreateFile(path)
	}
}

func NewBitCask(home string, index int, levelDB *leveldb.LevelDBDatabase) (*BitCask, error) {
	db := &BitCask{
		Home:         home,
		LevelDB:      levelDB,
		BitCaskIndex: index,
	}

	isExist, err := FileUtilsIsExist(home)
	if err != nil {
		return nil, err
	}

	if !isExist {
		err = db.homeIsNotExist(home)
		if err != nil {
			return nil, err
		} else {
			return db, nil
		}
	} else {
		err = db.homeIsExist(home)
		if err != nil {
			return nil, err
		} else {
			return db, nil
		}
	}
}

func (bitcask *BitCask) Put(flag uint32, key []byte, val []byte) error {
	bitcask.RW.Lock()
	defer bitcask.RW.Unlock()

	data, err := FileUtilsEncode(flag, key, val)
	length, err := bitcask.checkAndFlush(data)
	if err != nil {
		return err
	}

	offset := uint32(int(bitcask.CurOffset) | bitcask.CurIndex)
	err = leveldb.SetPos(bitcask.LevelDB, flag, key, &leveldb.Position{
		Flag:   flag,
		Offset: offset,
	})

	if err != nil {
		return err
	} else {
		bitcask.CurOffset += length
		offset = uint32(int(bitcask.CurOffset) | bitcask.CurIndex)
		return leveldb.SetCurrentPos(bitcask.LevelDB, bitcask.BitCaskIndex, offset)
	}
}

func (bitcask *BitCask) Get(flag uint32, key []byte) ([]byte, error) {
	bitcask.RW.RLock()
	defer bitcask.RW.RUnlock()

	pos, err := leveldb.GetPos(bitcask.LevelDB, flag, key)
	if err != nil {
		return nil, err
	}

	if pos == nil {
		return nil, nil
	}

	if pos.Flag != flag {
		log.Errorf("get db record pos.Flag(%d) != flag(%d)", pos.Flag, flag)
		return nil, nil
	}

	return bitcask.get(flag, key, pos.Offset)
}

func (bitcask *BitCask) Delete(flag uint32, key []byte) error {
	return leveldb.DelPos(bitcask.LevelDB, flag, key)
}

func (bitcask *BitCask) checkSize(size int64) error {
	if (bitcask.CurOffset + size) <= int64(maxFileSize) {
		return nil
	}

	tmpCurIndex := bitcask.CurIndex + 1
	path := bitcask.path(tmpCurIndex)
	err := FileUtilsCreateFile(path)
	if err != nil {
		return err
	}

	bitcask.CurIndex = tmpCurIndex
	bitcask.CurOffset = 0
	return nil
}

func (bitcask *BitCask) checkAndFlush(data []byte) (int64, error) {
	err := bitcask.checkSize(int64(len(data)))
	if err != nil {
		return -1, err
	}

	path := bitcask.path(bitcask.CurIndex)
	length, err := FileUtilsFlush(path, bitcask.CurOffset, data)
	if err != nil {
		return -1, err
	} else {
		return length, nil
	}
}

func (bitcask *BitCask) get(flag uint32, key []byte, offset uint32) ([]byte, error) {
	pos := offset & 0xffffff00
	bucketIndex := offset & 0xff
	dataPath := bitcask.path(int(bucketIndex))

	file, err := os.OpenFile(dataPath, os.O_RDONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	head, body, err := FileUtilsRead(file, int64(pos))
	if err != nil {
		return nil, err
	}

	if head.Flg != uint32(flag) {
		log.Errorf("get file record pos.Flag(%d) != flag(%d)", head, flag)
		return nil, nil
	}

	if bytes.Compare(body.Key, key) != 0 {
		log.Errorf("get file record body.key(%s) != key(%s)", common.ToHex(body.Key), common.ToHex(key))
		return nil, nil
	}

	return body.Val, nil
}
