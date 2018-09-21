package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

// SafeAccount is used to record modifications with changelog. So that the modifications can be reverted
type SafeAccount struct {
	rawAccount  *Account
	processor   *logProcessor
	origVersion uint32 // the version in Account from beginning
}

// NewSafeAccount creates an account object.
func NewSafeAccount(processor *logProcessor, account *Account) *SafeAccount {
	return &SafeAccount{
		rawAccount:  account,
		processor:   processor,
		origVersion: account.GetVersion(),
	}
}

func (a *SafeAccount) GetAddress() common.Address   { return a.rawAccount.GetAddress() }
func (a *SafeAccount) GetBalance() *big.Int         { return a.rawAccount.GetBalance() }
func (a *SafeAccount) GetVersion() uint32           { return a.rawAccount.GetVersion() }
func (a *SafeAccount) GetCodeHash() common.Hash     { return a.rawAccount.GetCodeHash() }
func (a *SafeAccount) GetCode() (types.Code, error) { return a.rawAccount.GetCode() }
func (a *SafeAccount) GetStorageRoot() common.Hash  { return a.rawAccount.GetStorageRoot() }
func (a *SafeAccount) GetStorageState(key common.Hash) ([]byte, error) {
	return a.rawAccount.GetStorageState(key)
}

// overwrite Account.SetXXX. Access Account with changelog
func (a *SafeAccount) SetBalance(balance *big.Int) {
	a.processor.PushChangeLog(NewBalanceLog(a.rawAccount, balance))
	a.rawAccount.SetBalance(balance)
}

func (a *SafeAccount) SetVersion(version uint32) {
	panic("SafeAccount.SetVersion should not be called")
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
	// the version in a.rawAccount has been changed in NewXXXLog()
	return a.rawAccount.GetVersion() != a.origVersion
}
