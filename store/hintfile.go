package store

import (
	"bytes"
	"encoding/binary"
	"os"
)

var M10 = int64(1024 * 1024 * 10)

type HintItem struct {
	Num uint32
	Pos uint32
}

type MFile struct {
	path  string
	size  int64
	start int64
	len   int64
	buf   []byte
}

func (mFile *MFile) Size() int64 {
	return mFile.size
}

func (mFile *MFile) Read(offset int64, size int64) ([]byte, error) {
	if (offset < 0) || (size > M10) {
		return nil, ErrArgInvalid
	}

	if (offset + size) > mFile.size {
		return nil, ErrEOF
	}

	result := make([]byte, size)
	if (offset >= mFile.start) && ((offset + size) <= (mFile.start + mFile.len)) {
		copy(result, mFile.buf[offset-mFile.start:offset+size-mFile.start])
	} else {
		file, err := os.OpenFile(mFile.path, os.O_RDONLY, os.ModePerm)
		defer file.Close()
		if err != nil {
			return nil, err
		}
		_, err = file.Seek(offset, os.SEEK_CUR)
		if err != nil {
			return nil, err
		}

		mFile.start = offset
		if mFile.size-offset > M10 {
			mFile.len = M10
			mFile.buf = make([]byte, mFile.len)
		} else {
			mFile.len = mFile.size - offset
			mFile.buf = make([]byte, mFile.len)
		}

		_, err = file.Read(mFile.buf)
		if err != nil {
			return nil, err
		}

		copy(result, mFile.buf[:size])
	}

	return result, nil
}

func OpenMFileForRead(path string) (*MFile, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	mFile := &MFile{
		path: path,
		size: info.Size(),
	}

	if mFile.size <= M10 {
		mFile.start = 0
		mFile.len = mFile.size
		mFile.buf = make([]byte, mFile.len)
	} else {
		mFile.start = 0
		mFile.len = M10
		mFile.buf = make([]byte, M10)
	}

	_, err = file.Read(mFile.buf)
	if err != nil {
		return nil, err
	}

	return mFile, nil
}

func OpenMFileForWrite(path string) (*MFile, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		if os.IsNotExist(err) {
			file, err = os.Create(path)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &MFile{
		path: path,
		size: 0,

		start: 0,
		len:   0,
		buf:   make([]byte, M10),
	}, nil
}

func (mFile *MFile) Write(data []byte) error {
	if int64(mFile.len)+int64(len(data)) >= M10 {
		err := mFile.Flush()
		if err != nil {
			return err
		}
	}

	copy(mFile.buf[mFile.len:], data)
	mFile.len += int64(len(data))
	return nil
}

func (mFile *MFile) Flush() error {
	file, err := os.OpenFile(mFile.path, os.O_APPEND|os.O_WRONLY, os.ModePerm)
	defer file.Close()

	if err != nil {
		return err
	}

	if mFile.len <= 0 {
		return nil
	}

	_, err = file.Write(mFile.buf[:mFile.len])
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}

	mFile.size = mFile.size + mFile.len
	mFile.start = mFile.size
	mFile.len = 0
	return nil
}

func ScanDataFile(index int, dataPath string, hintPath string) error {
	var dataFileInfo, err = os.Stat(dataPath)
	if err != nil {
		return err
	}

	hitFileInfo, err := os.Stat(hintPath)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(hintPath)
			file.Close()
			if err != nil {
				return err
			} else {
				dataMFile, err := OpenMFileForRead(dataPath)
				if err != nil {
					return err
				}

				hintMFile, err := OpenMFileForWrite(hintPath)
				if err != nil {
					return err
				}

				err = CompareFile(index, dataMFile, hintMFile, int64(0))
				if err != nil {
					return err
				}
			}
		} else {
			return err
		}
	} else {
		dataFileModTime := dataFileInfo.ModTime().UnixNano()
		hintFileModTime := hitFileInfo.ModTime().UnixNano()
		if dataFileModTime > hintFileModTime {
			dataMFile, err := OpenMFileForRead(dataPath)
			if err != nil {
				return err
			}

			hintMFile, err := OpenMFileForWrite(hintPath)
			if err != nil {
				return err
			}

			err = CompareFile(index, dataMFile, hintMFile, hitFileInfo.ModTime().UnixNano())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func CompareFile(index int, dataMFile *MFile, hintMFile *MFile, after int64) error {
	defer hintMFile.Flush()
	var hintItemLen = binary.Size(HintItem{})

	var offset = int64(0)
	var header RecordHeader
	var wHintItem = make([]byte, hintItemLen)
	for {
		data, err := dataMFile.Read(offset, int64(dataHeaderLen))
		if err != nil {
			if err != ErrEOF {
				return err
			} else {
				return nil
			}
		}

		err = binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &header)
		if err != nil {
			return err
		}

		key, err := dataMFile.Read(offset+int64(dataHeaderLen), int64(header.KLen))
		if err != nil {
			if err != ErrEOF {
				return err
			} else {
				return nil
			}
		}

		totalLen := int64(dataHeaderLen) + int64(header.KLen) + int64(header.VLen)
		if totalLen%256 != 0 {
			totalLen += 256 - totalLen%256
		}

		if int64(header.TimeStamp) <= after {
			offset += totalLen
			continue
		}

		if header.Flg&0x01 == 1 {
			continue
		}

		hintItem := &HintItem{
			Num: header.Num,
			Pos: uint32(offset) | uint32(index),
		}

		binary.Write(NewLmBuffer(wHintItem[0:hintItemLen]), binary.LittleEndian, hintItem)
		err = hintMFile.Write(wHintItem)
		if err != nil {
			return err
		}

		err = hintMFile.Write(key)
		if err != nil {
			return err
		}

		offset += totalLen
	}
	return nil
}
