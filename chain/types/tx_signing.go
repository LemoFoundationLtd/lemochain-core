package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
)

var (
	ErrPublicKey = errors.New("invalid public key")
)

// MakeSigner returns a Signer based on the given version and chainId.
func MakeSigner() Signer {
	return DefaultSigner{}
}

// SignTx signs the transaction using the given signer and private key
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

// Signer encapsulates transaction signature handling.
type Signer interface {
	// Sender returns the sender address of the transaction.
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
	V, R, S := tx.data.V, tx.data.R, tx.data.S

	if V.BitLen() > 32 {
		return common.Address{}, ErrInvalidSig
	}
	_, _, v, _ := ParseV(V)
	if !crypto.ValidateSignatureValues(v, R, S) {
		return common.Address{}, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	rb, sb := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	sig[64] = v
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
		tx.ChainId(),
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
