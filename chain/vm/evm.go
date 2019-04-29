package vm

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/LemoFoundationLtd/lemochain-core/store/protocol"
	"math/big"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

type (
	CanTransferFunc func(AccountManager, common.Address, *big.Int) bool
	TransferFunc    func(AccountManager, common.Address, common.Address, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint32) common.Hash
)

var (
	ErrIssueAssetAmount     = errors.New("issue asset amount can't be 0 or nil")
	ErrIssueAssetMetaData   = errors.New("the length of metaData more than limit")
	ErrReplenishAssetAmount = errors.New("replenish asset amount can't be 0 or nil")
	ErrFrozenAsset          = errors.New("can't replenish the frozen assets")
	ErrIsReplenishable      = errors.New("asset's \"IsReplenishable\" is false")
	ErrIsDivisible          = errors.New("this \"isDivisible == false\" kind of asset can't be replenished")
	ErrNotEqualAssetCode    = errors.New("assetCode not equal")
	ErrModifyAssetInfo      = errors.New("the struct of ModifyAssetInfo's Info can't be nil")
	ErrMarshalAssetLength   = errors.New("the length of data by marshal asset more than max length")
	ErrAssetEquity          = errors.New("asset equity can't be nil or 0")
	ErrTransferFrozenAsset  = errors.New("cannot trade frozen assets")
	ErrAssetCategory        = errors.New("assert's Category not exist")
	ErrAgainRegister        = errors.New("cannot register again after unregistering")
)

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *EVM, contract *Contract, input []byte) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContracts
		if p := precompiles[*contract.CodeAddr]; p != nil {
			// Determine whether the address to set the reward value is correct
			if *contract.CodeAddr == params.TermRewardContract && contract.caller.GetAddress() != evm.vmConfig.RewardManager {
				return nil, errors.New("Insufficient permission to call this Precompiled contract. ")
			}
			return RunPrecompiledContract(p, input, contract, evm)
		}
	}
	return evm.interpreter.Run(contract, input)
}

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient lemo to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers lemo from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Provides the current transaction hash which is used when EVM emits new contract events.
	TxIndex   uint
	TxHash    common.Hash
	BlockHash common.Hash

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	MinerAddress common.Address // Provides information for MinerAddress
	GasLimit     uint64         // Provides information for GASLIMIT
	BlockHeight  uint32         // Provides information for HEIGHT
	Time         uint32         // Provides information for TIME
}

// EVM is the Lemochain Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// am gives access to the underlying state
	am AccountManager
	// Depth is the current call stack
	depth int

	// virtual machine configuration options used to initialise the
	// evm.
	vmConfig Config
	// global (to this context) lemochain virtual machine
	// used throughout the execution of the tx.
	interpreter *Interpreter
	// abort is used to abort the EVM calling operations
	// NOTE: must be set atomically
	abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	callGasTemp uint64
}

// NewEVM returns a new EVM. The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVM(ctx Context, am AccountManager, vmConfig Config) *EVM {
	evm := &EVM{
		Context:  ctx,
		am:       am,
		vmConfig: vmConfig,
	}

	evm.interpreter = NewInterpreter(evm, vmConfig)
	return evm
}

// Cancel cancels any running EVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.Context.CanTransfer(evm.am, caller.GetAddress(), value) {
		return nil, gas, ErrInsufficientBalance
	}
	contractAccount := evm.am.GetAccount(addr)
	code, err := contractAccount.GetCode()
	if err != nil {
		return nil, gas, ErrContractCodeLoadFail
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.am.Snapshot()
	)
	if len(code) == 0 && PrecompiledContracts[addr] == nil && value.Sign() == 0 {
		return nil, gas, nil
	}
	evm.Transfer(evm.am, caller.GetAddress(), to.GetAddress(), value)
	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, contractAccount.GetCodeHash(), code)

	start := time.Now()

	// Capture the tracer start/end events in debug mode
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.GetAddress(), addr, false, input, gas, value)

		defer func() { // Lazy evaluation of the parameters
			evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}
	ret, err = run(evm, contract, input)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallVoteTx voting transaction call
func (evm *EVM) CallVoteTx(voter, node common.Address, gas uint64, initialBalance *big.Int) (leftgas uint64, err error) {
	nodeAccount := evm.am.GetAccount(node)

	profile := nodeAccount.GetCandidate()
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	if !ok || IsCandidate == params.NotCandidateNode {
		return gas, ErrOfNotCandidateNode
	}
	// var snapshot = evm.am.Snapshot()
	voterAccount := evm.am.GetAccount(voter)
	// Determine if the account has already voted
	if (voterAccount.GetVoteFor() != common.Address{}) {
		if voterAccount.GetVoteFor() == node {
			return gas, ErrOfAgainVote
		} else {
			oldNode := voterAccount.GetVoteFor()
			newNodeAccount := nodeAccount
			// Change in votes
			oldNodeAccount := evm.am.GetAccount(oldNode)
			// reduce the number of votes for old candidate nodes
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialBalance)
			oldNodeAccount.SetVotes(oldNodeVoters)
			// Increase the number of votes for new candidate nodes
			newNodeVoters := new(big.Int).Add(newNodeAccount.GetVotes(), initialBalance)
			newNodeAccount.SetVotes(newNodeVoters)
		}
	} else { // First vote
		// Increase the number of votes for candidate nodes
		nodeVoters := new(big.Int).Add(nodeAccount.GetVotes(), initialBalance)
		nodeAccount.SetVotes(nodeVoters)
	}
	// Set up voter account
	voterAccount.SetVoteFor(node)

	// if err != nil {
	// 	evm.am.RevertToSnapshot(snapshot)
	// }
	return gas, err
}

// RegisterOrUpdateToCandidate candidate node account transaction call
func (evm *EVM) RegisterOrUpdateToCandidate(candidateAddress, to common.Address, candiNode types.Profile, gas uint64, initialSenderBalance *big.Int) (leftgas uint64, err error) {
	// Candidate node information
	newIsCandidate, ok := candiNode[types.CandidateKeyIsCandidate]
	if !ok {
		newIsCandidate = params.IsCandidateNode
	}
	minerAddress, ok := candiNode[types.CandidateKeyIncomeAddress]
	if !ok {
		minerAddress = candidateAddress.String()
	}
	nodeID, ok := candiNode[types.CandidateKeyNodeID]
	if !ok {
		return gas, ErrOfRegisterNodeID
	}
	host, ok := candiNode[types.CandidateKeyHost]
	if !ok {
		return gas, ErrOfRegisterHost
	}
	port, ok := candiNode[types.CandidateKeyPort]
	if !ok {
		return gas, ErrOfRegisterPort
	}
	// Checking the balance is not enough
	if !evm.CanTransfer(evm.am, candidateAddress, params.RegisterCandidateNodeFees) {
		return gas, ErrInsufficientBalance
	}
	// var snapshot = evm.am.Snapshot()

	// Register as a candidate node account
	nodeAccount := evm.am.GetAccount(candidateAddress)
	// Check if the application address is already a candidate proxy node.
	profile := nodeAccount.GetCandidate()
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	// Set candidate node information if it is already a candidate node account
	if ok && IsCandidate == params.IsCandidateNode {
		// Determine whether to disqualify a candidate node
		if newIsCandidate == params.NotCandidateNode {
			profile[types.CandidateKeyIsCandidate] = params.NotCandidateNode
			nodeAccount.SetCandidate(profile)
			// Set the number of votes to 0
			nodeAccount.SetVotes(big.NewInt(0))
			// Transaction costs
			evm.Transfer(evm.am, candidateAddress, to, params.RegisterCandidateNodeFees)
			return gas, nil
		}

		profile[types.CandidateKeyIncomeAddress] = minerAddress
		profile[types.CandidateKeyHost] = host
		profile[types.CandidateKeyPort] = port
		nodeAccount.SetCandidate(profile)
	} else if ok && IsCandidate == params.NotCandidateNode {
		return gas, ErrAgainRegister
	} else {
		// Register candidate nodes
		newProfile := make(map[string]string, 5)
		newProfile[types.CandidateKeyIsCandidate] = params.IsCandidateNode
		newProfile[types.CandidateKeyIncomeAddress] = minerAddress
		newProfile[types.CandidateKeyNodeID] = nodeID
		newProfile[types.CandidateKeyHost] = host
		newProfile[types.CandidateKeyPort] = port
		nodeAccount.SetCandidate(newProfile)

		oldNodeAddress := nodeAccount.GetVoteFor()

		if (oldNodeAddress != common.Address{}) {
			oldNodeAccount := evm.am.GetAccount(oldNodeAddress)
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialSenderBalance)
			oldNodeAccount.SetVotes(oldNodeVoters)
		}
		nodeAccount.SetVoteFor(candidateAddress)
		nodeAccount.SetVotes(initialSenderBalance)

	}
	evm.Transfer(evm.am, candidateAddress, to, params.RegisterCandidateNodeFees)
	// if err != nil {
	// 	evm.am.RevertToSnapshot(snapshot)
	// }
	return gas, nil
}

// CreateAssetTx
func (evm *EVM) CreateAssetTx(sender common.Address, data []byte, txHash common.Hash) error {
	var err error
	issuerAcc := evm.am.GetAccount(sender)
	asset := &types.Asset{}
	err = json.Unmarshal(data, asset)
	if err != nil {
		return err
	}
	newAss := asset.Clone()
	newAss.Issuer = sender
	newAss.AssetCode = txHash
	newAss.TotalSupply = big.NewInt(0) // init 0
	newData, err := json.Marshal(newAss)
	if err != nil {
		return err
	}
	// judge data's length
	if len(newData) > types.MaxMarshalAssetLength {
		log.Errorf("the length of data by marshal asset more than max length,len(data) = %d ", len(newData))
		return ErrMarshalAssetLength
	}
	var snapshot = evm.am.Snapshot()
	err = issuerAcc.SetAssetCode(newAss.AssetCode, newAss)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
	}
	return err
}

// IssueAssetTx
func (evm *EVM) IssueAssetTx(sender, receiver common.Address, txHash common.Hash, data []byte) error {

	issueAsset := &types.IssueAsset{}
	err := json.Unmarshal(data, issueAsset)
	if err != nil {
		return err
	}
	// metaData length limit
	if len(issueAsset.MetaData) > types.MaxMetaDataLength {
		log.Errorf("the length of metaData more than limit, len(metaData) = %d ", len(issueAsset.MetaData))
		return ErrIssueAssetMetaData
	}
	// amount != nil && amount > 0
	if issueAsset.Amount == nil || issueAsset.Amount.Cmp(big.NewInt(0)) <= 0 {
		log.Errorf("issue asset amount must > 0 , currentAmount = %s", issueAsset.Amount.String())
		return ErrIssueAssetAmount
	}
	assetCode := issueAsset.AssetCode
	issuerAcc := evm.am.GetAccount(sender)
	asset, err := issuerAcc.GetAssetCode(assetCode)
	if err != nil {
		return err
	}
	// Determine whether asset is frozen
	isStop, err := issuerAcc.GetAssetCodeState(assetCode, types.AssetStop)
	if err == nil && isStop == "true" {
		return errors.New("Can't issue the frozen assets. ")
	}
	recAcc := evm.am.GetAccount(receiver)
	equity := &types.AssetEquity{}
	equity.AssetCode = asset.AssetCode
	equity.Equity = issueAsset.Amount

	// judge assert type
	AssType := asset.Category
	if AssType == types.Asset01 { // ERC20
		equity.AssetId = asset.AssetCode
	} else if AssType == types.Asset02 || AssType == types.Asset03 { // ERC721 or ERC721+20
		equity.AssetId = txHash
	} else {
		log.Errorf("Assert's Category not exist ,Category = %d ", AssType)
		return ErrAssetCategory
	}
	var snapshot = evm.am.Snapshot()
	newAsset := asset.Clone()
	// add totalSupply
	var oldTotalSupply *big.Int
	var newTotalSupply *big.Int
	oldTotalSupply, err = issuerAcc.GetAssetCodeTotalSupply(assetCode)
	if err != nil {
		return err
	}
	if !newAsset.IsDivisible {
		newTotalSupply = new(big.Int).Add(oldTotalSupply, big.NewInt(1))
	} else {
		newTotalSupply = new(big.Int).Add(oldTotalSupply, issueAsset.Amount)
	}
	// set new totalSupply
	err = issuerAcc.SetAssetCodeTotalSupply(assetCode, newTotalSupply)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return err
	}
	// set new asset equity for receiver
	err = recAcc.SetEquityState(equity.AssetId, equity)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return err
	}
	err = recAcc.SetAssetIdState(equity.AssetId, issueAsset.MetaData)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return err
	}
	return nil
}

// ReplenishAssetTx
func (evm *EVM) ReplenishAssetTx(sender, receiver common.Address, data []byte) error {
	repl := &types.ReplenishAsset{}
	err := json.Unmarshal(data, repl)
	if err != nil {
		return err
	}
	newAssetCode := repl.AssetCode
	newAssetId := repl.AssetId
	addAmount := repl.Amount
	// amount > 0
	if addAmount == nil || addAmount.Cmp(big.NewInt(0)) <= 0 {
		log.Errorf("Replenish asset amount must > 0,currentAmount = %s", addAmount.String())
		return ErrReplenishAssetAmount
	}
	// assert issuer account
	issuerAcc := evm.am.GetAccount(sender)
	asset, err := issuerAcc.GetAssetCode(newAssetCode)
	if err != nil {
		return err
	}
	// Determine whether asset is frozen
	isStop, err := issuerAcc.GetAssetCodeState(newAssetCode, types.AssetStop)
	if err == nil && isStop == "true" {
		return ErrFrozenAsset
	}
	// judge IsReplenishable
	if !asset.IsReplenishable {
		return ErrIsReplenishable
	}
	// erc721 asset can't be replenished
	if !asset.IsDivisible {
		return ErrIsDivisible
	}
	// receiver account
	recAcc := evm.am.GetAccount(receiver)
	equity, err := recAcc.GetEquityState(newAssetId)
	if err != nil && err != store.ErrNotExist {
		return err
	}
	if err == store.ErrNotExist {
		equity = &types.AssetEquity{
			AssetCode: newAssetCode,
			AssetId:   newAssetId,
			Equity:    big.NewInt(0),
		}
	}

	if newAssetCode != equity.AssetCode {
		log.Errorf("AssetCode not equal: newAssetCode = %s,originalAssetCode = %s. ", newAssetCode.String(), equity.AssetCode.String())
		return ErrNotEqualAssetCode
	}
	var snapshot = evm.am.Snapshot()
	// add amount
	newEquity := equity.Clone()
	newEquity.Equity = new(big.Int).Add(newEquity.Equity, addAmount)
	err = recAcc.SetEquityState(newEquity.AssetId, newEquity)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return err
	}
	// add asset totalSupply
	var oldTotalSupply *big.Int
	var newTotalSupply *big.Int
	oldTotalSupply, err = issuerAcc.GetAssetCodeTotalSupply(newAssetCode)
	if err != nil {
		return err
	}
	newTotalSupply = new(big.Int).Add(oldTotalSupply, addAmount)
	err = issuerAcc.SetAssetCodeTotalSupply(newAssetCode, newTotalSupply)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return err
	}
	return nil
}

// ModifyAssetProfileTx
func (evm *EVM) ModifyAssetProfileTx(sender common.Address, data []byte) error {
	modifyInfo := &types.ModifyAssetInfo{}
	err := json.Unmarshal(data, modifyInfo)
	if err != nil {
		return err
	}
	acc := evm.am.GetAccount(sender)
	info := modifyInfo.Info
	if info == nil || len(info) == 0 {
		return ErrModifyAssetInfo
	}
	var snapshot = evm.am.Snapshot()
	infoSlice := make([]string, 0, len(info))
	for k, _ := range info {
		infoSlice = append(infoSlice, strings.ToLower(k))
	}
	sort.Strings(infoSlice)
	for i := 0; i < len(infoSlice); i++ {
		err = acc.SetAssetCodeState(modifyInfo.AssetCode, infoSlice[i], info[infoSlice[i]])
		if err != nil {
			evm.am.RevertToSnapshot(snapshot)
			return err
		}
	}
	// modify profile
	// for k, v := range info {
	// 	err = acc.SetAssetCodeState(modifyInfo.AssetCode, strings.ToLower(k), v)
	// 	if err != nil {
	// 		evm.am.RevertToSnapshot(snapshot)
	// 		return err
	// 	}
	// }
	// 	judge profile size
	newAsset, err := acc.GetAssetCode(modifyInfo.AssetCode)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return err
	}
	newData, err := json.Marshal(newAsset)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return err
	}
	// judge data's length
	if len(newData) > types.MaxMarshalAssetLength {
		log.Errorf("The length of marshaling asset data exceed limit, len(data) = %d max = %d", len(data), types.MaxMarshalAssetLength)
		evm.am.RevertToSnapshot(snapshot)
		return ErrMarshalAssetLength
	}
	return nil
}

// TransferAssetTx
func (evm *EVM) TransferAssetTx(caller ContractRef, addr common.Address, gas uint64, assetId common.Hash, amount *big.Int, input []byte, chainDB protocol.ChainDB) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	senderAcc := evm.am.GetAccount(caller.GetAddress())
	senderEquity, err := senderAcc.GetEquityState(assetId)
	if err != nil {
		return nil, gas, err
	}
	if amount == nil || senderEquity.Equity == nil || senderEquity.Equity.Cmp(big.NewInt(0)) <= 0 {
		return nil, gas, ErrAssetEquity
	}
	// get asset
	issuer, err := chainDB.GetAssetCode(senderEquity.AssetCode)
	if err != nil {
		return nil, gas, err
	}
	issuerAcc := evm.am.GetAccount(issuer)
	isStop, err := issuerAcc.GetAssetCodeState(senderEquity.AssetCode, types.AssetStop)
	if err == nil && isStop == "true" {
		return nil, gas, ErrTransferFrozenAsset
	}

	asset, err := issuerAcc.GetAssetCode(senderEquity.AssetCode)
	if err != nil {
		return nil, gas, err
	}

	// Fail if we're trying to transfer more than the available balance
	if senderEquity.Equity.Cmp(amount) < 0 && asset.IsDivisible {
		return nil, gas, ErrInsufficientBalance
	}

	contractAccount := evm.am.GetAccount(addr)
	code, err := contractAccount.GetCode()
	if err != nil {
		return nil, gas, ErrContractCodeLoadFail
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.am.Snapshot()
	)
	if len(code) == 0 && PrecompiledContracts[addr] == nil && amount.Sign() == 0 {
		return nil, gas, nil
	}

	// if to == 0, then destruction of assets
	destroyAsset := addr == common.BytesToAddress([]byte{0})

	// Transfer assetId balance
	// if isDivisible == false,then we should set the amount equal to the balance value of assetId
	if !asset.IsDivisible {
		amount = senderEquity.Equity
	}
	if !destroyAsset {
		toEquity, err := contractAccount.GetEquityState(assetId)
		if err != nil && err == store.ErrNotExist { // not exist this kind of assetId
			// 	set new assetEquity for to
			newToEquity := senderEquity.Clone()
			newToEquity.Equity = amount
			err = contractAccount.SetEquityState(assetId, newToEquity)
			if err != nil {
				evm.am.RevertToSnapshot(snapshot)
				return nil, gas, err
			}
		} else if err != nil && err != store.ErrNotExist { // other err
			evm.am.RevertToSnapshot(snapshot)
			return nil, gas, err
		} else { // err == nil
			// 	add assetId's balance of to
			newToEquity := toEquity.Clone()
			newToEquity.Equity = new(big.Int).Add(newToEquity.Equity, amount)
			err = contractAccount.SetEquityState(assetId, newToEquity)
			if err != nil {
				evm.am.RevertToSnapshot(snapshot)
				return nil, gas, err
			}
		}
	} else {
		// reduce totalSupply
		var oldTotalSupply *big.Int
		var newTotalSupply *big.Int
		oldTotalSupply, err = issuerAcc.GetAssetCodeTotalSupply(senderEquity.AssetCode)
		if err != nil {
			return nil, gas, err
		}
		if asset.IsDivisible {
			newTotalSupply = new(big.Int).Sub(oldTotalSupply, amount)
		} else {
			newTotalSupply = new(big.Int).Sub(oldTotalSupply, big.NewInt(1))
		}
		err = issuerAcc.SetAssetCodeTotalSupply(senderEquity.AssetCode, newTotalSupply)
		if err != nil {
			evm.am.RevertToSnapshot(snapshot)
			return nil, gas, err
		}
	}

	// reduce sender's asset
	newSenderEquity := senderEquity.Clone()
	newSenderEquity.Equity = new(big.Int).Sub(newSenderEquity.Equity, amount)
	if newSenderEquity.Equity.Cmp(big.NewInt(0)) == 0 {
		// if assetId's balance == 0,then delete this kind of assetId
		err = senderAcc.SetEquityState(assetId, nil)
		if err != nil {
			evm.am.RevertToSnapshot(snapshot)
			return nil, gas, err
		}
	} else { // set new assetEquity for sender
		err = senderAcc.SetEquityState(assetId, newSenderEquity)
		if err != nil {
			evm.am.RevertToSnapshot(snapshot)
			return nil, gas, err
		}
	}

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, amount, gas)
	contract.SetCallCode(&addr, contractAccount.GetCodeHash(), code)

	start := time.Now()

	// Capture the tracer start/end events in debug mode
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.GetAddress(), addr, false, input, gas, nil)

		defer func() { // Lazy evaluation of the parameters
			evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}
	ret, err = run(evm, contract, input)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the caller as context.
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.CanTransfer(evm.am, caller.GetAddress(), value) {
		return nil, gas, ErrInsufficientBalance
	}
	contractAccount := evm.am.GetAccount(addr)
	code, err := contractAccount.GetCode()
	if err != nil {
		return nil, gas, ErrContractCodeLoadFail
	}

	var (
		snapshot = evm.am.Snapshot()
		to       = AccountRef(caller.GetAddress())
	)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, contractAccount.GetCodeHash(), code)

	ret, err = run(evm, contract, input)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	contractAccount := evm.am.GetAccount(addr)
	code, err := contractAccount.GetCode()
	if err != nil {
		return nil, gas, ErrContractCodeLoadFail
	}

	var (
		snapshot = evm.am.Snapshot()
		to       = AccountRef(caller.GetAddress())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, contractAccount.GetCodeHash(), code)

	ret, err = run(evm, contract, input)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Make sure the readonly is only set if we aren't in readonly yet
	// this makes also sure that the readonly flag isn't removed for
	// child calls.
	if !evm.interpreter.readOnly {
		evm.interpreter.readOnly = true
		defer func() { evm.interpreter.readOnly = false }()
	}
	contractAccount := evm.am.GetAccount(addr)
	code, err := contractAccount.GetCode()
	if err != nil {
		return nil, gas, ErrContractCodeLoadFail
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.am.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, contractAccount.GetCodeHash(), code)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(evm, contract, input)
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// Create creates a new contract using code as deployment code.
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if evm.depth > int(params.CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}
	if !evm.CanTransfer(evm.am, caller.GetAddress(), value) {
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}
	// Ensure there's no existing contract already at the designated address
	contractAddr = crypto.CreateAddress(caller.GetAddress(), evm.TxHash)
	// print out the contract address
	log.Warnf("Created the contract address = %v", contractAddr.String())
	contractAccount := evm.am.GetAccount(contractAddr)
	if !contractAccount.IsEmpty() {
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evm.am.Snapshot()
	// Add event to store the creation address.
	evm.AddEvent(contractAddr, []common.Hash{types.TopicContractCreation}, []byte{})
	evm.Transfer(evm.am, caller.GetAddress(), contractAddr, value)

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(contractAddr), value, gas)
	contract.SetCallCode(&contractAddr, crypto.Keccak256Hash(code), code)

	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, contractAddr, gas, nil
	}

	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.GetAddress(), contractAddr, true, code, gas, value)
	}
	start := time.Now()

	ret, err = run(evm, contract, nil)

	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			contractAccount.SetCode(types.Code(ret))
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || err != nil {
		evm.am.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	if err != nil && err != ErrInsufficientBalance {
		// Add event to record the error information.
		evm.AddEvent(contractAddr, []common.Hash{types.TopicRunFail}, []byte{})
	}
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}
	return ret, contractAddr, contract.Gas, err
}

// Interpreter returns the EVM interpreter
func (evm *EVM) Interpreter() *Interpreter { return evm.interpreter }

// AddEvent records the event during transaction's execution.
func (evm *EVM) AddEvent(address common.Address, topics []common.Hash, data []byte) {
	evm.am.AddEvent(&types.Event{
		Address: address,
		Topics:  topics,
		Data:    data,
		// This is a non-consensus field, but assigned here because
		// chain/account doesn't know the current block number.
		TxIndex: evm.TxIndex,
		TxHash:  evm.TxHash,
		// event.Term is set outside.
	})
}
