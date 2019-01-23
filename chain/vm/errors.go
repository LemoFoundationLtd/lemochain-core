package vm

import "errors"

var (
	ErrOutOfGas                    = errors.New("out of gas")
	ErrCodeStoreOutOfGas           = errors.New("contract creation code storage out of gas")
	ErrDepth                       = errors.New("max call depth exceeded")
	ErrTraceLimitReached           = errors.New("the number of logs reached the specified limit")
	ErrInsufficientBalance         = errors.New("insufficient balance for transfer")
	ErrContractAddressCollision    = errors.New("contract address collision")
	ErrContractCodeLoadFail        = errors.New("contract code load fail")
	ErrOfRegisterCandidateNodeFees = errors.New("Insufficient fees of registered candidate node ")
	ErrOfAgainVote                 = errors.New("already voted the same as candidate node")
	ErrOfNotCandidateNode          = errors.New("node address is not candidate account")
	ErrOfRegisterNodeID            = errors.New("can't get nodeId of RegisterInfo")
	ErrOfRegisterHost              = errors.New("can't get host of RegisterInfo")
	ErrOfRegisterPort              = errors.New("can't get port of RegisterInfo")
)
