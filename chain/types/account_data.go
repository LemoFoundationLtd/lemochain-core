package types

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"io"
	"math/big"
)

var ErrInvalidRecord = errors.New("invalid version records")

type VersionRecord struct {
	Version uint32
	Height  uint32
}

//go:generate gencodec -type AccountData --field-override accountDataMarshaling -out gen_account_data_json.go

// AccountData is the Lemochain consensus representation of accounts.
// These objects are stored in the store.
type AccountData struct {
	Address     common.Address           `json:"address" gencodec:"required"`
	Balance     *big.Int                 `json:"balance" gencodec:"required"`
	Versions    map[ChangeLogType]uint32 `json:"versions" gencodec:"required"`
	CodeHash    common.Hash              `json:"codeHash" gencodec:"required"`
	StorageRoot common.Hash              `json:"root" gencodec:"required"` // MPT root of the storage trie
	// It records the block height which contains any type of newest change log.
	NewestRecords map[ChangeLogType]VersionRecord
}

type accountDataMarshaling struct {
	Balance *hexutil.Big
}

type rlpAccountData struct {
	Address     common.Address
	Balance     *big.Int
	CodeHash    common.Hash
	StorageRoot common.Hash

	LogTypes      []uint32
	Versions      []uint32
	RalatedBlocks []uint32
}

// EncodeRLP implements rlp.Encoder.
func (a *AccountData) EncodeRLP(w io.Writer) error {
	var LogTypes, Versions, RalatedBlocks []uint32
	if len(a.Versions) != len(a.NewestRecords) {
		log.Error("unmatched array length for encoding AccountData. UpdateRecords should be called before encode", "Versions", len(a.Versions), "NewestRecords", len(a.NewestRecords))
		return ErrInvalidRecord
	}
	for logType, record := range a.NewestRecords {
		LogTypes = append(LogTypes, uint32(logType))
		Versions = append(Versions, record.Version)
		RalatedBlocks = append(RalatedBlocks, record.Height)
	}
	return rlp.Encode(w, rlpAccountData{
		Address:       a.Address,
		Balance:       a.Balance,
		CodeHash:      a.CodeHash,
		StorageRoot:   a.StorageRoot,
		LogTypes:      LogTypes,
		Versions:      Versions,
		RalatedBlocks: RalatedBlocks,
	})
}

// DecodeRLP implements rlp.Decoder.
func (a *AccountData) DecodeRLP(s *rlp.Stream) error {
	var dec rlpAccountData
	err := s.Decode(&dec)
	if err == nil {
		a.Address, a.Balance, a.CodeHash, a.StorageRoot = dec.Address, dec.Balance, dec.CodeHash, dec.StorageRoot
		if len(dec.LogTypes) != len(dec.Versions) || len(dec.LogTypes) != len(dec.RalatedBlocks) {
			log.Error("unmatched array length for decoding AccountData", "LogTypes", len(dec.LogTypes), "Versions", len(dec.Versions), "RalatedBlocks", len(dec.RalatedBlocks))
			return ErrInvalidRecord
		}
		a.Versions = make(map[ChangeLogType]uint32)
		a.NewestRecords = make(map[ChangeLogType]VersionRecord)

		for i, logType := range dec.LogTypes {
			a.Versions[ChangeLogType(logType)] = dec.Versions[i]
			a.NewestRecords[ChangeLogType(logType)] = VersionRecord{Version: dec.Versions[i], Height: dec.RalatedBlocks[i]}
		}
	}
	return err
}

func (a *AccountData) Copy() *AccountData {
	cpy := *a
	cpy.Balance = new(big.Int).Set(a.Balance)
	return &cpy
}

type Code []byte

func (c Code) String() string {
	return fmt.Sprintf("%v", []byte(c)) // strings.Join(asm.Disassemble(c), " ")
}

// UpdateRecords records the newest version's position in block chain
func (a *AccountData) UpdateRecords(blockHeight uint32) error {
	// save the newest record
	if a.NewestRecords == nil {
		a.NewestRecords = make(map[ChangeLogType]VersionRecord)
	}
	for logType, version := range a.Versions {
		record, ok := a.NewestRecords[logType]
		if !ok || record.Version != version {
			a.NewestRecords[logType] = VersionRecord{Height: blockHeight, Version: version}
		}
	}
	return nil
}

type AccountAccessor interface {
	GetAddress() common.Address
	GetBalance() *big.Int
	SetBalance(balance *big.Int)
	GetVersion(logType ChangeLogType) uint32
	SetVersion(logType ChangeLogType, version uint32)
	GetCodeHash() common.Hash
	SetCodeHash(codeHash common.Hash)
	GetCode() (Code, error)
	SetCode(code Code)
	GetStorageRoot() common.Hash
	SetStorageRoot(root common.Hash)
	GetStorageState(key common.Hash) ([]byte, error)
	SetStorageState(key common.Hash, value []byte) error
	IsEmpty() bool
	GetSuicide() bool
	SetSuicide(suicided bool)
	MarshalJSON() ([]byte, error)
}
