package txprocessor

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
	"time"
)

var (
	register            = common.HexToAddress("0x19999")
	normalNodeId        = "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	normalIncomeAddress = common.HexToAddress("0x12212").String()
	normalHost          = "www.lemochain.com"
	normalPort          = "7001"
)

// newCandidateTx 生成申请候选节点交易
func newCandidateTx(register common.Address, isCandidate bool, incomeAddress, nodeId, host, port string) *types.Transaction {
	profile := make(types.Profile)
	profile[types.CandidateKeyIsCandidate] = strconv.FormatBool(isCandidate)
	profile[types.CandidateKeyIncomeAddress] = incomeAddress
	if nodeId != "" {
		profile[types.CandidateKeyNodeID] = nodeId
	}
	if host != "" {
		profile[types.CandidateKeyHost] = host
	}
	if port != "" {
		profile[types.CandidateKeyPort] = port
	}
	data, _ := json.Marshal(profile)

	tx := types.NewTransaction(params.FeeReceiveAddress, params.RegisterCandidateNodeFees, 200000, common.Big1, data, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", "")
	// store tx register
	tx.StoreFromForTest(register)
	return tx
}

// Test_buildProfile 测试构建build profile函数
func Test_buildProfile(t *testing.T) {
	/*
		1. 检测传入的nodeID,host和incomeAddress
		2. 检测是否传入了nodeID,host和port
		3. 正常情况
	*/
	// 1.1 传入长度不对的nodeID
	nodeId_lengthErr := "7739f34055d3c0808683dbd77a937f8e28f707d5b1e873bbe61f6f2"
	tx01 := newCandidateTx(register, true, normalIncomeAddress, nodeId_lengthErr, normalHost, normalPort)
	_, err := buildProfile(tx01)
	assert.Equal(t, ErrInvalidNodeId, err)

	// 1.2 传入的nodeId不是通过链生成的标准的nodeId
	nodeId_invaliable := "444444444444444444444444444444444444444qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqfffffffffffffffffffffffffffffffffffsssssssssssss55555555"
	tx02 := newCandidateTx(register, true, normalIncomeAddress, nodeId_invaliable, normalHost, normalPort)
	_, err02 := buildProfile(tx02)
	assert.Equal(t, ErrInvalidNodeId, err02)

	// 1.3 传入的incomeAddress 地址不是标准的地址
	errAddress := "LemoDD7777777777777777EFFFDSDDCCCCCAAAAA"
	tx03 := newCandidateTx(register, true, errAddress, normalNodeId, normalHost, normalPort)
	_, err = buildProfile(tx03)
	assert.Equal(t, ErrInvalidAddress, err)

	// 1.4 传入的host长度超过了限制(128)
	errHost := "www.lemoSDDCCCCCAAAAA51bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d72814f0df789b46e9bc09f23SDDCCCCCAAAAAchain.com"
	tx04 := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, errHost, normalPort)
	_, err = buildProfile(tx04)
	assert.Equal(t, ErrInvalidHost, err)

	// 2.1 未传入nodeId
	tx05 := newCandidateTx(register, true, normalIncomeAddress, "", normalHost, normalPort)
	_, err = buildProfile(tx05)
	assert.Equal(t, ErrOfRegisterNodeID, err)

	// 2.2 未传入host
	tx06 := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, "", normalPort)
	_, err = buildProfile(tx06)
	assert.Equal(t, ErrOfRegisterHost, err)
	// 2.3 未传入port
	tx07 := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, normalHost, "")
	_, err = buildProfile(tx07)
	assert.Equal(t, ErrOfRegisterPort, err)

	// 3.1 正常情况
	normalTx := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	newProfile, err := buildProfile(normalTx)
	assert.NoError(t, err)
	assert.Equal(t, normalIncomeAddress, newProfile[types.CandidateKeyIncomeAddress])
	assert.Equal(t, normalNodeId, newProfile[types.CandidateKeyNodeID])
	assert.Equal(t, normalHost, newProfile[types.CandidateKeyHost])
	assert.Equal(t, normalPort, newProfile[types.CandidateKeyPort])
}

// TestCandidateVoteEnv_RegisterOrUpdateToCandidate 注册候选节点交易测试
func TestCandidateVoteEnv_RegisterOrUpdateToCandidate(t *testing.T) {
	/*
		1. balance不足的情况
		2. 已经注销候选节点之后再次注册的情况
		3. 第一次注册的正常情况
		4. 已经是候选节点，修改候选节点信息的情况(验证nodeId，不能被修改)
	*/
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	c := NewCandidateVoteEnv(am)
	initialSenderBalance := big.NewInt(5555)
	// 足够的balance给注册者
	registerAcc := c.am.GetAccount(register)
	registerAcc.SetBalance(new(big.Int).Sub(params.RegisterCandidateNodeFees, big.NewInt(2)))

	var snapshot = c.am.Snapshot()
	// 1. balance不足以支付质押lemo
	registerAcc.SetBalance(big.NewInt(0))
	tx01 := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err := c.RegisterOrUpdateToCandidate(tx01, initialSenderBalance)
	assert.Equal(t, ErrInsufficientBalance, err)

	c.am.RevertToSnapshot(snapshot)
	// 2. 注销之后的候选节点不能再次注册了
	// 构造注销状态
	registerAcc.SetCandidateState(types.CandidateKeyIsCandidate, params.NotCandidateNode)
	tx02 := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(tx02, initialSenderBalance)
	// 返回已经注销无法再次注册的错误
	assert.Equal(t, ErrAgainRegister, err)

	c.am.RevertToSnapshot(snapshot)
	// 3. 首次注册的正常情况
	tx03 := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(tx03, initialSenderBalance)
	assert.NoError(t, err)
	// 验证注册的候选节点信息
	p := registerAcc.GetCandidate()
	assert.Equal(t, normalIncomeAddress, p[types.CandidateKeyIncomeAddress])
	assert.Equal(t, normalNodeId, p[types.CandidateKeyNodeID])
	assert.Equal(t, normalHost, p[types.CandidateKeyHost])
	assert.Equal(t, normalPort, p[types.CandidateKeyPort])
	assert.Equal(t, "true", p[types.CandidateKeyIsCandidate])
	// 验证投票给自己，并且自己此时的票数为initialSenderBalance
	voteFor := registerAcc.GetVoteFor()
	assert.Equal(t, register, voteFor)
	votes := registerAcc.GetVotes()
	assert.Equal(t, initialSenderBalance, votes)

	c.am.RevertToSnapshot(snapshot)
	// 4. 已经是候选节点，修改候选节点信息
	// 注册候选节点
	rTx := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(rTx, initialSenderBalance)
	assert.NoError(t, err)
	// 4.1 修改信息为 注销候选节点
	tx04 := newCandidateTx(register, false, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(tx04, initialSenderBalance)
	assert.NoError(t, err)
	// 注销之后
	pro := registerAcc.GetCandidate()
	assert.Equal(t, params.NotCandidateNode, pro[types.CandidateKeyIsCandidate])
	votes = registerAcc.GetVotes() // 得票数变为0
	assert.Equal(t, big.NewInt(0), votes)
	// todo 之后如果修改为质押的话，注销候选节点会退还质押的lemo

	c.am.RevertToSnapshot(snapshot)
	// 4.2 修改包含nodeId信息，测试nodeId是否被修改和其他信息是否修改成功
	// 注册候选节点
	nTx := newCandidateTx(register, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(nTx, initialSenderBalance)
	assert.NoError(t, err)
	newNodeId := "5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797"
	newIncomeAddress := common.HexToAddress("0x99999999").String()
	newHost := "127.0.0.1"
	newPort := "5001"

	tx05 := newCandidateTx(register, true, newIncomeAddress, newNodeId, newHost, newPort)
	err = c.RegisterOrUpdateToCandidate(tx05, initialSenderBalance)
	assert.NoError(t, err)
	// 验证修改后的信息
	newPro := registerAcc.GetCandidate()
	assert.Equal(t, normalNodeId, newPro[types.CandidateKeyNodeID]) // 验证nodeId不能被修改
	assert.Equal(t, newIncomeAddress, newPro[types.CandidateKeyIncomeAddress])
	assert.Equal(t, newHost, newPro[types.CandidateKeyHost])
	assert.Equal(t, newPort, newPro[types.CandidateKeyPort])
}

// TestCandidateVoteEnv_CallVoteTx 投票交易测试
func TestCandidateVoteEnv_CallVoteTx(t *testing.T) {
	/*
		1. 测试投的不是候选节点
		2. 第一次投票
		3. 不是第一次投票：重复投同一个候选节点和转投其他节点
	*/
	ClearData()
	db := newDB()
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	c := NewCandidateVoteEnv(am)
	initialSenderBalance := big.NewInt(5000000) // 票数
	// 构造一个候选节点，该候选节点原本的票数为两倍于 initialSenderBalance
	candAddr := common.HexToAddress("0x13333000")
	candAcc := c.am.GetAccount(candAddr)
	candAcc.SetCandidateState(types.CandidateKeyIsCandidate, params.IsCandidateNode) // 设置为候选节点
	candAcc.SetVotes(new(big.Int).Mul(initialSenderBalance, big.NewInt(2)))
	// 构造一个投票者voter,balance数量为 initialSenderBalance
	voterAddr := common.HexToAddress("0x91191")
	voterAcc := c.am.GetAccount(voterAddr)
	voterAcc.SetBalance(initialSenderBalance) // 设置balance

	var snapshot = c.am.Snapshot()
	// 1. 测试投给不是候选节点
	err := c.CallVoteTx(voterAddr, common.HexToAddress("0x88787"), initialSenderBalance)
	assert.Equal(t, ErrOfNotCandidateNode, err)

	// 2. voter第一次投票
	assert.Equal(t, common.Address{}, voterAcc.GetVoteFor()) // 初始未投过票
	err = c.CallVoteTx(voterAddr, candAddr, initialSenderBalance)
	assert.NoError(t, err)
	// 候选节点票数变化
	newVotes := candAcc.GetVotes()
	assert.Equal(t, new(big.Int).Mul(initialSenderBalance, big.NewInt(3)), newVotes) // 在两倍initialSenderBalance的基础上再增加了initialSenderBalance
	// voteFor 变化
	assert.Equal(t, candAddr, voterAcc.GetVoteFor())

	c.am.RevertToSnapshot(snapshot)
	// 3.1 重复投给同一个候选节点
	voterAcc.SetVoteFor(candAddr) // 设置已经投过了candAddr候选节点
	// 给candAddr投票
	err = c.CallVoteTx(voterAddr, candAddr, initialSenderBalance)
	assert.Equal(t, ErrOfAgainVote, err) // 返回不能再次投同一个候选节点的错误

	c.am.RevertToSnapshot(snapshot)
	// 3.2 从投给的candAddr候选节点转投到其他节点
	voterAcc.SetVoteFor(candAddr) // 设置投给candAddr候选节点

	// 构造一个新的候选节点,初始得票数为0
	newCandAddr := common.HexToAddress("0x6666")
	newCandAcc := c.am.GetAccount(newCandAddr)
	newCandAcc.SetCandidateState(types.CandidateKeyIsCandidate, params.IsCandidateNode)

	// voter给newCandAddr投票
	err = c.CallVoteTx(voterAddr, newCandAddr, initialSenderBalance)
	assert.NoError(t, err)

	assert.Equal(t, newCandAddr, voterAcc.GetVoteFor())          // 查看voter的votFor
	assert.Equal(t, initialSenderBalance, newCandAcc.GetVotes()) // newCandAddr候选节点票数
	assert.Equal(t, initialSenderBalance, candAcc.GetVotes())    // candAddr候选节点由最初的两倍initialSenderBalance减少到一倍initialSenderBalance

}
