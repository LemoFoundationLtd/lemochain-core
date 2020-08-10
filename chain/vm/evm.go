package vm

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math/big"
	"strconv"
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

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *EVM, contract *Contract, input []byte) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContracts
		if p := precompiles[*contract.CodeAddr]; p != nil {
			// Determine whether the address to set the reward value is correct
			if *contract.CodeAddr == params.TermRewardContract && contract.caller.GetAddress() != evm.vmConfig.RewardManager {
				return nil, ErrTermReward
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
		log.Error("evm error", "error", err)
		evm.am.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
		evm.AddEvent(addr, []common.Hash{types.TopicRunFail}, []byte{})
	}
	return ret, contract.Gas, err
}

type AssetDb interface {
	GetIssurerByAssetCode(code common.Hash) (common.Address, error)
}

// TransferAssetTx
func (evm *EVM) TransferAssetTx(caller ContractRef, addr common.Address, gas uint64, txData []byte, assetDB AssetDb) (ret []byte, leftOverGas uint64, Err, vmErr error) {
	transferAsset, err := types.GetTransferAsset(txData)
	if err != nil {
		log.Errorf("Unmarshal transfer asset data err: %s", err)
		return nil, gas, err, nil
	}
	assetId := transferAsset.AssetId
	amount := transferAsset.Amount
	input := transferAsset.Input

	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, nil, ErrDepth
	}

	senderAcc := evm.am.GetAccount(caller.GetAddress())
	senderEquity, err := senderAcc.GetEquityState(assetId)
	if err != nil {
		return nil, gas, err, nil
	}
	if amount == nil || senderEquity.Equity == nil || senderEquity.Equity.Cmp(big.NewInt(0)) <= 0 {
		return nil, gas, ErrAssetEquity, nil
	}
	// get asset
	issuer, err := assetDB.GetIssurerByAssetCode(senderEquity.AssetCode)
	if err != nil {
		return nil, gas, err, nil
	}
	issuerAcc := evm.am.GetAccount(issuer)
	freeze, err := issuerAcc.GetAssetCodeState(senderEquity.AssetCode, types.AssetFreeze)
	if err == nil && freeze == "true" {
		return nil, gas, ErrTransferFrozenAsset, nil
	}

	asset, err := issuerAcc.GetAssetCode(senderEquity.AssetCode)
	if err != nil {
		return nil, gas, err, nil
	}

	// Fail if we're trying to transfer more than the available balance
	if senderEquity.Equity.Cmp(amount) < 0 && asset.IsDivisible {
		log.Errorf("Sender equity:%s, transaction amount:%s, asset.IsDivisible:%s", senderEquity.Equity.String(), amount.String(), strconv.FormatBool(asset.IsDivisible))
		return nil, gas, ErrInsufficientBalance, nil
	}

	contractAccount := evm.am.GetAccount(addr)
	code, err := contractAccount.GetCode()
	if err != nil {
		return nil, gas, ErrContractCodeLoadFail, nil
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.am.Snapshot()
	)
	if len(code) == 0 && PrecompiledContracts[addr] == nil && amount.Sign() == 0 {
		return nil, gas, nil, nil
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
		if err == types.ErrEquityNotExist { // not exist this kind of assetId
			// 	set new assetEquity for to
			newToEquity := senderEquity.Clone()
			newToEquity.Equity = amount
			err = contractAccount.SetEquityState(assetId, newToEquity)
			if err != nil {
				evm.am.RevertToSnapshot(snapshot)
				return nil, gas, err, nil
			}
		} else if err != nil && err != types.ErrEquityNotExist { // other err
			evm.am.RevertToSnapshot(snapshot)
			return nil, gas, err, nil
		} else { // err == nil
			// 	add assetId's balance of to
			newToEquity := toEquity.Clone()
			newToEquity.Equity = new(big.Int).Add(newToEquity.Equity, amount)
			err = contractAccount.SetEquityState(assetId, newToEquity)
			if err != nil {
				evm.am.RevertToSnapshot(snapshot)
				return nil, gas, err, nil
			}
		}
	} else {
		// reduce totalSupply
		var oldTotalSupply *big.Int
		var newTotalSupply *big.Int
		oldTotalSupply, err = issuerAcc.GetAssetCodeTotalSupply(senderEquity.AssetCode)
		if err != nil {
			return nil, gas, err, nil
		}
		if asset.IsDivisible {
			newTotalSupply = new(big.Int).Sub(oldTotalSupply, amount)
		} else {
			newTotalSupply = new(big.Int).Sub(oldTotalSupply, big.NewInt(1))
		}
		err = issuerAcc.SetAssetCodeTotalSupply(senderEquity.AssetCode, newTotalSupply)
		if err != nil {
			evm.am.RevertToSnapshot(snapshot)
			return nil, gas, err, nil
		}
	}

	// reduce sender's asset
	// 这里重新在account中获取一次sender的equity信息,解决from == to的时候前面to的account变化了,这里的account 变化要在前面to的account基础上,所以这里要重新读取一下account中的equity。
	newSenderEquity, err := senderAcc.GetEquityState(assetId)
	if err != nil {
		return nil, gas, err, nil
	}
	senderEquityCpy := newSenderEquity.Clone()
	senderEquityCpy.Equity = new(big.Int).Sub(senderEquityCpy.Equity, amount)
	if err := senderAcc.SetEquityState(assetId, senderEquityCpy); err != nil {
		evm.am.RevertToSnapshot(snapshot)
		return nil, gas, err, nil
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
	return ret, contract.Gas, nil, err
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
	contractAddr = crypto.CreateContractAddress(caller.GetAddress(), evm.TxHash)
	// print out the contract address
	log.Warnf("Created the contract address = %v", contractAddr.String())
	contractAccount := evm.am.GetAccount(contractAddr)
	if !contractAccount.IsEmpty() {
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evm.am.Snapshot()

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
	} else {
		// Add event to store the creation address.
		evm.AddEvent(contractAddr, []common.Hash{types.TopicContractCreation}, []byte{})
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
