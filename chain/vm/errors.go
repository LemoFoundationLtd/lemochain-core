package vm

import "errors"

var (
	ErrOutOfGas                 = errors.New("out of gas")
	ErrCodeStoreOutOfGas        = errors.New("contract creation code storage out of gas")
	ErrDepth                    = errors.New("max call depth exceeded")
	ErrTraceLimitReached        = errors.New("the number of logs reached the specified limit")
	ErrInsufficientBalance      = errors.New("insufficient balance for transfer")
	ErrContractAddressCollision = errors.New("contract address collision")
	ErrContractCodeLoadFail     = errors.New("contract code load fail")
	ErrAssetEquity              = errors.New("asset equity can't be nil or 0")
	ErrTransferFrozenAsset      = errors.New("cannot trade frozen assets")
	ErrTermReward               = errors.New("no permission to call this Precompiled contract")
)
