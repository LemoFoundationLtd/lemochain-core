package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

type BeansDB struct {
	height    uint
	bitcasks  []*BitCask
	indexDB   DB
	route2key map[string][]byte
	context   *ChainContext
}

func after(flag uint, route []byte, key []byte, val []byte, offset uint32) error {
	return nil
}

func NewBeansDB(home string, height int) *BeansDB {
	if height != 2 {
		panic("height != 2")
	}

	beansdb := &BeansDB{height: uint(height)}

	count := 1 << (beansdb.height * 4)
	beansdb.bitcasks = make([]*BitCask, count)
	beansdb.route2key = make(map[string][]byte)

	dns := "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"
	beansdb.indexDB = NewMySqlDB(DRIVER_MYSQL, dns)

	beansdb.context = new(ChainContext)
	isExist, err := beansdb.context.isExist(home)
	if err != nil {
		panic("check dir is exist err : " + err.Error())
	}

	if !isExist {
		err = os.MkdirAll(home, os.ModePerm)
		if err != nil {
			panic("mk dir is exist err : " + err.Error())
		}
	}

	beansdb.context.Path = filepath.Join(home, "context.data")
	err = beansdb.context.Load()
	if err != nil {
		panic("load context err : " + err.Error())
	}

	for index := 0; index < count; index++ {
		dataPath := filepath.Join(home, "/%02d/%02d/")
		str := fmt.Sprintf(dataPath, index>>4, index&0xf)

		last := beansdb.context.GetScanIndex(index)
		database, err := NewBitCask(str, int(last.Index), last.Offset, nil, beansdb.indexDB)
		if err != nil {
			panic("new bitcask error : " + err.Error())
		}

		beansdb.bitcasks[index] = database
	}

	return beansdb
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
			return nil, err
		}

		if route == nil {
			return nil, nil
		}

		bitcask := beansdb.route(route)
		val, err := bitcask.Get(uint(flg), route, key, offset)
		if err != nil {
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

func (beansdb *BeansDB) GetCurrentBlock() []byte {
	return beansdb.context.GetCurrentBlock()
}

func (beansdb *BeansDB) SetCurrentBlock(data []byte) error {
	beansdb.context.SetCurrentBlock(data)
	return beansdb.context.Flush()
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

type lastIndex struct {
	Index  uint32
	Offset uint32
}

type contextBody struct {
	ScanIndex    []lastIndex
	CurrentBlock []byte
}

type ChainContext struct {
	Path         string
	ScanIndex    []lastIndex
	CurrentBlock []byte
}

func (context *ChainContext) load() (*contextHead, *contextBody, error) {
	file, err := os.OpenFile(context.Path, os.O_RDONLY, 0666)
	defer file.Close()
	if err != nil {
		return nil, nil, err
	}

	headBuf := make([]byte, binary.Size(contextHead{}))
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, nil, err
	}

	_, err = file.Read(headBuf)
	if err != nil {
		return nil, nil, err
	}

	var head contextHead
	err = binary.Read(bytes.NewBuffer(headBuf), binary.LittleEndian, &head)
	if err != nil {
		return nil, nil, err
	}

	bodyBuf := make([]byte, head.FileLen)
	_, err = file.Read(bodyBuf)
	if err != nil {
		return nil, nil, err
	}

	var body contextBody
	err = rlp.DecodeBytes(bodyBuf, &body)
	if err != nil {
		return nil, nil, err
	} else {
		return &head, &body, nil
	}
}

func (context *ChainContext) isExist(path string) (bool, error) {
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

func (context *ChainContext) createFile() error {
	f, err := os.Create(context.Path)
	defer f.Close()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (context *ChainContext) Load() error {
	isExist, err := context.isExist(context.Path)
	if err != nil {
		return err
	}

	if !isExist {
		context.ScanIndex = make([]lastIndex, 256)
		for index := 0; index < 256; index++ {
			context.ScanIndex[index] = lastIndex{
				Index:  0,
				Offset: 0,
			}
		}
		err = context.createFile()
		if err != nil {
			return err
		} else {
			return context.Flush()
		}
	} else {
		_, body, err := context.load()
		if err != nil {
			return err
		}

		context.ScanIndex = body.ScanIndex
		context.CurrentBlock = body.CurrentBlock

		return nil
	}
}

func (context *ChainContext) GetScanIndex(index int) lastIndex {
	return context.ScanIndex[index]
}

func (context *ChainContext) SetScanIndex(index int, curIndex, curOffset uint32) {
	context.ScanIndex[index] = lastIndex{
		Index:  curIndex,
		Offset: curOffset,
	}
}

func (context *ChainContext) GetCurrentBlock() []byte {
	return context.CurrentBlock
}

func (context *ChainContext) SetCurrentBlock(val []byte) {
	context.CurrentBlock = val
}

func (context *ChainContext) encodeHead(fileLen uint32) ([]byte, error) {
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

func (context *ChainContext) encodeBody() ([]byte, error) {
	body := &contextBody{
		ScanIndex:    context.ScanIndex,
		CurrentBlock: context.CurrentBlock,
	}

	buf, err := rlp.EncodeToBytes(body)
	if err != nil {
		return nil, err
	} else {
		return buf, nil
	}
}

func (context *ChainContext) flush(buf []byte) error {
	file, err := os.OpenFile(context.Path, os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	n, err := file.Write(buf)
	if err != nil {
		return err
	}

	if n != len(buf) {
		panic("n != len(data)")
	}

	return file.Sync()
}

func (context *ChainContext) Flush() error {
	bodyBuf, err := context.encodeBody()
	if err != nil {
		return err
	}

	headBuf, err := context.encodeHead(uint32(len(bodyBuf)))
	if err != nil {
		return err
	}

	totalBuf := make([]byte, len(headBuf)+len(bodyBuf))
	copy(totalBuf[0:], headBuf[:])
	copy(totalBuf[len(headBuf):], bodyBuf[:])

	return context.flush(totalBuf)
}
