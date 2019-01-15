package node

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestAccountAPI_api account api test
func TestAccountAPI_api(t *testing.T) {
	db := newDB()
	defer store.ClearData()
	am := account.NewManager(common.Hash{}, db)
	acc := NewPublicAccountAPI(am)
	priAcc := NewPrivateAccountAPI(am)
	// Create key pair
	addressKeyPair, err := priAcc.NewKeyPair()
	assert.NoError(t, err)
	t.Log(addressKeyPair)

	// getBalance api
	b01 := acc.manager.GetCanonicalAccount(common.HexToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")).GetBalance().String()
	bb01, err := acc.GetBalance("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.NoError(t, err)
	assert.Equal(t, b01, bb01)

	address, err := common.StringToAddress("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.NoError(t, err)
	b02 := acc.manager.GetCanonicalAccount(address).GetBalance().String()
	bb02, err := acc.GetBalance("Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG")
	assert.NoError(t, err)
	assert.Equal(t, b02, bb02)

	b03 := acc.manager.GetCanonicalAccount(testAddr).GetBalance().String()
	bb03, err := acc.GetBalance(testAddr.String())
	assert.NoError(t, err)
	assert.Equal(t, b03, bb03)

	// get account api
	account01, err := acc.GetAccount("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.NoError(t, err)
	addr, err := common.StringToAddress("0x015780F8456F9c1532645087a19DcF9a7e0c7F97")
	assert.NoError(t, err)
	assert.Equal(t, acc.manager.GetCanonicalAccount(addr), account01)

}

// TestChainAPI_api chain api test
func TestChainAPI_api(t *testing.T) {
	bc := newChain()
	defer store.ClearData()
	c := NewPublicChainAPI(bc)

	// getBlockByHash
	exBlock1 := c.chain.GetBlockByHash(common.HexToHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211"))
	assert.Equal(t, exBlock1, c.GetBlockByHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211", true))
	Block1 := &types.Block{
		Header: exBlock1.Header,
	}
	assert.Equal(t, Block1, c.GetBlockByHash("0x3f4c3152fb02a7673bf804b1ddeb75542b6ef9a5a87501d9cfbbcf6c3632a211", false))

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
	assert.Equal(t, StaBlock, c.LatestStableBlock(true))
	sBlock := &types.Block{
		Header: StaBlock.Header,
	}
	assert.Equal(t, sBlock, c.LatestStableBlock(false))

	// get current chain height api
	assert.Equal(t, c.chain.CurrentBlock().Height(), c.CurrentHeight())

	// get latest stable block height
	assert.Equal(t, c.chain.StableBlock().Height(), c.LatestStableHeight())

	// get suggest gas price
	// todo

	// get nodeVersion
	// todo

}

// TestTxAPI_api send tx api test
func TestTxAPI_api(t *testing.T) {
	testTx := types.NewTransaction(common.HexToAddress("0x1"), common.Big1, 100, common.Big2, []byte{12}, 0, 200, uint64(1544596), "aa", string("send a Tx"))
	signTx := signTransaction(testTx, testPrivate)
	// txCh := make(chan types.Transactions, 100)
	node := &Node{}
	txAPI := NewPublicTxAPI(node)

	sendTxHash, err := txAPI.SendTx(signTx)
	assert.Nil(t, err)
	assert.Equal(t, signTx.Hash(), sendTxHash)
}

// // TestMineAPI_api miner api test // todo
// func TestMineAPI_api(t *testing.T) {
// 	lemoConf := &LemoConfig{
// 		Genesis:   chain.DefaultGenesisBlock(),
// 		NetworkId: 1,
// 		MaxPeers:  1000,
// 		Port:      7001,
// 		NodeKey:   "0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa",
// 		ExtraData: []byte{},
// 	}
// 	testNode, err := New(lemoConf, &DefaultNodeConfig, flag.NewCmdFlags(&cli.Context{}, []cli.Flag{}))
// 	t.Error(err)
//
// 	miner := (*testNode).miner
// 	m := NewMineAPI(miner)
// 	t.Log("after:", m.IsMining())
// 	m.MineStart()
// 	t.Log("then:", m.IsMining())
// 	// todo
// 	m.MineStop()
// 	t.Log("last:", m.IsMining())
//
// 	assert.Equal(t, "0x015780F8456F9c1532645087a19DcF9a7e0c7F97", m.MinerAddress())
//
// }

// TestNewPublicTxAPI_EstimateGas
func TestNewPublicTxAPI_EstimateGas(t *testing.T) {
	Chain := newChain()
	node := &Node{
		chain: Chain,
	}
	p := NewPublicTxAPI(node)
	// from, _ := common.StringToAddress("Lemo837QGPS3YNTYNF53CD88WA5DR3ABNA95W2DG")
	sdata := "608060405234801561001057600080fd5b506040516040806105e983398101806040528101908080519060200190929190805190602001909291905050508160028190555042600181905550806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505061054d8061009c6000396000f300608060405260043610610083576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063167fb50e146100955780631998aeef146100c057806338af3eed146100ca5780634f245ef714610121578063996657af1461014c578063b7db7e64146101a3578063d074a38d146101ba575b34801561008f57600080fd5b50600080fd5b3480156100a157600080fd5b506100aa6101e5565b6040518082815260200191505060405180910390f35b6100c86101eb565b005b3480156100d657600080fd5b506100df610371565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561012d57600080fd5b50610136610396565b6040518082815260200191505060405180910390f35b34801561015857600080fd5b5061016161039c565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156101af57600080fd5b506101b86103c2565b005b3480156101c657600080fd5b506101cf61051b565b6040518082815260200191505060405180910390f35b60045481565b600254600154014211156101fe57600080fd5b6004543411151561020e57600080fd5b6000600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561036f57600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166108fc6004549081150290604051600060405180830381858888f193505050501580156102ba573d6000803e3d6000fd5b5033600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550346004819055507fdfea07ab8527bd08519bfa633240757a7bb0a7f3c7adc98e30604ba73c70f4293334604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390a15b565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60015481565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60025460015401421115156103d657600080fd5b600560009054906101000a900460ff16156103f057600080fd5b7f917fd6d893e435f61cf143a3149d8db6cd1e06c6367f7448bda8e98d75e202f6600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16600454604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390a16000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166108fc3073ffffffffffffffffffffffffffffffffffffffff16319081150290604051600060405180830381858888f193505050501580156104fd573d6000803e3d6000fd5b506001600560006101000a81548160ff021916908315150217905550565b600254815600a165627a7a72305820a78d48dd525392b97d4830068c7fc783e921cf9fa197849dc35e18c1726f19c20029"
	data := common.FromHex(sdata)
	// args := NewCallArgs(from, nil, 0, big.NewInt(0), big.NewInt(0), data)
	gas, err := p.EstimateGas(nil, data)
	t.Log(gas, err)
}
