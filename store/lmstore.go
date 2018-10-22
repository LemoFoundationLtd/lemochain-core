package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	contextCurrentBlock = iota
	contextLen
)

type ContextFileHeader struct {
	Magic     uint8
	TCnt      uint8
	TLen      uint32
	TimeStamp uint64
	Crc       uint16
}

type RecordHeader struct {
	Flg       uint8
	Num       uint32
	KLen      uint8
	VLen      uint32
	TimeStamp uint64
	Crc       uint16
}

type LmDataBase struct {
	rw        sync.RWMutex
	HomePath  string
	CurIndex  int
	CurOffset int64
	Tree      *Tree
	Buf       *bytes.Buffer
	Context   [][]byte
}

func key2hash(key []byte) common.Hash {
	return crypto.Keccak256Hash(key)
}

func (database *LmDataBase) getDataPath(index int) string {
	dataPath := filepath.Join(database.HomePath, "%03d.data")
	return fmt.Sprintf(dataPath, index)
}

func (database *LmDataBase) getHintPath(index int) string {
	hintPath := filepath.Join(database.HomePath, "%03d.hint")
	return fmt.Sprintf(hintPath, index)
}

func (database *LmDataBase) fileIsExist(path string) (bool, error) {
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

func (database *LmDataBase) createFile(index int) error {
	dataPath := database.getDataPath(index)
	hintPath := database.getHintPath(index)

	_, err := os.Create(dataPath)
	if err != nil {
		return err
	}

	_, err = os.Create(hintPath)
	if err != nil {
		return err
	}

	return nil
}

func (database *LmDataBase) ScanDataFiles() error {
	for index := 0; index < int(maxBucketsCount); index++ {
		dataPath := database.getDataPath(index)
		hintPath := database.getHintPath(index)

		isExist, err := database.fileIsExist(hintPath)
		if err != nil {
			return err
		}

		if !isExist {
			return nil
		}

		err = ScanDataFile(index, dataPath, hintPath)
		if err != nil {
			return err
		}

		database.CurIndex = index
	}

	return nil
}

func (database *LmDataBase) initOffset() error {
	if database.CurIndex == -1 {
		database.CurIndex = database.CurIndex + 1
		database.CurOffset = 0
		return database.createFile(database.CurIndex)
	} else {
		info, err := os.Stat(database.getDataPath(database.CurIndex))
		if err != nil {
			return err
		} else {
			database.CurOffset = info.Size()
			return nil
		}
	}
}

func (database *LmDataBase) loadHintFile(mFile *MFile) error {
	offset := int64(0)
	hintItem := &HintItem{}
	for {
		data, err := mFile.Read(offset, int64(binary.Size(HintItem{})))
		if err != nil {
			if err == ErrEOF {
				return nil
			} else {
				return err
			}
		}

		err = binary.Read(NewLmBuffer(data[0:]), binary.LittleEndian, hintItem)
		if err != nil {
			return err
		}

		key, err := mFile.Read(offset+int64(binary.Size(HintItem{})), int64(keySize))
		if err != nil {
			return err
		}

		err = database.Tree.Add(key, hintItem.Pos)
		if err != nil {
			return err
		}

		offset += int64(binary.Size(HintItem{}))
		offset += int64(keySize)
	}
}

func (database *LmDataBase) loadHintFiles() error {
	for index := 0; index < int(maxBucketsCount); index++ {
		hintPath := database.getHintPath(index)

		isExist, err := database.fileIsExist(hintPath)
		if err != nil {
			return err
		}

		if !isExist {
			return nil
		}

		mFile, err := OpenMFileForRead(hintPath)
		if err != nil {
			return err
		}

		err = database.loadHintFile(mFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewLmDataBase(homePath string) (*LmDataBase, error) {
	_, err := os.Stat(homePath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(homePath, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	database := &LmDataBase{HomePath: homePath, CurIndex: -1}
	buf := make([]byte, 1024*1024*1)
	database.Buf = bytes.NewBuffer(buf)

	database.Context = make([][]byte, contextLen)
	database.loadCurrentBlock(database.HomePath)

	tree, err := NewTree()
	if err != nil {
		return nil, err
	} else {
		database.Tree = tree
	}

	err = database.ScanDataFiles()
	if err != nil {
		return nil, err
	}

	err = database.loadHintFiles()
	if err != nil {
		return nil, err
	}

	err = database.initOffset()
	if err != nil {
		return nil, err
	}

	return database, nil
}

func (database *LmDataBase) loadCurrentBlock(homePath string) error {
	dataPath := filepath.Join(database.HomePath, "data.context")
	isExit, err := database.fileIsExist(dataPath)
	if err != nil {
		return err
	}

	if !isExit {
		_, err := os.Create(dataPath)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(dataPath, os.O_RDONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}

	contextFileHeaderLen := binary.Size(ContextFileHeader{})
	if info.Size() <= int64(contextFileHeaderLen) {
		return nil
	}

	headerBuf := make([]byte, contextFileHeaderLen)
	_, err = file.Read(headerBuf)
	if err != nil {
		return err
	}

	var contextFileHeader ContextFileHeader
	err = binary.Read(bytes.NewBuffer(headerBuf[:]), binary.LittleEndian, &contextFileHeader)
	if err != nil {
		return err
	}

	contextFileDataLen := contextFileHeader.TLen
	dataBuf := make([]byte, contextFileDataLen)
	_, err = file.Seek(int64(contextFileHeaderLen), 0)
	if err != nil {
		return err
	}

	_, err = file.Read(dataBuf)
	if err != nil {
		return err
	}

	return database.initContext(contextFileHeader, dataBuf)
}

func (database *LmDataBase) initContext(contextFileHeader ContextFileHeader, buf []byte) error {
	totalLen := len(buf)
	headerStart := 0
	for {
		if headerStart >= totalLen {
			return nil
		}

		var dHeader RecordHeader
		err := binary.Read(bytes.NewBuffer(buf[headerStart:headerStart+int(dataHeaderLen)]), binary.LittleEndian, &dHeader)
		if err != nil {
			return err
		}

		if dHeader.Flg&0x01 == 1 {
			continue
		}

		keyStart := headerStart + int(dataHeaderLen)
		valStart := keyStart + int(dHeader.KLen)
		val := buf[valStart : valStart+int(dHeader.VLen)]

		database.Context[int(dHeader.Flg)] = val

		headerStart = valStart + int(dHeader.VLen)
	}
}

func (database *LmDataBase) SetCurrentBlock(block []byte) error {
	database.Context[contextCurrentBlock] = block

	buf := make([]byte, 1024*1024)
	totalLen := 0
	totalCnt := 0
	for index := 0; index < contextLen; index++ {
		rHeader := &RecordHeader{
			Flg:       uint8(index),
			KLen:      0,
			VLen:      uint32(len(database.Context[index])),
			Num:       0,
			TimeStamp: uint64(time.Now().UnixNano()),
			Crc:       CheckSum(database.Context[index]),
		}

		err := binary.Write(NewLmBuffer(buf[totalLen:totalLen+int(dataHeaderLen)]), binary.LittleEndian, rHeader)
		if err != nil {
			return err
		}
		totalLen = totalLen + int(dataHeaderLen)
		totalLen = totalLen + int(rHeader.KLen)
		err = binary.Write(NewLmBuffer(buf[totalLen:totalLen+int(rHeader.VLen)]), binary.LittleEndian, database.Context[index])
		if err != nil {
			return err
		}
		totalLen = totalLen + int(rHeader.VLen)
		totalCnt = totalCnt + 1
	}

	dataPath := filepath.Join(database.HomePath, "data.context")
	isExit, err := database.fileIsExist(dataPath)
	if err != nil {
		return err
	}

	if !isExit {
		_, err := os.Create(dataPath)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(dataPath, os.O_RDWR, os.ModePerm)
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	contextFileHeader := &ContextFileHeader{
		Magic:     99,
		TCnt:      uint8(totalCnt),
		TLen:      uint32(totalLen),
		TimeStamp: 0,
		Crc:       0,
	}

	headerBuf := make([]byte, binary.Size(ContextFileHeader{}))
	err = binary.Write(NewLmBuffer(headerBuf[:]), binary.LittleEndian, contextFileHeader)
	if err != nil {
		return err
	}

	_, err = file.Write(headerBuf[:])
	if err != nil {
		return err
	}

	_, err = file.Write(buf[0:totalLen])
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (database *LmDataBase) isExist(key []byte) (bool, error) {
	item, err := database.Tree.Get(key)
	if err != nil {
		return false, err
	}

	if item == nil {
		return false, nil
	}

	return true, nil
}

func (database *LmDataBase) Close() error {
	return nil
}

func (database *LmDataBase) Has(key []byte) (bool, error) {
	database.rw.RLock()
	defer database.rw.RUnlock()

	val, err := database.Get(key)
	if err != nil {
		if err == ErrNotExist {
			return false, nil
		} else {
			return false, err
		}
	}

	if val == nil {
		return false, nil
	} else {
		return true, nil
	}
}

func (database *LmDataBase) Get(key []byte) ([]byte, error) {
	database.rw.RLock()
	defer database.rw.RUnlock()

	key = key2hash(key).Bytes()
	item, err := database.Tree.Get(key)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, ErrNotExist
	}

	pos := item.Pos & 0xffffff00
	bucketIndex := item.Pos & 0xff

	dataPath := database.getDataPath(int(bucketIndex))
	file, err := os.OpenFile(dataPath, os.O_RDONLY, 0666)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	_, err = file.Seek(int64(pos), 0)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, dataHeaderLen)
	_, err = file.Read(buf)
	if err != nil {
		return nil, err
	}

	var dHeader RecordHeader
	err = binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &dHeader)
	if err != nil {
		return nil, err
	}

	if dHeader.Flg&0x01 == 1 {
		return nil, nil
	}

	tBuf := make([]byte, uint32(dHeader.KLen)+dHeader.VLen)
	_, err = file.Read(tBuf)
	if err != nil {
		return nil, err
	}

	return tBuf[dHeader.KLen:], nil
}

func (database *LmDataBase) align(tLen uint32) uint32 {
	if tLen%256 != 0 {
		tLen += 256 - tLen%256
	}

	return tLen
}

func (database *LmDataBase) encode(dataHeader []byte, key []byte, val []byte) []byte {
	tLen := database.align(dataHeaderLen + uint32(len(key)) + uint32(len(val)))

	pBuf := make([]byte, tLen)
	copy(pBuf, dataHeader)
	copy(pBuf[dataHeaderLen:], key)
	copy(pBuf[dataHeaderLen+uint32(len(key)):], val)

	return pBuf
}

func (database *LmDataBase) Delete(key []byte) error {
	database.rw.RLock()
	defer database.rw.RUnlock()
	item, err := database.Tree.Get(key)
	if err != nil {
		return err
	}

	if item == nil {
		return nil
	}

	pos := item.Pos & 0xffffff00
	bucketIndex := item.Pos & 0xff

	dataPath := database.getDataPath(int(bucketIndex))
	file, err := os.OpenFile(dataPath, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.Seek(int64(pos), 0)
	if err != nil {
		return err
	}

	buf := make([]byte, dataHeaderLen)
	_, err = file.Read(buf)
	if err != nil {
		return err
	}

	var dHeader RecordHeader
	err = binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &dHeader)
	if err != nil {
		return err
	}

	dHeader.Flg = dHeader.Flg | 0x1
	_, err = file.Seek(int64(pos), 0)
	if err != nil {
		return err
	}

	database.Buf.Reset()
	err = binary.Write(database.Buf, binary.LittleEndian, dHeader)
	if err != nil {
		return err
	}

	_, err = file.Write(database.Buf.Bytes())
	if err != nil {
		return err
	} else {
		err = file.Sync()
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (database *LmDataBase) Put(key []byte, val []byte) error {
	database.rw.Lock()
	defer database.rw.Unlock()

	key = key2hash(key).Bytes()

	rHeader := &RecordHeader{
		KLen:      uint8(len(key)),
		VLen:      uint32(len(val)),
		Num:       Byte2Uint32(key),
		TimeStamp: uint64(time.Now().UnixNano()),
		Crc:       CheckSum(val),
	}

	database.Buf.Reset()
	err := binary.Write(database.Buf, binary.LittleEndian, rHeader)
	if err != nil {
		return err
	}

	pBuf := database.encode(database.Buf.Bytes(), key, val)

	if database.CurOffset+int64(len(pBuf)) > int64(maxFileSize) {
		database.CurIndex = database.CurIndex + 1
		err := database.createFile(database.CurIndex)
		if err != nil {
			return err
		} else {
			database.CurOffset = 0

			database.Buf.Reset()
			rHeader.TimeStamp = rHeader.TimeStamp + 1000000000
			err := binary.Write(database.Buf, binary.LittleEndian, rHeader)
			if err != nil {
				return err
			}

			pBuf = database.encode(database.Buf.Bytes(), key, val)
		}
	}

	file, err := os.OpenFile(database.getDataPath(database.CurIndex), os.O_APPEND|os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.Seek(int64(database.CurOffset), 0)
	if err != nil {
		return err
	}

	n, err := file.Write(pBuf)
	if err != nil || n < len(pBuf) {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}

	pos := int(database.CurOffset) | database.CurIndex
	err = database.Tree.Add(key, uint32(pos))
	if err != nil {
		return err
	} else {
		database.CurOffset += int64(n)
	}

	return nil
}

func (database *LmDataBase) CurrentBlock() []byte {
	return database.Context[contextCurrentBlock]
}

type BatchItem struct {
	Key []byte
	Val []byte
}

func (database *LmDataBase) addTree(offsets []uint32, keys [][]byte) error {
	for index := 0; index < len(keys); index++ {
		if keys[index] == nil {
			break
		}

		offset := database.CurOffset + int64(offsets[index])
		pos := int(offset) | database.CurIndex
		err := database.Tree.Add(keys[index], uint32(pos))
		if err != nil {
			return err
		}
	}

	return nil
}

func (database *LmDataBase) Commit(items []*BatchItem) error {
	database.rw.Lock()
	defer database.rw.Unlock()
	if len(items) <= 0 {
		return nil
	}

	file, err := os.OpenFile(database.getDataPath(database.CurIndex), os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	var dHeader RecordHeader
	var wOffset uint32 = 0

	bufLen := 256 * 1024 * 1024
	buf := make([]byte, 0, bufLen)

	wIndex := 0
	sOffset := make([]uint32, len(items))
	keys := make([][]byte, len(items))
	for index := 0; index < len(items); index++ {
		if items[index] == nil {
			break
		}

		keys[index] = key2hash(items[index].Key).Bytes()
		dHeader.KLen = uint8(len(keys[index]))
		dHeader.VLen = uint32(len(items[index].Val))
		dHeader.TimeStamp = uint64(time.Now().UnixNano())
		dHeader.Crc = CheckSum(items[index].Val)
		database.Buf.Reset()
		err := binary.Write(database.Buf, binary.LittleEndian, &dHeader)
		if err != nil {
			return err
		}

		tLen := database.align(dataHeaderLen + uint32(len(keys[index])) + uint32(len(items[index].Val)))
		if (uint32(bufLen) - uint32(wOffset)) >= uint32(tLen) {
			sOffset[index] = wOffset

			pBuf := database.encode(database.Buf.Bytes(), keys[index], items[index].Val)
			err = binary.Write(NewLmBuffer(buf[wOffset:wOffset+tLen]), binary.LittleEndian, pBuf)
			if err != nil {
				return err
			}

			wOffset = wOffset + tLen
		} else {
			_, err = file.Seek(int64(database.CurOffset), 0)
			if err != nil {
				return err
			}

			n, err := file.Write(buf[0:wOffset])
			if err != nil || n < len(buf) {
				return err
			}

			err = file.Sync()
			if err != nil {
				return err
			}

			err = database.addTree(sOffset[wIndex:index], keys[wIndex:index])
			if err != nil {
				return err
			}

			database.CurOffset = database.CurOffset + int64(wOffset)

			wOffset = 0
			sOffset[index] = wOffset

			pBuf := database.encode(database.Buf.Bytes(), keys[index], items[index].Val)
			err = binary.Write(NewLmBuffer(buf[wOffset:wOffset+tLen]), binary.LittleEndian, pBuf)
			if err != nil {
				return err
			}

			wIndex = index
			wOffset = wOffset + tLen
		}
	}

	if wOffset > 0 {
		_, err = file.Seek(int64(database.CurOffset), 0)
		if err != nil {
			return err
		}

		n, err := file.Write(buf[0:wOffset])
		if err != nil || n < len(buf) {
			return err
		}

		err = file.Sync()
		if err != nil {
			return err
		}

		err = database.addTree(sOffset[wIndex:], keys[wIndex:])
		if err != nil {
			return err
		}

		database.CurOffset = database.CurOffset + int64(wOffset)
	}

	return nil
}
