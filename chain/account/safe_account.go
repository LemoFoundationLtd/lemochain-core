package account

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"math/big"
)

// SafeAccount is used to record modifications with changelog. So that the modifications can be reverted
type SafeAccount struct {
	rawAccount *Account
	processor  *LogProcessor
}

func (a *SafeAccount) SetSingers(signers types.Signers) error {
	return a.rawAccount.SetSingers(signers)
}

func (a *SafeAccount) GetSigners() types.Signers {
	return a.rawAccount.GetSigners()
}

func (a *SafeAccount) GetNextVersion(logType types.ChangeLogType) uint32 {
	return a.rawAccount.GetNextVersion(logType)
}

func (a *SafeAccount) PushEvent(event *types.Event) {
	newLog := NewAddEventLog(a.GetAddress(), a.processor, event)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.PushEvent(event)
}

func (a *SafeAccount) PopEvent() error {
	return a.rawAccount.PopEvent()
}

func (a *SafeAccount) GetEvents() []*types.Event {
	return a.rawAccount.GetEvents()
}

func (a *SafeAccount) GetCandidate() types.Profile { return a.rawAccount.GetCandidate() }

func (a *SafeAccount) SetCandidate(profile types.Profile) {
	newLog := NewCandidateLog(a.GetAddress(), a.processor, profile)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetCandidate(profile)
}

func (a *SafeAccount) GetCandidateState(key string) string {
	return a.rawAccount.GetCandidateState(key)
}

func (a *SafeAccount) SetCandidateState(key string, val string) {
	newLog := NewCandidateStateLog(a.GetAddress(), a.processor, key, val)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetCandidateState(key, val)
}

func (a *SafeAccount) GetAssetCode(code common.Hash) (*types.Asset, error) {
	return a.rawAccount.GetAssetCode(code)
}

func (a *SafeAccount) SetAssetCode(code common.Hash, asset *types.Asset) error {
	newLog, err := NewAssetCodeLog(a.GetAddress(), a.processor, code, asset)
	if err != nil {
		return err
	} else {
		a.processor.PushChangeLog(newLog)
		return a.rawAccount.SetAssetCode(code, asset)
	}
}

func (a *SafeAccount) GetAssetCodeTotalSupply(code common.Hash) (*big.Int, error) {
	return a.rawAccount.GetAssetCodeTotalSupply(code)
}

func (a *SafeAccount) SetAssetCodeTotalSupply(code common.Hash, val *big.Int) error {
	newLog, err := NewAssetCodeTotalSupplyLog(a.GetAddress(), a.processor, code, val)
	if err != nil {
		return err
	} else {
		a.processor.PushChangeLog(newLog)
		return a.rawAccount.SetAssetCodeTotalSupply(code, val)
	}
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

func (a *SafeAccount) GetVoteFor() common.Address { return a.rawAccount.GetVoteFor() }

func (a *SafeAccount) SetVoteFor(addr common.Address) {
	newLog := NewVoteForLog(a.GetAddress(), a.processor, addr)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetVoteFor(addr)
}

func (a *SafeAccount) GetVotes() *big.Int {
	return a.rawAccount.GetVotes()
}

func (a *SafeAccount) SetVotes(votes *big.Int) {
	newLog := NewVotesLog(a.GetAddress(), a.processor, votes)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetVotes(votes)
}

func (a *SafeAccount) GetAddress() common.Address { return a.rawAccount.GetAddress() }
func (a *SafeAccount) GetBalance() *big.Int       { return a.rawAccount.GetBalance() }

// GetBaseVersion returns the version of specific change log from the base block. It is not changed by tx processing until the finalised
func (a *SafeAccount) GetVersion(logType types.ChangeLogType) uint32 {
	return a.rawAccount.GetVersion(logType)
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
	newLog, err := NewStorageLog(a.GetAddress(), a.processor, key, value)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(newLog)
	return a.rawAccount.SetStorageState(key, value)
}

func (a *SafeAccount) GetAssetCodeState(code common.Hash, key string) (string, error) {
	return a.rawAccount.GetAssetCodeState(code, key)
}

func (a *SafeAccount) SetAssetCodeState(code common.Hash, key string, val string) error {
	newLog, err := NewAssetCodeStateLog(a.GetAddress(), a.processor, code, key, val)
	if err != nil {
		return err
	} else {
		a.processor.PushChangeLog(newLog)
		return a.rawAccount.SetAssetCodeState(code, key, val)
	}
}

func (a *SafeAccount) GetAssetIdState(id common.Hash) (string, error) {
	return a.rawAccount.GetAssetIdState(id)
}

func (a *SafeAccount) SetAssetIdState(id common.Hash, val string) error {
	newLog, err := NewAssetIdLog(a.GetAddress(), a.processor, id, val)
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
	newLog, err := NewEquityLog(a.GetAddress(), a.processor, id, equity)
	if err != nil {
		return err
	}
	a.processor.PushChangeLog(newLog)
	return a.rawAccount.SetEquityState(id, equity)
}

// overwrite Account.SetXXX. Access Account with changelog
func (a *SafeAccount) SetBalance(balance *big.Int) {
	newLog := NewBalanceLog(a.GetAddress(), a.processor, balance)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetBalance(balance)
}

func (a *SafeAccount) SetSuicide(suicided bool) {
	newLog := NewSuicideLog(a.GetAddress(), a.processor)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetSuicide(suicided)
}

func (a *SafeAccount) SetCodeHash(codeHash common.Hash) {
	panic("SafeAccount.SetCodeHash should not be called")
}

func (a *SafeAccount) SetCode(code types.Code) {
	newLog := NewCodeLog(a.GetAddress(), a.processor, code)
	a.processor.PushChangeLog(newLog)
	a.rawAccount.SetCode(code)
}
