package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	CACHE_FLG_STOP         = uint(0)
	CACHE_FLG_BLOCK        = uint(1)
	CACHE_FLG_BLOCK_HEIGHT = uint(2)
	CACHE_FLG_TRIE         = uint(3)
	CACHE_FLG_ACT          = uint(4)
	CACHE_FLG_TX_INDEX     = uint(5)
	CACHE_FLG_CODE         = uint(6)
	CACHE_FLG_KV           = uint(7)
)

type BeansDB struct {
	height    uint
	bitcasks  []*BitCask
	indexDB   *leveldb.LevelDBDatabase
	route2key sync.Map
	after     BizAfterScan
}

type BizAfterScan func(flag uint, key []byte, val []byte) error

func (beansdb *BeansDB) initBitCasks(home string, count int) {
	for index := 0; index < count; index++ {
		pathModule := filepath.Join(home, "/%02d/%02d/")
		path := fmt.Sprintf(pathModule, index>>4, index&0xf)

		bitcask, err := NewBitCask(path, beansdb.AfterScan, beansdb.indexDB)
		if err != nil {
			panic("new bitcask err : " + err.Error())
		} else {
			beansdb.bitcasks[index] = bitcask
		}
	}
}

func NewBeansDB(home string, height int, DB *leveldb.LevelDBDatabase, after BizAfterScan) *BeansDB {
	if height != 2 {
		panic("beansdb height != 2")
	}

	count := 1 << (uint(height) * 4)
	beansdb := &BeansDB{
		height:   uint(height),
		bitcasks: make([]*BitCask, count),
		indexDB:  DB, after: after,
	}

	beansdb.initBitCasks(home, count)
	return beansdb
}

func (beansdb *BeansDB) AfterScan(flag uint, route []byte, key []byte, val []byte, offset uint32) error {
	if beansdb.after == nil {
		return nil
	}

	err := beansdb.after(flag, key, val)
	if err != nil {
		return err
	} else {
		beansdb.route2key.Delete(string(key))
		return nil
	}
}

func (beansdb *BeansDB) route(route []byte) *BitCask {
	num := Byte2Uint32(route)
	index := num >> ((8 - beansdb.height) * 4)
	return beansdb.bitcasks[index]
}

func (beansdb *BeansDB) NewBatch(route []byte) Batch {
	batch := &LmDBBatch{
		db:    beansdb,
		items: make([]*BatchItem, 0),
		size:  0,
	}

	batch.route = make([]byte, len(route))
	copy(batch.route, route)
	return batch
}

func (beansdb *BeansDB) Commit(batch Batch) error {
	route := batch.Route()
	bitcask := beansdb.route(route)
	err := bitcask.Commit(batch)
	if err != nil {
		return err
	} else {
		items := batch.Items()
		route := batch.Route()

		for index := 0; index < len(items); index++ {
			beansdb.route2key.Store(string(items[index].Key), route)
		}
		return nil
	}
}

func (beansdb *BeansDB) Put(flg uint, route []byte, key []byte, val []byte) error {
	bitcask := beansdb.route(route)
	return bitcask.Put(flg, route, key, val)
}

func (beansdb *BeansDB) Has(key []byte) (bool, error) {
	val, err := beansdb.Get(key)
	if err != nil {
		return false, err
	}

	if val == nil {
		return false, nil
	} else {
		return true, nil
	}
}

func (beansdb *BeansDB) Get(key []byte) ([]byte, error) {
	route, ok := beansdb.route2key.Load(string(key))
	if !ok {
		position, err := leveldb.GetPos(beansdb.indexDB, key)
		if err != nil {
			return nil, err
		}

		if position == nil {
			return nil, nil
		}

		bitcask := beansdb.route(position.Route)
		val, err := bitcask.Get(uint(position.Flag), position.Route, key, int64(position.Offset))
		if err != nil {
			log.Error("get data from disk err : " + err.Error())
			return nil, err
		} else {
			return val, nil
		}
	} else {
		bitcask := beansdb.route(route.([]byte))
		val, err := bitcask.Get4Cache(route.([]byte), key)
		if err != nil {
			return nil, err
		} else {
			return val, nil
		}
	}
}

func (beansdb *BeansDB) Close() error {
	for index := 0; index < len(beansdb.bitcasks); index++ {
		beansdb.bitcasks[index].Close()
	}
	return nil
}

// ///////////////////////
type contextHead struct {
	FileLen   uint32
	Version   uint32
	TimeStamp uint32
	Crc       uint16
}

type contextItemHead struct {
	Flg uint32
	Len uint32
}

type CandidatePos struct {
	Pos uint32
	Len uint32
}

type CandidateCache struct {
	Candidates   map[common.Address]CandidatePos
	ItemMaxSize  int
	Cap          int
	Cur          int
	CandidateBuf []byte
}

func NewCandidateCache() *CandidateCache {
	return &CandidateCache{
		Candidates:   make(map[common.Address]CandidatePos),
		ItemMaxSize:  64,
		Cap:          64 * 64,
		Cur:          0,
		CandidateBuf: make([]byte, 64*64),
	}
}

func (cache *CandidateCache) Set(candidate *Candidate) error {
	if candidate == nil {
		return nil
	}

	buf, err := rlp.EncodeToBytes(candidate)
	if err != nil {
		return err
	}

	candidatePosLen := binary.Size(CandidatePos{})
	if cache.ItemMaxSize < (len(buf) + candidatePosLen) {
		panic("candidate buf is larger than ItemMaxSize")
	}

	pos, ok := cache.Candidates[candidate.Address]
	if ok {
		pos.Len = uint32(len(buf))
		err = binary.Write(NewLmBuffer(cache.CandidateBuf[pos.Pos:]), binary.LittleEndian, &pos)
		if err != nil {
			return err
		} else {
			copy(cache.CandidateBuf[(pos.Pos+uint32(candidatePosLen)):], buf[:])
			cache.Candidates[candidate.Address] = pos
		}
	} else {
		if cache.Cur+cache.ItemMaxSize > cache.Cap {
			tmp := make([]byte, cache.Cap*2)
			copy(tmp[:], cache.CandidateBuf[:])
			cache.Cap = cache.Cap * 2
			cache.CandidateBuf = tmp
		}

		pos := CandidatePos{
			Pos: uint32(cache.Cur),
			Len: uint32(len(buf)),
		}

		err = binary.Write(NewLmBuffer(cache.CandidateBuf[pos.Pos:]), binary.LittleEndian, &pos)
		if err != nil {
			return err
		} else {
			copy(cache.CandidateBuf[(pos.Pos+uint32(candidatePosLen)):], buf[:])
			cache.Candidates[candidate.Address] = pos
			cache.Cur = cache.Cur + cache.ItemMaxSize
		}
	}

	return nil
}

func (cache *CandidateCache) IsExist(address common.Address) bool {
	_, ok := cache.Candidates[address]
	if !ok {
		return false
	} else {
		return true
	}
}

func (cache *CandidateCache) Encode() ([]byte, int) {
	return cache.CandidateBuf, cache.Cur
}

func (cache *CandidateCache) Decode(buf []byte, length int) error {
	if len(buf) < length {
		panic("decode candidate: len(buf) litter than len")
	}

	candidatePosLen := binary.Size(CandidatePos{})
	count := length / cache.ItemMaxSize
	for index := 0; index < count; index++ {
		start := index * cache.ItemMaxSize
		var pos CandidatePos
		err := binary.Read(bytes.NewBuffer(buf[start:start+candidatePosLen]), binary.LittleEndian, &pos)
		if err != nil {
			return err
		}

		if start != int(pos.Pos) || pos.Len == 0 {
			panic("start != pos.")
		}

		var candidate Candidate
		err = rlp.DecodeBytes(buf[start+candidatePosLen:start+candidatePosLen+int(pos.Len)], &candidate)
		if err != nil {
			return err
		}

		cache.Candidates[candidate.Address] = pos
	}

	cache.CandidateBuf = buf
	cache.Cap = len(buf)
	cache.Cur = cache.Cap
	return nil
}

func (cache *CandidateCache) GetCandidates() ([]*Candidate, error) {
	if len(cache.Candidates) <= 0 {
		return make([]*Candidate, 0), nil
	}

	candidatePosLen := binary.Size(CandidatePos{})
	result := make([]*Candidate, 0, len(cache.Candidates))
	for _, v := range cache.Candidates {
		var pos CandidatePos
		err := binary.Read(bytes.NewBuffer(cache.CandidateBuf[int(v.Pos):int(v.Pos)+candidatePosLen]), binary.LittleEndian, &pos)
		if err != nil {
			return nil, err
		}

		var candidate Candidate
		err = rlp.DecodeBytes(cache.CandidateBuf[int(v.Pos)+candidatePosLen:int(v.Pos)+candidatePosLen+int(v.Len)], &candidate)
		if err != nil {
			return nil, err
		}

		result = append(result, &candidate)
	}
	return result, nil
}

func (cache *CandidateCache) GetCandidatePage(index int, size int) ([]common.Address, uint32, error) {
	total := len(cache.Candidates)
	if index > total {
		return make([]common.Address, 0), uint32(total), nil
	}

	result := make([]common.Address, 0, size)
	start := index
	candidatePosLen := binary.Size(CandidatePos{})
	for ; (index < start+size) && (index < total); index++ {
		start := index * cache.ItemMaxSize
		var pos CandidatePos
		err := binary.Read(bytes.NewBuffer(cache.CandidateBuf[start:start+candidatePosLen]), binary.LittleEndian, &pos)
		if err != nil {
			return nil, uint32(total), err
		}

		var candidate Candidate
		err = rlp.DecodeBytes(cache.CandidateBuf[start+candidatePosLen:start+candidatePosLen+int(pos.Len)], &candidate)
		if err != nil {
			return nil, uint32(total), err
		}

		result = append(result, candidate.Address)
	}
	return result, uint32(total), nil
}

type RunContext struct {
	Path       string
	Candidates *CandidateCache
}

func NewRunContext(path string) *RunContext {
	path = filepath.Join(path, "/context.data")
	context := &RunContext{
		Path:       path,
		Candidates: NewCandidateCache(),
	}

	err := context.Load()
	if err != nil {
		panic("load run context error : " + err.Error())
	}

	return context
}

func (context *RunContext) load() error {
	file, err := os.OpenFile(context.Path, os.O_RDONLY, 0666)
	defer file.Close()
	if err != nil {
		return err
	}

	headBuf := make([]byte, binary.Size(contextHead{}))
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = file.Read(headBuf)
	if err != nil {
		return err
	}

	var head contextHead
	err = binary.Read(bytes.NewBuffer(headBuf), binary.LittleEndian, &head)
	if err != nil {
		return err
	}

	bodyBuf := make([]byte, head.FileLen)
	_, err = file.Read(bodyBuf)
	if err != nil {
		return err
	}

	offset := 0
	itemHeadLen := binary.Size(contextItemHead{})
	for {
		if offset >= len(bodyBuf) {
			break
		}

		var itemHead contextItemHead
		err = binary.Read(bytes.NewBuffer(bodyBuf[offset:offset+itemHeadLen]), binary.LittleEndian, &itemHead)
		if err != nil {
			return err
		}

		if itemHead.Flg == 1 { // stable block
			// nil
		}

		if itemHead.Flg == 2 { // addresses
			if itemHead.Len == 0 {
				// make(map[common.address]bool)
			} else {
				curPos := offset + itemHeadLen
				context.Candidates.Decode(bodyBuf[curPos:(curPos+int(itemHead.Len))], int(itemHead.Len))
			}
		}

		offset = offset + itemHeadLen + int(itemHead.Len)
	}

	return nil
}

func (context *RunContext) createFile() error {
	f, err := os.Create(context.Path)
	defer f.Close()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (context *RunContext) Load() error {
	isExist, err := IsExist(context.Path)
	if err != nil {
		return err
	}

	if !isExist {
		err = context.createFile()
		if err != nil {
			return err
		} else {
			return context.Flush()
		}
	} else {
		err := context.load()
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (context *RunContext) SetCandidate(candidate *Candidate) error {
	return context.Candidates.Set(candidate)
}

func (context *RunContext) SetCandidates(candidates []*Candidate) error {
	for index := 0; index < len(candidates); index++ {
		candidate := &Candidate{
			Address: candidates[index].Address,
			Total:   new(big.Int).Set(candidates[index].Total),
		}

		err := context.SetCandidate(candidate)
		if err != nil {
			return err
		}
	}
	return nil
}

func (context *RunContext) GetCandidates() ([]*Candidate, error) {
	return context.Candidates.GetCandidates()
}

func (context *RunContext) GetCandidatePage(index int, size int) ([]common.Address, uint32, error) {
	return context.Candidates.GetCandidatePage(index, size)
}

func (context *RunContext) CandidateIsExist(address common.Address) bool {
	return context.Candidates.IsExist(address)
}

func (context *RunContext) encodeHead(fileLen uint32) ([]byte, error) {
	head := contextHead{
		FileLen:   fileLen,
		Version:   1,
		TimeStamp: uint32(time.Now().Unix()),
		Crc:       0,
	}

	buf := make([]byte, binary.Size(contextHead{}))
	err := binary.Write(NewLmBuffer(buf[:]), binary.LittleEndian, &head)
	if err != nil {
		return nil, err
	} else {
		return buf, nil
	}
}

func (context *RunContext) encodeBody() ([]byte, error) {
	candidatesBuf, bufLen := context.Candidates.Encode()
	candidatesItemHead := contextItemHead{
		Flg: 2,
		Len: uint32(bufLen),
	}

	totalLen := binary.Size(candidatesItemHead) + int(candidatesItemHead.Len)
	totalBuf := make([]byte, totalLen)

	// candidate
	err := binary.Write(NewLmBuffer(totalBuf[0:]), binary.LittleEndian, &candidatesItemHead)
	if err != nil {
		return nil, err
	}

	if candidatesItemHead.Len > 0 {
		copy(totalBuf[binary.Size(candidatesItemHead):], candidatesBuf[:])
	}

	return totalBuf, nil
}

func (context *RunContext) flush(headBuf, bodyBuf []byte) error {
	file, err := os.OpenFile(context.Path, os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	n, err := file.Write(headBuf)
	if err != nil {
		return err
	}

	if n != len(headBuf) {
		panic("n != len(head data)")
	}

	_, err = file.Seek(int64(len(headBuf)), 0)
	if err != nil {
		return err
	}

	n, err = file.Write(bodyBuf)
	if err != nil {
		return err
	}

	if n != len(bodyBuf) {
		panic("n != len(body data)")
	}

	return file.Sync()
}

func (context *RunContext) Flush() error {
	bodyBuf, err := context.encodeBody()
	if err != nil {
		return err
	}

	headBuf, err := context.encodeHead(uint32(len(bodyBuf)))
	if err != nil {
		return err
	}

	return context.flush(headBuf, bodyBuf)
}
