package transaction

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
	register02          = common.HexToAddress("0x9999111")
	register03          = common.HexToAddress("0x99998888")
	normalNodeId        = "34f0df789b46e9bc09f23d5315b951bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d7281e4d63baba913e9fa6908f"
	normalIncomeAddress = common.HexToAddress("0x12212").String()
	normalHost          = "www.lemochain.com"
	normalPort          = "7001"
)

// newCandidateTx 生成申请候选节点交易
func newCandidateTx(register common.Address, amount *big.Int, isCandidate bool, incomeAddress, nodeId, host, port string) *types.Transaction {
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

	tx := types.NewTransaction(register, params.CandidateDepositAddress, amount, 200000, common.Big1, data, params.RegisterTx, chainID, uint64(time.Now().Unix()+300), "", "")

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
	tx01 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, nodeId_lengthErr, normalHost, normalPort)
	_, err := buildProfile(tx01)
	assert.Equal(t, ErrInvalidNodeId, err)

	// 1.2 传入的nodeId不是通过链生成的标准的nodeId
	nodeId_invaliable := "444444444444444444444444444444444444444qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqfffffffffffffffffffffffffffffffffffsssssssssssss55555555"
	tx02 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, nodeId_invaliable, normalHost, normalPort)
	_, err02 := buildProfile(tx02)
	assert.Equal(t, ErrInvalidNodeId, err02)

	// 1.3 传入的incomeAddress 地址不是标准的地址
	errAddress := "LemoDD7777777777777777EFFFDSDDCCCCCAAAAA"
	tx03 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, errAddress, normalNodeId, normalHost, normalPort)
	_, err = buildProfile(tx03)
	assert.Equal(t, ErrInvalidAddress, err)

	// 1.4 传入的host长度超过了限制(128)
	errHost := "www.lemoSDDCCCCCAAAAA51bc77bbfeda653ae6f5aab564c9b4619322fddb3b1f28d1c434250e9d4dd8f51aa8334573d72814f0df789b46e9bc09f23SDDCCCCCAAAAAchain.com"
	tx04 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, normalNodeId, errHost, normalPort)
	_, err = buildProfile(tx04)
	assert.Equal(t, ErrInvalidHost, err)

	// 2.1 未传入nodeId
	tx05 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, "", normalHost, normalPort)
	_, err = buildProfile(tx05)
	assert.Equal(t, ErrOfRegisterNodeID, err)

	// 2.2 未传入host
	tx06 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, normalNodeId, "", normalPort)
	_, err = buildProfile(tx06)
	assert.Equal(t, ErrOfRegisterHost, err)
	// 2.3 未传入port
	tx07 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, normalNodeId, normalHost, "")
	_, err = buildProfile(tx07)
	assert.Equal(t, ErrOfRegisterPort, err)

	// 3.1 正常情况
	normalTx := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
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
	// 足够的balance给注册者
	registerAcc := c.am.GetAccount(register)
	registerAcc.SetBalance(new(big.Int).Mul(params.RegisterCandidatePledgeAmount, big.NewInt(2)))
	register02Acc := c.am.GetAccount(register02)
	register02Acc.SetBalance(new(big.Int).Mul(params.RegisterCandidatePledgeAmount, big.NewInt(2)))
	register03Acc := c.am.GetAccount(register03)
	register03Acc.SetBalance(new(big.Int).Mul(params.RegisterCandidatePledgeAmount, big.NewInt(2)))

	var snapshot = c.am.Snapshot()
	// 1. balance不足以支付质押lemo
	registerAcc.SetBalance(big.NewInt(0))
	tx01 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err := c.RegisterOrUpdateToCandidate(tx01)
	assert.Equal(t, ErrInsufficientBalance, err)

	c.am.RevertToSnapshot(snapshot)
	var snap = c.am.Snapshot()
	// 2. 注销之后的候选节点不能再次注册了
	// 构造注销状态
	registerAcc.SetCandidateState(types.CandidateKeyIsCandidate, params.NotCandidateNode)
	tx02 := newCandidateTx(register, nil, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(tx02)
	// 返回已经注销无法再次注册的错误
	assert.Equal(t, ErrRegisterAgain, err)
	acc := c.am.GetAccount(register)
	t.Log(acc.GetCandidateState(types.CandidateKeyIsCandidate))
	c.am.RevertToSnapshot(snap)
	accc := c.am.GetAccount(register)
	t.Log(accc.GetCandidateState(types.CandidateKeyIsCandidate))
	// 3. 首次注册的正常情况
	tx03 := newCandidateTx(register, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(tx03)
	assert.NoError(t, err)
	// 验证注册的候选节点信息
	p := registerAcc.GetCandidate()
	assert.Equal(t, normalIncomeAddress, p[types.CandidateKeyIncomeAddress])
	assert.Equal(t, normalNodeId, p[types.CandidateKeyNodeID])
	assert.Equal(t, normalHost, p[types.CandidateKeyHost])
	assert.Equal(t, normalPort, p[types.CandidateKeyPort])
	assert.Equal(t, "true", p[types.CandidateKeyIsCandidate])
	assert.Equal(t, params.RegisterCandidatePledgeAmount.String(), p[types.CandidateKeyPledgeAmount])
	// 自己此时的票数为 params.RegisterCandidatePledgeAmount / 75LEMO
	votes := registerAcc.GetVotes()
	realVotes := new(big.Int).Div(params.RegisterCandidatePledgeAmount, params.PledgeExchangeRate)
	assert.Equal(t, realVotes, votes)

	// 4. 已经是候选节点，修改候选节点信息
	// 注册候选节点
	rTx := newCandidateTx(register02, params.RegisterCandidatePledgeAmount, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(rTx)
	assert.NoError(t, err)
	// 4.1 修改信息为 注销候选节点
	tx04 := newCandidateTx(register02, nil, false, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(tx04)
	assert.NoError(t, err)
	// 注销之后，不会立即退回押金，要等到换届发放奖励的区块中退回押金。
	pro := register02Acc.GetCandidate()
	assert.Equal(t, params.NotCandidateNode, pro[types.CandidateKeyIsCandidate])
	votes = register02Acc.GetVotes() // 得票数变为0
	assert.Equal(t, big.NewInt(0), votes)

	// 4.2 修改包含nodeId信息，测试nodeId是否被修改和其他信息是否修改成功
	// 注册候选节点
	PledgeAmount := common.Lemo2Mo("7500050") // 质押的金额为750万零50 LEMO,换算为票数为10万票
	nTx := newCandidateTx(register03, PledgeAmount, true, normalIncomeAddress, normalNodeId, normalHost, normalPort)
	err = c.RegisterOrUpdateToCandidate(nTx)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100000), register03Acc.GetVotes())
	// 测试修改注册候选节点的信息中的所有字段，包括质押金额的追加
	newNodeId := "5b980ffb1b463fce4773a22ebf376c07c6207023b016b36ccfaba7be1cd1ab4a91737741cd43b7fcb10879e0fcf314d69fa953daec0f02be0f8f9cedb0cb3797"
	newIncomeAddress := common.HexToAddress("0x99999999").String()
	newHost := "127.0.0.1"
	newPort := "5001"
	addPledgeAmount := common.Lemo2Mo("50") // 追加50LEMO,加上之前的质押金额，换算为票数为100001票
	tx05 := newCandidateTx(register03, addPledgeAmount, true, newIncomeAddress, newNodeId, newHost, newPort)
	err = c.RegisterOrUpdateToCandidate(tx05)
	assert.NoError(t, err)
	// 验证修改后的信息
	newPro := register03Acc.GetCandidate()
	assert.Equal(t, normalNodeId, newPro[types.CandidateKeyNodeID]) // 验证nodeId不能被修改
	assert.Equal(t, newIncomeAddress, newPro[types.CandidateKeyIncomeAddress])
	assert.Equal(t, newHost, newPro[types.CandidateKeyHost])
	assert.Equal(t, newPort, newPro[types.CandidateKeyPort])
	assert.Equal(t, common.Lemo2Mo("7500100").String(), newPro[types.CandidateKeyPledgeAmount])
	assert.Equal(t, big.NewInt(100001), register03Acc.GetVotes())
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
	initialSenderBalance := common.Lemo2Mo("2090") // 兑换为票数为20票
	// 构造一个候选节点，该候选节点原本的票数为两倍于 initialSenderBalance
	candAddr := common.HexToAddress("0x13333000")
	candAcc := c.am.GetAccount(candAddr)
	candAcc.SetCandidateState(types.CandidateKeyIsCandidate, params.IsCandidateNode) // 设置为候选节点
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
	assert.Equal(t, new(big.Int).Div(initialSenderBalance, params.VoteExchangeRate), newVotes)
	// voteFor 变化
	assert.Equal(t, candAddr, voterAcc.GetVoteFor())

	c.am.RevertToSnapshot(snapshot)
	// 3.1 重复投给同一个候选节点
	voterAcc.SetVoteFor(candAddr) // 设置已经投过了candAddr候选节点
	// 给candAddr投票
	err = c.CallVoteTx(voterAddr, candAddr, initialSenderBalance)
	assert.Equal(t, ErrAlreadyVoted, err) // 返回不能再次投同一个候选节点的错误

	// 3.2 从投给的candAddr候选节点转投到其他节点
	voterAcc.SetVoteFor(candAddr)    // 设置投给candAddr候选节点
	candAcc.SetVotes(big.NewInt(50)) // 设置候选节点的票数为50票
	// 构造一个新的候选节点,初始得票数为0
	newCandAddr := common.HexToAddress("0x6666")
	newCandAcc := c.am.GetAccount(newCandAddr)
	newCandAcc.SetCandidateState(types.CandidateKeyIsCandidate, params.IsCandidateNode)

	// voter给newCandAddr投票,20 票
	err = c.CallVoteTx(voterAddr, newCandAddr, initialSenderBalance)
	assert.NoError(t, err)

	assert.Equal(t, newCandAddr, voterAcc.GetVoteFor())                                                     // 查看voter的votFor
	assert.Equal(t, new(big.Int).Div(initialSenderBalance, params.VoteExchangeRate), newCandAcc.GetVotes()) // newCandAddr候选节点票数
	assert.Equal(t, big.NewInt(30), candAcc.GetVotes())                                                     // candAddr候选节点票数为50 -20 = 30

}
