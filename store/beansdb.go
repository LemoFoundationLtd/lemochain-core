package store

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type BeansDB struct {
	Home    string
	LevelDB *leveldb.LevelDBDatabase
	Queue   *FileQueue
}

func NewBeansDB(home string, levelDB *leveldb.LevelDBDatabase) *BeansDB {
	return &BeansDB{
		Home:    home,
		LevelDB: levelDB,
	}
}

func (beansdb *BeansDB) Start() {
	beansdb.Queue = NewFileQueue(beansdb.Home, beansdb.LevelDB, beansdb)
	beansdb.Queue.Start()
}

func (beansdb *BeansDB) After(flg uint32, key []byte, val []byte) error {
	if flg == leveldb.ItemFlagBlock {
		log.Infof("after flag: ItemFlagBlock")
		return beansdb.afterBlock(key, val)
	} else if flg == leveldb.ItemFlagBlockHeight {
		log.Infof("after flag: ItemFlagBlockHeight")
		return nil
	} else if flg == leveldb.ItemFlagTrie {
		log.Infof("after flag: ItemFlagTrie")
		return nil
	} else if flg == leveldb.ItemFlagAct {
		log.Infof("after flag: ItemFlagAct")
		return nil
	} else if flg == leveldb.ItemFlagTxIndex {
		log.Infof("after flag: ItemFlagTxIndex")
		return nil
	} else if flg == leveldb.ItemFlagCode {
		log.Infof("after flag: ItemFlagCode")
		return nil
	} else if flg == leveldb.ItemFlagKV {
		log.Infof("after flag: ItemFlagKV")
		return nil
	} else if flg == leveldb.ItemFlagAssetCode {
		log.Infof("after flag: ItemFlagAssetCode")
		return nil
	} else if flg == leveldb.ItemFlagAssetId {
		log.Infof("after flag: ItemFlagAssetId")
		return nil
	} else {
		panic("after! unknown flag.flag = " + strconv.Itoa(int(flg)))
	}

	return nil
}

func (beansdb *BeansDB) afterBlock(key []byte, val []byte) error {
	var block types.Block
	err := rlp.DecodeBytes(val, &block)
	if err != nil {
		return err
	}

	txs := block.Txs
	if len(txs) <= 0 {
		return nil
	}

	for index := 0; index < len(txs); index++ {
		tx := txs[index]
		from, err := tx.From()
		if err != nil {
			return err
		}

		if tx.Type() == params.CreateAssetTx {
			err := UtilsSetAssetCode(beansdb, tx.Hash(), from)
			if err != nil {
				return err
			} else {
				continue
			}
		} else if tx.Type() == params.IssueAssetTx {
			extendData := tx.Data()
			if len(extendData) <= 0 {
				panic("tx is issue asset. but data is nil.")
			}

			issueAsset := &types.IssueAsset{}
			err := json.Unmarshal(extendData, issueAsset)
			if err != nil {
				return err
			}

			err = UtilsSetAssetId(beansdb, tx.Hash(), issueAsset.AssetCode)
			if err != nil {
				return err
			} else {
				continue
			}
		}
	}

	return nil
}

func (beansdb *BeansDB) NewBatch() Batch {
	return &LmDBBatch{
		db:    beansdb,
		items: make([]*BatchItem, 0),
		size:  0,
	}
}

func (beansdb *BeansDB) Commit(batch Batch) error {
	items := batch.Items()
	if len(items) <= 0 {
		return nil
	} else {
		return beansdb.Queue.PutBatch(items)
	}
}

func (beansdb *BeansDB) Put(flag uint32, key []byte, val []byte) error {
	if !leveldb.CheckItemFlag(flag) || len(key) <= 0 || len(val) <= 0 {
		return ErrArgInvalid
	}

	return beansdb.Queue.Put(flag, key, val)
}

func (beansdb *BeansDB) Has(flag uint32, key []byte) (bool, error) {
	if !leveldb.CheckItemFlag(flag) || len(key) <= 0 {
		return false, ErrArgInvalid
	}

	val, err := beansdb.Get(flag, key)
	if err != nil {
		return false, err
	}

	if val == nil {
		return false, nil
	} else {
		return true, nil
	}
}

func (beansdb *BeansDB) Get(flg uint32, key []byte) ([]byte, error) {
	if !leveldb.CheckItemFlag(flg) || len(key) <= 0 {
		return nil, ErrArgInvalid
	} else {
		return beansdb.Queue.Get(flg, key)
	}
}

func (beansdb *BeansDB) Delete(flg uint32, key []byte) error {
	panic("implement me")
}

func (beansdb *BeansDB) Close() {
	beansdb.Queue.Close()
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
	isExist, err := FileUtilsIsExist(context.Path)
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
