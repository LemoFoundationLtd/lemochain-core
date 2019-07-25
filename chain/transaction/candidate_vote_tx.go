package transaction

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math/big"
)

var (
	ErrOfAgainVote              = errors.New("already voted the same as candidate node")
	ErrOfNotCandidateNode       = errors.New("node address is not candidate account")
	ErrOfRegisterNodeID         = errors.New("can't get nodeId of RegisterInfo")
	ErrOfRegisterHost           = errors.New("can't get host of RegisterInfo")
	ErrOfRegisterPort           = errors.New("can't get port of RegisterInfo")
	ErrAgainRegister            = errors.New("cannot register again after unregistering")
	ErrIsCandidate              = errors.New("get an unexpected character")
	ErrInsufficientBalance      = errors.New("the balance is insufficient to deduct the deposit for candidate register")
	ErrInsufficientPledgeAmount = errors.New("the pledge amount is insufficient for candidate register")
	ErrParsePledgeAmount        = errors.New("parse pledge amount failed")
	ErrDepositPoolInsufficient  = errors.New("insufficient deposit pool balance")
	ErrFailedGetPledgeBalacne   = errors.New("failed to get pledge balance")
)

type CandidateVoteEnv struct {
	am          *account.Manager
	CanTransfer func(vm.AccountManager, common.Address, *big.Int) bool
	Transfer    func(vm.AccountManager, common.Address, common.Address, *big.Int)
}

func NewCandidateVoteEnv(am *account.Manager) *CandidateVoteEnv {
	return &CandidateVoteEnv{
		am:          am,
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
	}
}

// checkRegisterTxProfile
func checkRegisterTxProfile(profile types.Profile) error {
	// check income address
	if strIncomeAddress, ok := profile[types.CandidateKeyIncomeAddress]; ok {
		if !common.CheckLemoAddress(strIncomeAddress) {
			log.Errorf("Income address failed verification,please check whether the input is correct. incomeAddress = %s", strIncomeAddress)
			return ErrInvalidAddress
		}
	}
	// check nodeId
	if nodeId, ok := profile[types.CandidateKeyNodeID]; ok {
		nodeIdLength := len(nodeId)
		if nodeIdLength != StandardNodeIdLength {
			log.Errorf("The nodeId length [%d] is not equal the standard length [%d] ", nodeIdLength, StandardNodeIdLength)
			return ErrInvalidNodeId
		}
		// check nodeId is available
		if !crypto.CheckPublic(nodeId) {
			log.Errorf("Invalid nodeId, nodeId = %s", nodeId)
			return ErrInvalidNodeId
		}
	}

	if host, ok := profile[types.CandidateKeyHost]; ok {
		hostLength := len(host)
		if hostLength > MaxDeputyHostLength {
			log.Errorf("The length of host field in transaction is out of max length limit. host length = %d. max length limit = %d. ", hostLength, MaxDeputyHostLength)
			return ErrInvalidHost
		}
	}
	return nil
}

// buildProfile
func buildProfile(tx *types.Transaction) (types.Profile, error) {
	// Unmarshal tx data
	txData := tx.Data()
	profile := make(types.Profile)
	err := json.Unmarshal(txData, &profile)
	if err != nil {
		log.Errorf("Unmarshal Candidate node error: %s", err)
		return nil, err
	}
	// check nodeID host and incomeAddress
	if err = checkRegisterTxProfile(profile); err != nil {
		return nil, err
	}
	candidateAddress := tx.From()
	// Candidate node information
	_, ok := profile[types.CandidateKeyIsCandidate]
	if !ok {
		profile[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	}
	_, ok = profile[types.CandidateKeyIncomeAddress]
	if !ok {
		profile[types.CandidateKeyIncomeAddress] = candidateAddress.String()
	}
	_, ok = profile[types.CandidateKeyNodeID]
	if !ok {
		return nil, ErrOfRegisterNodeID
	}
	_, ok = profile[types.CandidateKeyHost]
	if !ok {
		return nil, ErrOfRegisterHost
	}
	_, ok = profile[types.CandidateKeyPort]
	if !ok {
		return nil, ErrOfRegisterPort
	}
	return profile, nil
}

// firstRegisterCandidate 第一次注册候选节点处理逻辑
func (c *CandidateVoteEnv) firstRegisterCandidate(pledgeAmount *big.Int, register common.Address, txBuildProfile types.Profile) error {
	// 1. 判断注册的押金必须要大于等于规定的押金限制(500万LEMO)
	if pledgeAmount.Cmp(params.RegisterCandidatePledgeAmount) < 0 {
		return ErrInsufficientPledgeAmount
	}

	// Check if the balance is not enough
	if !c.CanTransfer(c.am, register, params.RegisterCandidatePledgeAmount) {
		return ErrInsufficientBalance
	}

	registerAcc := c.am.GetAccount(register)
	// 设置candidate info
	endProfile := make(map[string]string, 6)
	endProfile[types.CandidateKeyIsCandidate] = params.IsCandidateNode
	endProfile[types.CandidateKeyIncomeAddress] = txBuildProfile[types.CandidateKeyIncomeAddress]
	endProfile[types.CandidateKeyNodeID] = txBuildProfile[types.CandidateKeyNodeID]
	endProfile[types.CandidateKeyHost] = txBuildProfile[types.CandidateKeyHost]
	endProfile[types.CandidateKeyPort] = txBuildProfile[types.CandidateKeyPort]
	endProfile[types.CandidateKeyPledgeAmount] = pledgeAmount.String()
	registerAcc.SetCandidate(endProfile)

	// cash pledge
	c.Transfer(c.am, register, params.CandidateDepositAddress, pledgeAmount)

	initialPledgeVoteNum := new(big.Int).Div(pledgeAmount, params.PledgeExchangeRate) // 质押金额兑换所得票数
	// 设置自己所得到的初始票数,初始票数为 质押所得票数
	registerAcc.SetVotes(initialPledgeVoteNum)

	return nil
}

// modifyCandidateInfo 修改candidate info 操作
func (c *CandidateVoteEnv) modifyCandidateInfo(amount *big.Int, senderAddr common.Address, txBuildProfile types.Profile) error {
	senderAcc := c.am.GetAccount(senderAddr)
	candidateProfile := senderAcc.GetCandidate()

	// 注销候选节点操作,注：注销之后不能再次注册，质押押金退还会在换届奖励块中进行
	if txBuildProfile[types.CandidateKeyIsCandidate] == params.NotCandidateNode {
		senderAcc.SetCandidateState(types.CandidateKeyIsCandidate, params.NotCandidateNode)
		// Set the number of votes to 0
		senderAcc.SetVotes(big.NewInt(0))
		return nil
	}
	// 修改候选节点注册信息
	// 判断交易的amount字段的值是否大于0,大于0则为追加质押金额
	if amount.Cmp(big.NewInt(0)) > 0 {
		if c.CanTransfer(c.am, senderAddr, amount) {
			c.Transfer(c.am, senderAddr, params.CandidateDepositAddress, amount)
		} else {
			return ErrInsufficientBalance
		}
		// 修改质押押金和对应的票数变化
		if pledgeAmount, ok := candidateProfile[types.CandidateKeyPledgeAmount]; ok {
			oldPledge, success := new(big.Int).SetString(pledgeAmount, 10)
			if !success {
				log.Errorf("Fatal error!!! Parse pledge balance failed. CandidateAddress: %s", senderAddr.String())
				return ErrParsePledgeAmount
			}
			newPledge := new(big.Int).Add(oldPledge, amount)
			// 修改质押押金
			candidateProfile[types.CandidateKeyPledgeAmount] = newPledge.String()
			// 修改votes
			// 新老质押的金额与75Lemo相除，把求的数比较如果增加了则增加相应的票数
			oldNum := new(big.Int).Div(oldPledge, params.PledgeExchangeRate)
			newNum := new(big.Int).Div(newPledge, params.PledgeExchangeRate)
			addVotes := new(big.Int).Sub(newNum, oldNum)
			if addVotes.Cmp(big.NewInt(0)) > 0 { // 达到增加vote的条件
				newVotes := new(big.Int).Add(senderAcc.GetVotes(), addVotes)
				senderAcc.SetVotes(newVotes)
			}
		} else {
			log.Errorf("Failed to get pledge balance. CandidateAddress: %s", senderAddr.String())
			return ErrFailedGetPledgeBalacne
		}
	}
	// nodeId不能修改
	candidateProfile[types.CandidateKeyIncomeAddress] = txBuildProfile[types.CandidateKeyIncomeAddress]
	candidateProfile[types.CandidateKeyHost] = txBuildProfile[types.CandidateKeyHost]
	candidateProfile[types.CandidateKeyPort] = txBuildProfile[types.CandidateKeyPort]
	senderAcc.SetCandidate(candidateProfile)
	return nil
}

// RegisterOrUpdateToCandidate candidate node account transaction call
func (c *CandidateVoteEnv) RegisterOrUpdateToCandidate(tx *types.Transaction) error {
	// 解析data并生成新的profile
	txBuildProfile, err := buildProfile(tx)
	if err != nil {
		return err
	}
	senderAddr := tx.From()
	// Register as a candidate node account
	senderAcc := c.am.GetAccount(senderAddr)

	// Check if the application address is already a candidate proxy node.
	candidateProfile := senderAcc.GetCandidate()
	candidateState, ok := candidateProfile[types.CandidateKeyIsCandidate]

	if !ok || candidateState == "" { // 表示第一次注册候选节点，等于""为如果一个账户第一次注册候选交易失败之后，回滚会让map中的值回滚为零值，string类型的0值为"".箱子交易中容易出现此情况。
		if err := c.firstRegisterCandidate(tx.Amount(), senderAddr, txBuildProfile); err != nil {
			return err
		}
	} else { // 已经注册过候选节点的情况
		if candidateState == params.NotCandidateNode { // 如果此账户注册之后又注销了候选节点，则不能重新注册
			return ErrAgainRegister
		} else if candidateState == params.IsCandidateNode { // 此账户已经是一个候选节点账户
			if err := c.modifyCandidateInfo(tx.Amount(), senderAddr, txBuildProfile); err != nil {
				return err
			}
		} else {
			log.Errorf("Get an unexpected character [%s] when we call \"candidateProfile[types.CandidateKeyIsCandidate]\". Expected value: [%s] and [%s]", candidateState, params.NotCandidateNode, params.IsCandidateNode)
			return ErrIsCandidate
		}
	}
	return nil
}

// CallVoteTx voting transaction call
func (c *CandidateVoteEnv) CallVoteTx(voter, newCandidateAddr common.Address, initialBalance *big.Int) error {
	newCandidateAcc := c.am.GetAccount(newCandidateAddr)

	profile := newCandidateAcc.GetCandidate()
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	if !ok || IsCandidate == params.NotCandidateNode || IsCandidate == "" {
		return ErrOfNotCandidateNode
	}

	voterAccount := c.am.GetAccount(voter)
	exchangeVotes := new(big.Int).Div(initialBalance, params.VoteExchangeRate)

	// 不能再次投同一个候选节点
	if voterAccount.GetVoteFor() == newCandidateAddr {
		return ErrOfAgainVote
	}

	c.modifyCandidateVotes(voterAccount, newCandidateAcc, exchangeVotes)
	// Set up voter account
	voterAccount.SetVoteFor(newCandidateAddr)

	return nil
}

// modifyCandidateVotes
func (c *CandidateVoteEnv) modifyCandidateVotes(voterAccount, newCandidateAccount types.AccountAccessor, modifyVotes *big.Int) {
	// 如果票数等于0则没必要对candidate的票数进行修改
	if modifyVotes.Cmp(big.NewInt(0)) <= 0 {
		return
	}
	// 如果已经投给了其他节点，则减少该节点的票数
	oldCandidateAddr := voterAccount.GetVoteFor()
	if (oldCandidateAddr != common.Address{}) {
		oldCandidateAccount := c.am.GetAccount(oldCandidateAddr)
		// 判断前一个投票的候选节点是否已经取消候选节点列表
		if oldCandidateAccount.GetCandidateState(types.CandidateKeyIsCandidate) == params.IsCandidateNode {
			// reduce the number of votes for old candidate nodes
			oldNodeVoters := new(big.Int).Sub(oldCandidateAccount.GetVotes(), modifyVotes)
			oldCandidateAccount.SetVotes(oldNodeVoters)
		}
	}

	// 增加新节点的票数
	nodeVoters := new(big.Int).Add(newCandidateAccount.GetVotes(), modifyVotes)
	newCandidateAccount.SetVotes(nodeVoters)
}
