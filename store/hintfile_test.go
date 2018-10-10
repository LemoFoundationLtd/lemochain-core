package store

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func PathIsExist(path string) (bool, error) {
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

func TestMFile_Write2(t *testing.T) {
	ClearData()

	path := GetStorePath()
	isExist, err := PathIsExist(path)
	assert.NoError(t, err)

	if !isExist {
		err = os.MkdirAll(path, os.ModePerm)
		assert.NoError(t, err)
	}

	path1 := filepath.Join(path, "test1_hint.hint")
	file, err := OpenMFileForWrite(path1)
	assert.NoError(t, err)

	uintSize := 32
	wBuf, err := CreateBufWithNumber(uintSize)
	assert.NoError(t, err)

	totalCnt := 256 * 256 * 16
	for index := 0; index < totalCnt; index++ {
		err = file.Write(wBuf)
		assert.NoError(t, err)
	}
	file.Flush()

	// test file size
	info, err := os.Stat(path1)
	assert.NoError(t, err)

	totalSize := totalCnt * uintSize
	assert.Equal(t, info.Size(), int64(totalSize))

	path2 := filepath.Join(path, "test2_hint.hint")
	file1, err := OpenMFileForRead(path1)
	assert.NoError(t, err)

	file2, err := OpenMFileForWrite(path2)
	assert.NoError(t, err)

	offset := 0
	for {
		if offset > totalSize {
			break
		}

		rBuf, err := file1.Read(int64(offset), int64(uintSize))
		if err == ErrEOF {
			break
		} else {
			assert.NoError(t, err)
		}

		err = file2.Write(rBuf)
		assert.NoError(t, err)
		offset = offset + uintSize
	}
	file2.Flush()

	info, err = os.Stat(path2)
	assert.NoError(t, err)
	assert.Equal(t, info.Size(), int64(totalSize))
}
