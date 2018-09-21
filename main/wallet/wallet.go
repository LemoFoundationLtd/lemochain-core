package wallet

import (
	"crypto/ecdsa"
)

type Wallet struct {
	// todo
}

// NewWallet generate wallet
func NewWallet() (*Wallet, error) {

	return newWalletFromECDSA(nil), nil
}

// newWalletFromECDSA return the address wallet with the private key
func newWalletFromECDSA(privateKey *ecdsa.PrivateKey) *Wallet {
	w := &Wallet{}
	return w
}
