package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"math/big"
)

// SafeAccount is used to record modifications with changelog. So that the modifications can be reverted
type SafeAccount struct {
	rawAccount *Account
	processor  *LogProcessor
}

// NewSafeAccount creates an account object.
func NewSafeAccount(processor *LogProcessor, account *Account) *SafeAccount {
	return &SafeAccount{
		rawAccount: account,
		processor:  processor,
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

func (a *SafeAccount) GetTxCount() uint32 { return a.rawAccount.GetTxCount() }

func (a *SafeAccount) SetTxCount(count uint32) {
	newLog := NewTxCountLog(a.processor, a.rawAccount, count)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetTxCount(count)
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

func (a *SafeAccount) GetCandidateProfile() types.Profile {
	return a.rawAccount.GetCandidateProfile()
}

func (a *SafeAccount) SetCandidateProfile(profile types.Profile) {
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

func (a *SafeAccount) GetStorageRoot() common.Hash { return a.rawAccount.GetStorageRoot() }

func (a *SafeAccount) SetStorageRoot(root common.Hash) {
	panic("SafeAccount.SetStorageRoot should not be called")
}

func (a *SafeAccount) GetAssetCodeRoot() common.Hash { return a.rawAccount.GetAssetCodeRoot() }

func (a *SafeAccount) SetAssetCodeRoot(root common.Hash) {
	panic("SafeAccount.SetAssetCodeRoot should not be called")
}

func (a *SafeAccount) GetAssetIdRoot() common.Hash { return a.rawAccount.GetAssetIdRoot() }

func (a *SafeAccount) SetAssetIdRoot(root common.Hash) {
	panic("SafeAccount.SetAssetIdRoot should not be called")
}

func (a *SafeAccount) GetEquityRoot() common.Hash { return a.rawAccount.GetEquityRoot() }

func (a *SafeAccount) SetEquityRoot(root common.Hash) {
	panic("SafeAccount.SetEquityRoot should not be called")
}

func (a *SafeAccount) GetStorageState(key common.Hash) ([]byte, error) {
	return a.rawAccount.GetStorageState(key)
}

func (a *SafeAccount) SetStorageState(key common.Hash, value []byte) error {
	newLog, err := NewStorageLog(a.processor, a.rawAccount, key, value)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(newLog)
	return a.rawAccount.SetStorageState(key, value)
}

func (a *SafeAccount) GetAssetCodeState(code common.Hash) (*types.Asset, error) {
	return a.rawAccount.GetAssetCodeState(code)
}

func (a *SafeAccount) SetAssetCodeState(code common.Hash, asset *types.Asset) error {
	newLog, err := NewAssetCodeLog(a.processor, a.rawAccount, code, asset)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(newLog)
	return a.rawAccount.SetAssetCodeState(code, asset)
}

func (a *SafeAccount) GetAssetIdState(id common.Hash) (string, error) {
	return a.rawAccount.GetAssetIdState(id)
}

func (a *SafeAccount) SetAssetIdState(id common.Hash, val string) error {
	newLog, err := NewAssetIdLog(a.processor, a.rawAccount, id, val)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(newLog)
	return a.rawAccount.SetAssetIdState(id, val)
}

func (a *SafeAccount) GetEquityState(id common.Hash) (*types.AssetEquity, error) {
	return a.rawAccount.GetEquityState(id)
}

func (a *SafeAccount) SetEquityState(id common.Hash, equity *types.AssetEquity) error {
	newLog, err := NewEquityLog(a.processor, a.rawAccount, id, equity)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(newLog)
	return a.rawAccount.SetEquityState(id, equity)
}

// func (a *SafeAccount) GetTxHashList() []common.Hash { return a.rawAccount.GetTxHashList() }

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

func (a *SafeAccount) IsDirty() bool {
	logs := a.processor.GetLogsByAddress(a.GetAddress())
	return len(logs) != 0
}
