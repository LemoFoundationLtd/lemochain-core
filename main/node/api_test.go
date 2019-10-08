package node

import (
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/testchain"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

// TestAccountAPI_api account api test
func TestAccountAPI_api(t *testing.T) {
	bc, db := testchain.NewTestChain()
	defer testchain.CloseTestChain(bc, db)

	am := bc.AccountManager()
	acc := NewPublicAccountAPI(am)
	priAcc := NewPrivateAccountAPI(am)
	// Create key pair
	addressKeyPair, err := priAcc.NewKeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, addressKeyPair.Private)

	// getBalance api
	_, err = acc.GetBalance("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.Equal(t, ErrLemoAddress, err)

	address, err := common.StringToAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.NoError(t, err)
	b02 := acc.manager.GetCanonicalAccount(address).GetBalance().String()
	bb02, err := acc.GetBalance("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.NoError(t, err)
	assert.Equal(t, b02, bb02)

	b03 := acc.manager.GetCanonicalAccount(testchain.FounderAddr).GetBalance().String()
	bb03, err := acc.GetBalance(testchain.FounderAddr.String())
	assert.NoError(t, err)
	assert.Equal(t, b03, bb03)

	// get account api
	_, err = acc.GetAccount("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.Equal(t, ErrLemoAddress, err)
}

// TestChainAPI_api chain api test
func TestChainAPI_api(t *testing.T) {
	bc, db := testchain.NewTestChain()
	defer testchain.CloseTestChain(bc, db)
	c := NewPublicChainAPI(bc)

	// getBlockByHash
	targetBlock := testchain.LoadDefaultBlock(1)
	exBlock1 := c.chain.GetBlockByHash(targetBlock.Hash())

	assert.Equal(t, exBlock1.VersionRoot(), targetBlock.VersionRoot())
	assert.Equal(t, exBlock1.Height(), targetBlock.Height())
	assert.Equal(t, exBlock1.ParentHash(), targetBlock.ParentHash())
	assert.Equal(t, exBlock1.Header.LogRoot, targetBlock.Header.LogRoot)
	assert.Equal(t, exBlock1.Header.TxRoot, targetBlock.Header.TxRoot)

	assert.Equal(t, exBlock1, c.GetBlockByHash(targetBlock.Hash().String(), true))
	Block1 := &types.Block{
		Header: exBlock1.Header,
	}
	assert.Equal(t, Block1, c.GetBlockByHash(targetBlock.Hash().String(), false))

	// getBlockByHeight
	exBlock2 := c.chain.GetBlockByHeight(1)
	assert.Equal(t, exBlock2, c.GetBlockByHeight(1, true))
	Block2 := &types.Block{
		Header: exBlock2.Header,
	}
	assert.Equal(t, Block2, c.GetBlockByHeight(1, false))

	// get chain ID api
	assert.Equal(t, c.chain.ChainID(), c.ChainID())

	// get genesis block api
	assert.Equal(t, c.chain.Genesis(), c.Genesis())

	// get current block api
	curBlock := c.chain.CurrentBlock()
	assert.Equal(t, curBlock, c.CurrentBlock(true))
	cBlock := &types.Block{
		Header: curBlock.Header,
	}
	assert.Equal(t, cBlock, c.CurrentBlock(false))

	// get stable block api
	StaBlock := c.chain.StableBlock()
	assert.Equal(t, StaBlock, c.CurrentBlock(true))
	sBlock := &types.Block{
		Header: StaBlock.Header,
	}
	assert.Equal(t, sBlock, c.CurrentBlock(false))

	// get current chain height api
	assert.Equal(t, c.chain.CurrentBlock().Height(), c.CurrentHeight())

	// get latest stable block height
	assert.Equal(t, c.chain.StableBlock().Height(), c.CurrentHeight())

	// get suggest gas price
	// todo

	// get nodeVersion
	// todo
}

// TestTxAPI_api send tx api test
func TestTxAPI_api(t *testing.T) {
	bc, db := testchain.NewTestChain()
	defer testchain.CloseTestChain(bc, db)

	from := crypto.PubkeyToAddress(testchain.FounderPrivate.PublicKey)
	testTx := types.NewTransaction(from, common.HexToAddress("0x1"), common.Big1, 100, big.NewInt(1000000000), []byte{12}, 0, 100, uint64(time.Now().Unix()+60*30), "aa", string("send a Tx"))
	tx := testchain.SignTx(testTx, testchain.FounderPrivate)
	node := &Node{
		chainID: 100,
		chain:   bc,
		txPool:  txpool.NewTxPool(),
	}
	txAPI := NewPublicTxAPI(node)

	sendTxHash, err := txAPI.SendTx(tx)
	assert.NoError(t, err)
	assert.Equal(t, tx.Hash(), sendTxHash)
}

// 序列化注册候选节点所用data
func Test_CreatRegisterTxData(t *testing.T) {
	pro1 := make(types.Profile)
	pro1[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro1[types.CandidateKeyPort] = "1111"
	pro1[types.CandidateKeyNodeID] = "11f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	pro1[types.CandidateKeyIncomeAddress] = "Lemo11111"
	pro1[types.CandidateKeyHost] = "1111"
	marPro1, _ := json.Marshal(pro1)
	fmt.Println("txData1:", common.ToHex(marPro1))

	pro2 := make(types.Profile)
	pro2[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro2[types.CandidateKeyPort] = "2222"
	pro2[types.CandidateKeyNodeID] = "22f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	pro2[types.CandidateKeyIncomeAddress] = "Lemo2222"
	pro2[types.CandidateKeyHost] = "2222"
	marPro2, _ := json.Marshal(pro2)
	fmt.Println("txData2:", common.ToHex(marPro2))

	pro3 := make(types.Profile)
	pro3[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro3[types.CandidateKeyPort] = "3333"
	pro3[types.CandidateKeyNodeID] = "33f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	pro3[types.CandidateKeyIncomeAddress] = "Lemo3333"
	pro3[types.CandidateKeyHost] = "3333"
	marPro3, _ := json.Marshal(pro3)
	fmt.Println("txData3:", common.ToHex(marPro3))

	pro4 := make(types.Profile)
	pro4[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro4[types.CandidateKeyPort] = "4444"
	pro4[types.CandidateKeyNodeID] = "44f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	pro4[types.CandidateKeyIncomeAddress] = "Lemo4444"
	pro4[types.CandidateKeyHost] = "4444"
	marPro4, _ := json.Marshal(pro4)
	fmt.Println("txData4:", common.ToHex(marPro4))

	pro5 := make(types.Profile)
	pro5[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	pro5[types.CandidateKeyPort] = "5555"
	pro5[types.CandidateKeyNodeID] = "55f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	pro5[types.CandidateKeyIncomeAddress] = "Lemo5555"
	pro5[types.CandidateKeyHost] = "5555"
	marPro5, _ := json.Marshal(pro5)
	fmt.Println("txData5:", common.ToHex(marPro5))
}
