package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"os"
	"path/filepath"
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

type LastIndex struct {
	Index  uint32
	Offset uint32
}

type BitcaskIndexes [256]LastIndex

func (bitcaskIndexes *BitcaskIndexes) load(home string) error {
	path := filepath.Join(home, "/hit.index")
	isExist, err := IsExist(path)
	if err != nil {
		return err
	}

	if !isExist {
		return nil
	}

	file, err := os.OpenFile(path, os.O_RDONLY, 0666)
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

	return rlp.DecodeBytes(bodyBuf, bitcaskIndexes)
}

func (bitcaskIndexes *BitcaskIndexes) flush(home string) error {
	path := filepath.Join(home, "/hit.index")
	isExist, err := IsExist(path)
	if err != nil {
		return err
	}

	if !isExist {
		err = CreateFile(path)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return err
	}

	bodyBuf, err := rlp.EncodeToBytes(bitcaskIndexes)
	if err != nil {
		return err
	}

	head := contextHead{
		FileLen:   uint32(len(bodyBuf)),
		Version:   1,
		TimeStamp: uint32(time.Now().Unix()),
		Crc:       0,
	}

	headBuf := make([]byte, binary.Size(contextHead{}))
	err = binary.Write(NewLmBuffer(headBuf[:]), binary.LittleEndian, &head)
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
		panic("n != len(headBuf)")
	}

	n, err = file.Write(bodyBuf)
	if err != nil {
		return err
	}

	if n != len(bodyBuf) {
		panic("n != len(bodyBuf)")
	}

	return file.Sync()
}

type BeansDB struct {
	height    uint
	bitcasks  []*BitCask
	indexDB   DB
	route2key map[string][]byte
	scanIndex BitcaskIndexes

	after BizAfterScan
}

type BizAfterScan func(flag uint, key []byte, val []byte) error

func NewBeansDB(home string, height int, DB *MySqlDB, after BizAfterScan) *BeansDB {
	if height != 2 {
		panic("beansdb height != 2")
	}

	count := 1 << (uint(height) * 4)
	beansdb := &BeansDB{height: uint(height)}
	beansdb.bitcasks = make([]*BitCask, count)
	beansdb.route2key = make(map[string][]byte)
	beansdb.indexDB = DB
	beansdb.after = after

	err := beansdb.scanIndex.load(home)
	if err != nil {
		panic("load scan index err : " + err.Error())
	}

	for index := 0; index < count; index++ {
		dataPath := filepath.Join(home, "/%02d/%02d/")
		str := fmt.Sprintf(dataPath, index>>4, index&0xf)

		last := beansdb.scanIndex[index]
		database, err := NewBitCask(str, int(last.Index), last.Offset, beansdb.AfterScan, beansdb.indexDB)
		if err != nil {
			panic("new bitcask err : " + err.Error())
		} else {
			beansdb.bitcasks[index] = database
			beansdb.scanIndex[index].Index = uint32(database.CurIndex)
			beansdb.scanIndex[index].Offset = uint32(database.CurOffset)
		}
	}

	err = beansdb.scanIndex.flush(home)
	if err != nil {
		panic("flush context to disk err : " + err.Error())
	}

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
		delete(beansdb.route2key, string(key))
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
			beansdb.route2key[string(items[index].Key)] = route
		}
		return nil
	}
}

func (beansdb *BeansDB) Put(flg uint, route []byte, key []byte, val []byte) error {
	bitcask := beansdb.route(route)
	err := bitcask.Put(flg, route, key, val)
	if err != nil {
		return err
	} else {
		beansdb.route2key[string(key)] = route
		return nil
	}
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
	route, ok := beansdb.route2key[string(key)]
	if !ok {
		flg, route, offset, err := beansdb.indexDB.GetIndex(key)
		if err != nil {
			log.Errorf("get index from db err : " + err.Error())
			return nil, err
		}

		if route == nil {
			//log.Error("get index from db is not exist.")
			return nil, nil
		}

		// str := common.BytesToHash(route).Hex()
		// log.Error("str:" + str)

		bitcask := beansdb.route(route)
		val, err := bitcask.Get(uint(flg), route, key, offset)
		if err != nil {
			log.Error("get data from disk err : " + err.Error())
			return nil, err
		} else {
			return val, nil
		}
	} else {
		bitcask := beansdb.route(route)
		val, err := bitcask.Get4Cache(route, key)
		if err != nil {
			return nil, err
		} else {
			return val, nil
		}
	}
}

func (beansdb *BeansDB) Close() error {
	return nil
}

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
		Cap:          64 * 1024,
		Cur:          0,
		CandidateBuf: make([]byte, 64*1024),
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

		var candidate Candidate
		err = rlp.DecodeBytes(buf[start+candidatePosLen:pos.Len], &candidate)
		if err != nil {
			return err
		}

		cache.Candidates[candidate.Address] = pos
	}

	return nil
}

func (cache *CandidateCache) GetCandidates() ([]*Candidate, error) {
	if len(cache.Candidates) <= 0 {
		return make([]*Candidate, 0), nil
	}

	candidatePosLen := binary.Size(CandidatePos{})
	result := make([]*Candidate, 0, len(cache.Candidates))
	for k, v := range cache.Candidates {
		var candidate Candidate
		err := rlp.DecodeBytes(cache.CandidateBuf[int(v.Pos)+candidatePosLen:int(v.Pos)+candidatePosLen+int(v.Len)], &candidate)
		if err != nil {
			return nil, err
		}

		result = append(result, &Candidate{
			Address: k,
			Total:   candidate.Total,
		})
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
	Path        string
	StableBlock *types.Block
	Candidates  *CandidateCache
}

func NewRunContext(path string) *RunContext {
	path = filepath.Join(path, "/context.data")
	context := &RunContext{
		Path:        path,
		StableBlock: nil,
		Candidates:  NewCandidateCache(),
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
			if itemHead.Len == 0 {
				context.StableBlock = nil
			} else {
				var stableBlock types.Block
				err := rlp.DecodeBytes(bodyBuf[offset+itemHeadLen:offset+itemHeadLen+int(itemHead.Len)], &stableBlock)
				if err != nil {
					return err
				} else {
					context.StableBlock = &stableBlock
				}
			}
		}

		if itemHead.Flg == 2 { // addresses
			if itemHead.Len == 0 {
				//make(map[common.address]bool)
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

func (context *RunContext) GetStableBlock() *types.Block {
	return context.StableBlock
}

func (context *RunContext) SetStableBlock(block *types.Block) {
	context.StableBlock = block
}

func (context *RunContext) SetCandidate(candidate *Candidate) {
	context.Candidates.Set(candidate)
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
	stableItemHead := contextItemHead{Flg: 1, Len: 0}

	stableBlockBuf := []byte(nil)
	err := error(nil)
	if context.StableBlock != nil {
		stableBlockBuf, err = rlp.EncodeToBytes(context.StableBlock)
		if err != nil {
			return nil, err
		} else {
			stableItemHead.Len = uint32(len(stableBlockBuf))
		}
	}

	candidatesBuf, bufLen := context.Candidates.Encode()
	candidatesItemHead := contextItemHead{
		Flg: 2,
		Len: uint32(bufLen),
	}

	candidatesOffset := binary.Size(stableItemHead) + int(stableItemHead.Len) + binary.Size(candidatesItemHead)
	totalLen := candidatesOffset + int(candidatesItemHead.Len)
	totalBuf := make([]byte, totalLen)

	// stable block
	err = binary.Write(NewLmBuffer(totalBuf[0:]), binary.LittleEndian, &stableItemHead)
	if err != nil {
		return nil, err
	}

	if stableItemHead.Len > 0 {
		copy(totalBuf[binary.Size(stableItemHead):], stableBlockBuf[:])
	}

	// candidate
	err = binary.Write(NewLmBuffer(totalBuf[binary.Size(stableItemHead)+int(stableItemHead.Len):]), binary.LittleEndian, &candidatesItemHead)
	if err != nil {
		return nil, err
	}

	if candidatesItemHead.Len > 0 {
		copy(totalBuf[candidatesOffset:], candidatesBuf[:])
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
