package vm

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
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
			return RunPrecompiledContract(p, input, contract)
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

// 投票交易调用
func (evm *EVM) CallVoteTx(voter, node common.Address, gas uint64, initialBalance *big.Int) (leftgas uint64, err error) {
	nodeAccount := evm.am.GetAccount(node)
	// 	判断node是否为候选节点的竞选账户
	profile := nodeAccount.GetCandidateProfile()
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	if !ok || IsCandidate == "false" {
		return gas, ErrOfNotCandidateNode
	}
	var snapshot = evm.am.Snapshot() // 回滚操作
	voterAccount := evm.am.GetAccount(voter)
	// 查看voter是否已经投过票了
	if (voterAccount.GetVoteFor() != common.Address{}) {
		if voterAccount.GetVoteFor() == node { // 已经投过此竞选节点了
			return gas, ErrOfAgainVote
		} else { // 转投其他竞选节点
			oldNode := voterAccount.GetVoteFor()
			newNodeAccount := nodeAccount
			// 减少旧节点对应的票数，增加新节点对应的票数，票数为账户的balance
			oldNodeAccount := evm.am.GetAccount(oldNode)
			// 减少旧的竞选节点的票数
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialBalance)
			oldNodeAccount.SetVotes(oldNodeVoters)
			// 增加新的竞选节点的票数
			newNodeVoters := new(big.Int).Add(newNodeAccount.GetVotes(), initialBalance)
			newNodeAccount.SetVotes(newNodeVoters)
		}
	} else { // 第一次投票
		// 增加竞选节点的票数
		nodeVoters := new(big.Int).Add(nodeAccount.GetVotes(), initialBalance)
		nodeAccount.SetVotes(nodeVoters)
	}
	// 修改投票者指定的竞选节点
	voterAccount.SetVoteFor(node)
	// 回滚
	if err != nil {
		evm.am.RevertToSnapshot(snapshot)
	}
	return gas, err
}

// 申请注册参加竞选代理节点的交易调用,sender为发起申请交易的用户地址，to为接收注册费用的账户地址，CandidateAddress为要成为候选节点的地址，Host为节点ip或者域名
func (evm *EVM) RegisterOrModifyOfCandidate(CandidateAddress, to, minerAddress common.Address, isCandidate bool, nodeID, host, port string, gas uint64, value, initialSenderBalance *big.Int) (leftgas uint64, err error) {
	// value不能小于规定的注册费用
	if value.Cmp(params.RegisterCandidateNodeFees) < 0 {
		return gas, ErrOfRegisterCandidateNodeFees
	}
	// 查看余额够不够
	if !evm.CanTransfer(evm.am, CandidateAddress, value) {
		return gas, ErrInsufficientBalance
	}
	// var snapshot = evm.am.Snapshot() // 回滚操作

	// 申请为候选节点账户
	nodeAccount := evm.am.GetAccount(CandidateAddress)
	// 查看申请地址是否已经为竞选代理节点
	profile := nodeAccount.GetCandidateProfile()
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	// 如果已经是候选节点账户则查看传入的候选节点参数是否需要改变或者是否为一个取消候选人资格的交易
	if ok && IsCandidate == "true" {
		// 判断是否要注销候选者资格
		if !isCandidate {
			profile[types.CandidateKeyIsCandidate] = "false"
			nodeAccount.SetCandidateProfile(profile)
			// 注销后的用户的票数为0
			nodeAccount.SetVotes(big.NewInt(0))
			// 注销也需要消耗1000LEMO
			evm.Transfer(evm.am, CandidateAddress, to, value)
			return gas, nil
		}
		if profile[types.CandidateKeyMinerAddress] != minerAddress.Hex() {
			profile[types.CandidateKeyMinerAddress] = minerAddress.Hex()
		}
		// if profile[types.CandidateKeyNodeID] != nodeID {
		// 	profile[types.CandidateKeyNodeID] = nodeID
		// }
		if profile[types.CandidateKeyHost] != host {
			profile[types.CandidateKeyHost] = host
		}
		if profile[types.CandidateKeyPort] != port {
			profile[types.CandidateKeyPort] = port
		}
		nodeAccount.SetCandidateProfile(profile)
	} else {
		// 初始化map
		// 注册为竞选节点
		newProfile := make(types.CandidateProfile, 5)
		newProfile[types.CandidateKeyIsCandidate] = "true"
		newProfile[types.CandidateKeyMinerAddress] = minerAddress.Hex()
		newProfile[types.CandidateKeyNodeID] = nodeID
		newProfile[types.CandidateKeyHost] = host
		newProfile[types.CandidateKeyPort] = port
		nodeAccount.SetCandidateProfile(newProfile)

		// 设置自己的账户投给自己，票数为执行交易前的Balance，所以要加上购买gas所用的balance
		oldNodeAddress := nodeAccount.GetVoteFor()
		// 如果投过票要减少所投候选节点的票数
		if (oldNodeAddress != common.Address{}) { // 曾经投过别的候选节点票
			oldNodeAccount := evm.am.GetAccount(oldNodeAddress) // 旧的候选节点的账户
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialSenderBalance)
			oldNodeAccount.SetVotes(oldNodeVoters)
		}
		// 设置投票候选节点为自己地址
		nodeAccount.SetVoteFor(CandidateAddress)
		// 设置自己的票数，此时自己的票数为自己的balance.
		nodeAccount.SetVotes(initialSenderBalance)

	}
	// 转账操作，必须放在票数改变逻辑后面，保证不会因为balance改变导致票数的改变出错
	evm.Transfer(evm.am, CandidateAddress, to, value)
	// // 回滚
	// if err != nil {
	// 	evm.am.RevertToSnapshot(snapshot)
	// }
	return gas, nil
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
		BlockHeight: evm.BlockHeight,
		TxIndex:     evm.TxIndex,
		TxHash:      evm.TxHash,
		BlockHash:   evm.BlockHash,
		// event.Index is set outside.
	})
}
