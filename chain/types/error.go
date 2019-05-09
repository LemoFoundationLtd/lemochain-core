package types

import (
	"errors"
)

var (
	// ErrKnownBlock is returned when a block to import is already known locally.
	ErrKnownBlock = errors.New("block already known")

	// ErrGasLimitReached is returned by the gas pool if the amount of gas required
	// by a transaction is higher than what's left in the block.
	ErrGasLimitReached = errors.New("block gas limit reached")

	// ErrBlacklistedHash is returned if a block to import is on the blacklist.
	ErrBlacklistedHash = errors.New("blacklisted hash")

	ErrInvalidSig     = errors.New("invalid transaction sig")
	ErrInvalidVersion = errors.New("invalid transaction version")
	ErrToName         = errors.New("the length of toName field in transaction is out of max length limit")
	ErrTxMessage      = errors.New("the length of message field in transaction is out of max length limit")
	ErrCreateContract = errors.New("the data of create contract transaction can't be null")
	ErrSpecialTx      = errors.New("the data of special transaction can't be null")
	ErrTxType         = errors.New("the transaction type does not exit")
	ErrTxExpiration   = errors.New("transaction is out of date")
	ErrNegativeValue  = errors.New("transaction amount can't be negative")
	ErrTxChainID      = errors.New("transaction chainID is incorrect")
)
