package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type RHead struct {
	Flg       uint32
	Len       uint32
	TimeStamp uint64
	Crc       uint16
}

var RHeadLength = binary.Size(RHead{})

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

type WritingElement struct {
	failCount   int
	writingData []*element
}

type BitCask struct {
	HomePath string

	CurIndex  int
	CurOffset int64
	IndexDB   *leveldb.LevelDBDatabase

	Q            *queue
	IndexWriting WritingElement
	IndexWritEnd chan *inject

	RW sync.RWMutex
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
	return err
}

func (bitcask *BitCask) homeIsNotExist(home string) error {
	err := os.MkdirAll(home, os.ModePerm)
	if err != nil {
		return err
	}

	bitcask.CurIndex = 0
	bitcask.CurOffset = 0
	return bitcask.createFile(bitcask.CurIndex)
}

func (bitcask *BitCask) homeIsExist(home string) error {
	pos, err := leveldb.GetScanPos(bitcask.IndexDB, home)
	if err != nil {
		return err
	}

	bitcask.CurIndex = int(pos & 0xFF)
	bitcask.CurOffset = int64(pos & 0xFFFFFF00)
	isExist, err := IsExist(bitcask.path(bitcask.CurIndex))
	if err != nil {
		return err
	}

	if isExist {
		return nil
	} else {
		return bitcask.createFile(bitcask.CurIndex)
	}
}

func NewBitCask(home string, after AfterScan, indexDB *leveldb.LevelDBDatabase) (*BitCask, error) {
	db := &BitCask{HomePath: home, IndexDB: indexDB, Q: NewQueue(home, indexDB, after)}
	db.Q.start()

	isExist, err := db.isExist(home)
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

func (bitcask *BitCask) InitBitCask() error {
	pos, err := leveldb.GetScanPos(bitcask.IndexDB, bitcask.HomePath)
	if err != nil {
		return err
	}

	err = bitcask.scan(pos)
	if err != nil {
		return err
	}

	isExist, err := IsExist(bitcask.path(bitcask.CurIndex))
	if err != nil {
		return err
	}

	if !isExist {
		err = bitcask.createFile(bitcask.CurIndex)
		if err != nil {
			return err
		}
	}

	pos = uint32(int(bitcask.CurOffset) | bitcask.CurIndex)
	return leveldb.SetScanPos(bitcask.IndexDB, bitcask.HomePath, pos)
}

func (bitcask *BitCask) scanFile(filePath string, offset int64) (int64, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	defer file.Close()

	if err != nil {
		return -1, err
	}

	nextOffset := offset
	for {
		head, body, err := bitcask.read(file, nextOffset)
		if err == io.EOF {
			return nextOffset, ErrEOF
		}

		if err != nil {
			return -1, err
		}

		length := bitcask.align(uint32(RHeadLength) + uint32(head.Len))
		curPos := uint32(nextOffset) | uint32(bitcask.CurIndex)
		bitcask.Q.set(curPos, body.Route, &element{
			item: &BatchItem{
				Flg: uint(head.Flg),
				Key: body.Key,
				Val: body.Val,
			},
			offset: curPos,
			len:    int(length),
		})

		nextOffset = nextOffset + int64(length)
	}
}

func (bitcask *BitCask) scan(lastPos uint32) error {
	nextIndex := lastPos & 0xFF
	nextOffset := lastPos & 0xFFFFFF00
	for {
		nextPath := bitcask.path(int(nextIndex))
		isExist, err := bitcask.isExist(nextPath)
		if err != nil {
			return err
		}

		if !isExist { // 刚刚写满
			bitcask.CurIndex = int(nextIndex)
			bitcask.CurOffset = 0
			return nil
		}

		offset, err := bitcask.scanFile(nextPath, int64(nextOffset))
		if err == ErrEOF {
			if offset < int64(maxFileSize) {
				bitcask.CurIndex = int(nextIndex)
				bitcask.CurOffset = offset
				return nil
			} else {
				nextIndex = nextIndex + 1
				nextOffset = 0
				continue
			}
		} else {
			return err
		}
	}
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

	buf := make([]byte, RHeadLength)
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

func (bitcask *BitCask) checkAndFlush(data []byte) (int64, error) {
	bitcask.RW.Lock()
	defer bitcask.RW.Unlock()

	err := bitcask.checkSize(int64(len(data)))
	if err != nil {
		return -1, err
	}

	offset, err := bitcask.flush(data)
	if err != nil {
		return -1, err
	} else {
		return offset, nil
	}
}

func (bitcask *BitCask) checkSize(size int64) error {
	if (bitcask.CurOffset + size) <= int64(maxFileSize) {
		return nil
	}

	tmpCurIndex := bitcask.CurIndex + 1
	err := bitcask.createFile(tmpCurIndex)
	if err != nil {
		return err
	} else {
		bitcask.CurIndex = tmpCurIndex
		bitcask.CurOffset = 0
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
	body, err := bitcask.encodeBody(route, key, val)
	if err != nil {
		return err
	}

	head, err := bitcask.encodeHead(flag, body)
	if err != nil {
		return err
	}

	data := bitcask.encode(head, body)

	offset, err := bitcask.checkAndFlush(data)
	if err != nil {
		return err
	}

	return leveldb.SetPos(bitcask.IndexDB, key, &leveldb.Position{
		Flag:   uint32(flag),
		Route:  route,
		Offset: uint32(int(offset) | bitcask.CurIndex),
	})
	return nil
}

func (bitcask *BitCask) read(file *os.File, offset int64) (*RHead, *RBody, error) {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return nil, nil, err
	}

	heaBuf := make([]byte, RHeadLength)
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
	}

	if head.Flg != uint32(flag) {
		return nil, nil
	} else {
		return body.Val, nil
	}
}

func (bitcask *BitCask) Get(flag uint, route []byte, key []byte, offset int64) ([]byte, error) {
	bitcask.RW.RLock()
	defer bitcask.RW.RUnlock()

	return bitcask.get(flag, route, key, offset)
}

func (bitcask *BitCask) Get4Cache(route []byte, key []byte) ([]byte, error) {
	bitcask.RW.RLock()
	defer bitcask.RW.RUnlock()

	index := bitcask.Q.get(key)
	if index != nil {
		return bitcask.get(index.flg, route, key, int64(index.pos))
	}

	position, err := leveldb.GetPos(bitcask.IndexDB, key)
	if err != nil {
		return nil, err
	}

	if position == nil {
		return nil, ErrNotExist
	}

	return bitcask.get(uint(position.Flag), position.Route, key, int64(position.Offset))
}

func (bitcask *BitCask) Close() error {
	return bitcask.Q.stop()
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

func (bitcask *BitCask) encodeBatchItem(route []byte, item *BatchItem) ([]byte, error) {
	body, err := bitcask.encodeBody(route, item.Key, item.Val)
	if err != nil {
		return nil, err
	}

	head, err := bitcask.encodeHead(item.Flg, body)
	if err != nil {
		return nil, err
	}

	return bitcask.encode(head, body), nil
}

func (bitcask *BitCask) encodeBatchItems(route []byte, items []*BatchItem) ([]int64, [][]byte, int, error) {
	if len(items) <= 0 {
		return nil, nil, 0, nil
	}

	tmpPos := make([]int64, len(items))
	tmpBuf := make([][]byte, len(items))
	totalSize := 0
	for index := 0; index < len(items); index++ {
		buf, err := bitcask.encodeBatchItem(route, items[index])
		if err != nil {
			return nil, nil, -1, err
		}

		tmpBuf[index] = buf
		totalSize = totalSize + len(tmpBuf[index])
		if index == 0 {
			tmpPos[index] = 0
		} else {
			tmpPos[index] = tmpPos[index-1] + int64(len(tmpBuf[index-1]))
		}
	}

	return tmpPos, tmpBuf, totalSize, nil
}

func (bitcask *BitCask) mergeBatchItems(tmpPos []int64, tmpBuf [][]byte, totalSize int) []byte {
	totalBuf := make([]byte, totalSize)
	for index := 0; index < len(tmpPos); index++ {
		copy(totalBuf[tmpPos[index]:], tmpBuf[index])
	}

	return totalBuf
}

func (bitcask *BitCask) writeBatchItemsIndex(curPos int64, tmpPos []int64, tmpBuf [][]byte, route []byte, items []*BatchItem) {
	elements := make([]*element, len(items))
	for index := 0; index < len(items); index++ {
		curPos := uint32(int(curPos+tmpPos[index]) | bitcask.CurIndex)
		elements[index] = &element{
			item:   items[index],
			offset: curPos,
			len:    len(tmpBuf[index]),
		}
	}

	scanPos := uint32(bitcask.CurIndex) | uint32(bitcask.CurOffset)
	bitcask.Q.setBatch(scanPos, route, elements)
}

func (bitcask *BitCask) Commit(batch Batch) error {
	route := batch.Route()
	items := batch.Items()

	tmpPos, tmpBuf, totalSize, err := bitcask.encodeBatchItems(route, items)
	if err != nil {
		return err
	}
	if totalSize <= 0 {
		return nil
	}

	totalBuf := bitcask.mergeBatchItems(tmpPos, tmpBuf, totalSize)
	pos, err := bitcask.checkAndFlush(totalBuf)
	if err != nil {
		return err
	}

	bitcask.writeBatchItemsIndex(pos, tmpPos, tmpBuf, route, items)
	return nil
}

// //////////////////
type element struct {
	item   *BatchItem
	offset uint32
	len    int
}

type inject struct {
	scanPos  uint32
	route    []byte
	elements []*element
}

type queue struct {
	path string

	cache   sync.Map
	writing []*inject
	inject  chan *inject
	quit    chan struct{}
	indexDB *leveldb.LevelDBDatabase

	RW          sync.RWMutex
	repeatTimer *time.Timer
	repeatChan  chan struct{}
	repeatCount int

	after AfterScan
}

func NewQueue(path string, indexDB *leveldb.LevelDBDatabase, after AfterScan) *queue {
	return &queue{
		path: path,

		writing: make([]*inject, 0),
		inject:  make(chan *inject),
		quit:    make(chan struct{}),
		indexDB: indexDB,

		repeatChan: make(chan struct{}),

		after:       after,
		repeatCount: 0,
	}
}

func (q *queue) set(scanPos uint32, route []byte, item *element) {
	items := make([]*element, 1)
	items[0] = item
	op := q.batch(scanPos, route, items)
	if op == nil {
		return
	}

	select {
	case q.inject <- op:
		return
	case <-q.quit:
		return
	}
}

func (q *queue) setBatch(scanPos uint32, route []byte, items []*element) {
	op := q.batch(scanPos, route, items)
	if op == nil {
		return
	}

	select {
	case q.inject <- op:
		return
	case <-q.quit:
		return
	}
}

func (q *queue) batch(scanPos uint32, route []byte, items []*element) *inject {
	if len(items) <= 0 {
		return nil
	}

	for index := 0; index < len(items); index++ {
		q.cache.Store(string(items[index].item.Key), RIndex{
			flg: items[index].item.Flg,
			pos: items[index].offset,
			len: items[index].len,
		})
	}

	return &inject{
		scanPos:  scanPos,
		route:    route,
		elements: items,
	}
}

func (q *queue) get(key []byte) *RIndex {
	index, ok := q.cache.Load(string(key))
	if !ok {
		return nil
	} else {
		tmp := index.(RIndex)
		return &RIndex{
			flg: tmp.flg,
			pos: tmp.pos,
			len: tmp.len,
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

		case <-q.repeatChan:
			q.RW.Lock()
			if len(q.writing) <= 0 {
				q.RW.Unlock()
				continue
			} else {
				op := q.writing[0]
				err := q.process(op)
				if err != nil {
					log.Errorf("q.repeat channel. len(writing):" + strconv.Itoa(len(q.writing)) + "|err: " + err.Error())
					q.repeat(op)
					q.RW.Unlock()
					continue
				} else {
					copy(q.writing[0:], q.writing[1:])
					q.repeatCount = 0
					q.RW.Unlock()
					continue
				}
			}
		case op := <-q.inject:
			q.RW.Lock()
			if len(q.writing) > 0 {
				log.Errorf("q.inject channel. len(writing):" + strconv.Itoa(len(q.writing)))
				q.writing = append(q.writing, op)
				q.RW.Unlock()
				continue
			} else {
				err := q.process(op)
				if err != nil {
					q.writing = append(q.writing, op)
					log.Errorf("q.inject channel. len(writing):" + strconv.Itoa(len(q.writing)) + "|err: " + err.Error())
					q.repeat(op)
					q.RW.Unlock()
					continue
				} else {
					q.repeatCount = 0
					q.RW.Unlock()
					continue
				}
			}
		}
	}
}

func (q *queue) repeat(op *inject) {
	q.repeatCount = q.repeatCount + 1
	if q.repeatCount > 1024 {
		panic("repeat set index to level db greater than 3.")
	} else {
		log.Errorf("repeat set index to level db.count: " + strconv.Itoa(q.repeatCount))
		q.repeatTimer = time.AfterFunc(time.Duration(q.repeatCount*int(time.Second)), func() {
			q.repeatChan <- struct{}{}
		})
	}
}

func (q *queue) process(op *inject) error {
	route := op.route
	elements := op.elements
	scanPos := op.scanPos

	for index := 0; index < len(elements); index++ {
		element := elements[index]
		err := leveldb.SetPos(q.indexDB, element.item.Key, &leveldb.Position{
			Flag:   uint32(element.item.Flg),
			Route:  route,
			Offset: uint32(element.offset),
		})

		if err != nil {
			return err
		}

		if q.after == nil {
			log.Errorf("q.after is nil.")
		} else {
			err = q.after(element.item.Flg, route, element.item.Key, element.item.Val, element.offset)
			if err != nil {
				return err
			}
		}
	}

	return leveldb.SetScanPos(q.indexDB, q.path, scanPos)
}

func (q *queue) stop() error {
	if q.repeatTimer != nil {
		q.repeatTimer.Stop()
	}

	close(q.quit)

	return nil
}
