package miner

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

type NodePair struct {
	privateKey  string
	publicKey   string
	address     string
	lemoAddress string
}

var Nodes = []*NodePair{
	&NodePair{
		privateKey:  "c21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa",
		publicKey:   "5e3600755f9b512a65603b38e30885c98cbac70259c3235c9b3f42ee563b480edea351ba0ff5748a638fe0aeff5d845bf37a3b437831871b48fd32f33cd9a3c0",
		address:     "0x015780F8456F9c1532645087a19DcF9a7e0c7F97",
		lemoAddress: "Lemo83GN72GYH2NZ8BA729Z9TCT7KQ5FC3CR6DJG",
	},
	&NodePair{
		privateKey:  "9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb",
		publicKey:   "ddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0",
		address:     "016ad4Fc7e1608685Bf5fe5573973BF2B1Ef9B8A",
		lemoAddress: "Lemo83JW7TBPA7P2P6AR9ZC2WCQJYRNHZ4NJD4CY",
	},
	&NodePair{
		privateKey:  "ba9b51e59ec57d66b30b9b868c76d6f4d386ce148d9c6c1520360d92ef0f27ae",
		publicKey:   "7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2d0347692f36ef736f342fb5ce4710f7e337f062cc2110d134b63a9575f78cb167bfae2f43",
		address:     "01f98855Be9ecc5c23A28Ce345D2Cc04686f2c61",
		lemoAddress: "Lemo842BJZ4DKCC764C63Y6A943775JH6NQ3Z33Y",
	},
	&NodePair{
		privateKey:  "b381bad69ad4b200462a0cc08fcb8ba64d26efd4f49933c2c2448cb23f2cd9d0",
		publicKey:   "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f",
		address:     "0112fDDcF0C08132A5dcd9ED77e1a3348ff378D2",
		lemoAddress: "Lemo837QGPS3YNTYNF53CD88WA5DR3ABNA95W2DG",
	},
	&NodePair{
		privateKey:  "56b5fe1b8c40f0dec29b621a16ffcbc7a1bb5c0b0f910c5529f991273cd0569c",
		publicKey:   "5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797",
		address:     "016017aF50F4bB67101CE79298ACBdA1A3c12C15",
		lemoAddress: "Lemo83HKZK68JQZDRGS5PWT2ZBSKR5CRADCSJB9B",
	},
	&NodePair{
		privateKey:  "56b5fe1b8c40f0dec29b621a16ffcbc7a1bb5c0b0f910c5529f991273cd056ff",
		publicKey:   "5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb37ff",
		address:     "016017aF50F4bB67101CE79298ACBdA1A3c12Cff",
		lemoAddress: "Lemo83HKZK68JQZDRGS5PWT2ZBSKR5CRADCSJBff",
	},
}

var Cnf = &MineConfig{
	SleepTime: 3000,
	Timeout:   10000,
}

type blockInfo struct {
	hash        common.Hash
	parentHash  common.Hash
	height      uint32
	author      common.Address
	versionRoot common.Hash
	txRoot      common.Hash
	logRoot     common.Hash
	txList      []*types.Transaction
	gasLimit    uint64
	time        uint32
	deputyRoot  []byte
	deputyNodes deputynode.DeputyNodes
}

func makeBlock(db protocol.ChainDB, info blockInfo, save bool) *types.Block {
	manager := account.NewManager(info.parentHash, db)
	// sign transactions
	var err error
	var gasUsed uint64 = 0
	txRoot := types.DeriveTxsSha(info.txList)
	if txRoot != info.txRoot {
		if info.txRoot != (common.Hash{}) {
			fmt.Printf("%d txRoot hash error. except: %s, got: %s\n", info.height, info.txRoot.Hex(), txRoot.Hex())
		}
		info.txRoot = txRoot
	}

	var deputyRoot []byte
	if len(info.deputyNodes) > 0 {
		deputyRoot = types.DeriveDeputyRootSha(info.deputyNodes).Bytes()
	}
	if bytes.Compare(deputyRoot, info.deputyRoot) != 0 {
		if len(info.deputyNodes) > 0 || len(info.deputyRoot) != 0 {
			fmt.Printf("%d deputyRoot error. except: %s, got: %s\n", info.height, common.ToHex(info.deputyRoot), common.ToHex(deputyRoot))
		}
		info.deputyRoot = deputyRoot
	}

	// genesis coin
	if info.height == 0 {
		return nil
	}
	// account
	salary := new(big.Int)
	for _, tx := range info.txList {
		gas := params.TxGas + params.TxDataNonZeroGas*uint64(len(tx.Data()))
		fromAddr, err := tx.From()
		if err != nil {
			panic(err)
		}
		from := manager.GetAccount(fromAddr)
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		cost := new(big.Int).Add(tx.Amount(), fee)
		to := manager.GetAccount(*tx.To())
		// make sure the change log has right order
		if fromAddr.Hex() < tx.To().Hex() {
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Amount()))
		} else {
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Amount()))
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
		}
		gasUsed += gas
		salary.Add(salary, fee)
	}
	if salary.Cmp(new(big.Int)) != 0 {
		miner := manager.GetAccount(info.author)
		miner.SetBalance(new(big.Int).Add(miner.GetBalance(), salary))
	}
	err = manager.Finalise()
	if err != nil {
		panic(err)
	}
	// header
	if manager.GetVersionRoot() != info.versionRoot {
		if info.versionRoot != (common.Hash{}) {
			fmt.Printf("%d version root error. except: %s, got: %s\n", info.height, info.versionRoot.Hex(), manager.GetVersionRoot().Hex())
		}
		info.versionRoot = manager.GetVersionRoot()
	}
	changeLogs := manager.GetChangeLogs()
	// fmt.Printf("%d changeLogs %v\n", info.height, changeLogs)
	logRoot := types.DeriveChangeLogsSha(changeLogs)
	if logRoot != info.logRoot {
		if info.logRoot != (common.Hash{}) {
			fmt.Printf("%d change logs root error. except: %s, got: %s\n", info.height, info.logRoot.Hex(), logRoot.Hex())
		}
		info.logRoot = logRoot
	}
	if info.time == 0 {
		info.time = uint32(time.Now().Unix())
	}
	if info.gasLimit == 0 {
		info.gasLimit = 1000000
	}
	header := &types.Header{
		ParentHash:   info.parentHash,
		MinerAddress: info.author,
		VersionRoot:  info.versionRoot,
		TxRoot:       info.txRoot,
		LogRoot:      info.logRoot,
		// Bloom:        types.CreateBloom(nil),
		Height:   info.height,
		GasLimit: info.gasLimit,
		GasUsed:  gasUsed,
		Time:     info.time,
		Extra:    []byte{},
	}
	if len(info.deputyRoot) > 0 {
		header.DeputyRoot = make([]byte, len(info.deputyRoot))
		copy(header.DeputyRoot[:], info.deputyRoot[:])
	}

	blockHash := header.Hash()
	if blockHash != info.hash {
		if info.hash != (common.Hash{}) {
			fmt.Printf("%d block hash error. except: %s, got: %s\n", info.height, info.hash.Hex(), blockHash.Hex())
		}
		info.hash = blockHash
	}
	// block
	block := &types.Block{
		Txs:         info.txList,
		ChangeLogs:  changeLogs,
		DeputyNodes: info.deputyNodes,
	}
	block.SetHeader(header)
	if save {
		err = db.SetBlock(blockHash, block)
		if err != nil && err != store.ErrExist {
			panic(err)
		}
		err = manager.Save(blockHash)
		if err != nil {
			panic(err)
		}
	}
	return block
}

func makeSignBlock(key string, db protocol.ChainDB, info blockInfo, save bool) (*types.Block, error) {
	block := makeBlock(db, info, false)

	tmp, err := crypto.HexToECDSA(key)
	if err != nil {
		return nil, err
	}

	hash := block.Hash()
	signData, err := crypto.Sign(hash[:], tmp)
	if err != nil {
		return nil, err
	}

	block.Header.SignData = signData
	return block, nil
}

type EngineTestForMiner struct{}

func (engine *EngineTestForMiner) VerifyHeader(block *types.Block) error { return nil }

func (engine *EngineTestForMiner) Seal(header *types.Header, txs []*types.Transaction, changeLog []*types.ChangeLog) (*types.Block, error) {
	return nil, nil
}

func (engine *EngineTestForMiner) Finalize(header *types.Header, am *account.Manager) {}

func newBlockChain() (*chain.BlockChain, chan *types.Block, error) {
	chainID := uint16(99)
	db := store.NewChainDataBase(store.GetStorePath(), store.DRIVER_MYSQL, store.DNS_MYSQL)
	genesis := chain.DefaultGenesisBlock()
	_, err := chain.SetupGenesisBlock(db, genesis)
	if err != nil {
		return nil, nil, err
	}

	var engine EngineTestForMiner
	ch := make(chan *types.Block)
	blockChain, err := chain.NewBlockChain(chainID, &engine, db, nil)
	if err != nil {
		return nil, nil, err
	}

	return blockChain, ch, nil
}

func setSelfNodeKey(key string) {
	tmp, _ := crypto.HexToECDSA(key)
	deputynode.SetSelfNodeKey(tmp)
}

func newMiner(key string) (*Miner, error) {
	store.ClearData()
	setSelfNodeKey(key)

	blockChain, _, err := newBlockChain()
	if err != nil {
		return nil, err
	}

	txPool := chain.NewTxPool(100)
	return New(Cnf, blockChain, txPool, new(EngineTestForMiner)), nil
}

func calDeviation(ex int, src int) bool {
	if ex == 0 {
		if src == 0 {
			return true
		} else {
			return false
		}
	} else {
		if (ex >= (src - 1000)) && (ex <= (src + 1000)) {
			return true
		} else {
			return false
		}
	}
}

func init() {
	log.Setup(log.LevelDebug, false, true)
}

func TestMiner_GetSleepGenesis(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)

	me := Nodes[0].privateKey
	miner, err := newMiner(me)
	assert.NoError(t, err)

	reset0 := miner.getSleepTime()

	setSelfNodeKey(Nodes[1].privateKey)
	reset1 := miner.getSleepTime()

	setSelfNodeKey(Nodes[2].privateKey)
	reset2 := miner.getSleepTime()

	setSelfNodeKey(Nodes[3].privateKey)
	reset3 := miner.getSleepTime()

	setSelfNodeKey(Nodes[4].privateKey)
	reset4 := miner.getSleepTime()

	if (reset0 == 0) || (reset1 == 0) || (reset2 == 0) || (reset3 == 0) || (reset4 == 0) {
		assert.Equal(t, true, true)
	} else {
		assert.Equal(t, true, false)
	}
}

func TestMine_GetSleepNotSelf(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)

	miner, err := newMiner(Nodes[0].privateKey)
	assert.NoError(t, err)

	genesis := miner.chain.GetBlockByHeight(0)
	info := blockInfo{parentHash: genesis.Hash(), height: 1, author: common.HexToAddress(Nodes[0].address)}
	block, err := makeSignBlock(Nodes[0].privateKey, miner.chain.Db(), info, false)
	assert.NoError(t, err)

	err = miner.chain.InsertChain(block, true)
	assert.NoError(t, err)
	miner.chain.SetStableBlock(block.Hash(), block.Height())
	assert.NoError(t, err)

	setSelfNodeKey(Nodes[5].privateKey)
	reset := miner.getSleepTime()
	assert.Equal(t, -1, reset)
}

func TestMiner_GetSleep1Deputy(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, deputynode.DeputyNodes{chain.DefaultDeputyNodes[0]})
	setSelfNodeKey(Nodes[0].privateKey)

	miner, err := newMiner(Nodes[0].privateKey)
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	assert.Equal(t, int(Cnf.SleepTime), reset)
}

func TestMiner_GetSleepValidAuthor(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()

	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)
	miner, err := newMiner(Nodes[0].privateKey)
	assert.NoError(t, err)

	genesis := miner.chain.GetBlockByHeight(0)
	info := blockInfo{parentHash: genesis.Hash(), height: 1, author: common.HexToAddress(Nodes[5].address)}
	block, err := makeSignBlock(Nodes[5].privateKey, miner.chain.Db(), info, false)
	assert.NoError(t, err)
	err = miner.chain.InsertChain(block, true)
	assert.NoError(t, err)
	miner.chain.SetStableBlock(block.Hash(), block.Height())
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	assert.Equal(t, -1, reset)
}

func TestMiner_GetSleepSlot1(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)

	miner, err := newMiner(Nodes[0].privateKey)
	assert.NoError(t, err)

	genesis := miner.chain.GetBlockByHeight(0)
	wait := 1
	info := blockInfo{
		parentHash: genesis.Hash(),
		height:     1,
		author:     common.HexToAddress(Nodes[0].address),
		time:       uint32(time.Now().Unix()) - uint32(wait),
	}
	block, err := makeSignBlock(Nodes[0].privateKey, miner.chain.Db(), info, false)
	assert.NoError(t, err)

	err = miner.chain.InsertChain(block, true)
	assert.NoError(t, err)
	miner.chain.SetStableBlock(block.Hash(), block.Height())
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	fmt.Printf("NODE[0]: %d, blocktime: %d, currenttime: %d\r\n", reset, block.Time(), time.Now().Unix())
	assert.Equal(t, calDeviation(40000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[1].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[1]:", reset)
	assert.Equal(t, calDeviation(int(Cnf.SleepTime)-wait*1000, reset), true)

	setSelfNodeKey(Nodes[2].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[2]:", reset)
	assert.Equal(t, calDeviation(10000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[3].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[3]:", reset)
	assert.Equal(t, calDeviation(20000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[4].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[4]:", reset)
	assert.Equal(t, calDeviation(30000-wait*1000, reset), true)
}

func TestMiner_GetSleepSlot2(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)

	miner, err := newMiner(Nodes[0].privateKey)
	assert.NoError(t, err)

	genesis := miner.chain.GetBlockByHeight(0)
	wait := 31
	info := blockInfo{
		parentHash: genesis.Hash(),
		height:     1,
		author:     common.HexToAddress(Nodes[0].address),
		time:       uint32(time.Now().Unix()) - uint32(wait),
	}
	block, err := makeSignBlock(Nodes[0].privateKey, miner.chain.Db(), info, false)
	assert.NoError(t, err)

	err = miner.chain.InsertChain(block, true)
	assert.NoError(t, err)
	miner.chain.SetStableBlock(block.Hash(), block.Height())
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	fmt.Println("NODE[0]:", reset)
	assert.Equal(t, calDeviation(40000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[1].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[1]:", reset)
	assert.Equal(t, calDeviation(50000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[2].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[2]:", reset)
	assert.Equal(t, calDeviation(50000+10000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[3].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[3]:", reset)
	assert.Equal(t, calDeviation(50000+20000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[4].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[4]:", reset)
	assert.Equal(t, 0, reset)
}

func TestMiner_GetSleepSlot3(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)

	miner, err := newMiner(Nodes[0].privateKey)
	assert.NoError(t, err)

	genesis := miner.chain.GetBlockByHeight(0)
	wait := 45
	info := blockInfo{
		parentHash: genesis.Hash(),
		height:     1,
		author:     common.HexToAddress(Nodes[0].address),
		time:       uint32(time.Now().Unix()) - uint32(wait),
	}
	block, err := makeSignBlock(Nodes[0].privateKey, miner.chain.Db(), info, false)
	assert.NoError(t, err)

	err = miner.chain.InsertChain(block, true)
	assert.NoError(t, err)
	miner.chain.SetStableBlock(block.Hash(), block.Height())
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	fmt.Println("NODE[0]:", reset)
	assert.Equal(t, 0, reset)

	setSelfNodeKey(Nodes[1].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[1]:", reset)
	assert.Equal(t, calDeviation(50000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[2].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[2]:", reset)
	assert.Equal(t, calDeviation(50000+10000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[3].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[3]:", reset)
	assert.Equal(t, calDeviation(50000+20000-wait*1000, reset), true)

	setSelfNodeKey(Nodes[4].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[4]:", reset)
	assert.Equal(t, calDeviation(50000+30000-wait*1000, reset), true)
}

func TestMiner_GetSleepSlot4(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)

	miner, err := newMiner(Nodes[0].privateKey)
	assert.NoError(t, err)

	genesis := miner.chain.GetBlockByHeight(0)
	wait := 45
	info := blockInfo{
		parentHash: genesis.Hash(),
		height:     1,
		author:     common.HexToAddress(Nodes[0].address),
		time:       uint32(time.Now().Unix()) - uint32(wait),
	}
	block, err := makeSignBlock(Nodes[0].privateKey, miner.chain.Db(), info, false)
	assert.NoError(t, err)

	err = miner.chain.InsertChain(block, true)
	assert.NoError(t, err)
	miner.chain.SetStableBlock(block.Hash(), block.Height())
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	fmt.Println("NODE[0]:", reset)
	assert.Equal(t, 0, reset)

	setSelfNodeKey(Nodes[1].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[1]:", reset)
	assert.Equal(t, calDeviation(10000-(wait*1000)%40000, reset), true)

	setSelfNodeKey(Nodes[2].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[2]:", reset)
	assert.Equal(t, calDeviation(20000-(wait*1000)%40000, reset), true)

	setSelfNodeKey(Nodes[3].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[3]:", reset)
	assert.Equal(t, calDeviation(30000-(wait*1000)%40000, reset), true)

	setSelfNodeKey(Nodes[4].privateKey)
	reset = miner.getSleepTime()
	fmt.Println("NODE[4]:", reset)
	assert.Equal(t, calDeviation(40000-(wait*1000)%40000, reset), true)
}

// func TestMiner_GetSleepSlot5(t *testing.T) {
// 	store.ClearData()
// 	deputynode.Instance().Clear()
// 	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)
//
// 	miner, err := newMiner(Nodes[0].privateKey)
// 	assert.NoError(t, err)
//
// 	genesis := miner.chain.GetBlockByHeight(0)
// 	wait := 95
// 	info := blockInfo{
// 		parentHash: genesis.Hash(),
// 		height:     1,
// 		author:     common.HexToAddress(Nodes[0].address),
// 		time:       uint32(time.Now().Unix()) - uint32(wait),
// 	}
// 	block, err := makeSignBlock(Nodes[0].privateKey, miner.chain.Db(), info, false)
// 	assert.NoError(t, err)
//
// 	err = miner.chain.InsertChain(block, true)
// 	assert.NoError(t, err)
// 	miner.chain.SetStableBlock(block.Hash(), block.Height(), false)
// 	assert.NoError(t, err)
//
// 	reset := miner.getSleepTime()
// 	fmt.Println("NODE[0]:", reset)
// 	assert.Equal(t, 0, reset)
//
// 	setSelfNodeKey(Nodes[1].privateKey)
// 	reset = miner.getSleepTime()
// 	fmt.Println("NODE[1]:", reset)
// 	assert.Equal(t, calDeviation(50000-(wait*1000)%50000, reset), true)
//
// 	setSelfNodeKey(Nodes[2].privateKey)
// 	reset = miner.getSleepTime()
// 	fmt.Println("NODE[2]:", reset)
// 	//assert.Equal(t, calDeviation(20000-(wait*1000)%40000, reset), true)
// 	assert.Equal(t, 0, reset)
//
// 	setSelfNodeKey(Nodes[3].privateKey)
// 	reset = miner.getSleepTime()
// 	fmt.Println("NODE[3]:", reset)
// 	assert.Equal(t, calDeviation(30000-(wait*1000)%40000, reset), true)
//
// 	setSelfNodeKey(Nodes[4].privateKey)
// 	reset = miner.getSleepTime()
// 	fmt.Println("NODE[4]:", reset)
// 	assert.Equal(t, calDeviation(40000-(wait*1000)%40000, reset), true)
// }

func TestMiner_GetSleepNormal(t *testing.T) {
	store.ClearData()
	deputynode.Instance().Clear()
	deputynode.Instance().Add(0, chain.DefaultDeputyNodes)

	me := Nodes[0].privateKey
	miner, err := newMiner(me)
	assert.NoError(t, err)

	genesis := miner.chain.GetBlockByHeight(0)
	info := blockInfo{parentHash: genesis.Hash(), height: 1, author: common.HexToAddress(Nodes[0].address)}
	block, err := makeSignBlock(Nodes[0].privateKey, miner.chain.Db(), info, false)
	assert.NoError(t, err)

	err = miner.chain.InsertChain(block, true)
	assert.NoError(t, err)
	miner.chain.SetStableBlock(block.Hash(), block.Height())
	assert.NoError(t, err)

	reset := miner.getSleepTime()
	assert.Equal(t, calDeviation(40000, reset), true)

	setSelfNodeKey(Nodes[1].privateKey)
	reset = miner.getSleepTime()
	assert.Equal(t, calDeviation(int(Cnf.SleepTime), reset), true)

	setSelfNodeKey(Nodes[2].privateKey)
	reset = miner.getSleepTime()
	assert.Equal(t, calDeviation(10000, reset), true)

	setSelfNodeKey(Nodes[3].privateKey)
	reset = miner.getSleepTime()
	assert.Equal(t, calDeviation(20000, reset), true)

	setSelfNodeKey(Nodes[4].privateKey)
	reset = miner.getSleepTime()
	assert.Equal(t, calDeviation(30000, reset), true)
}

func Test_NextTerm(t *testing.T) {

}
