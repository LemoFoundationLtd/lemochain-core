package chain

//
// import (
// 	"errors"
// 	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
// 	"math"
// 	"math/big"
//
// 	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
// 	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
// 	"github.com/LemoFoundationLtd/lemochain-go/common"
// 	"github.com/LemoFoundationLtd/lemochain-go/common/log"
// )
//
// var (
// 	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
// )
//
// /*
// The State Transitioning Model
//
// A state transition is a change made when a transaction is applied to the current world state
// The state transitioning model does all all the necessary work to work out a valid new state root.
//
// 1) Nonce handling
// 2) Pre pay gas
// 3) Create a new state object if the recipient is \0*32
// 4) Value transfer
// == If contract creation ==
//   4a) Attempt to run transaction data
//   4b) If valid, use result as code for the new state object
// == end ==
// 5) Run Script section
// 6) Derive new state root
// */
// type StateTransition struct {
// 	gp         *types.GasPool
// 	msg        Message
// 	gas        uint64
// 	gasPrice   *big.Int
// 	initialGas uint64
// 	value      *big.Int
// 	data       []byte
// 	state      vm.AccountManager
// 	evm        *vm.EVM
// }
//
// // Message represents a message sent to a contract.
// type Message interface {
// 	From() common.Address
// 	To() *common.Address
//
// 	GasPrice() *big.Int
// 	Gas() uint64
// 	Value() *big.Int
//
// 	Data() []byte
// }
//
// // IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
// func IntrinsicGas(data []byte, contractCreation bool) (uint64, error) {
// 	// Set the starting gas for the raw transaction
// 	var gas uint64
// 	if contractCreation {
// 		gas = params.TxGasContractCreation
// 	} else {
// 		gas = params.TxGas
// 	}
// 	// Bump the required gas by the amount of transactional data
// 	if len(data) > 0 {
// 		// Zero and non-zero bytes are priced differently
// 		var nz uint64
// 		for _, byt := range data {
// 			if byt != 0 {
// 				nz++
// 			}
// 		}
// 		// Make sure we don't exceed uint64 for all data combinations
// 		if (math.MaxUint64-gas)/params.TxDataNonZeroGas < nz {
// 			return 0, vm.ErrOutOfGas
// 		}
// 		gas += nz * params.TxDataNonZeroGas
//
// 		z := uint64(len(data)) - nz
// 		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
// 			return 0, vm.ErrOutOfGas
// 		}
// 		gas += z * params.TxDataZeroGas
// 	}
// 	return gas, nil
// }
//
// // NewStateTransition initialises and returns a new state transition object.
// func NewStateTransition(evm *vm.EVM, msg Message, gp *types.GasPool) *StateTransition {
// 	return &StateTransition{
// 		gp:       gp,
// 		evm:      evm,
// 		msg:      msg,
// 		gasPrice: msg.GasPrice(),
// 		value:    msg.Value(),
// 		data:     msg.Data(),
// 		state:    evm.AccountManager,
// 	}
// }
//
// // ApplyMessage computes the new state by applying the given message
// // against the old state within the environment.
// //
// // ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// // the gas used (which includes gas refunds) and an error if it failed. An error always
// // indicates a core error meaning that the message would always fail for that particular
// // state and would never be accepted within a block.
// func ApplyMessage(evm *vm.EVM, msg Message, gp *types.GasPool) ([]byte, uint64, bool, error) {
// 	return NewStateTransition(evm, msg, gp).TransitionDb()
// }
//
// func (st *StateTransition) from() vm.AccountRef {
// 	f := st.msg.From()
// 	if !st.state.Exist(f) {
// 		st.state.CreateAccount(f)
// 	}
// 	return vm.AccountRef(f)
// }
//
// func (st *StateTransition) to() vm.AccountRef {
// 	if st.msg == nil {
// 		return vm.AccountRef{}
// 	}
// 	to := st.msg.To()
// 	if to == nil {
// 		return vm.AccountRef{} // contract creation
// 	}
//
// 	reference := vm.AccountRef(*to)
// 	if !st.state.Exist(*to) {
// 		st.state.CreateAccount(*to)
// 	}
// 	return reference
// }
//
// func (st *StateTransition) useGas(amount uint64) error {
// 	if st.gas < amount {
// 		return vm.ErrOutOfGas
// 	}
// 	st.gas -= amount
//
// 	return nil
// }
//
// func (st *StateTransition) buyGas() error {
// 	var (
// 		state  = st.state
// 		sender = st.from()
// 	)
// 	mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.gasPrice)
// 	if state.GetBalance(sender.GetAddress()).Cmp(mgval) < 0 {
// 		return errInsufficientBalanceForGas
// 	}
// 	if err := st.gp.SubGas(st.msg.Gas()); err != nil {
// 		return err
// 	}
// 	st.gas += st.msg.Gas()
//
// 	st.initialGas = st.msg.Gas()
// 	state.SubBalance(sender.GetAddress(), mgval)
// 	return nil
// }
//
// func (st *StateTransition) preCheck() error {
// 	return st.buyGas()
// }
//
// // TransitionDb will transition the state by applying the current message and
// // returning the result including the the used gas. It returns an error if it
// // failed. An error indicates a consensus issue.
// func (st *StateTransition) TransitionDb() (ret []byte, usedGas uint64, failed bool, err error) {
// 	if err = st.preCheck(); err != nil {
// 		return
// 	}
// 	msg := st.msg
// 	sender := st.from() // err checked in preCheck
//
// 	contractCreation := msg.To() == nil
//
// 	// Pay intrinsic gas
// 	gas, err := IntrinsicGas(st.data, contractCreation)
// 	if err != nil {
// 		return nil, 0, false, err
// 	}
// 	if err = st.useGas(gas); err != nil {
// 		return nil, 0, false, err
// 	}
//
// 	var (
// 		evm = st.evm
// 		// vm errors do not effect consensus and are therefor
// 		// not assigned to err, except for insufficient balance
// 		// error.
// 		vmerr error
// 	)
// 	if contractCreation {
// 		ret, _, st.gas, vmerr = evm.Create(sender, st.data, st.gas, st.value)
// 	} else {
// 		// Increment the nonce for the next transaction
// 		st.state.SetNonce(sender.GetAddress(), st.state.GetNonce(sender.GetAddress())+1)
// 		ret, st.gas, vmerr = evm.Call(sender, st.to().GetAddress(), st.data, st.gas, st.value)
// 	}
// 	if vmerr != nil {
// 		log.Debug("VM returned with error", "err", vmerr)
// 		// The only possible consensus-error would be if there wasn't
// 		// sufficient balance to make the transfer happen. The first
// 		// balance transfer may never fail.
// 		if vmerr == vm.ErrInsufficientBalance {
// 			return nil, 0, false, vmerr
// 		}
// 	}
// 	st.refundGas()
// 	st.state.AddBalance(st.evm.Lemobase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))
//
// 	return ret, st.gasUsed(), vmerr != nil, err
// }
//
// func (st *StateTransition) refundGas() {
// 	// Apply refund counter, capped to half of the used gas.
// 	refund := st.gasUsed() / 2
// 	if refund > st.state.GetRefund() {
// 		refund = st.state.GetRefund()
// 	}
// 	st.gas += refund
//
// 	// Return ETH for remaining gas, exchanged at the original rate.
// 	sender := st.from()
//
// 	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
// 	st.state.AddBalance(sender.GetAddress(), remaining)
//
// 	// Also return remaining gas to the block gas counter so it is
// 	// available for the next transaction.
// 	st.gp.AddGas(st.gas)
// }
//
// // gasUsed returns the amount of gas used up by the state transition.
// func (st *StateTransition) gasUsed() uint64 {
// 	return st.initialGas - st.gas
// }
