package store

import (
	"bytes"
	"encoding/binary"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"io"
	"os"
	"time"
)

func FileUtilsIsExist(path string) (bool, error) {
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

func FileUtilsCreateFile(path string) error {
	f, err := os.Create(path)
	defer f.Close()
	return err
}

func FileUtilsAlign(length uint32) uint32 {
	if length%256 != 0 {
		length += 256 - length%256
	}

	return length
}

func fileUtilsEncodeHead(flag uint32, data []byte) ([]byte, error) {
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

func fileUtilsEncodeBody(key []byte, val []byte) ([]byte, error) {
	return rlp.EncodeToBytes(&RecordBody{
		Key: key,
		Val: val,
	})
}

func fileUtilsMergeHB(hBuf []byte, bBuf []byte) []byte {
	tLen := FileUtilsAlign(uint32(len(hBuf)) + uint32(len(bBuf)))

	tBuf := make([]byte, tLen)
	copy(tBuf[0:], hBuf[:])
	copy(tBuf[len(hBuf):], bBuf[:])

	return tBuf
}

func FileUtilsEncode(flag uint32, key []byte, val []byte) ([]byte, error) {
	body, err := fileUtilsEncodeBody(key, val)
	if err != nil {
		return nil, err
	}

	head, err := fileUtilsEncodeHead(flag, body)
	if err != nil {
		return nil, err
	}

	return fileUtilsMergeHB(head, body), nil
}

func FileUtilsFlush(path string, offset int64, data []byte) (int64, error) {
	file, err := os.OpenFile(path, os.O_APPEND, os.ModePerm)
	defer file.Close()
	if err != nil {
		return -1, err
	}

	_, err = file.Seek(offset, 0)
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

	return int64(n), nil
}

func FileUtilsRead(file *os.File, offset int64) (*RecordHead, *RecordBody, error) {
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
