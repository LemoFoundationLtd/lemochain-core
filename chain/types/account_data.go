package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
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

//go:generate gencodec -type Candidate --field-override candidateMarshaling -out gen_candidate_json.go
//go:generate gencodec -type AccountData --field-override accountDataMarshaling -out gen_account_data_json.go

// AccountData is the Lemochain consensus representation of accounts.
// These objects are stored in the store.

const (
	// candidate profile
	CandidateKeyIsCandidate   string = "isCandidate"
	CandidateKeyNodeID        string = "nodeID"
	CandidateKeyHost          string = "host"
	CandidateKeyPort          string = "port"
	CandidateKeyIncomeAddress string = "incomeAddress"
	CandidateKeyPledgeAmount  string = "pledgeBalance" // 质押金额
	// asset profile
	AssetName              string = "name"
	AssetSymbol            string = "symbol"
	AssetDescription       string = "description"
	AssetFreeze            string = "freeze"
	AssetSuggestedGasLimit string = "suggestedGasLimit"
)

type Pair struct {
	Key string
	Val string
}

type Profile map[string]string

func (a *Profile) Clone() *Profile {
	if a == nil {
		return nil
	}

	result := make(Profile)
	for k, v := range *a {
		result[k] = v
	}
	return &result
}

func (a *Profile) EncodeRLP(w io.Writer) error {
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
		return rlp.Encode(w, tmp)
	}
}

func (a *Profile) DecodeRLP(s *rlp.Stream) error {
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
	Votes   *big.Int `json:"votes"`
	Profile Profile  `json:"profile"`
}

type candidateMarshaling struct {
	Votes *hexutil.Big10
}

//go:generate gencodec -type SignAccount --field-override signAccountMarshaling -out gen_sign_account_json.go
type SignAccount struct {
	Address common.Address `json:"address" gencodec:"required"`
	Weight  uint8          `json:"weight" gencodec:"required"`
}
type signAccountMarshaling struct {
	Weight hexutil.Uint8
}
type Signers []SignAccount

func (signers Signers) Len() int {
	return len(signers)
}

func (signers Signers) Less(i, j int) bool {
	return signers[i].Address.Hex() < signers[j].Address.Hex()
}

func (signers Signers) Swap(i, j int) {
	signers[i], signers[j] = signers[j], signers[i]
}

type SignerMap map[common.Address]uint8

func (signers Signers) ToSignerMap() SignerMap {
	m := make(SignerMap)
	for _, v := range signers {
		m[v.Address] = v.Weight
	}
	return m
}

func (signers Signers) String() string {
	if len(signers) > 0 {
		records := make([]string, 0, len(signers))
		for index := 0; index < len(signers); index++ {
			records = append(records, fmt.Sprintf("{Addr: %s, Weight: %d}", signers[index].Address.Hex(), signers[index].Weight))
		}
		return fmt.Sprintf("[%s]", strings.Join(records, ", "))
	} else {
		return "[]"
	}
}

func (signers Signers) Set(address common.Address, weight uint8) {
	isExist := false
	for index := 0; index < len(signers); index++ {
		if signers[index].Address == address {
			signers[index].Weight = weight
			isExist = true
			break
		}
	}

	if !isExist {
		signers[len(signers)] = SignAccount{
			Address: address,
			Weight:  weight,
		}
	}

	return
}

type AccountData struct {
	Address  common.Address `json:"address" gencodec:"required"`
	Balance  *big.Int       `json:"balance" gencodec:"required"`
	CodeHash common.Hash    `json:"codeHash" gencodec:"required"`

	StorageRoot   common.Hash `json:"root" gencodec:"required"`
	AssetCodeRoot common.Hash `json:"assetCodeRoot" gencodec:"required"`
	AssetIdRoot   common.Hash `json:"assetIdRoot" gencodec:"required"`
	EquityRoot    common.Hash `json:"equityRoot" gencodec:"required"`

	VoteFor   common.Address `json:"voteFor"`
	Candidate Candidate      `json:"candidate"`

	// It records the block height which contains any type of newest change log. It is updated in finalize step
	NewestRecords map[ChangeLogType]VersionRecord `json:"records" gencodec:"required"`
	Signers       Signers                         `json:"signers"`
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
	Profile *Profile
}

// rlpAccountData defines the fields which would be encode/decode by rlp
type rlpAccountData struct {
	Address       common.Address
	Balance       *big.Int
	CodeHash      common.Hash
	StorageRoot   common.Hash
	AssetCodeRoot common.Hash
	AssetIdRoot   common.Hash
	EquityRoot    common.Hash
	TxHashList    []common.Hash
	VoteFor       common.Address
	Candidate     rlpCandidate
	TxCount       uint32
	NewestRecords []rlpVersionRecord
	Signers       Signers
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
		Address:       a.Address,
		Balance:       a.Balance,
		CodeHash:      a.CodeHash,
		StorageRoot:   a.StorageRoot,
		AssetCodeRoot: a.AssetCodeRoot,
		AssetIdRoot:   a.AssetIdRoot,
		EquityRoot:    a.EquityRoot,
		VoteFor:       a.VoteFor,
		Candidate:     candidate,
		NewestRecords: NewestRecords,
		Signers:       a.Signers,
	})
}

// DecodeRLP implements rlp.Decoder.
func (a *AccountData) DecodeRLP(s *rlp.Stream) error {
	var dec rlpAccountData

	profile := make(Profile)
	dec.Candidate.Profile = &profile

	err := s.Decode(&dec)
	if err == nil {
		a.Address, a.Balance, a.CodeHash, a.StorageRoot, a.AssetCodeRoot, a.AssetIdRoot, a.EquityRoot, a.VoteFor, a.Signers =
			dec.Address, dec.Balance, dec.CodeHash, dec.StorageRoot, dec.AssetCodeRoot, dec.AssetIdRoot, dec.EquityRoot, dec.VoteFor, dec.Signers
		a.NewestRecords = make(map[ChangeLogType]VersionRecord)

		a.Candidate.Votes = dec.Candidate.Votes
		a.Candidate.Profile = *dec.Candidate.Profile

		for _, record := range dec.NewestRecords {
			a.NewestRecords[ChangeLogType(record.LogType)] = VersionRecord{Version: record.Version, Height: record.Height}
		}
	}
	return err
}

func (a *AccountData) Clone() NodeData {
	return a.Copy()
}

func (a *AccountData) Copy() *AccountData {
	cpy := *a
	cpy.Balance = new(big.Int).Set(a.Balance)

	if a.Candidate.Votes != nil {
		cpy.Candidate.Votes = new(big.Int).Set(a.Candidate.Votes)
	}

	if len(a.Candidate.Profile) > 0 {
		cpy.Candidate.Profile = make(Profile)
		for k, v := range a.Candidate.Profile {
			cpy.Candidate.Profile[k] = v
		}
	}

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
		fmt.Sprintf("VoteFor: %s", a.VoteFor.String()),
	}

	if a.Candidate.Votes != nil {
		set = append(set, fmt.Sprintf("Votes: %s", a.Candidate.Votes.String()))
	}
	if a.CodeHash != (common.Hash{}) {
		set = append(set, fmt.Sprintf("CodeHash: %s", a.CodeHash.Hex()))
	}
	if a.StorageRoot != (common.Hash{}) {
		set = append(set, fmt.Sprintf("StorageRoot: %s", a.StorageRoot.Hex()))
	}

	if a.AssetCodeRoot != (common.Hash{}) {
		set = append(set, fmt.Sprintf("AssetCodeRoot: %s", a.AssetCodeRoot.Hex()))
	}

	if a.AssetIdRoot != (common.Hash{}) {
		set = append(set, fmt.Sprintf("AssetIdRoot: %s", a.AssetIdRoot.Hex()))
	}

	if a.EquityRoot != (common.Hash{}) {
		set = append(set, fmt.Sprintf("EquityRoot: %s", a.EquityRoot.Hex()))
	}

	if len(a.NewestRecords) > 0 {
		records := make([]string, 0, len(a.NewestRecords))
		for logType, record := range a.NewestRecords {
			records = append(records, fmt.Sprintf("%s: {v: %d, h: %d}", logType, record.Version, record.Height))
		}
		set = append(set, fmt.Sprintf("NewestRecords: {%s}", strings.Join(records, ", ")))
	}

	if a.VoteFor != (common.Address{}) {
		set = append(set, fmt.Sprintf("VoteFor: %s", a.VoteFor.String()))
	}
	if a.Candidate.Votes != nil || len(a.Candidate.Profile) != 0 {
		set = append(set, fmt.Sprintf("Candidate: {Votes: %s, Profile: %v}", a.Candidate.Votes.String(), a.Candidate.Profile))
	}

	if len(a.Candidate.Profile) > 0 {
		records := make([]string, 0, len(a.Candidate.Profile))
		for k, v := range a.Candidate.Profile {
			records = append(records, fmt.Sprintf("%s => %s", k, v))
		}
		set = append(set, fmt.Sprintf("Profiles: {%s}", strings.Join(records, ", ")))
	}

	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
}

type Code []byte

func (c Code) String() string {
	return fmt.Sprintf("%#x", []byte(c)) // strings.Join(asm.Disassemble(c), " ")
}

type AccountAccessor interface {
	GetAddress() common.Address
	GetVersion(logType ChangeLogType) uint32
	GetNextVersion(logType ChangeLogType) uint32

	GetVoteFor() common.Address
	SetVoteFor(addr common.Address)

	GetVotes() *big.Int
	SetVotes(votes *big.Int)

	GetCandidate() Profile
	SetCandidate(profile Profile)
	GetCandidateState(key string) string
	SetCandidateState(key string, val string)

	GetBalance() *big.Int
	SetBalance(balance *big.Int)

	GetCodeHash() common.Hash
	SetCodeHash(codeHash common.Hash)

	GetCode() (Code, error)
	SetCode(code Code)

	GetStorageRoot() common.Hash
	SetStorageRoot(root common.Hash)
	GetAssetCodeRoot() common.Hash
	SetAssetCodeRoot(root common.Hash)
	GetAssetIdRoot() common.Hash
	SetAssetIdRoot(root common.Hash)
	GetEquityRoot() common.Hash
	SetEquityRoot(root common.Hash)

	GetStorageState(key common.Hash) ([]byte, error)
	SetStorageState(key common.Hash, value []byte) error

	GetAssetCode(code common.Hash) (*Asset, error)
	SetAssetCode(code common.Hash, asset *Asset) error
	GetAssetCodeTotalSupply(code common.Hash) (*big.Int, error)
	SetAssetCodeTotalSupply(code common.Hash, val *big.Int) error
	GetAssetCodeState(code common.Hash, key string) (string, error)
	SetAssetCodeState(code common.Hash, key string, val string) error

	GetAssetIdState(id common.Hash) (string, error)
	SetAssetIdState(id common.Hash, data string) error

	GetEquityState(id common.Hash) (*AssetEquity, error)
	SetEquityState(id common.Hash, equity *AssetEquity) error

	SetSingers(signers Signers) error
	GetSigners() Signers

	PushEvent(event *Event)
	PopEvent() error
	GetEvents() []*Event

	GetSuicide() bool
	SetSuicide(suicided bool)

	IsEmpty() bool
	MarshalJSON() ([]byte, error)
}
