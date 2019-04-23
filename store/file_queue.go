package store

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type item struct {
	flg    uint32
	key    []byte
	val    []byte
	offset int64
	refCnt int
}

type FileQueue struct {
	Home   string
	Offset int64

	IndexRW sync.RWMutex
	Index   map[string]*item

	LevelDB *leveldb.LevelDBDatabase

	SyncFileDB *SyncFileDB
	DoneChan   chan *Inject
	ErrChan    chan *Inject
	Quit       chan struct{}
}

func (queue *FileQueue) path() string {
	return filepath.Join(queue.Home, "tmp.data")
}

func NewFileQueue(home string, levelDB *leveldb.LevelDBDatabase, extend WriteExtend) *FileQueue {
	doneChan := make(chan *Inject, 1024*256)
	errChan := make(chan *Inject)
	quit := make(chan struct{})
	return &FileQueue{
		Home:       home,
		Index:      make(map[string]*item),
		Offset:     0,
		LevelDB:    levelDB,
		SyncFileDB: NewSyncFileDB(home, levelDB, doneChan, errChan, quit, extend),
		DoneChan:   doneChan,
		ErrChan:    errChan,
		Quit:       quit,
	}
}

func (queue *FileQueue) Start() {
	queue.start()
	queue.SyncFileDB.Open()

	err := queue.checkFile()
	if err != nil {
		panic("start queue.check tmp file err: " + err.Error())
	}
}

func (queue *FileQueue) start() {
	go func() {
		for {
			select {
			case <-queue.Quit:
				return
			case done := <-queue.DoneChan:
				queue.afterPut(done)
				// continue
			case <-queue.ErrChan:
				// continue
			}
		}
	}()
}

func (queue *FileQueue) Close() {
	close(queue.Quit)
}

func (queue *FileQueue) setIndex(item *item) {
	queue.IndexRW.Lock()
	defer queue.IndexRW.Unlock()

	tmp, ok := queue.Index[common.ToHex(item.key)]
	if !ok {
		item.refCnt = 1
	} else {
		item.refCnt = tmp.refCnt + 1
	}

	queue.Index[common.ToHex(item.key)] = item
}

func (queue *FileQueue) delIndex(flag uint32, key []byte) {
	queue.IndexRW.Lock()
	defer queue.IndexRW.Unlock()

	val, ok := queue.Index[common.ToHex(key)]
	if !ok {
		log.Errorf("del index.done is not exist.flg: %d, key: %s", flag, common.ToHex(key))
	} else {
		if val.flg != flag {
			panic(fmt.Sprintf("del index.val.flag(%d) != flag(%d)", val.flg, flag))
		}

		if val.refCnt <= 1 {
			delete(queue.Index, common.ToHex(key))
			if len(queue.Index) <= 0 {
				// del tmp file.
			}
		} else {
			val.refCnt = val.refCnt - 1
			queue.Index[common.ToHex(key)] = val
		}
	}
}

func (queue *FileQueue) getIndex(flag uint32, key []byte) []byte {
	queue.IndexRW.Lock()
	defer queue.IndexRW.Unlock()

	val, ok := queue.Index[common.ToHex(key)]
	if !ok {
		return nil
	} else {
		if val.flg == flag {
			return val.val
		} else {
			log.Errorf("val.flag(%d) != flag(%d)", val.flg, flag)
			return nil
		}
	}
}

func (queue *FileQueue) checkFile() error {
	filePath := queue.path()
	isExist, err := FileUtilsIsExist(filePath)
	if err != nil {
		return err
	}

	if !isExist {
		err = FileUtilsCreateFile(filePath)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else {
		offset, err := queue.scanFile(queue.path(), queue.Offset)
		if err == nil || err == ErrEOF {
			queue.Offset = offset
			return nil
		} else {
			return err
		}
	}
}

func (queue *FileQueue) emptyFile(path string) error {
	return nil
}

func (queue *FileQueue) scanFile(filePath string, offset int64) (int64, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	defer file.Close()

	if err != nil {
		return -1, err
	}

	queue.Offset = offset
	for {
		head, body, err := FileUtilsRead(file, queue.Offset)
		if err == io.EOF {
			return queue.Offset, ErrEOF
		}

		if err != nil {
			return -1, err
		}

		length := FileUtilsAlign(uint32(RecordHeadLength) + uint32(head.Len))
		queue.deliver(head.Flg, body.Key, body.Val)
		queue.Offset += int64(length)
	}
}

func (queue *FileQueue) encodeBatchItems(items []*BatchItem) ([][]byte, error) {
	if len(items) <= 0 {
		return nil, nil
	}

	tmpBuf := make([][]byte, len(items))
	for index := 0; index < len(items); index++ {
		tmp := items[index]
		item := &BatchItem{
			Flg: tmp.Flg,
			Key: make([]byte, len(tmp.Key)),
			Val: make([]byte, len(tmp.Val)),
		}

		copy(item.Key, tmp.Key)
		copy(item.Val, tmp.Val)
		items[index] = item

		buf, err := FileUtilsEncode(items[index].Flg, items[index].Key, items[index].Val)
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

func (queue *FileQueue) Get(flag uint32, key []byte) ([]byte, error) {
	val := queue.getIndex(flag, key)
	if val != nil {
		return val, nil
	} else {
		return queue.SyncFileDB.Get(flag, key)
	}
}

func (queue *FileQueue) Put(flag uint32, key []byte, val []byte) error {
	buf, err := FileUtilsEncode(flag, key, val)
	if err != nil {
		return err
	}

	path := queue.path()
	length, err := FileUtilsFlush(path, queue.Offset, buf)
	if err != nil {
		return err
	} else {
		queue.deliver(flag, key, val)
		queue.Offset += length
		return nil
	}
}

func (queue *FileQueue) PutBatch(items []*BatchItem) error {
	tmpBuf, err := queue.encodeBatchItems(items)
	if err != nil {
		return err
	}

	path := queue.path()
	totalBuf := queue.mergeBatchItems(tmpBuf)
	_, err = FileUtilsFlush(path, queue.Offset, totalBuf)
	if err != nil {
		return err
	}

	queue.deliverBatch(tmpBuf, items)
	return nil
}

func (queue *FileQueue) deliverBatch(tmpBuf [][]byte, items []*BatchItem) {
	for index := 0; index < len(tmpBuf); index++ {
		tmp := items[index]
		queue.deliver(tmp.Flg, tmp.Key, tmp.Val)
		queue.Offset += int64(len(tmpBuf[index]))
	}
}

func (queue *FileQueue) deliver(flag uint32, key []byte, val []byte) {
	queue.setIndex(&item{
		flg:    flag,
		key:    key,
		val:    val,
		offset: queue.Offset,
		refCnt: 1,
	})

	queue.SyncFileDB.Put(flag, key, val)
}

func (queue *FileQueue) afterPut(op *Inject) {
	queue.delIndex(op.Flg, op.Key)
}
