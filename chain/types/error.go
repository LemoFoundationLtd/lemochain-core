package types

import (
	"errors"
)

var (
	ErrNegativeBalance = errors.New("balance can't be negative")
	ErrLoadCodeFail    = errors.New("can't load contract code")
	ErrAssetNotExist   = errors.New("asset dose not exist")
	ErrAssetIdNotExist = errors.New("assetId dose not exist")
	ErrEquityNotExist  = errors.New("equity dose not exist")
	ErrTrieFail        = errors.New("can't load contract storage trie")
	ErrTrieChanged     = errors.New("the trie has changed after Finalise")

	// ErrGasLimitReached is returned by the gas pool if the amount of gas required
	// by a transaction is higher than what's left in the block.
	ErrGasLimitReached = errors.New("block gas limit reached")

	// ErrBlacklistedHash is returned if a block to import is on the blacklist.
	ErrBlacklistedHash = errors.New("blacklisted hash")
	ErrInvalidSig      = errors.New("invalid transaction sig")
	ErrInvalidVersion  = errors.New("invalid transaction version")
	ErrToNameLength    = errors.New("the length of 'toName' field in transaction is out of limit")
	ErrToNameCharacter = errors.New("the 'toName' field in transaction contains illegal characters")
	ErrTxMessage       = errors.New("the length of 'message' field in transaction is out of limit")
	ErrCreateContract  = errors.New("the 'data' field of create contract transaction can't be null")
	ErrSpecialTx       = errors.New("the 'data' field of special transaction can't be null")
	ErrTxType          = errors.New("the 'type' field of transaction does not exist")
	ErrGasPrice        = errors.New("the 'gasPrice' filed of transaction is too low")
	ErrTxExpired       = errors.New("the 'expirationTime' field of transaction must be later than current time")
	ErrTxExpiration    = errors.New("the 'expirationTime' field of transaction must not be later than 30 minutes")
	ErrNegativeValue   = errors.New("the 'amount' field of transaction can't be negative")
	ErrTxChainID       = errors.New("the 'chainID' field of transaction is incorrect")
	ErrBoxTx           = errors.New("the 'expirationTime' field of box transaction must be later than all sub transactions")
	ErrVerifyBoxTx     = errors.New("box transaction cannot be in another box transaction")
	ErrToExist         = errors.New("the 'to' field of transaction is incorrect")
)
