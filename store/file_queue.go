package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WriteExtend interface {
	After(flg uint32, key []byte, val []byte) error
}

type item struct {
	key    []byte
	val    []byte
	offset int64
	refCnt int
}

type FileQueue struct {
	Home string
	RW   sync.RWMutex

	Index  map[string]*item
	Offset int64

	Height   uint
	BitCasks []*BitCask

	LevelDB *leveldb.LevelDBDatabase
	Extend  WriteExtend
	Quit    chan struct{}
}

func (queue *FileQueue) path() string {
	return filepath.Join(queue.Home, "%tmp.data")
}

func NewFileQueue(home string, levelDB *leveldb.LevelDBDatabase, extend WriteExtend) *FileQueue {
	return &FileQueue{
		Home:    home,
		Index:   make(map[string]*item),
		Offset:  0,
		Height:  2,
		LevelDB: levelDB,
		Extend:  extend,
		Quit:    make(chan struct{}),
	}
}

func (queue *FileQueue) Start() {
	count := 1 << (uint(queue.Height) * 4)
	queue.BitCasks = make([]*BitCask, count)
	for index := 0; index < count; index++ {
		dataPathModule := filepath.Join(queue.Home, "/%02d/%02d/")
		dataPath := fmt.Sprintf(dataPathModule, index>>4, index&0xf)

		bitCask, err := NewBitCask(dataPath, index, queue.LevelDB, queue.Quit)
		if err != nil {
			panic("create bit cask err: " + err.Error())
		} else {
			queue.BitCasks[index] = bitCask
		}
		go queue.StartBitCask(queue.BitCasks[index])
	}

	err := queue.checkFile()
	if err != nil {
		panic("start queue.check tmp file err: " + err.Error())
	}
}

func (queue *FileQueue) Close() {

}

func (queue *FileQueue) checkFile() error {
	filePath := queue.path()
	isExist, err := queue.isExist(filePath)
	if err != nil {
		return err
	}

	if !isExist {
		err = queue.createFile(filePath)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else {
		offset, err := queue.scanFile(queue.path(), 0)
		if err == nil || err == ErrEOF {
			queue.Offset = offset
			return nil
		} else {
			return err
		}
	}
}

func (queue *FileQueue) StartBitCask(bitCask *BitCask) {
	doneChan := make(chan *Inject)
	errChan := make(chan *Inject)

	go bitCask.Start(doneChan, errChan)

	for {
		select {
		case done := <-doneChan:
			index := 0
			for ; index < 1024; index++ {
				err := queue.Extend.After(done.Flg, done.Key, done.Val)
				if err != nil {
					log.Errorf("write extend data err: " + err.Error())
					continue
				} else {
					val, ok := queue.Index[common.ToHex(done.Key)]
					if !ok {
						log.Errorf("done is not exist.flag: %d", done.Flg)
					} else {
						if val.refCnt <= 1 {
							delete(queue.Index, common.ToHex(done.Key))
						} else {
							val.refCnt = val.refCnt - 1
							queue.Index[common.ToHex(done.Key)] = val
						}
						break
					}

					break
				}
			}

			if index == 1024 {
				panic("write extend data err !!!")
			}
		case <-errChan:
			continue
		}
	}
}

func (queue *FileQueue) isExist(path string) (bool, error) {
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

func (queue *FileQueue) createFile(path string) error {
	f, err := os.Create(path)
	defer f.Close()
	return err
}

func (queue *FileQueue) emptyFile(path string) error {
	return nil
}

func (queue *FileQueue) read(file *os.File, offset int64) (*RecordHead, *RecordBody, error) {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return nil, nil, err
	}

	heaBuf := make([]byte, RecordHeadLength)
	_, err = file.Read(heaBuf)
	if err != nil {
		return nil, nil, err
	}

	var head RecordHead
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

	var body RecordBody
	err = rlp.DecodeBytes(bodyBuf, &body)
	if err != nil {
		return nil, nil, err
	} else {
		return &head, &body, nil
	}
}

func (queue *FileQueue) scanFile(filePath string, offset int64) (int64, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	defer file.Close()

	if err != nil {
		return -1, err
	}

	next := offset
	for {
		head, body, err := queue.read(file, next)
		if err == io.EOF {
			return next, ErrEOF
		}

		if err != nil {
			return -1, err
		}

		tmp, ok := queue.Index[common.ToHex(body.Key)]
		refCnt := 0
		if !ok {
			refCnt = 1
		} else {
			refCnt = tmp.refCnt + 1
		}

		length := queue.align(uint32(RecordHeadLength) + uint32(head.Len))
		queue.Index[common.ToHex(body.Key)] = &item{
			key:    body.Key,
			val:    body.Val,
			offset: next,
			refCnt: refCnt,
		}

		bitcask := queue.route(body.Key)
		bitcask.Put(head.Flg, body.Key, body.Val)

		next = next + int64(length)
	}
}

func (queue *FileQueue) align(tLen uint32) uint32 {
	if tLen%256 != 0 {
		tLen += 256 - tLen%256
	}

	return tLen
}

func (queue *FileQueue) encode(hBuf []byte, bBuf []byte) []byte {
	tLen := queue.align(uint32(len(hBuf)) + uint32(len(bBuf)))

	tBuf := make([]byte, tLen)
	copy(tBuf[0:], hBuf[:])
	copy(tBuf[len(hBuf):], bBuf[:])

	return tBuf
}

func (queue *FileQueue) encodeHead(flag uint32, data []byte) ([]byte, error) {
	head := RecordHead{
		Flg:       flag,
		Len:       uint32(len(data)),
		TimeStamp: uint64(time.Now().Unix()),
		Crc:       CheckSum(data),
	}

	buf := make([]byte, RecordHeadLength)
	err := binary.Write(NewLmBuffer(buf[:]), binary.LittleEndian, &head)
	if err != nil {
		return nil, err
	} else {
		return buf, nil
	}
}

func (queue *FileQueue) encodeBody(key []byte, val []byte) ([]byte, error) {
	body := &RecordBody{
		Key: key,
		Val: val,
	}

	return rlp.EncodeToBytes(body)
}

func (queue *FileQueue) encodeBatchItem(item *BatchItem) ([]byte, error) {
	body, err := queue.encodeBody(item.Key, item.Val)
	if err != nil {
		return nil, err
	}

	head, err := queue.encodeHead(item.Flg, body)
	if err != nil {
		return nil, err
	}

	return queue.encode(head, body), nil
}

func (queue *FileQueue) encodeBatchItems(items []*BatchItem) ([][]byte, error) {
	if len(items) <= 0 {
		return nil, nil
	}

	tmpBuf := make([][]byte, len(items))
	for index := 0; index < len(items); index++ {
		buf, err := queue.encodeBatchItem(items[index])
		if err != nil {
			return nil, err
		}

		tmpBuf[index] = buf
	}

	return tmpBuf, nil
}

func (queue *FileQueue) mergeBatchItems(tmpBuf [][]byte) []byte {
	totalSize := 0
	for index := 0; index < len(tmpBuf); index++ {
		totalSize = totalSize + len(tmpBuf[index])
	}

	totalBuf := make([]byte, totalSize)
	curOffset := 0
	for index := 0; index < len(tmpBuf); index++ {
		copy(totalBuf[curOffset:], tmpBuf[index])
		curOffset = curOffset + len(tmpBuf[index])
	}

	return totalBuf
}

func (queue *FileQueue) flush(data []byte) (int64, error) {
	file, err := os.OpenFile(queue.path(), os.O_APPEND, os.ModePerm)
	defer file.Close()
	if err != nil {
		return -1, err
	}

	_, err = file.Seek(queue.Offset, 0)
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

	tmp := queue.Offset
	queue.Offset += int64(n)
	return tmp, nil
}

func (queue *FileQueue) route(key []byte) *BitCask {
	num := Byte2Uint32(key)
	index := num >> ((8 - queue.Height) * 4)
	return queue.BitCasks[index]
}

func (queue *FileQueue) Get(flag uint32, key []byte) ([]byte, error) {
	tmp, ok := queue.Index[common.ToHex(key)]
	if !ok {
		bitcask := queue.route(key)
		if bitcask == nil {
			panic(fmt.Sprintf("queue get k/v. bitcask is nil.key: %s", common.ToHex(key)))
		} else {
			return bitcask.Get(flag, key)
		}
	} else {
		return tmp.val, nil
	}
}

func (queue *FileQueue) Put(flag uint32, key []byte, val []byte) error {
	buf, err := queue.encodeBatchItem(&BatchItem{
		Flg: flag,
		Key: key,
		Val: val,
	})

	if err != nil {
		return err
	}

	offset, err := queue.flush(buf)
	if err != nil {
		return err
	} else {
		queue.set(flag, key, val, offset)
		return nil
	}
}

func (queue *FileQueue) PutBatch(items []*BatchItem) error {
	tmpBuf, err := queue.encodeBatchItems(items)
	if err != nil {
		return err
	}

	totalBuf := queue.mergeBatchItems(tmpBuf)
	offset, err := queue.flush(totalBuf)
	if err != nil {
		return err
	}

	queue.setBatch(tmpBuf, items, offset)
	return nil
}

func (queue *FileQueue) set(flag uint32, key []byte, val []byte, offset int64) {
	tmp, ok := queue.Index[common.ToHex(key)]
	refCnt := 0
	if !ok {
		refCnt = 1
	} else {
		refCnt = tmp.refCnt + 1
	}

	queue.Index[common.ToHex(key)] = &item{
		key:    key,
		val:    val,
		offset: offset,
		refCnt: refCnt,
	}

	queue.deliver(flag, key, val)
}

func (queue *FileQueue) setBatch(tmpBuf [][]byte, items []*BatchItem, offset int64) {
	curOffset := offset
	for index := 0; index < len(tmpBuf); index++ {
		tmp := items[index]
		queue.set(tmp.Flg, tmp.Key, tmp.Val, curOffset)
		curOffset = curOffset + int64(len(tmpBuf[index]))
	}
	queue.Offset = curOffset
}

func (queue *FileQueue) deliver(flag uint32, key []byte, val []byte) {
	bitcask := queue.route(key)
	bitcask.Put(flag, key, val)
}
