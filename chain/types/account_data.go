package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

type VersionRecord struct {
	Version uint32
	Height  uint32
}

// AccountData is the Lemochain consensus representation of accounts.
// These objects are stored in the store.
type AccountData struct {
	Address     common.Address
	Balance     *big.Int
	Version     uint32
	CodeHash    common.Hash
	StorageRoot common.Hash // MPT root of the storage trie
	// One block may contains lost of change logs, but there is only one record in this array for a block. It is the record of the last change log version in related block.
	VersionRecords []VersionRecord
}

type Code []byte

func (c Code) String() string {
	return fmt.Sprintf("%v", []byte(c)) // strings.Join(asm.Disassemble(c), " ")
}

type AccountAccessor interface {
	GetAddress() common.Address
	GetBalance() *big.Int
	SetBalance(balance *big.Int)
	GetVersion() uint32
	SetVersion(version uint32)
	GetCodeHash() common.Hash
	SetCodeHash(codeHash common.Hash)
	GetCode() (Code, error)
	SetCode(code Code)
	GetStorageRoot() common.Hash
	SetStorageRoot(root common.Hash)
	GetStorageState(key common.Hash) ([]byte, error)
	SetStorageState(key common.Hash, value []byte) error
}
