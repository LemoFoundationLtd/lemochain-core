package account

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"math/big"
)

const (
	BalanceLog types.ChangeLogType = iota + 1
	StorageLog
	CodeLog
	AddEventLog
	SuicideLog
	VoteForLog
	VotesLog
	CandidateProfileLog
	TxCountLog
	LOG_TYPE_STOP
)

func init() {
	types.RegisterChangeLog(BalanceLog, "BalanceLog", decodeBigInt, decodeEmptyInterface, redoBalance, undoBalance)
	types.RegisterChangeLog(StorageLog, "StorageLog", decodeBytes, decodeBytes, redoStorage, undoStorage)
	types.RegisterChangeLog(CodeLog, "CodeLog", decodeCode, decodeEmptyInterface, redoCode, undoCode)
	types.RegisterChangeLog(AddEventLog, "AddEventLog", decodeEvent, decodeEmptyInterface, redoAddEvent, undoAddEvent)
	types.RegisterChangeLog(SuicideLog, "SuicideLog", decodeEmptyInterface, decodeEmptyInterface, redoSuicide, undoSuicide)
	types.RegisterChangeLog(VoteForLog, "VoteForLog", decodeAddress, decodeEmptyInterface, redoVoteFor, undoVoteFor)
	types.RegisterChangeLog(VotesLog, "VotesLog", decodeBigInt, decodeEmptyInterface, redoVotes, undoVotes)
	types.RegisterChangeLog(CandidateProfileLog, "CandidateProfileLog", decodeCandidateProfile, decodeEmptyInterface, redoCandidateProfile, undoCandidateProfile)
	types.RegisterChangeLog(TxCountLog, "TxCountLog", decodeUInt32, decodeEmptyInterface, redoTxCount, undoTxCount)
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
	case CandidateProfileLog:
		fallthrough
	default:
		valuable = log.OldVal != log.NewVal
	}
	return valuable
}

func isEmptyHash(hash common.Hash) bool {
	return hash == (common.Hash{}) || hash == sha3Nil
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

func decodeCandidateProfile(s *rlp.Stream) (interface{}, error) {
	_, size, _ := s.Kind()
	result := make(types.CandidateProfile)
	if size <= 0 {
		return &result, nil
	} else {
		err := s.Decode(&result)
		return &result, err
	}
}

func decodeUInt32(s *rlp.Stream) (interface{}, error) {
	var result uint32
	err := s.Decode(&result)
	return &result, err
}

//
// ChangeLog definitions
//

func NewVotesLog(processor types.ChangeLogProcessor, account types.AccountAccessor, newVotes *big.Int) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: VotesLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(VotesLog, account.GetAddress()),
		OldVal:  *(new(big.Int).Set(account.GetVotes())),
		NewVal:  *(new(big.Int).Set(newVotes)),
	}
}

func redoVotes(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newValue, ok := c.NewVal.(big.Int)
	if !ok {
		log.Errorf("expected NewVal big.Int, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVotes(&newValue)
	return nil
}

func undoVotes(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldValue, ok := c.OldVal.(big.Int)
	if !ok {
		log.Errorf("expected OldVal big.Int, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVotes(&oldValue)
	return nil
}

func NewVoteForLog(processor types.ChangeLogProcessor, account types.AccountAccessor, newVoteFor common.Address) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: VoteForLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(VoteForLog, account.GetAddress()),
		OldVal:  account.GetVoteFor(),
		NewVal:  newVoteFor,
	}
}

func redoVoteFor(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(common.Address)
	if !ok {
		log.Errorf("expected NewVal common.Address, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVoteFor(newVal)
	return nil
}

func undoVoteFor(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(common.Address)
	if !ok {
		log.Errorf("expected NewVal common.Address, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetVoteFor(oldVal)
	return nil
}

func cloneCandidateProfile(src types.CandidateProfile) types.CandidateProfile {
	if src == nil {
		return nil
	}
	result := make(types.CandidateProfile)
	for k, v := range src {
		result[k] = v
	}
	return result
}

func NewCandidateProfileLog(processor types.ChangeLogProcessor, account types.AccountAccessor, newProfile types.CandidateProfile) *types.ChangeLog {
	oldVal := cloneCandidateProfile(account.GetCandidateProfile())
	newProfile = cloneCandidateProfile(newProfile)
	return &types.ChangeLog{
		LogType: CandidateProfileLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(CandidateProfileLog, account.GetAddress()),
		OldVal:  &oldVal,
		NewVal:  &newProfile,
	}
}

func redoCandidateProfile(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.(*types.CandidateProfile)
	if !ok {
		log.Errorf("expected NewVal *CandidateProfile, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetCandidateProfile(*newVal)
	return nil
}

func undoCandidateProfile(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.(*types.CandidateProfile)
	if !ok {
		log.Errorf("expected NewVal map[string]string, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetCandidateProfile(*oldVal)
	return nil
}

func NewTxCountLog(processor types.ChangeLogProcessor, account types.AccountAccessor, newTxCount uint32) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: TxCountLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(TxCountLog, account.GetAddress()),
		OldVal:  account.GetTxCount(),
		NewVal:  newTxCount,
	}
}

func redoTxCount(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newValue, ok := c.NewVal.(uint32)
	if !ok {
		log.Errorf("expected NewVal uint32, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetTxCount(newValue)
	return nil
}

func undoTxCount(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldValue, ok := c.OldVal.(uint32)
	if !ok {
		log.Errorf("expected OldVal uint32, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetTxCount(oldValue)
	return nil
}

// NewBalanceLog records balance change
func NewBalanceLog(processor types.ChangeLogProcessor, account types.AccountAccessor, newBalance *big.Int) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: BalanceLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(BalanceLog, account.GetAddress()),
		OldVal:  *(new(big.Int).Set(account.GetBalance())),
		NewVal:  *(new(big.Int).Set(newBalance)),
	}
}

func redoBalance(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newValue, ok := c.NewVal.(big.Int)
	if !ok {
		log.Errorf("expected NewVal big.Int, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetBalance(&newValue)
	return nil
}

func undoBalance(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldValue, ok := c.OldVal.(big.Int)
	if !ok {
		log.Errorf("expected OldVal big.Int, got %T", c.OldVal)
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
func NewStorageLog(processor types.ChangeLogProcessor, account types.AccountAccessor, key common.Hash, newVal []byte) (*types.ChangeLog, error) {
	oldValue, err := account.GetStorageState(key)
	if err != nil {
		return nil, fmt.Errorf("can't create storage log: %v", err)
	}
	return &types.ChangeLog{
		LogType: StorageLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(StorageLog, account.GetAddress()),
		OldVal:  cloneBytes(oldValue),
		NewVal:  cloneBytes(newVal),
		Extra:   key,
	}, nil
}

func redoStorage(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newVal, ok := c.NewVal.([]byte)
	if !ok {
		log.Errorf("expected NewVal []byte, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	key, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetStorageState(key, newVal)
}

func undoStorage(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	oldVal, ok := c.OldVal.([]byte)
	if !ok {
		log.Errorf("expected NewVal []byte, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	key, ok := c.Extra.(common.Hash)
	if !ok {
		log.Errorf("expected Extra common.Hash, got %T", c.Extra)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	return accessor.SetStorageState(key, oldVal)
}

// NewCodeLog records contract code setting
func NewCodeLog(processor types.ChangeLogProcessor, account types.AccountAccessor, code types.Code) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: CodeLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(CodeLog, account.GetAddress()),
		NewVal:  code,
	}
}

func redoCode(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	code, ok := c.NewVal.(types.Code)
	if !ok {
		log.Errorf("expected NewVal Code, got %T", c.NewVal)
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
func NewAddEventLog(processor types.ChangeLogProcessor, account types.AccountAccessor, newEvent *types.Event) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: AddEventLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(AddEventLog, account.GetAddress()),
		NewVal:  newEvent,
	}
}

func redoAddEvent(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	newEvent, ok := c.NewVal.(*types.Event)
	if !ok {
		log.Errorf("expected NewVal types.Event, got %T", c.NewVal)
		return types.ErrWrongChangeLogData
	}
	processor.PushEvent(newEvent)
	return nil
}

func undoAddEvent(c *types.ChangeLog, processor types.ChangeLogProcessor) error {
	return processor.PopEvent()
}

// NewSuicideLog records balance change
func NewSuicideLog(processor types.ChangeLogProcessor, account types.AccountAccessor) *types.ChangeLog {
	oldAccount := &types.AccountData{
		Balance:     new(big.Int).Set(account.GetBalance()),
		CodeHash:    account.GetCodeHash(),
		StorageRoot: account.GetStorageRoot(),
	}
	return &types.ChangeLog{
		LogType: SuicideLog,
		Address: account.GetAddress(),
		Version: processor.GetNextVersion(SuicideLog, account.GetAddress()),
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
		log.Errorf("expected OldVal big.Int, got %T", c.OldVal)
		return types.ErrWrongChangeLogData
	}
	accessor := processor.GetAccount(c.Address)
	accessor.SetBalance(oldValue.Balance)
	accessor.SetCodeHash(oldValue.CodeHash)
	accessor.SetStorageRoot(oldValue.StorageRoot)
	accessor.SetSuicide(false)
	return nil
}
