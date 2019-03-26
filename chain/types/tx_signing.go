package types

import (
	"crypto/ecdsa"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
)

var (
	ErrPublicKey = errors.New("invalid public key")
)

// MakeSigner returns a Signer based on the given version and chainID.
func MakeSigner() Signer {
	return DefaultSigner{}
}

// MakeReimbursementTxSigner returns instead of pay gas transaction signer
func MakeReimbursementTxSigner() Signer {
	return ReimbursementTxSigner{}
}

// MakeGasPayerSigner returns gas payer signer
func MakeGasPayerSigner() Signer {
	return GasPayerSigner{}
}

// Signer encapsulates transaction signature handling.
type Signer interface {
	// SignTx returns transaction after signature
	SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error)

	// GetSigner returns the sender address of the transaction.
	GetSigner(tx *Transaction) (common.Address, error)

	// Hash returns the hash to be signed.
	Hash(tx *Transaction) common.Hash
}

// DefaultSigner implements Signer.
type DefaultSigner struct {
}

func (s DefaultSigner) SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data, gasPayer: tx.gasPayer}
	cpy.data.Sig = sig
	return cpy, nil
}

func (s DefaultSigner) GetSigner(tx *Transaction) (common.Address, error) {
	sigHash := s.Hash(tx)
	sig := tx.data.Sig
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sigHash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, ErrPublicKey
	}
	addr := crypto.PubToAddress(pub)
	return addr, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s DefaultSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Type(),
		tx.Version(),
		tx.ChainID(),
		tx.data.Recipient,
		tx.data.RecipientName,
		tx.data.GasPrice,
		tx.data.GasLimit,
		tx.data.Amount,
		tx.data.Data,
		tx.data.Expiration,
		tx.data.Message,
	})
}

type ReimbursementTxSigner struct {
}

// SignTx returns first signature to reimbursement gas transaction
func (s ReimbursementTxSigner) SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data, gasPayer: tx.gasPayer}
	cpy.data.Sig = sig
	return cpy, nil
}

// Hash excluding gasLimit and gasPrice
func (s ReimbursementTxSigner) Hash(tx *Transaction) common.Hash {
	gasPayer, err := tx.GasPayer()
	if err != nil {
		return common.Hash{}
	}
	return rlpHash([]interface{}{
		tx.Type(),
		tx.Version(),
		tx.ChainID(),
		tx.data.Recipient,
		tx.data.RecipientName,
		tx.data.Amount,
		tx.data.Data,
		tx.data.Expiration,
		tx.data.Message,
		gasPayer,
	})
}

// GetSigner
func (s ReimbursementTxSigner) GetSigner(tx *Transaction) (common.Address, error) {
	sigHash := s.Hash(tx)
	sig := tx.data.Sig
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sigHash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, ErrPublicKey
	}
	addr := crypto.PubToAddress(pub)
	return addr, nil
}

type GasPayerSigner struct {
}

// SignTx returns last signature to reimbursement gas transaction
func (g GasPayerSigner) SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	gasPayer, err := tx.GasPayer()
	if err != nil {
		return nil, err
	}
	addr := crypto.PubkeyToAddress(prv.PublicKey)
	if gasPayer != addr {
		return nil, errors.New("not expect gas payer")
	}

	h := g.Hash(tx)

	lastSignData, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.GasPayerSig = lastSignData
	return cpy, nil
}

// GetGasPayer returns gas payer address
func (g GasPayerSigner) GetSigner(tx *Transaction) (common.Address, error) {
	sigHash := g.Hash(tx)

	sig := tx.data.GasPayerSig
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sigHash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, ErrPublicKey
	}
	addr := crypto.PubToAddress(pub)
	return addr, nil
}

// Hash returns sign hash
func (g GasPayerSigner) Hash(tx *Transaction) common.Hash {
	firstSignData := tx.data.Sig
	return rlpHash([]interface{}{
		firstSignData,
		tx.GasPrice(),
		tx.GasLimit(),
	})
}
