package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"math/big"
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

	// ParseSignature returns the raw R, S, V values corresponding to the
	// given signature.
	ParseSignature(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
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
	return tx.WithSignature(s, sig)
}

func (s DefaultSigner) GetSigner(tx *Transaction) (common.Address, error) {
	sigHash := s.Hash(tx)
	sig, err := recoverSignData(tx)
	if err != nil {
		return common.Address{}, err
	}
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

// ParseSignature returns a new transaction with the given signature. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s DefaultSigner) ParseSignature(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	R = new(big.Int).SetBytes(sig[:32])
	S = new(big.Int).SetBytes(sig[32:64])
	V = SetSecp256k1V(tx.data.V, sig[64])
	return R, S, V, nil
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
	return tx.WithSignature(s, sig)
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
	sig, err := recoverSignData(tx)
	if err != nil {
		return common.Address{}, err
	}
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

// ParseSignature
func (s ReimbursementTxSigner) ParseSignature(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	R = new(big.Int).SetBytes(sig[:32])
	S = new(big.Int).SetBytes(sig[32:64])
	// if sig[64] == 1, then V will change. else V remains unchanged
	V = SetSecp256k1V(tx.data.V, sig[64])
	return R, S, V, nil
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
	firstSignData, err := recoverSignData(tx)
	if err != nil {
		return common.Hash{}
	}
	return rlpHash([]interface{}{
		firstSignData,
		tx.GasPrice(),
		tx.GasLimit(),
	})
}

// ParseSignature
func (g GasPayerSigner) ParseSignature(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	return
}

// recoverSignData recover tx sign data by V, R, S
func recoverSignData(tx *Transaction) ([]byte, error) {
	V, R, S := tx.data.V, tx.data.R, tx.data.S

	if V.BitLen() > 32 {
		return nil, ErrInvalidSig
	}
	_, _, v, _ := ParseV(V)
	if !crypto.ValidateSignatureValues(v, R, S) {
		return nil, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	rb, sb := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	sig[64] = v
	return sig, nil
}
