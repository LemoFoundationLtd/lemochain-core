package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RHead struct {
	Flg       uint32
	Len       uint32
	TimeStamp uint64
	Crc       uint16
}

var RHEAD_LENGTH = binary.Size(RHead{})

type RBody struct {
	Route []byte
	Key   []byte
	Val   []byte
}

type RIndex struct {
	flg uint
	pos uint32
	len int
}

type AfterScan func(flag uint, route []byte, key []byte, val []byte, offset uint32) error

type IndexReader interface {
	Get(key []byte)
}

type BitCask struct {
	rw sync.RWMutex

	HomePath string

	CurIndex  int
	CurOffset int64

	Cache map[string]RIndex

	After   AfterScan
	IndexDB DB
}

func (bitcask *BitCask) path(index int) string {
	dataPath := filepath.Join(bitcask.HomePath, "%03d.data")
	return fmt.Sprintf(dataPath, index)
}

func (bitcask *BitCask) isExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (bitcask *BitCask) createFile(index int) error {
	dataPath := bitcask.path(index)
	f, err := os.Create(dataPath)
	defer f.Close()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func NewBitCask(homePath string, lastIndex int, lastOffset uint32, after AfterScan, indexDB DB) (*BitCask, error) {
	db := &BitCask{HomePath: homePath, After: after, IndexDB: indexDB}
	isExist, err := db.isExist(homePath)
	if err != nil {
		return nil, err
	}

	db.Cache = make(map[string]RIndex)

	if !isExist {
		err = os.MkdirAll(homePath, os.ModePerm)
		if err != nil {
			return nil, err
		}

		db.CurIndex = 0
		db.CurOffset = 0
		err = db.createFile(db.CurIndex)
		if err != nil {
			return nil, err
		} else {
			return db, nil
		}
	} else {
		err = db.scan(lastIndex, lastOffset)
		if err != nil {
			return nil, err
		}

		isExist, err := IsExist(db.path(db.CurIndex))
		if err != nil {
			return nil, err
		}

		if !isExist {
			err = db.createFile(db.CurIndex)
			if err != nil {
				return nil, err
			} else {
				return db, nil
			}
		} else {
			return db, nil
		}
	}
}

func (bitcask *BitCask) scan(lastIndex int, lastOffset uint32) error {

	nextIndex := lastIndex
	nextOffset := lastOffset

	for {
		nextPath := bitcask.path(nextIndex)
		isExist, err := bitcask.isExist(nextPath)
		if err != nil {
			return err
		}

		if !isExist {
			break
		}

		file, err := os.OpenFile(nextPath, os.O_RDONLY, os.ModePerm)
		defer file.Close()
		if err != nil {
			return err
		}

		lastIndex = nextIndex
		for {
			head, body, err := bitcask.read(file, int64(nextOffset))
			if err == io.EOF {
				break
			}

			if err != nil {
				return err
			}

			err = bitcask.After(uint(head.Flg), body.Route, body.Key, body.Val, lastOffset)
			if err != nil {
				return err
			} else {
				delete(bitcask.Cache, string(body.Key))
			}

			nextOffset = nextOffset + uint32(RHEAD_LENGTH) + uint32(head.Len)
			lastOffset = nextOffset
		}

		nextIndex = nextIndex + 1
		nextOffset = 0
	}

	bitcask.CurIndex = lastIndex
	bitcask.CurOffset = int64(lastOffset)

	return nil
}

func (bitcask *BitCask) align(tLen uint32) uint32 {
	if tLen%256 != 0 {
		tLen += 256 - tLen%256
	}

	return tLen
}

func (bitcask *BitCask) encode(hBuf []byte, bBuf []byte) []byte {
	tLen := bitcask.align(uint32(len(hBuf)) + uint32(len(bBuf)))

	tBuf := make([]byte, tLen)
	copy(tBuf[:], hBuf[:])
	copy(tBuf[len(hBuf):], bBuf[:])

	return tBuf
}

func (bitcask *BitCask) Delete(flag int, route []byte, key []byte) error {
	return nil
}

func (bitcask *BitCask) encodeHead(flag uint, data []byte) ([]byte, error) {
	head := RHead{
		Flg:       uint32(flag),
		Len:       uint32(len(data)),
		TimeStamp: uint64(time.Now().Unix()),
		Crc:       CheckSum(data),
	}

	buf := make([]byte, RHEAD_LENGTH)
	err := binary.Write(NewLmBuffer(buf[:]), binary.LittleEndian, &head)
	if err != nil {
		return nil, err
	} else {
		return buf, nil
	}
}

func (bitcask *BitCask) encodeBody(route []byte, key []byte, val []byte) ([]byte, error) {
	body := &RBody{
		Route: route,
		Key:   key,
		Val:   val,
	}

	return rlp.EncodeToBytes(body)
}

func (bitcask *BitCask) checkSize(size int64) error {
	if (bitcask.CurOffset + size) > int64(maxFileSize) {
		bitcask.CurIndex = bitcask.CurIndex + 1
		err := bitcask.createFile(bitcask.CurIndex)
		if err != nil {
			return err
		} else {
			bitcask.CurOffset = 0
			return nil
		}
	} else {
		return nil
	}
}

func (bitcask *BitCask) flush(data []byte) (int64, error) {
	file, err := os.OpenFile(bitcask.path(bitcask.CurIndex), os.O_APPEND|os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return -1, err
	}

	_, err = file.Seek(int64(bitcask.CurOffset), 0)
	if err != nil {
		return -1, err
	}

	n, err := file.Write(data)
	if err != nil {
		return -1, err
	}

	if n != len(data) {
		panic("n != len(data)")
	}

	err = file.Sync()
	if err != nil {
		return -1, err
	}

	tmp := bitcask.CurOffset
	bitcask.CurOffset += int64(n)
	return tmp, nil
}

func (bitcask *BitCask) Put(flag uint, route []byte, key []byte, val []byte) error {
	bitcask.rw.Lock()
	defer bitcask.rw.Unlock()

	body, err := bitcask.encodeBody(route, key, val)
	if err != nil {
		return err
	}

	head, err := bitcask.encodeHead(flag, body)
	if err != nil {
		return err
	}

	data := bitcask.encode(head, body)
	err = bitcask.checkSize(int64(len(data)))
	if err != nil {
		return err
	}

	offset, err := bitcask.flush(data)
	if err != nil {
		return err
	}

	bitcask.Cache[string(key[:])] = RIndex{
		flg: flag,
		pos: uint32(int(offset) | bitcask.CurIndex),
		len: len(data),
	}
	return nil
}

func (bitcask *BitCask) read(file *os.File, offset int64) (*RHead, *RBody, error) {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return nil, nil, err
	}

	heaBuf := make([]byte, RHEAD_LENGTH)
	_, err = file.Read(heaBuf)
	if err != nil {
		return nil, nil, err
	}

	var head RHead
	err = binary.Read(bytes.NewBuffer(heaBuf), binary.LittleEndian, &head)
	if err == io.EOF {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	bodyBuf := make([]byte, head.Len)
	_, err = file.Read(bodyBuf)
	if err != nil {
		return nil, nil, err
	}

	var body RBody
	err = rlp.DecodeBytes(bodyBuf, &body)
	if err != nil {
		return nil, nil, err
	} else {
		return &head, &body, nil
	}
}

func (bitcask *BitCask) get(flag uint, route []byte, key []byte, offset int64) ([]byte, error) {
	pos := uint32(offset) & 0xffffff00
	bucketIndex := uint32(offset) & 0xff

	dataPath := bitcask.path(int(bucketIndex))
	file, err := os.OpenFile(dataPath, os.O_RDONLY, 0666)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	head, body, err := bitcask.read(file, int64(pos))
	if err != nil {
		return nil, err
	} else {
		if head.Flg != uint32(flag) {
			return nil, nil
		}

		// if (bytes.Compare(body.Route, route) != 0) || (bytes.Compare(body.Key, key) != 0) {
		// 	return nil, nil
		// } else {
		// 	return body.Val, nil
		// }
		str := common.BytesToHash(body.Key).Hex()
		log.Errorf("str:" + str)
		return body.Val, nil
	}
}

func (bitcask *BitCask) Get(flag uint, route []byte, key []byte, offset int64) ([]byte, error) {
	bitcask.rw.RLock()
	defer bitcask.rw.RUnlock()
	return bitcask.get(flag, route, key, offset)
}

func (bitcask *BitCask) Get4Cache(route []byte, key []byte) ([]byte, error) {
	bitcask.rw.RLock()
	defer bitcask.rw.RUnlock()

	index, ok := bitcask.Cache[string(key)]
	if !ok {
		flg, route, offset, err := bitcask.IndexDB.GetIndex(key)
		if err != nil {
			return nil, err
		} else {
			return bitcask.get(uint(flg), route, key, offset)
		}
	} else {
		return bitcask.get(index.flg, route, key, int64(index.pos))
	}
}

func (bitcask *BitCask) Close() error {
	return nil
}

func (bitcask *BitCask) NewBatch(route []byte) Batch {
	batch := &LmDBBatch{
		db:    bitcask,
		items: make([]*BatchItem, 0),
		size:  0,
	}

	batch.route = make([]byte, len(route))
	copy(batch.route, route)
	return batch
}

func (bitcask *BitCask) Commit(batch Batch) error {
	bitcask.rw.Lock()
	defer bitcask.rw.Unlock()

	route := batch.Route()
	items := batch.Items()

	if len(items) <= 0 {
		return nil
	}

	tmpPos := make([]int64, len(items))
	tmpBuf := make([][]byte, len(items))
	totalSize := 0
	for index := 0; index < len(items); index++ {
		item := items[index]

		body, err := bitcask.encodeBody(route, item.Key, item.Val)
		if err != nil {
			return err
		}

		head, err := bitcask.encodeHead(item.Flg, body)
		if err != nil {
			return err
		}

		tmpBuf[index] = bitcask.encode(head, body)
		totalSize = totalSize + len(tmpBuf[index])
		if index == 0 {
			tmpPos[index] = 0
		} else {
			tmpPos[index] = tmpPos[index-1] + int64(len(tmpBuf[index-1]))
		}
	}

	buf := make([]byte, totalSize)
	for index := 0; index < len(items); index++ {
		copy(buf[tmpPos[index]:], tmpBuf[index])
	}

	err := bitcask.checkSize(int64(len(buf)))
	if err != nil {
		return err
	}

	pos, err := bitcask.flush(buf)
	if err != nil {
		return err
	} else {
		for index := 0; index < len(items); index++ {
			pos := uint32(int(pos+tmpPos[index]) | bitcask.CurIndex)
			bitcask.Cache[string(items[index].Key)] = RIndex{
				flg: items[index].Flg,
				pos: pos,
				len: len(tmpBuf[index]),
			}

			bitcask.IndexDB.SetIndex(int(items[index].Flg), batch.Route(), items[index].Key, int64(pos))
		}
		return nil
	}
}
