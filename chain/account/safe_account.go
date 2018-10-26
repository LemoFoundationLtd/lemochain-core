package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

// SafeAccount is used to record modifications with changelog. So that the modifications can be reverted
type SafeAccount struct {
	rawAccount   *Account
	processor    *logProcessor
	origVersions map[types.ChangeLogType]uint32 // the versions in Account from beginning
	origTxCount  int
}

// NewSafeAccount creates an account object.
func NewSafeAccount(processor *logProcessor, account *Account) *SafeAccount {
	origVersions := make(map[types.ChangeLogType]uint32)
	for logType, record := range account.data.NewestRecords {
		origVersions[logType] = record.Version
	}
	return &SafeAccount{
		rawAccount:   account,
		processor:    processor,
		origVersions: origVersions,
		origTxCount:  len(account.GetTxHashList()),
	}
}

// MarshalJSON encodes the lemoClient RPC safeAccount format.
func (a *SafeAccount) MarshalJSON() ([]byte, error) {
	return a.rawAccount.MarshalJSON()
}

// UnmarshalJSON decodes the lemoClient RPC safeAccount format.
func (a *SafeAccount) UnmarshalJSON(input []byte) error {
	var dec Account
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	// TODO a.processor is nil
	*a = *NewSafeAccount(a.processor, &dec)
	return nil
}

func (a *SafeAccount) String() string {
	return a.rawAccount.String()
}

func (a *SafeAccount) GetAddress() common.Address { return a.rawAccount.GetAddress() }
func (a *SafeAccount) GetBalance() *big.Int       { return a.rawAccount.GetBalance() }
func (a *SafeAccount) GetVersion(logType types.ChangeLogType) uint32 {
	return a.rawAccount.GetVersion(logType)
}
func (a *SafeAccount) GetSuicide() bool             { return a.rawAccount.GetSuicide() }
func (a *SafeAccount) GetCodeHash() common.Hash     { return a.rawAccount.GetCodeHash() }
func (a *SafeAccount) GetCode() (types.Code, error) { return a.rawAccount.GetCode() }
func (a *SafeAccount) IsEmpty() bool                { return a.rawAccount.IsEmpty() }
func (a *SafeAccount) GetStorageRoot() common.Hash  { return a.rawAccount.GetStorageRoot() }
func (a *SafeAccount) GetStorageState(key common.Hash) ([]byte, error) {
	return a.rawAccount.GetStorageState(key)
}
func (a *SafeAccount) GetBaseHeight() uint32        { return a.rawAccount.baseHeight }
func (a *SafeAccount) GetTxHashList() []common.Hash { return a.rawAccount.GetTxHashList() }

// overwrite Account.SetXXX. Access Account with changelog
func (a *SafeAccount) SetBalance(balance *big.Int) {
	a.processor.PushChangeLog(NewBalanceLog(a.rawAccount, balance))
	a.rawAccount.SetBalance(balance)
}

func (a *SafeAccount) SetVersion(logType types.ChangeLogType, version uint32) {
	panic("SafeAccount.SetVersion should not be called")
}

func (a *SafeAccount) SetSuicide(suicided bool) {
	a.processor.PushChangeLog(NewSuicideLog(a.rawAccount))
	a.rawAccount.SetSuicide(suicided)
}

func (a *SafeAccount) SetCodeHash(codeHash common.Hash) {
	panic("SafeAccount.SetCodeHash should not be called")
}

func (a *SafeAccount) SetCode(code types.Code) {
	a.processor.PushChangeLog(NewCodeLog(a.rawAccount, code))
	a.rawAccount.SetCode(code)
}

func (a *SafeAccount) SetStorageRoot(root common.Hash) {
	panic("SafeAccount.SetStorageRoot should not be called")
}

func (a *SafeAccount) SetStorageState(key common.Hash, value []byte) error {
	log, err := NewStorageLog(a.rawAccount, key, value)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(log)
	a.rawAccount.SetStorageState(key, value)
	return nil
}

func (a *SafeAccount) IsDirty() bool {
	if a.origTxCount != len(a.GetTxHashList()) {
		return true
	}
	// the version in a.rawAccount has been changed in NewXXXLog()
	if len(a.origVersions) != len(a.rawAccount.data.NewestRecords) {
		return true
	}
	for logType, version := range a.origVersions {
		if version != a.rawAccount.GetVersion(logType) {
			return true
		}
	}
	return false
}

func (a *SafeAccount) AppendTx(hash common.Hash) {
	for _, exist := range a.rawAccount.data.TxHashList {
		if exist == hash {
			return
		}
	}
	a.rawAccount.data.TxHashList = append(a.rawAccount.data.TxHashList, hash)
}
