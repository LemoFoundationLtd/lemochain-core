package merkle

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"testing"
)

var (
	src = []string{
		"0x5f30cc80133b9394156e24b233f0c4be32b24e44bb3381f02c7ba52619d0febc",
		"0xbdd637c523ed5c0eab792b986db18850c239a2e23802b36aff26bb68fb3fe008",
		"0x2ed873ae0dd2372777c57a685d907083ca97290e508f15478ca98bad3225ebbc",
		"0x1e4c3d286aada9b795029df6a2da044699943ea25a18adfa2995ad798dbb922b",
		"0x4b7471ea1795646eac817cf8a39e94a0f3a120e3affe4ad69240b97c74896c25",
	}

	root = "0x4755fbecb3a0ba3df2be677d35b309ea398bc10c08bbb89962c72e8c6b6cff2a"
)

func makeHashes() []common.Hash {
	res := make([]common.Hash, 0)
	for _, s := range src {
		res = append(res, common.HexToHash(s))
	}
	return res
}

func Test_HashNodes(t *testing.T) {
	src := makeHashes()
	for _, hash := range src {
		fmt.Println(common.ToHex(hash[:]))
	}
	m := New(src)
	hashes := m.HashNodes()
	fmt.Printf("\r\ntotal size:%d\r\n", len(hashes))
	for _, hash := range hashes {
		fmt.Println(common.ToHex(hash[:]))
	}
}

func Test_FindSiblingNodes(t *testing.T) {
	src := makeHashes()
	m := New(src)
	hashes := m.HashNodes()
	hash := common.HexToHash("0x4b7471ea1795646eac817cf8a39e94a0f3a120e3affe4ad69240b97c74896c25")
	sibing, err := FindSiblingNodes(hash, hashes)
	if err != nil {
		t.Error(err)
	}
	verifyRoot := verifyRoot(hash, sibing)
	dstRoot := common.HexToHash(root)
	if bytes.Compare(verifyRoot[:], dstRoot[:]) != 0 {
		t.Error("not match")
	}
}

func verifyRoot(hash common.Hash, sibing []MerkleNode) common.Hash {
	for _, item := range sibing {
		fmt.Println(common.ToHex(item.Hash[:]))
		if item.NodeType == LeftNode {
			hash = crypto.Keccak256Hash(append(item.Hash[:], hash[:]...))
		} else if item.NodeType == RightNode {
			hash = crypto.Keccak256Hash(append(hash[:], item.Hash[:]...))
		}
	}
	return hash
}
