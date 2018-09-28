package account

import (
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
)

func init() {
	types.RegisterChangeLog(BalanceLog, "BalanceLog", decodeBigInt, decodeEmptyInterface, redoBalance, undoBalance)
	types.RegisterChangeLog(StorageLog, "StorageLog", decodeBytes, decodeBytes, redoStorage, undoStorage)
	types.RegisterChangeLog(CodeLog, "CodeLog", decodeBytes, decodeEmptyInterface, redoCode, undoCode)
	types.RegisterChangeLog(AddEventLog, "AddEventLog", decodeEvent, decodeEmptyInterface, redoAddEvent, undoAddEvent)
	types.RegisterChangeLog(SuicideLog, "SuicideLog", decodeEmptyInterface, decodeEmptyInterface, redoSuicide, undoSuicide)
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

// decodeUint64 decode an interface which contains an uint64
func decodeUint64(s *rlp.Stream) (interface{}, error) {
	var result uint64
	err := s.Decode(&result)
	return result, err
}

// decodeHash decode an interface which contains an common.Hash
func decodeHash(s *rlp.Stream) (interface{}, error) {
	var result common.Hash
	err := s.Decode(&result)
	return result, err
}

// decodeBytes decode an interface which contains an []byte
func decodeBytes(s *rlp.Stream) (interface{}, error) {
	var result []byte
	err := s.Decode(&result)
	return result, err
}

// decodeEvents decode an interface which contains an *types.Event
func decodeEvent(s *rlp.Stream) (interface{}, error) {
	var result types.Event
	err := s.Decode(&result)
	return &result, err
}

//
// ChangeLog definitions
//

// increaseVersion increases account version by one
func increaseVersion(account types.AccountAccessor) uint32 {
	newVersion := account.GetVersion() + 1
	account.SetVersion(newVersion)
	return newVersion
}

// NewBalanceLog records balance change
func NewBalanceLog(account types.AccountAccessor, newBalance *big.Int) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: BalanceLog,
		Address: account.GetAddress(),
		Version: increaseVersion(account),
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

// NewStorageLog records contract storage value change
func NewStorageLog(account types.AccountAccessor, key common.Hash, newVal []byte) (*types.ChangeLog, error) {
	oldValue, err := account.GetStorageState(key)
	if err != nil {
		return nil, fmt.Errorf("can't create storage log: %v", err)
	}
	return &types.ChangeLog{
		LogType: StorageLog,
		Address: account.GetAddress(),
		Version: increaseVersion(account),
		OldVal:  oldValue,
		NewVal:  newVal,
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
	accessor.SetStorageState(key, newVal)
	return nil
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
	accessor.SetStorageState(key, oldVal)
	return nil
}

// NewCodeLog records contract code setting
func NewCodeLog(account types.AccountAccessor, code types.Code) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: CodeLog,
		Address: account.GetAddress(),
		Version: increaseVersion(account),
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
func NewAddEventLog(account types.AccountAccessor, newEvent *types.Event) *types.ChangeLog {
	return &types.ChangeLog{
		LogType: AddEventLog,
		Address: account.GetAddress(),
		Version: increaseVersion(account),
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
func NewSuicideLog(account types.AccountAccessor) *types.ChangeLog {
	oldAccount := &types.AccountData{
		Balance:     new(big.Int).Set(account.GetBalance()),
		CodeHash:    account.GetCodeHash(),
		StorageRoot: account.GetStorageRoot(),
	}
	return &types.ChangeLog{
		LogType: SuicideLog,
		Address: account.GetAddress(),
		Version: increaseVersion(account),
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
