package store

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type item struct {
	key    []byte
	val    []byte
	offset int64
	refCnt int
}

type FileQueue struct {
	RW         sync.RWMutex
	Home       string
	Index      map[string]*item
	Offset     int64
	LevelDB    *leveldb.LevelDBDatabase
	SyncFileDB *SyncFileDB

	DoneChan chan *Inject
	ErrChan  chan *Inject
	Quit     chan struct{}
}

func (queue *FileQueue) path() string {
	return filepath.Join(queue.Home, "%tmp.data")
}

func NewFileQueue(home string, levelDB *leveldb.LevelDBDatabase, extend WriteExtend) *FileQueue {
	doneChan := make(chan *Inject)
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
				continue
			case <-queue.ErrChan:
				continue
			}
		}
	}()
}

func (queue *FileQueue) Close() {
	close(queue.Quit)
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
		offset, err := queue.scanFile(queue.path(), 0)
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

	next := offset
	for {
		head, body, err := FileUtilsRead(file, next)
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

		length := FileUtilsAlign(uint32(RecordHeadLength) + uint32(head.Len))
		queue.Index[common.ToHex(body.Key)] = &item{
			key:    body.Key,
			val:    body.Val,
			offset: next,
			refCnt: refCnt,
		}

		queue.deliver(head.Flg, body.Key, body.Val)
		next = next + int64(length)
	}
}

func (queue *FileQueue) encodeBatchItems(items []*BatchItem) ([][]byte, error) {
	if len(items) <= 0 {
		return nil, nil
	}

	tmpBuf := make([][]byte, len(items))
	for index := 0; index < len(items); index++ {
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
	queue.RW.Lock()
	defer queue.RW.Unlock()

	tmp, ok := queue.Index[common.ToHex(key)]
	if !ok {
		return queue.SyncFileDB.Get(flag, key)
	} else {
		return tmp.val, nil
	}
}

func (queue *FileQueue) Put(flag uint32, key []byte, val []byte) error {
	queue.RW.Lock()
	defer queue.RW.Unlock()

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
	queue.RW.Lock()
	defer queue.RW.Unlock()

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
		offset: queue.Offset,
		refCnt: refCnt,
	}

	queue.SyncFileDB.Put(flag, key, val)
}

func (queue *FileQueue) afterPut(op *Inject) {
	queue.RW.Lock()
	defer queue.RW.Unlock()

	val, ok := queue.Index[common.ToHex(op.Key)]
	if !ok {
		log.Errorf("done is not exist.flag: %d", op.Flg)
	} else {
		if val.refCnt <= 1 {
			delete(queue.Index, common.ToHex(op.Key))
		} else {
			val.refCnt = val.refCnt - 1
			queue.Index[common.ToHex(op.Key)] = val
		}
	}
}
