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

// RegisterOrUpdateToCandidate candidate node account transaction call
func (c *CandidateVoteEnv) RegisterOrUpdateToCandidate(tx *types.Transaction, initialSenderBalance *big.Int) error {
	// 解析data并生成新的profile
	newProfile, err := buildProfile(tx)
	if err != nil {
		return err
	}
	candidateAddress := tx.From()
	// Register as a candidate node account
	nodeAccount := c.am.GetAccount(candidateAddress)
	// Check if the application address is already a candidate proxy node.
	profile := nodeAccount.GetCandidate()
	candidateState, ok := profile[types.CandidateKeyIsCandidate]
	// Set candidate node information if it is already a candidate node account
	if ok && candidateState == params.IsCandidateNode {
		// Determine whether to disqualify a candidate node
		if newProfile[types.CandidateKeyIsCandidate] == params.NotCandidateNode { // 此账户为注销候选节点操作,注：注销之后不能再次注册
			nodeAccount.SetCandidateState(types.CandidateKeyIsCandidate, params.NotCandidateNode)
			// Set the number of votes to 0
			nodeAccount.SetVotes(big.NewInt(0))
			return nil
		}
		// 修改候选节点注册信息
		// 判断交易的amount字段的值是否大于0,大于0则为追加质押金额
		addBalance := tx.Amount()
		if addBalance.Cmp(big.NewInt(0)) > 0 {
			if c.CanTransfer(c.am, candidateAddress, addBalance) {
				c.Transfer(c.am, candidateAddress, params.CandidateDepositAddress, addBalance)
			} else {
				return ErrInsufficientBalance
			}
			// 修改质押押金和对应的票数变化
			if pledgeAmount, ok := profile[types.CandidateKeyPledgeAmount]; ok {
				oldPledge, success := new(big.Int).SetString(pledgeAmount, 10)
				if !success {
					log.Errorf("Fatal error!!! Parse pledge balance failed. CandidateAddress: %s", candidateAddress.String())
					return ErrParsePledgeAmount
				}
				newPledge := new(big.Int).Add(oldPledge, addBalance)
				// 修改质押押金
				profile[types.CandidateKeyPledgeAmount] = newPledge.String()
				// 修改votes
				// 新老质押的金额与75Lemo求模，把求的数比较如果增加了则增加相应的票数
				oldNum := new(big.Int).Div(oldPledge, params.PledgeExchangeRate)
				newNum := new(big.Int).Div(newPledge, params.PledgeExchangeRate)
				addVotes := new(big.Int).Sub(newNum, oldNum)
				if addVotes.Cmp(big.NewInt(0)) > 0 { // 达到增加vote的条件
					newVotes := new(big.Int).Add(nodeAccount.GetVotes(), addVotes)
					nodeAccount.SetVotes(newVotes)
				}
			} else {
				log.Errorf("Failed to get pledge balance. CandidateAddress: %s", candidateAddress.String())
				return ErrFailedGetPledgeBalacne
			}
		}
		profile[types.CandidateKeyIncomeAddress] = newProfile[types.CandidateKeyIncomeAddress]
		profile[types.CandidateKeyHost] = newProfile[types.CandidateKeyHost]
		profile[types.CandidateKeyPort] = newProfile[types.CandidateKeyPort]
		nodeAccount.SetCandidate(profile)
		return nil
	} else if ok && candidateState == params.NotCandidateNode { // 注册之后的候选节点不能再次注册
		return ErrAgainRegister
	} else { // 注：candidateState 是直接从数据库返回的account状态,在存储candidateState到数据库的时候只会存储"true"或"false",所以读取candidateState出来只会有三种情况："true"、"false"和空(ok == false)。
		// Register candidate nodes
		// 1. 判断注册的押金必须要大于等于规定的押金限制(500万LEMO)
		if tx.Amount().Cmp(params.RegisterCandidatePledgeAmount) < 0 {
			return ErrInsufficientPledgeAmount
		}

		// Checking the balance is not enough
		if !c.CanTransfer(c.am, candidateAddress, params.RegisterCandidatePledgeAmount) {
			return ErrInsufficientBalance
		}

		endProfile := make(map[string]string, 6)
		endProfile[types.CandidateKeyIsCandidate] = params.IsCandidateNode
		endProfile[types.CandidateKeyIncomeAddress] = newProfile[types.CandidateKeyIncomeAddress]
		endProfile[types.CandidateKeyNodeID] = newProfile[types.CandidateKeyNodeID]
		endProfile[types.CandidateKeyHost] = newProfile[types.CandidateKeyHost]
		endProfile[types.CandidateKeyPort] = newProfile[types.CandidateKeyPort]
		endProfile[types.CandidateKeyPledgeAmount] = tx.Amount().String()
		nodeAccount.SetCandidate(endProfile)

		oldNodeAddress := nodeAccount.GetVoteFor()
		initialVoteNum := new(big.Int).Div(initialSenderBalance, params.VoteExchangeRate) // 当前账户投票票数
		initialPledgeVoteNum := new(big.Int).Div(tx.Amount(), params.PledgeExchangeRate)  // 质押金额兑换所得票数

		if (oldNodeAddress != common.Address{}) { // 因为注册候选节点之后当前账户默认投票给自己，所以要进行修改投票人的操作
			oldNodeAccount := c.am.GetAccount(oldNodeAddress)
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialVoteNum)
			oldNodeAccount.SetVotes(oldNodeVoters)
		}
		// 设置当前账户的投票者为自己并设置自己所得到的初始票数,初始票数为 投票票数 + 质押所得票数
		nodeAccount.SetVoteFor(candidateAddress)
		initialTotalVotes := new(big.Int).Add(initialVoteNum, initialPledgeVoteNum)
		nodeAccount.SetVotes(initialTotalVotes)
		// cash pledge
		c.Transfer(c.am, candidateAddress, params.CandidateDepositAddress, tx.Amount())
		return nil
	}
}

// CallVoteTx voting transaction call
func (c *CandidateVoteEnv) CallVoteTx(voter, node common.Address, initialBalance *big.Int) error {
	nodeAccount := c.am.GetAccount(node)

	profile := nodeAccount.GetCandidate()
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	if !ok || IsCandidate == params.NotCandidateNode {
		return ErrOfNotCandidateNode
	}
	// var snapshot = evm.am.Snapshot()
	voterAccount := c.am.GetAccount(voter)
	exchangeVotes := new(big.Int).Div(initialBalance, params.VoteExchangeRate)
	// 如果票数等于0则没必要对candidate的票数进行修改
	if exchangeVotes.Cmp(big.NewInt(0)) <= 0 {
		// Set up voter account
		if voterAccount.GetVoteFor() == node {
			return ErrOfAgainVote
		}
		voterAccount.SetVoteFor(node)
		return nil
	}

	// Determine if the account has already voted
	if (voterAccount.GetVoteFor() != common.Address{}) {
		if voterAccount.GetVoteFor() == node {
			return ErrOfAgainVote
		} else {
			oldNode := voterAccount.GetVoteFor()
			newNodeAccount := nodeAccount
			// Change in votes
			oldNodeAccount := c.am.GetAccount(oldNode)
			// reduce the number of votes for old candidate nodes
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), exchangeVotes)
			oldNodeAccount.SetVotes(oldNodeVoters)
			// Increase the number of votes for new candidate nodes
			newNodeVoters := new(big.Int).Add(newNodeAccount.GetVotes(), exchangeVotes)
			newNodeAccount.SetVotes(newNodeVoters)
		}
	} else { // First vote
		// Increase the number of votes for candidate nodes
		nodeVoters := new(big.Int).Add(nodeAccount.GetVotes(), exchangeVotes)
		nodeAccount.SetVotes(nodeVoters)
	}
	// Set up voter account
	voterAccount.SetVoteFor(node)

	return nil
}
