package consensus

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
)

// cache confirm to save CPU. This confirm may not be used at last
var sigCache struct {
	Hash common.Hash
	Sig  []byte
}

// SignBlock sign a block hash by node key
func SignBlock(blockHash common.Hash) ([]byte, error) {
	if sigCache.Hash == blockHash {
		return sigCache.Sig, nil
	}

	// sign
	sig, err := crypto.Sign(hash[:], deputynode.GetSelfNodeKey())
	if err != nil {
		return []byte{}, err
	}

	// save to cache
	sigCache.Hash = hash
	sigCache.Sig = sig

	return sigCache.Sig, nil
}
