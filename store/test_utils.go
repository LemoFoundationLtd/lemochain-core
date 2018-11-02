package store

import (
	"bytes"
	"encoding/binary"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"math/big"
	"os"
)

func GetStorePath() string {
	return "../../db"
}

func ClearData() error {
	return os.RemoveAll(GetStorePath())
}

func CreateBlock(hash common.Hash, parent common.Hash, height uint32) *types.Block {
	header := &types.Header{VersionRoot: hash}
	header.Height = height
	header.ParentHash = parent
	header.VersionRoot = hash
	block := &types.Block{}
	block.SetHeader(header)
	return block
}

func GetBlock0() *types.Block {
	hash := common.HexToHash("0000000000000000")
	return CreateBlock(hash, common.Hash{}, 0)
}

func GetBlock1() *types.Block {
	parentBlock := GetBlock0()
	childHash := common.HexToHash("1111111111111111")
	return CreateBlock(childHash, parentBlock.Hash(), 1)
}

func GetBlock2() *types.Block {
	parentBlock := GetBlock1()
	childHash := common.HexToHash("2222222222222222")
	return CreateBlock(childHash, parentBlock.Hash(), 2)
}

func GetAccount(address string, balance int64, version uint32) *types.AccountData {
	newestRecords := make(map[types.ChangeLogType]types.VersionRecord)
	newestRecords[0] = types.VersionRecord{Version: version, Height: 100}
	return &types.AccountData{
		Address:       common.HexToAddress(address),
		Balance:       big.NewInt(balance),
		NewestRecords: newestRecords,
		CodeHash:      common.HexToHash("0x1d5f11eaa13e02cdca886181dc38ab4cb8cf9092e86c000fb42d12c8b504500e"),
		StorageRoot:   common.HexToHash("0xcbeb7c7e36b846713bc99b8fa527e8d552e31bfaa1ac0f2b773958cda3aba3ed"),
		TxHashList: []common.Hash{
			common.HexToHash("0x11"),
			common.HexToHash("0x22"),
		},
	}
}

func GetAccounts() []*types.AccountData {
	accounts := make([]*types.AccountData, 2)
	accounts[0] = GetAccount("100", 5, 1)
	accounts[1] = GetAccount("200", 6, 2)
	return accounts
}

func NewKey1() ([]byte, uint32) {
	key := common.HexToHash("0x5fa2358263196dbbf23d1ca7a509451f7a2f64c15837bfbb81298b1e3e24e4fa")
	return key.Bytes(), 500
}

func NewKey2() ([]byte, uint32) {
	key := common.HexToHash("0x6fa2358263196dbbf23d1ca7a509451f7a2f64c15837bfbb81298b1e3e24e4fa")
	return key.Bytes(), 600
}

func NewKey3() ([]byte, uint32) {
	key := common.HexToHash("0x7fa2358263196dbbf23d1ca7a509451f7a2f64c15837bfbb81298b1e3e24e4fa")
	return key.Bytes(), 700
}

func CreateSign(cnt int) ([]types.SignData, error) {
	if cnt <= 0 || cnt >= 256 {
		return nil, ErrArgInvalid
	}

	result := make([]types.SignData, cnt)
	for index := 0; index < cnt; index++ {
		s1 := bytes.NewBuffer([]byte{})
		err := binary.Write(s1, binary.BigEndian, uint8(index))
		if err != nil {
			return nil, err
		} else {
			val := make([]byte, 65)
			val1 := crypto.Keccak256(s1.Bytes())
			copy(val[0:32], val1)
			copy(val[32:64], val1)

			var sign types.SignData
			err := binary.Read(bytes.NewBuffer(val), binary.LittleEndian, &sign)
			if err != nil {
				return nil, err
			}

			result[index] = sign
		}
	}

	return result, nil
}

func CreateBufWithNumber(size int) ([]byte, error) {
	if size < 32 || size > 1024*1024*1024 {
		return nil, ErrArgInvalid
	}

	buf := make([]byte, size)
	wLen := 0
	for index := 0; index < size; index++ {
		s1 := bytes.NewBuffer([]byte{})
		err := binary.Write(s1, binary.BigEndian, uint32(index))
		if err != nil {
			return nil, err
		}

		val := crypto.Keccak256(s1.Bytes())
		if size-wLen >= len(val) {
			copy(buf[wLen:], val[:])
			wLen = wLen + len(val)
		} else {
			break
		}
	}
	return buf, nil
}

func CreateBufWithNumberBatch(cnt int, template []byte) ([][]byte, error) {
	if cnt <= 0 || cnt > 256*256*256*256 || len(template) < 32 {
		return nil, ErrArgInvalid
	}

	result := make([][]byte, cnt)
	for index := 0; index < cnt; index++ {
		s1 := bytes.NewBuffer([]byte{})
		err := binary.Write(s1, binary.BigEndian, uint32(index))
		if err != nil {
			return nil, err
		} else {
			buf := make([]byte, len(template))
			copy(buf[:], template[:])
			copy(buf[0:4], s1.Bytes()[0:4])
			copy(buf[14:18], s1.Bytes()[0:4])
			copy(buf[28:32], s1.Bytes()[0:4])
			result[index] = crypto.Keccak256(buf)
		}
	}

	return result, nil
}
