package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"io"
	"math/big"
	"sort"
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

const (
	CandidateKeyIsCandidate  string = "isCandidate"
	CandidateKeyNodeID       string = "nodeID"
	CandidateKeyHost         string = "host"
	CandidateKeyPort         string = "port"
	CandidateKeyMinerAddress string = "minerAddress"
)

type Pair struct {
	Key string
	Val string
}

type CandidateProfile map[string]string

func (a *CandidateProfile) EncodeRLP(w io.Writer) error {
	tmp := make([]Pair, 0)
	if len(*a) <= 0 {
		return rlp.Encode(w, tmp)
	} else {
		keys := make([]string, 0, len(*a))
		for k, _ := range *a {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for index := 0; index < len(keys); index++ {
			tmp = append(tmp, Pair{
				Key: keys[index],
				Val: (*a)[keys[index]],
			})
		}

		// for k, v := range *a {
		// 	tmp = append(tmp, Pair{
		// 		Key: k,
		// 		Val: v,
		// 	})
		// }

		return rlp.Encode(w, tmp)
	}
}

func (a *CandidateProfile) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	if size <= 0 {
		return nil
	}

	dec := make([]Pair, 0)
	err := s.Decode(&dec)
	if err != nil {
		return err
	}

	for index := 0; index < len(dec); index++ {
		(*a)[dec[index].Key] = dec[index].Val
	}
	return nil
}

type Candidate struct {
	Votes   *big.Int
	Profile CandidateProfile
}

type AccountData struct {
	Address     common.Address `json:"address" gencodec:"required"`
	Balance     *big.Int       `json:"balance" gencodec:"required"`
	CodeHash    common.Hash    `json:"codeHash" gencodec:"required"`
	StorageRoot common.Hash    `json:"root" gencodec:"required"` // MPT root of the storage trie
	// It records the block height which contains any type of newest change log. It is updated in finalize step
	NewestRecords map[ChangeLogType]VersionRecord `json:"records" gencodec:"required"`

	VoteFor   common.Address `json:"voteFor"`
	Candidate Candidate      `json:"candidate"`
	TxCount   int            `json:"txCount"`
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

type rlpCandidate struct {
	Votes   *big.Int
	Profile *CandidateProfile
}

// rlpAccountData defines the fields which would be encode/decode by rlp
type rlpAccountData struct {
	Address       common.Address
	Balance       *big.Int
	CodeHash      common.Hash
	StorageRoot   common.Hash
	TxHashList    []common.Hash
	VoteFor       common.Address
	Candidate     rlpCandidate
	NewestRecords []rlpVersionRecord
}

// EncodeRLP implements rlp.Encoder.
func (a *AccountData) EncodeRLP(w io.Writer) error {
	var NewestRecords []rlpVersionRecord
	for logType, record := range a.NewestRecords {
		NewestRecords = append(NewestRecords, rlpVersionRecord{logType, record.Version, record.Height})
	}

	candidate := rlpCandidate{
		Votes:   a.Candidate.Votes,
		Profile: &(a.Candidate.Profile),
	}

	return rlp.Encode(w, rlpAccountData{
		Address:     a.Address,
		Balance:     a.Balance,
		CodeHash:    a.CodeHash,
		StorageRoot: a.StorageRoot,
		// TxHashList:    a.TxHashList,
		VoteFor:       a.VoteFor,
		Candidate:     candidate,
		NewestRecords: NewestRecords,
	})
}

// DecodeRLP implements rlp.Decoder.
func (a *AccountData) DecodeRLP(s *rlp.Stream) error {
	var dec rlpAccountData

	profile := make(CandidateProfile)
	dec.Candidate.Profile = &profile

	err := s.Decode(&dec)
	if err == nil {
		a.Address, a.Balance, a.CodeHash, a.StorageRoot, a.VoteFor =
			dec.Address, dec.Balance, dec.CodeHash, dec.StorageRoot, dec.VoteFor
		a.NewestRecords = make(map[ChangeLogType]VersionRecord)

		a.Candidate.Votes = dec.Candidate.Votes
		a.Candidate.Profile = *dec.Candidate.Profile

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
	GetTxCount() int
	SetTxCount(count int)

	GetVoteFor() common.Address
	SetVoteFor(addr common.Address)

	GetVotes() *big.Int
	SetVotes(votes *big.Int)

	GetCandidateProfile() CandidateProfile
	SetCandidateProfile(profile CandidateProfile)

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
	IsEmpty() bool
	GetSuicide() bool
	SetSuicide(suicided bool)
	MarshalJSON() ([]byte, error)
}
