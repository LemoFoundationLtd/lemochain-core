package vm

import "errors"

var (
	ErrOutOfGas                   = errors.New("out of gas")
	ErrCodeStoreOutOfGas          = errors.New("contract creation code storage out of gas")
	ErrDepth                      = errors.New("max call depth exceeded")
	ErrTraceLimitReached          = errors.New("the number of logs reached the specified limit")
	ErrInsufficientBalance        = errors.New("insufficient balance for transfer")
	ErrContractAddressCollision   = errors.New("contract address collision")
	ErrContractCodeLoadFail       = errors.New("contract code load fail")
	ErrOfRegisterCampaignNodeFees = errors.New("Insufficient fees of registered campaign node ")
	ErrOfAgainVote                = errors.New("already voted the same as campaign node")
	ErrOfNotCampaignNode          = errors.New("node address is not candidate account")
)
