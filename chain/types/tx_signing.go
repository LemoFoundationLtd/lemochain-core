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

// SignTx signs the transaction using the given signer and private key
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	fmt.Println("sigdata:", sig)
	return tx.WithSignature(s, sig)
}

// Signer encapsulates transaction signature handling.
type Signer interface {
	// GetSender returns the sender address of the transaction.
	GetSender(tx *Transaction) (common.Address, error)
	// ParseSignature returns the raw R, S, V values corresponding to the
	// given signature.
	ParseSignature(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *Transaction) common.Hash
}

// DefaultSigner implements Signer.
type DefaultSigner struct {
}

func (s DefaultSigner) GetSender(tx *Transaction) (common.Address, error) {
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

// Hash excluding gasLimit and gasPrice
func (s ReimbursementTxSigner) Hash(tx *Transaction) common.Hash {
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
	})
}

// GetSender
func (s ReimbursementTxSigner) GetSender(tx *Transaction) (common.Address, error) {
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

// MakeGasPayerSigner returns gas payer signer
func MakeGasPayerSigner() GasPayerSigner {
	return GasPayerSigner{}
}

type GasPayerSigner struct {
}

// GasPayerSign gas payer signs the signature of transaction
func (g GasPayerSigner) GasPayerSignTx(tx *Transaction, gasPrice *big.Int, gasLimit uint64, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h, err := g.SignHash(tx, gasPrice, gasLimit)
	if err != nil {
		return nil, err
	}
	lastSignData, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.GasPrice = gasPrice
	cpy.data.GasLimit = gasLimit
	cpy.data.GasPayerSign = lastSignData
	return cpy, nil
}

// GetGasPayer returns gas payer address
func (g GasPayerSigner) GasPayer(tx *Transaction) (common.Address, error) {
	sigHash, err := g.SignHash(tx, tx.GasPrice(), tx.GasLimit())
	if err != nil {
		return common.Address{}, err
	}
	sig := tx.data.GasPayerSign
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

// SignHash returns sign hash
func (g GasPayerSigner) SignHash(tx *Transaction, gasPrice *big.Int, gasLimit uint64) (common.Hash, error) {
	firstSignData, err := recoverSignData(tx)
	if err != nil {
		return common.Hash{}, err
	}
	return rlpHash([]interface{}{
		firstSignData,
		gasPrice,
		gasLimit,
	}), nil
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
