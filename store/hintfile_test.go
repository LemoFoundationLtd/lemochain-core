package store

import (
	"io"
	"testing"
)

func TestMFile_Write(t *testing.T) {
	mfile, err := OpenMFileForWrite("../../lmstore/test.hint")
	if err != nil {
		t.Errorf("OPEN FILE FOR WRITE FAIL.%s", err.Error())
	}

	data := []byte{'x', 'g', 'g', 'y', 'x', 'g', 'g', 'y', 'x', 'g', 'g', 'y', 'x', 'g', 'g', 'y', 'x', 'g', 'g', 'y', 'x', 'g', 'g', 'y', 'x', 'g', 'g', 'y', 'x', 'g', 'g', 'y'}
	for index := 0; index < 10000; index++ {
		err = mfile.Write(data)
		if err != nil {
			t.Errorf(" MFILE WRITE FAIL.%s", err.Error())
		} else {
			//t.Logf("MFILE WRITE SUCCESS.")
		}
	}
	mfile.Flush()

	rfile, err := OpenMFileForRead("../../lmstore/test.hint")
	if err != nil {
		t.Errorf("OPEN FILE FOR READ FAIL.%s", err.Error())
	}

	offset := int64(0)
	cnt := 0
	for {
		_, err := rfile.Read(offset, 32)
		if err != nil {
			if err != io.EOF {
				t.Errorf("MFILE READ FAIL.%s", err.Error())
			}

			break
		} else {
			offset += 32
			cnt += 1
		}
	}
}
