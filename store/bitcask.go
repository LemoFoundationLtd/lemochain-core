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

type RecordHead struct {
	Flg       uint32
	Len       uint32
	TimeStamp uint64
	Crc       uint16
}

type RecordBody struct {
	Key []byte
	Val []byte
}

type Inject struct {
	Flg uint32
	Key []byte
	Val []byte
}

var RecordHeadLength = binary.Size(RecordHead{})

type BitCask struct {
	RW   sync.RWMutex
	Home string

	BitCaskIndex int
	CurIndex     int
	CurOffset    int64
	LevelDB      *leveldb.LevelDBDatabase

	InjectsChan chan *Inject
	Quit        chan struct{}
}

func (bitcask *BitCask) path(index int) string {
	dataPath := filepath.Join(bitcask.Home, "%03d.data")
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
	pos, err := leveldb.GetCurrentPos(bitcask.LevelDB, bitcask.BitCaskIndex)
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

func NewBitCask(home string, index int, levelDB *leveldb.LevelDBDatabase, quit chan struct{}) (*BitCask, error) {
	db := &BitCask{
		Home:         home,
		LevelDB:      levelDB,
		InjectsChan:  make(chan *Inject, 1024),
		Quit:         quit,
		BitCaskIndex: index,
	}

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

func (bitcask *BitCask) Start(Done chan *Inject, Err chan *Inject) {
	for {
		select {
		case <-bitcask.Quit:
			return
		case op := <-bitcask.InjectsChan:
			err := bitcask.put(op.Flg, op.Key, op.Val)
			if err != nil {
				Err <- op
			} else {
				Done <- op
			}
			continue
		}
	}
}

func (bitcask *BitCask) Put(flag uint32, key []byte, val []byte) {
	op := &Inject{
		Flg: flag,
		Key: key,
		Val: val,
	}

	select {
	case <-bitcask.Quit:
		return
	case bitcask.InjectsChan <- op:
		return
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

func (bitcask *BitCask) encodeHead(flag uint32, data []byte) ([]byte, error) {
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

func (bitcask *BitCask) encodeBody(key []byte, val []byte) ([]byte, error) {
	body := &RecordBody{
		Key: key,
		Val: val,
	}

	return rlp.EncodeToBytes(body)
}

func (bitcask *BitCask) checkAndFlush(data []byte) (int64, error) {
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
	file, err := os.OpenFile(bitcask.path(bitcask.CurIndex), os.O_APPEND, os.ModePerm)
	defer file.Close()
	if err != nil {
		return -1, err
	}

	_, err = file.Seek(bitcask.CurOffset, 0)
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

func (bitcask *BitCask) put(flag uint32, key []byte, val []byte) error {
	bitcask.RW.Lock()
	defer bitcask.RW.Unlock()

	if common.ToHex(key) == "0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed" {
		log.Errorf("put: %s", common.ToHex(key))
	}

	body, err := bitcask.encodeBody(key, val)
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

	curOffset := uint32(int(offset) | bitcask.CurIndex)
	err = leveldb.SetPos(bitcask.LevelDB, flag, key, &leveldb.Position{
		Flag:   flag,
		Offset: curOffset,
	})

	if err != nil {
		return err
	}

	return leveldb.SetCurrentPos(bitcask.LevelDB, bitcask.BitCaskIndex, curOffset)
}

func (bitcask *BitCask) read(file *os.File, offset int64) (*RecordHead, *RecordBody, error) {
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

func (bitcask *BitCask) get(flag uint32, key []byte, offset uint32) ([]byte, error) {
	pos := offset & 0xffffff00
	bucketIndex := offset & 0xff
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
		log.Errorf("get file record pos.Flag(%d) != flag(%d)", head, flag)
		return nil, nil
	}

	if bytes.Compare(body.Key, key) != 0 {
		log.Errorf("get file record body.key(%s) != key(%s)", common.ToHex(body.Key), common.ToHex(key))
		return nil, nil
	}

	return body.Val, nil
}

func (bitcask *BitCask) Get(flag uint32, key []byte) ([]byte, error) {
	bitcask.RW.RLock()
	defer bitcask.RW.RUnlock()

	pos, err := leveldb.GetPos(bitcask.LevelDB, flag, key)
	if err != nil {
		return nil, err
	}

	if pos == nil {
		return nil, nil
	}

	if pos.Flag != flag {
		log.Errorf("get db record pos.Flag(%d) != flag(%d)", pos.Flag, flag)
		return nil, nil
	}

	return bitcask.get(flag, key, pos.Offset)
}
