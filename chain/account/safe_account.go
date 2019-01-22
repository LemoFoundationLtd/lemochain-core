package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

// SafeAccount is used to record modifications with changelog. So that the modifications can be reverted
type SafeAccount struct {
	rawAccount  *Account
	processor   *LogProcessor
	origTxCount int
}

// NewSafeAccount creates an account object.
func NewSafeAccount(processor *LogProcessor, account *Account) *SafeAccount {
	return &SafeAccount{
		rawAccount:  account,
		processor:   processor,
		origTxCount: len(account.GetTxHashList()),
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

func (a *SafeAccount) GetVoteFor() common.Address { return a.rawAccount.GetVoteFor() }

func (a *SafeAccount) SetVoteFor(addr common.Address) {
	newLog := NewVoteForLog(a.processor, a.rawAccount, addr)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetVoteFor(addr)
}

func (a *SafeAccount) GetVotes() *big.Int {
	return a.rawAccount.GetVotes()
}

func (a *SafeAccount) SetVotes(votes *big.Int) {
	newLog := NewVotesLog(a.processor, a.rawAccount, votes)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetVotes(votes)
}

func (a *SafeAccount) GetCandidateProfile() *types.CandidateProfile {
	return a.rawAccount.GetCandidateProfile()
}

func (a *SafeAccount) SetCandidateProfile(profile *types.CandidateProfile) {
	newLog := NewCandidateProfileLog(a.processor, a.rawAccount, profile)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetCandidateProfile(profile)
}

func (a *SafeAccount) GetAddress() common.Address { return a.rawAccount.GetAddress() }
func (a *SafeAccount) GetBalance() *big.Int       { return a.rawAccount.GetBalance() }

// GetBaseVersion returns the version of specific change log from the base block. It is not changed by tx processing until the finalised
func (a *SafeAccount) GetBaseVersion(logType types.ChangeLogType) uint32 {
	return a.rawAccount.GetBaseVersion(logType)
}
func (a *SafeAccount) GetSuicide() bool             { return a.rawAccount.GetSuicide() }
func (a *SafeAccount) GetCodeHash() common.Hash     { return a.rawAccount.GetCodeHash() }
func (a *SafeAccount) GetCode() (types.Code, error) { return a.rawAccount.GetCode() }
func (a *SafeAccount) IsEmpty() bool                { return a.rawAccount.IsEmpty() }
func (a *SafeAccount) GetStorageRoot() common.Hash  { return a.rawAccount.GetStorageRoot() }
func (a *SafeAccount) GetStorageState(key common.Hash) ([]byte, error) {
	return a.rawAccount.GetStorageState(key)
}
func (a *SafeAccount) GetTxHashList() []common.Hash { return a.rawAccount.GetTxHashList() }

// overwrite Account.SetXXX. Access Account with changelog
func (a *SafeAccount) SetBalance(balance *big.Int) {
	newLog := NewBalanceLog(a.processor, a.rawAccount, balance)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetBalance(balance)
}

func (a *SafeAccount) SetSuicide(suicided bool) {
	newLog := NewSuicideLog(a.processor, a.rawAccount)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetSuicide(suicided)
}

func (a *SafeAccount) SetCodeHash(codeHash common.Hash) {
	panic("SafeAccount.SetCodeHash should not be called")
}

func (a *SafeAccount) SetCode(code types.Code) {
	newLog := NewCodeLog(a.processor, a.rawAccount, code)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetCode(code)
}

func (a *SafeAccount) SetStorageRoot(root common.Hash) {
	panic("SafeAccount.SetStorageRoot should not be called")
}

func (a *SafeAccount) SetStorageState(key common.Hash, value []byte) error {
	newLog, err := NewStorageLog(a.processor, a.rawAccount, key, value)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(newLog)
	return a.rawAccount.SetStorageState(key, value)
}

func (a *SafeAccount) IsDirty() bool {
	if a.origTxCount != len(a.GetTxHashList()) {
		return true
	}
	logs := a.processor.GetLogsByAddress(a.GetAddress())
	return len(logs) != 0
}

func (a *SafeAccount) AppendTx(hash common.Hash) {
	a.rawAccount.AppendTx(hash)
}
