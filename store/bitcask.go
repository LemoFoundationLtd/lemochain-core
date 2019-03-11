package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-go/store/leveldb"
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
	rw        sync.RWMutex
	HomePath  string
	CurIndex  int
	CurOffset int64
	IndexDB   *leveldb.LevelDBDatabase

	q *queue
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

func NewBitCask(homePath string, lastIndex int, lastOffset uint32, after AfterScan, indexDB *leveldb.LevelDBDatabase) (*BitCask, error) {
	db := &BitCask{HomePath: homePath, IndexDB: indexDB}
	db.q = NewQueue(after, db.IndexDB)
	db.q.start()

	isExist, err := db.isExist(homePath)
	if err != nil {
		return nil, err
	}

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

			length := bitcask.align(uint32(RHEAD_LENGTH) + uint32(head.Len))
			bitcask.q.set(body.Route, &element{
				item: &BatchItem{
					Flg: uint(head.Flg),
					Key: body.Key,
					Val: body.Val,
				},
				offset: lastOffset,
				len:    int(length),
			})

			nextOffset = nextOffset + length
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
	copy(tBuf[0:], hBuf[:])
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

	err = leveldb.SetPos(bitcask.IndexDB, key, &leveldb.Position{
		Flag:   uint32(flag),
		Route:  route,
		Offset: uint32(int(offset) | bitcask.CurIndex),
	})

	if err != nil {
		return err
	} else {
		return nil
	}
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
	file, err := os.OpenFile(dataPath, os.O_RDONLY, os.ModePerm)
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

	index := bitcask.q.get(key)
	if index != nil {
		return bitcask.get(index.flg, route, key, int64(index.pos))
	} else {
		position, err := leveldb.GetPos(bitcask.IndexDB, key)
		if err != nil {
			return nil, err
		} else {
			return bitcask.get(uint(position.Flag), position.Route, key, int64(position.Offset))
		}
	}
}

func (bitcask *BitCask) Close() error {
	return bitcask.q.stop()
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
		elements := make([]*element, len(items))
		for index := 0; index < len(items); index++ {
			curPos := uint32(int(pos+tmpPos[index]) | bitcask.CurIndex)
			elements[index] = &element{
				item:   items[index],
				offset: curPos,
				len:    len(tmpBuf[index]),
			}
		}

		bitcask.q.setBatch(route, elements)
		return nil
	}
}

type element struct {
	item   *BatchItem
	offset uint32
	len    int
}

type inject struct {
	route    []byte
	elements []*element
}

type queue struct {
	after   AfterScan
	cache   map[string]RIndex
	inject  chan *inject
	quit    chan struct{}
	indexDB *leveldb.LevelDBDatabase
}

func NewQueue(after AfterScan, indexDB *leveldb.LevelDBDatabase) *queue {
	return &queue{
		after:   after,
		cache:   make(map[string]RIndex),
		inject:  make(chan *inject),
		quit:    make(chan struct{}),
		indexDB: indexDB,
	}
}

func (q *queue) set(route []byte, item *element) {
	items := make([]*element, 1)
	items[0] = item
	q.setBatch(route, items)
}

func (q *queue) setBatch(route []byte, items []*element) {
	for index := 0; index < len(items); index++ {
		q.cache[string(items[index].item.Key)] = RIndex{
			flg: items[index].item.Flg,
			pos: items[index].offset,
			len: items[index].len,
		}
	}

	op := &inject{
		route:    route,
		elements: items,
	}

	select {
	case q.inject <- op:
		return
	case <-q.quit:
		return
	}
}

func (q *queue) get(key []byte) *RIndex {
	index, ok := q.cache[string(key)]
	if !ok {
		return nil
	} else {
		return &RIndex{
			flg: index.flg,
			pos: index.pos,
			len: index.len,
		}
	}
}

func (q *queue) start() {
	go q.loop()
}

func (q *queue) loop() {
	for {
		select {
		case <-q.quit:
			return

		case op := <-q.inject:
			route := op.route
			elements := op.elements

			for index := 0; index < len(elements); index++ {
				element := elements[index]
				_, ok := q.cache[string(element.item.Key)]
				if !ok {

				} else {
					err := leveldb.SetPos(q.indexDB, element.item.Key, &leveldb.Position{
						Flag:   uint32(element.item.Flg),
						Route:  route,
						Offset: uint32(element.offset),
					})

					if err != nil {
						panic("q.after err: " + err.Error())
					}

					if q.after == nil {
						log.Errorf("q.after is nil.")
						return
					}

					err = q.after(element.item.Flg, route, element.item.Key, element.item.Val, element.offset)
					if err != nil {
						panic("q.after err: " + err.Error())
					} else {
						delete(q.cache, string(element.item.Key))
					}
				}
			}
		}
	}
}

func (q *queue) stop() error {
	close(q.quit)
	return nil
}
