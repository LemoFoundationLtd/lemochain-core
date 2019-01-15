package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"io"
	"math/big"
	"strings"
)

//go:generate gencodec -type VersionRecord --field-override versionRecordMarshaling -out gen_version_record_json.go

type VersionRecord struct {
	Version uint32 `json:"version" gencodec:"required"`
	Height  uint32 `json:"height" gencodec:"required"`
}

type versionRecordMarshaling struct {
	Version hexutil.Uint32
	Height  hexutil.Uint32
}

//go:generate gencodec -type AccountData --field-override accountDataMarshaling -out gen_account_data_json.go

// AccountData is the Lemochain consensus representation of accounts.
// These objects are stored in the store.
type Candidate struct {
	IsCandidate bool
	Votes       *big.Int
	Profile     map[string]string
}

type AccountData struct {
	Address     common.Address `json:"address" gencodec:"required"`
	Balance     *big.Int       `json:"balance" gencodec:"required"`
	CodeHash    common.Hash    `json:"codeHash" gencodec:"required"`
	StorageRoot common.Hash    `json:"root" gencodec:"required"` // MPT root of the storage trie
	// It records the block height which contains any type of newest change log. It is updated in finalize step
	NewestRecords map[ChangeLogType]VersionRecord `json:"records" gencodec:"required"`

	VoteFor   common.Address
	Candidate Candidate

	// related transactions include income and outcome
	TxHashList []common.Hash   `json:"-"`
	IsNode     bool            // todo 临时字段，标记此账户是否为代理代理节点账户
	VoteTo     *common.Address // todo 投票给谁
	Votes      *big.Int        // todo 票数
}

type accountDataMarshaling struct {
	Balance *hexutil.Big10
}

// rlpVersionRecord defines the fields which would be encode/decode by rlp
type rlpVersionRecord struct {
	LogType ChangeLogType
	Version uint32
	Height  uint32
}

// rlpAccountData defines the fields which would be encode/decode by rlp
type rlpAccountData struct {
	Address     common.Address
	Balance     *big.Int
	CodeHash    common.Hash
	StorageRoot common.Hash
	TxHashList  []common.Hash

	NewestRecords []rlpVersionRecord
}

// EncodeRLP implements rlp.Encoder.
func (a *AccountData) EncodeRLP(w io.Writer) error {
	var NewestRecords []rlpVersionRecord
	for logType, record := range a.NewestRecords {
		NewestRecords = append(NewestRecords, rlpVersionRecord{logType, record.Version, record.Height})
	}
	return rlp.Encode(w, rlpAccountData{
		Address:       a.Address,
		Balance:       a.Balance,
		CodeHash:      a.CodeHash,
		StorageRoot:   a.StorageRoot,
		TxHashList:    a.TxHashList,
		NewestRecords: NewestRecords,
	})
}

// DecodeRLP implements rlp.Decoder.
func (a *AccountData) DecodeRLP(s *rlp.Stream) error {
	var dec rlpAccountData
	err := s.Decode(&dec)
	if err == nil {
		a.Address, a.Balance, a.CodeHash, a.StorageRoot, a.TxHashList = dec.Address, dec.Balance, dec.CodeHash, dec.StorageRoot, dec.TxHashList
		a.NewestRecords = make(map[ChangeLogType]VersionRecord)

		for _, record := range dec.NewestRecords {
			a.NewestRecords[ChangeLogType(record.LogType)] = VersionRecord{Version: record.Version, Height: record.Height}
		}
	}
	return err
}

func (a *AccountData) Copy() *AccountData {
	cpy := *a
	cpy.Balance = new(big.Int).Set(a.Balance)
	if len(a.NewestRecords) > 0 {
		cpy.NewestRecords = make(map[ChangeLogType]VersionRecord)
		for logType, record := range a.NewestRecords {
			cpy.NewestRecords[logType] = record
		}
	}
	if len(a.TxHashList) > 0 {
		cpy.TxHashList = make([]common.Hash, 0, len(a.TxHashList))
		for _, hash := range a.TxHashList {
			cpy.TxHashList = append(cpy.TxHashList, hash)
		}
	}
	return &cpy
}

func (a *AccountData) String() string {
	set := []string{
		fmt.Sprintf("Address: %s", a.Address.String()),
		fmt.Sprintf("Balance: %s", a.Balance.String()),
	}
	if a.CodeHash != (common.Hash{}) {
		set = append(set, fmt.Sprintf("CodeHash: %s", a.CodeHash.Hex()))
	}
	if a.StorageRoot != (common.Hash{}) {
		set = append(set, fmt.Sprintf("StorageRoot: %s", a.StorageRoot.Hex()))
	}
	if len(a.TxHashList) > 0 {
		set = append(set, fmt.Sprintf("TxHashList: %v", a.TxHashList))
	}
	if len(a.NewestRecords) > 0 {
		records := make([]string, 0, len(a.NewestRecords))
		for logType, record := range a.NewestRecords {
			records = append(records, fmt.Sprintf("%s: {v: %d, h: %d}", logType, record.Version, record.Height))
		}
		set = append(set, fmt.Sprintf("NewestRecords: {%s}", strings.Join(records, ", ")))
	}

	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
}

type Code []byte

func (c Code) String() string {
	return fmt.Sprintf("%#x", []byte(c)) // strings.Join(asm.Disassemble(c), " ")
}

type AccountAccessor interface {
	GetVoteFor() common.Address
	SetVoteFor(addr common.Address)

	GetCandidate() Candidate
	SetCandidate(candidate Candidate)

	GetAddress() common.Address
	GetBalance() *big.Int
	SetBalance(balance *big.Int)
	// GetBaseVersion returns the version of specific change log from the base block. It is not changed by tx processing until the finalised
	GetBaseVersion(logType ChangeLogType) uint32
	GetCodeHash() common.Hash
	SetCodeHash(codeHash common.Hash)
	GetCode() (Code, error)
	SetCode(code Code)
	GetStorageRoot() common.Hash
	SetStorageRoot(root common.Hash)
	GetStorageState(key common.Hash) ([]byte, error)
	SetStorageState(key common.Hash, value []byte) error
	GetTxHashList() []common.Hash
	IsEmpty() bool
	GetSuicide() bool
	SetSuicide(suicided bool)
	MarshalJSON() ([]byte, error)

	IsdeputyNode() bool                      // todo 临时函数
	SetdeputyNode(isnode bool) error         // todo
	SetVoteTo(address *common.Address) error // todo
	GetVoteTo() *common.Address              // todo
	SetVotes(votes *big.Int) error           // todo
	GetVotes() *big.Int                      // todo

}
