package account

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"math/big"
	"strings"
)

const (
	BalanceLog types.ChangeLogType = iota + 1
	StorageLog
	StorageRootLog

	AssetCodeLog
	AssetCodeStateLog
	AssetCodeRootLog
	AssetCodeTotalSupplyLog
	AssetIdLog
	AssetIdRootLog
	EquityLog
	EquityRootLog
	CandidateLog
	CandidateStateLog

	CodeLog
	AddEventLog
	SuicideLog
	VoteForLog
	VotesLog
	LOG_TYPE_STOP
)

type ProfileChangeLogExtra struct {
	UUID common.Hash
	Key  string
}

func (extra *ProfileChangeLogExtra) String() string {
	set := []string{
		fmt.Sprintf("UUID: %s", extra.UUID.String()),
		fmt.Sprintf("Key: %s", extra.Key),
	}

	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
}

func init() {
	types.RegisterChangeLog(BalanceLog, "BalanceLog", decodeBigInt, decodeEmptyInterface, redoBalance, undoBalance)
	types.RegisterChangeLog(StorageLog, "StorageLog", decodeBytes, decodeHash, redoStorage, undoStorage)
	types.RegisterChangeLog(StorageRootLog, "StorageRootLog", decodeHash, decodeEmptyInterface, redoStorageRoot, undoStorageRoot)
	types.RegisterChangeLog(AssetCodeLog, "AssetCodeLog", decodeAsset, decodeHash, redoAssetCode, undoAssetCode)
	types.RegisterChangeLog(AssetCodeRootLog, "AssetCodeRootLog", decodeHash, decodeEmptyInterface, redoAssetCodeRoot, undoAssetCodeRoot)
	types.RegisterChangeLog(AssetCodeStateLog, "AssetCodeStateLog", decodeString, decodeProfileChangeLogExtra, redoAssetCodeState, undoAssetCodeState)
	types.RegisterChangeLog(AssetCodeTotalSupplyLog, "AssetCodeTotalSupplyLog", decodeBigInt, decodeHash, redoAssetCodeTotalSupply, undoAssetCodeTotalSupply)
	types.RegisterChangeLog(AssetIdLog, "AssetIdLog", decodeString, decodeHash, redoAssetId, undoAssetId)
	types.RegisterChangeLog(AssetIdRootLog, "AssetIdRootLog", decodeHash, decodeEmptyInterface, redoAssetIdRoot, undoAssetIdRoot)
	types.RegisterChangeLog(EquityLog, "EquityLog", decodeEquity, decodeHash, redoEquity, undoEquity)
	types.RegisterChangeLog(EquityRootLog, "EquityRootLog", decodeHash, decodeEmptyInterface, redoEquityRoot, undoEquityRoot)
	types.RegisterChangeLog(CodeLog, "CodeLog", decodeCode, decodeEmptyInterface, redoCode, undoCode)
	types.RegisterChangeLog(AddEventLog, "AddEventLog", decodeEvent, decodeEmptyInterface, redoAddEvent, undoAddEvent)
	types.RegisterChangeLog(SuicideLog, "SuicideLog", decodeEmptyInterface, decodeEmptyInterface, redoSuicide, undoSuicide)
	types.RegisterChangeLog(VoteForLog, "VoteForLog", decodeAddress, decodeEmptyInterface, redoVoteFor, undoVoteFor)
	types.RegisterChangeLog(VotesLog, "VotesLog", decodeBigInt, decodeEmptyInterface, redoVotes, undoVotes)
	types.RegisterChangeLog(CandidateLog, "CandidateLog", decodeCandidate, decodeEmptyInterface, redoCandidate, undoCandidate)
	types.RegisterChangeLog(CandidateStateLog, "CandidateStateLog", decodeString, decodeString, redoCandidateState, undoCandidateState)
}

// IsValuable returns true if the change log contains some data change
func IsValuable(log *types.ChangeLog) bool {
	valuable := true
	switch log.LogType {
	case BalanceLog:
		oldVal := log.OldVal.(big.Int)
		newVal := log.NewVal.(big.Int)
		valuable = oldVal.Cmp(&newVal) != 0
	case StorageLog:
		oldVal := log.OldVal.([]byte)
		newVal := log.NewVal.([]byte)
		valuable = bytes.Compare(oldVal, newVal) != 0
	case CodeLog:
		valuable = log.NewVal != nil && len(log.NewVal.(types.Code)) > 0
	case AddEventLog:
		valuable = log.NewVal != (*types.Event)(nil)
	case SuicideLog:
		oldAccount := log.OldVal.(*types.AccountData)
		valuable = oldAccount != nil && (big.NewInt(0).Cmp(oldAccount.Balance) != 0 || !isEmptyHash(oldAccount.CodeHash) || !isEmptyHash(oldAccount.StorageRoot))
	case VotesLog:
		oldVal := log.OldVal.(big.Int)
		newVal := log.NewVal.(big.Int)
		valuable = oldVal.Cmp(&newVal) != 0
	case VoteForLog:
		oldVal := log.OldVal.(common.Address)
		newVal := log.NewVal.(common.Address)
		valuable = oldVal != newVal
	case AssetCodeStateLog:
		fallthrough
	default:
		valuable = log.OldVal != log.NewVal
	}
	return valuable
}

func isEmptyHash(hash common.Hash) bool {
	return hash == (common.Hash{}) || hash == common.Sha3Nil
}

// decodeEmptyInterface decode an interface which contains an empty interface{}. its encoded data is [192], same as rlp([])
func decodeEmptyInterface(s *rlp.Stream) (interface{}, error) {
	_, size, _ := s.Kind()
	if size > 0 {
		log.Errorf("expected nil, got data size %d", size)
		return nil, types.ErrWrongChangeLogData
	}
	var result interface{}
	err := s.Decode(&result)
	return nil, err
}

// decodeBigInt decode an interface which contains an big.Int
func decodeBigInt(s *rlp.Stream) (interface{}, error) {
	var result big.Int
	err := s.Decode(&result)
	return result, err
}

// decodeBytes decode an interface which contains an []byte
func decodeBytes(s *rlp.Stream) (interface{}, error) {
	var result []byte
	err := s.Decode(&result)
	return result, err
}

// decodeCode decode an interface which contains an types.Code
func decodeCode(s *rlp.Stream) (interface{}, error) {
	var result []byte
	err := s.Decode(&result)
	return types.Code(result), err
}

func decodeAddress(s *rlp.Stream) (interface{}, error) {
	var result []byte
	err := s.Decode(&result)
	return common.BytesToAddress(result), err
}

// decodeEvents decode an interface which contains an *types.Event
func decodeEvent(s *rlp.Stream) (interface{}, error) {
	var result types.Event
	err := s.Decode(&result)
	return &result, err
}

func decodeCandidate(s *rlp.Stream) (interface{}, error) {
	_, size, _ := s.Kind()
	if size <= 0 {
		var result interface{}
		err := s.Decode(&result)
		return &result, err
	} else {
		result := make(types.Profile)
		err := s.Decode(&result)
		return &result, err
	}
}

func decodeUInt32(s *rlp.Stream) (interface{}, error) {
	var result uint32
	err := s.Decode(&result)
	return &result, err
}

func decodeHash(s *rlp.Stream) (interface{}, error) {
	var result []byte
	err := s.Decode(&result)
	return common.BytesToHash(result), err
}

func decodeString(s *rlp.Stream) (interface{}, error) {
	var result []byte
	err := s.Decode(&result)
	return string(result), err
}

func decodeAsset(s *rlp.Stream) (interface{}, error) {
	_, size, _ := s.Kind()
	if size <= 0 {
		var result interface{}
		err := s.Decode(&result)
		return nil, err
	} else {
		var result types.Asset
		result.TotalSupply = new(big.Int)
		result.Profile = make(types.Profile)
		err := s.Decode(&result)
		return &result, err
	}
}

func decodeEquity(s *rlp.Stream) (interface{}, error) {
	_, size, _ := s.Kind()
	if size <= 0 {
		var result interface{}
		err := s.Decode(&result)
		return nil, err
	} else {
		var result types.AssetEquity
		result.Equity = new(big.Int)
		err := s.Decode(&result)
		return &result, err
	}
}

func decodeProfileChangeLogExtra(s *rlp.Stream) (interface{}, error) {
	_, size, _ := s.Kind()
	if size <= 0 {
		var result interface{}
		err := s.Decode(&result)
		return nil, err
	} else {
		var result ProfileChangeLogExtra
		err := s.Decode(&result)
		return &result, err
	}
}

//
// ChangeLog definitions
//
func NewVotesLog(address common.Address, processor types.ChangeLogProcessor, newVotes *big.Int) *types.ChangeLog {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: VotesLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(VotesLog),
		OldVal:  *(new(big.Int).Set(account.GetVotes())),
		NewVal:  *(new(big.Int).Set(newVotes)),
	}
}

func redoVotes(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newValue, ok := c.NewVal.(big.Int)
	if !ok {
		log.Errorf("redoVotes expected NewVal big.Int, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVotes(&newValue)
	return nil
}

func undoVotes(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldValue, ok := c.OldVal.(big.Int)
	if !ok {
		log.Errorf("undoVotes expected OldVal big.Int, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVotes(&oldValue)
	return nil
}

func NewVoteForLog(address common.Address, processor types.ChangeLogProcessor, newVoteFor common.Address) *types.ChangeLog {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: VoteForLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(VoteForLog),
		OldVal:  account.GetVoteFor(),
		NewVal:  newVoteFor,
	}
}

func redoVoteFor(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(common.Address)
	if !ok {
		log.Errorf("redoVoteFor expected NewVal common.Address, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVoteFor(newVal)
	return nil
}

func undoVoteFor(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(common.Address)
	if !ok {
		log.Errorf("undoVoteFor expected OldVal common.Address, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVoteFor(oldVal)
	return nil
}

func cloneCandidateProfile(src types.Profile) types.Profile {
	if src == nil {
		return nil
	}
	result := make(types.Profile)
	for k, v := range src {
		result[k] = v
	}
	return result
}

func NewCandidateLog(address common.Address, processor types.ChangeLogProcessor, newVal types.Profile) *types.ChangeLog {
	account := processor.GetAccount(address)
	oldVal := cloneCandidateProfile(account.GetCandidate())
	newVal = cloneCandidateProfile(newVal)
	return &types.ChangeLog{
		LogType: CandidateLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(CandidateLog),
		OldVal:  &oldVal,
		NewVal:  &newVal,
	}
}

func redoCandidate(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(*types.Profile)
	if !ok {
		log.Errorf("redoCandidate expected NewVal *Profile, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetCandidate(*newVal)
	return nil
}

func undoCandidate(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(*types.Profile)
	if !ok {
		log.Errorf("undoCandidate expected OldVal *types.Profile, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetCandidate(*oldVal)
	return nil
}

func NewCandidateStateLog(address common.Address, processor types.ChangeLogProcessor, key string, newVal string) *types.ChangeLog {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: CandidateStateLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(CandidateStateLog),
		OldVal:  account.GetCandidateState(key),
		NewVal:  newVal,
		Extra:   key,
	}
}

func redoCandidateState(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(string)
	if !ok {
		log.Errorf("redoCandidateState expected NewVal string, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}

	key, ok := c.Extra.(string)
	if !ok {
		log.Errorf("redoCandidateState expected Extra string, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}

	accessor := processor.GetAccount(c.Address)
	accessor.SetCandidateState(key, newVal)
	return nil
}

func undoCandidateState(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(string)
	if !ok {
		log.Errorf("undoCandidateState expected NewVal string, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}

	key, ok := c.Extra.(string)
	if !ok {
		log.Errorf("undoCandidateState expected Extra string, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetCandidateState(key, oldVal)
	return nil
}

// NewBalanceLog records balance change
func NewBalanceLog(address common.Address, processor types.ChangeLogProcessor, newBalance *big.Int) *types.ChangeLog {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: BalanceLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(BalanceLog),
		OldVal:  *(new(big.Int).Set(account.GetBalance())),
		NewVal:  *(new(big.Int).Set(newBalance)),
	}
}

func redoBalance(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newValue, ok := c.NewVal.(big.Int)
	if !ok {
		log.Errorf("redoBalance expected NewVal big.Int, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetBalance(&newValue)
	return nil
}

func undoBalance(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldValue, ok := c.OldVal.(big.Int)
	if !ok {
		log.Errorf("undoBalance expected OldVal big.Int, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetBalance(&oldValue)
	return nil
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	result := make([]byte, len(src))
	copy(result, src)
	return result
}

// NewStorageLog records contract storage value change
func NewStorageLog(address common.Address, processor types.ChangeLogProcessor, key common.Hash, newVal []byte) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)
	oldValue, err := account.GetStorageState(key)
	if err != nil {
		return nil, fmt.Errorf("can't create storage log: %v", err)
	}
	return &types.ChangeLog{
		LogType: StorageLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(StorageLog),
		OldVal:  cloneBytes(oldValue),
		NewVal:  cloneBytes(newVal),
		Extra:   key,
	}, nil
}

func redoStorage(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.([]byte)
	if !ok {
		log.Errorf("redoStorage expected NewVal []byte, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	key, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("redoStorage expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetStorageState(key, newVal)
}

func undoStorage(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.([]byte)
	if !ok {
		log.Errorf("undoStorage expected OldVal []byte, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	key, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("undoStorage expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetStorageState(key, oldVal)
}

func NewStorageRootLog(address common.Address, processor types.ChangeLogProcessor, oldVal common.Hash, newVal common.Hash) *types.ChangeLog {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: StorageRootLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(StorageRootLog),
		OldVal:  oldVal,
		NewVal:  newVal,
	}
}

func redoStorageRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(common.Hash)
	if !ok {
		log.Errorf("redoStorageRoot expected NewVal common.Hash, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetStorageRoot(newVal)
	return nil
}

func undoStorageRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(common.Hash)
	if !ok {
		log.Errorf("redoStorageRoot expected OldVal common.Hash, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetStorageRoot(oldVal)
	return nil
}

func NewAssetCodeLog(address common.Address, processor types.ChangeLogProcessor, code common.Hash, asset *types.Asset) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)
	oldValue, err := account.GetAssetCode(code)
	if err != nil && err != store.ErrNotExist {
		return nil, fmt.Errorf("can't create asset log: %v", err)
	}

	return &types.ChangeLog{
		LogType: AssetCodeLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(AssetCodeLog),
		OldVal:  oldValue.Clone(),
		NewVal:  asset.Clone(),
		Extra:   code,
	}, nil
}

func redoAssetCode(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(*types.Asset)
	if !ok {
		log.Errorf("redoAssetCode expected NewVal *types.Asset, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	hash, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("redoAssetCode expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetCode(hash, newVal)
}

func undoAssetCode(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(*types.Asset)
	if !ok {
		log.Errorf("undoAssetCode expected OldVal *types.Asset, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	hash, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("undoAssetCode expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetCode(hash, oldVal)
}

func NewAssetCodeStateLog(address common.Address, processor types.ChangeLogProcessor, id common.Hash, key string, newVal string) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)

	oldVal, err := account.GetAssetCodeState(id, key)
	if err != nil && err != store.ErrNotExist {
		return nil, fmt.Errorf("can't create asset code state log: %v", err)
	}

	return &types.ChangeLog{
		LogType: AssetCodeStateLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(AssetCodeStateLog),
		OldVal:  oldVal,
		NewVal:  newVal,
		Extra: &ProfileChangeLogExtra{
			UUID: id,
			Key:  key,
		},
	}, nil
}

func redoAssetCodeState(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(string)
	if !ok {
		log.Errorf("redoAssetCodeState expected NewVal string, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	extra, ok := c.Extra.(*ProfileChangeLogExtra)
	if !ok {
		log.Errorf("redoAssetCodeState expected Extra ProfileChangeLogExtra, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}

	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetCodeState(extra.UUID, extra.Key, newVal)
}

func undoAssetCodeState(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(string)
	if !ok {
		log.Errorf("undoAssetCodeState expected OldVal string, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	extra, ok := c.Extra.(*ProfileChangeLogExtra)
	if !ok {
		log.Errorf("undoAssetCodeState expected Extra common.Token, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetCodeState(extra.UUID, extra.Key, oldVal)
}

func NewAssetCodeTotalSupplyLog(address common.Address, processor types.ChangeLogProcessor, code common.Hash, newVal *big.Int) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)

	oldVal, err := account.GetAssetCodeTotalSupply(code)
	if err != nil && err != store.ErrNotExist {
		return nil, fmt.Errorf("can't create total supply log: %v", err)
	}

	return &types.ChangeLog{
		LogType: AssetCodeTotalSupplyLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(AssetCodeTotalSupplyLog),
		OldVal:  *(new(big.Int).Set(oldVal)),
		NewVal:  *(new(big.Int).Set(newVal)),
		Extra:   code,
	}, nil
}

func redoAssetCodeTotalSupply(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(big.Int)
	if !ok {
		log.Errorf("redoAssetCodeTotalSupply expected NewVal big.Int, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}

	code, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("redoAssetCodeTotalSupply expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}

	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetCodeTotalSupply(code, &newVal)
}

func undoAssetCodeTotalSupply(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(big.Int)
	if !ok {
		log.Errorf("undoAssetCodeTotalSupply expected OldVal big.Int, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}

	code, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("undoAssetCodeTotalSupply expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}

	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetCodeTotalSupply(code, &oldVal)
}

func NewAssetCodeRootLog(address common.Address, processor types.ChangeLogProcessor, oldVal common.Hash, newVal common.Hash) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: AssetCodeRootLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(AssetCodeRootLog),
		OldVal:  oldVal,
		NewVal:  newVal,
	}, nil
}

func redoAssetCodeRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(common.Hash)
	if !ok {
		log.Errorf("redoAssetCodeRoot expected NewVal common.Hash, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetAssetCodeRoot(newVal)
	return nil
}

func undoAssetCodeRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(common.Hash)
	if !ok {
		log.Errorf("redoAssetCodeRoot expected OldVal common.Hash, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetAssetCodeRoot(oldVal)
	return nil
}

func NewAssetIdLog(address common.Address, processor types.ChangeLogProcessor, id common.Hash, newVal string) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)
	oldValue, err := account.GetAssetIdState(id)
	if err != nil && err != store.ErrNotExist {
		return nil, fmt.Errorf("can't create asset log: %v", err)
	}
	return &types.ChangeLog{
		LogType: AssetIdLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(AssetIdLog),
		OldVal:  oldValue,
		NewVal:  newVal,
		Extra:   id,
	}, nil
}

func redoAssetId(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(string)
	if !ok {
		log.Errorf("redoAssetId expected NewVal string, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	id, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("redoAssetId expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetIdState(id, newVal)
}

func undoAssetId(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(string)
	if !ok {
		log.Errorf("undoAssetId expected OldVal *types.Asset, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	id, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("undoAssetId expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetAssetIdState(id, oldVal)
}

func NewAssetIdRootLog(address common.Address, processor types.ChangeLogProcessor, oldVal common.Hash, newVal common.Hash) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: AssetIdRootLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(AssetIdRootLog),
		OldVal:  oldVal,
		NewVal:  newVal,
	}, nil
}

func redoAssetIdRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(common.Hash)
	if !ok {
		log.Errorf("redoAssetIdRoot expected NewVal common.Hash, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetAssetIdRoot(newVal)
	return nil
}

func undoAssetIdRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(common.Hash)
	if !ok {
		log.Errorf("undoAssetIdRoot expected OldVal common.Hash, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetAssetIdRoot(oldVal)
	return nil
}

func NewEquityLog(address common.Address, processor types.ChangeLogProcessor, id common.Hash, newVal *types.AssetEquity) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)

	oldValue, err := account.GetEquityState(id)
	if err != nil && err != store.ErrNotExist {
		return nil, fmt.Errorf("can't create equity log: %v", err)
	}

	log := &types.ChangeLog{
		LogType: EquityLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(EquityLog),
		Extra:   id,
	}

	if oldValue == nil {
		log.OldVal = nil
	} else {
		log.OldVal = oldValue.Clone()
	}

	if newVal == nil {
		log.NewVal = nil
	} else {
		log.NewVal = newVal.Clone()
	}

	return log, nil
}

func redoEquity(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(*types.AssetEquity)
	if !ok {
		log.Errorf("redoEquity expected NewVal *types.AssetEquity, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	id, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("redoEquity expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetEquityState(id, newVal)
}

func undoEquity(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(*types.AssetEquity)
	if !ok {
		log.Errorf("undoEquity expected OldVal *types.AssetEquity, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	id, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("undoEquity expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetEquityState(id, oldVal)
}

func NewEquityRootLog(address common.Address, processor types.ChangeLogProcessor, oldVal common.Hash, newVal common.Hash) (*types.ChangeLog, error) {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: EquityRootLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(EquityRootLog),
		OldVal:  oldVal,
		NewVal:  newVal,
	}, nil
}

func redoEquityRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(common.Hash)
	if !ok {
		log.Errorf("redoEquityRoot expected NewVal common.Hash, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetEquityRoot(newVal)
	return nil
}

func undoEquityRoot(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(common.Hash)
	if !ok {
		log.Errorf("undoEquityRoot expected OldVal common.Hash, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetEquityRoot(oldVal)
	return nil
}

// NewCodeLog records contract code setting
func NewCodeLog(address common.Address, processor types.ChangeLogProcessor, code types.Code) *types.ChangeLog {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: CodeLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(CodeLog),
		NewVal:  code,
	}
}

func redoCode(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	code, ok := c.NewVal.(types.Code)
	if !ok {
		log.Errorf("redoCode expected NewVal Code, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetCode(code)
	return nil
}

func undoCode(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	accessor := processor.GetAccount(c.Address)
	accessor.SetCode(nil)
	return nil
}

// NewAddEventLog records contract code change
func NewAddEventLog(address common.Address, processor types.ChangeLogProcessor, newEvent *types.Event) *types.ChangeLog {
	account := processor.GetAccount(address)
	return &types.ChangeLog{
		LogType: AddEventLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(AddEventLog),
		NewVal:  newEvent,
	}
}

func redoAddEvent(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newEvent, ok := c.NewVal.(*types.Event)
	if !ok {
		log.Errorf("redoAddEvent expected NewVal types.Event, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	account := processor.GetAccount(c.Address)
	account.PushEvent(newEvent)
	return nil
}

func undoAddEvent(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	// 	account := processor.GetAccount(c.Address)
	// 	return account.PopEvent()
	return nil
}

// NewSuicideLog records balance change
func NewSuicideLog(address common.Address, processor types.ChangeLogProcessor) *types.ChangeLog {
	account := processor.GetAccount(address)
	oldAccount := &types.AccountData{
		Balance:     new(big.Int).Set(account.GetBalance()),
		CodeHash:    account.GetCodeHash(),
		StorageRoot: account.GetStorageRoot(),
	}
	return &types.ChangeLog{
		LogType: SuicideLog,
		Address: account.GetAddress(),
		Version: account.GetNextVersion(SuicideLog),
		OldVal:  oldAccount,
	}
}

func redoSuicide(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	accessor := processor.GetAccount(c.Address)
	accessor.SetSuicide(true)
	return nil
}

func undoSuicide(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldValue, ok := c.OldVal.(*types.AccountData)
	if !ok {
		log.Errorf("undoSuicide expected OldVal big.Int, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetBalance(oldValue.Balance)
	accessor.SetCodeHash(oldValue.CodeHash)
	accessor.SetStorageRoot(oldValue.StorageRoot)
	accessor.SetSuicide(false)
	return nil
}
