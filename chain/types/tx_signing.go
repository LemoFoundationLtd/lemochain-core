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

// recoverSigners
func recoverSigners(s Signer, tx *Transaction) ([]common.Address, error) {
	sigHash := s.Hash(tx)
	sigs := s.Sigs(tx)
	length := len(sigs)
	signers := make([]common.Address, length, length)
	for i := 0; i < length; i++ {
		// recover the public key from the signature
		pub, err := crypto.Ecrecover(sigHash[:], sigs[i])
		if err != nil {
			return nil, err
		}
		if len(pub) == 0 || pub[0] != 4 {
			return nil, ErrPublicKey
		}
		addr := crypto.PubToAddress(pub)
		signers[i] = addr
	}
	return signers, nil
}

// Signer encapsulates transaction signature handling.
type Signer interface {
	// SignTx returns transaction after signature
	SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error)

	// GetSigners returns the sender address of the transaction.
	GetSigners(tx *Transaction) ([]common.Address, error)

	// Hash returns the hash to be signed.
	Hash(tx *Transaction) common.Hash

	// get transaction sigs or gasPayerSigs
	Sigs(tx *Transaction) [][]byte
}

// DefaultSigner implements Signer.
type DefaultSigner struct {
}

func (s DefaultSigner) Sigs(tx *Transaction) [][]byte {
	return tx.Sigs()
}

func (s DefaultSigner) SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := tx.Clone()
	cpy.data.Sigs = append(cpy.data.Sigs, sig)
	return cpy, nil
}

func (s DefaultSigner) GetSigners(tx *Transaction) ([]common.Address, error) {
	signers, err := recoverSigners(s, tx)
	return signers, err
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s DefaultSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Type(),
		tx.Version(),
		tx.ChainID(),
		tx.data.From,
		tx.data.GasPayer,
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

func (s ReimbursementTxSigner) Sigs(tx *Transaction) [][]byte {
	return tx.Sigs()
}

// SignTx returns first signature to reimbursement gas transaction
func (s ReimbursementTxSigner) SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := tx.Clone()
	cpy.data.Sigs = append(cpy.data.Sigs, sig)
	return cpy, nil
}

// Hash excluding gasLimit and gasPrice
func (s ReimbursementTxSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Type(),
		tx.Version(),
		tx.ChainID(),
		tx.data.From,
		tx.data.GasPayer,
		tx.data.Recipient,
		tx.data.RecipientName,
		tx.data.Amount,
		tx.data.Data,
		tx.data.Expiration,
		tx.data.Message,
	})
}

// GetSigners
func (s ReimbursementTxSigner) GetSigners(tx *Transaction) ([]common.Address, error) {
	signers, err := recoverSigners(s, tx)
	return signers, err
}

type GasPayerSigner struct {
}

func (g GasPayerSigner) Sigs(tx *Transaction) [][]byte {
	return tx.GasPayerSigs()
}

// SignTx returns last signature to reimbursement gas transaction
func (g GasPayerSigner) SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := g.Hash(tx)

	lastSignData, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := tx.Clone()
	cpy.data.GasPayerSigs = append(cpy.data.GasPayerSigs, lastSignData)
	return cpy, nil
}

// GetGasPayer returns gas payer address
func (g GasPayerSigner) GetSigners(tx *Transaction) ([]common.Address, error) {
	signers, err := recoverSigners(g, tx)
	return signers, err
}

// Hash returns sign hash
func (g GasPayerSigner) Hash(tx *Transaction) common.Hash {
	firstSignData := tx.data.Sigs
	return rlpHash([]interface{}{
		firstSignData,
		tx.GasPrice(),
		tx.GasLimit(),
	})
}
