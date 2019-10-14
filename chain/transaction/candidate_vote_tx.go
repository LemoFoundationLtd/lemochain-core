package transaction

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math/big"
	"strconv"
)

var (
	ErrAlreadyVoted              = errors.New("already voted the same as candidate node")
	ErrOfNotCandidateNode        = errors.New("node address is not candidate account")
	ErrOfRegisterNodeID          = errors.New("can't get nodeId of RegisterInfo")
	ErrOfRegisterHost            = errors.New("can't get host of RegisterInfo")
	ErrOfRegisterPort            = errors.New("can't get port of RegisterInfo")
	ErrRegisterAgain             = errors.New("cannot register again after unregistering")
	ErrIsCandidate               = errors.New("get an unexpected character")
	ErrInsufficientBalance       = errors.New("the balance is insufficient to deduct the deposit for candidate register")
	ErrInsufficientDepositAmount = errors.New("the deposit amount is not enough for candidate register")
	ErrParseDepositAmount        = errors.New("parse deposit amount failed")
	ErrDepositPoolInsufficient   = errors.New("insufficient deposit pool balance")
	ErrFailedGetDepositBalacne   = errors.New("failed to get deposit balance")
)

type CandidateVoteEnv struct {
	am          *account.Manager
	dm          *deputynode.Manager
	CanTransfer func(vm.AccountManager, common.Address, *big.Int) bool
	Transfer    func(vm.AccountManager, common.Address, common.Address, *big.Int)
}

func NewCandidateVoteEnv(am *account.Manager, dm *deputynode.Manager) *CandidateVoteEnv {
	return &CandidateVoteEnv{
		am:          am,
		dm:          dm,
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
	}
}

/*
检查：
1. nodeID必须存在并检验是否可用。
2. 如果有introduction长度必须小于10245，其他字段长度必须小于130.
3. incomeAddress host port 限制性判断
*/
// CheckRegisterTxProfile
func CheckRegisterTxProfile(profile types.Profile) error {
	// 检查传入profile信息的长度大小
	for key, val := range profile {
		if key == types.CandidateKeyIntroduction { // 检查introduction
			if len(val) > MaxIntroductionLength {
				log.Errorf("The length of introduction field in transaction is out of max length limit. introduction length = %d. max length limit = %d.", len(val), MaxIntroductionLength)
				return ErrInvalidIntroduction
			}
		} else if key == types.CandidateKeyNodeID {
			if len(val) > NodeIDFieldLength {
				log.Errorf("The length of candidate nodeId field in transaction is out of max length limit. fieldName: %s, field length = %d. max length limit = %d. ", key, len(val), NodeIDFieldLength)
				return ErrInvalidNodeId
			}
		} else {
			// 其他字段长度不能大于128
			if len(val) > MaxProfileFieldLength {
				log.Errorf("The length of candidate profile field in transaction is out of max length limit. fieldName: %s, field length = %d. max length limit = %d. ", key, len(val), MaxProfileFieldLength)
				return ErrInvalidProfile
			}
		}
	}
	// check income address
	if strIncomeAddress, ok := profile[types.CandidateKeyIncomeAddress]; ok {
		if !common.CheckLemoAddress(strIncomeAddress) {
			log.Errorf("Income address failed verification,please check whether the input is correct. incomeAddress = %s", strIncomeAddress)
			return ErrInvalidAddress
		}
	}

	// check nodeId
	if nodeId, ok := profile[types.CandidateKeyNodeID]; ok {
		nodeIdLength := len(common.FromHex(nodeId)) // nodeId转换为[]byte始终为64位
		if nodeIdLength != StandardNodeIdLength {
			log.Errorf("The nodeId length [%d] is not equal the standard length [%d] ", nodeIdLength, StandardNodeIdLength)
			return ErrInvalidNodeId
		}
		// check nodeId is available
		if !crypto.CheckPublic(nodeId) {
			log.Errorf("Invalid nodeId, nodeId = %s", nodeId)
			return ErrInvalidNodeId
		}
	} else {
		return ErrOfRegisterNodeID
	}

	// check host 必须存在
	if _, ok := profile[types.CandidateKeyHost]; !ok {
		return ErrOfRegisterHost
	}

	// check port
	if port, ok := profile[types.CandidateKeyPort]; ok {
		if portNum, err := strconv.Atoi(port); err == nil {
			if portNum > 65535 || portNum < 1024 {
				return ErrInvalidPort
			}
		} else {
			log.Errorf("Strconv.Atoi(port) error. port: %s, error: %v", port, err)
			return ErrInvalidPort
		}
	} else {
		return ErrOfRegisterPort
	}
	return nil
}

// buildProfile
func buildProfile(tx *types.Transaction) (types.Profile, error) {
	// Unmarshal tx data
	profile := make(types.Profile)
	err := json.Unmarshal(tx.Data(), &profile)
	if err != nil {
		log.Errorf("Unmarshal Candidate node error: %s", err)
		return nil, err
	}
	// check nodeID host and incomeAddress
	if err = CheckRegisterTxProfile(profile); err != nil {
		return nil, err
	}
	if _, ok := profile[types.CandidateKeyIsCandidate]; !ok {
		profile[types.CandidateKeyIsCandidate] = types.IsCandidateNode
	}
	if _, ok := profile[types.CandidateKeyIncomeAddress]; !ok {
		profile[types.CandidateKeyIncomeAddress] = tx.From().String()
	}
	if _, ok := profile[types.CandidateKeyIntroduction]; !ok {
		profile[types.CandidateKeyIntroduction] = ""
	}
	return profile, nil
}

// InitCandidateProfile
func InitCandidateProfile(registerAcc types.AccountAccessor, newProfile types.Profile) {
	registerAcc.SetCandidate(newProfile)
}

// registerCandidate 注册候选节点处理逻辑
func (c *CandidateVoteEnv) registerCandidate(depositAmount *big.Int, register common.Address, p types.Profile) error {
	// 1. 判断注册的押金必须要大于等于规定的押金限制(500万LEMO)
	if depositAmount.Cmp(params.MinCandidateDeposit) < 0 {
		return ErrInsufficientDepositAmount
	}

	// Check if the balance is not enough
	if !c.CanTransfer(c.am, register, params.MinCandidateDeposit) {
		return ErrInsufficientBalance
	}

	registerAcc := c.am.GetAccount(register)

	// 设置candidate info
	p[types.CandidateKeyDepositAmount] = depositAmount.String()
	InitCandidateProfile(registerAcc, p)

	// cash deposit
	c.Transfer(c.am, register, params.DepositPoolAddress, depositAmount)

	initialDepositVoteNum := new(big.Int).Div(depositAmount, params.DepositExchangeRate) // 质押金额兑换所得票数
	// 设置自己所得到的初始票数,初始票数为 质押所得票数
	registerAcc.SetVotes(initialDepositVoteNum)

	return nil
}

// unRegisterCandidate 注销候选节点操作, 注：注销之后不能再次注册，质押押金退还会在换届奖励块中进行
func (c *CandidateVoteEnv) unRegisterCandidate(candidateAcc types.AccountAccessor, txBuildProfile types.Profile) bool {
	if txBuildProfile[types.CandidateKeyIsCandidate] == types.NotCandidateNode {
		candidateAcc.SetCandidateState(types.CandidateKeyIsCandidate, types.NotCandidateNode)
		// Set the number of votes to 0
		candidateAcc.SetVotes(big.NewInt(0))
		// 退还候选节点的押金
		currentHeight := c.am.CurrentBlockHeight()
		c.refundDeposit(candidateAcc.GetAddress(), currentHeight)
		return true
	}
	return false
}

// refundDeposit
func (c *CandidateVoteEnv) refundDeposit(candidateAddress common.Address, height uint32) {
	// 判断当前是否在过度期，过度期不得退款
	num := height % params.TermDuration
	// 1. 在过渡期,延后到过渡期之后发放奖励区块中退款,如果他已经被选成了下一届的共识节点的情况，发放奖励区块中退押金的时候会判断这种情况。
	if num <= params.InterimDuration && height > params.InterimDuration {
		return
	}
	// 2. 不在过渡期
	candidateAcc := c.am.GetAccount(candidateAddress)
	nodeId := candidateAcc.GetCandidateState(types.CandidateKeyNodeID)
	if nodeId == "" {
		panic("Can not get candidate profile!!!")
	}

	if c.dm.IsNodeDeputy(height, common.FromHex(nodeId)) { // 为当前共识节点，则在发放奖励区块退款
		return
	} else { // 不是共识节点，则立即退款
		Refund(candidateAddress, c.am)
	}
}

// Refund 进行退款操作
func Refund(candidateAddress common.Address, am *account.Manager) {
	log.Info("Refund candidate deposit", "address", candidateAddress)
	// 判断addr的candidate信息
	candidateAcc := am.GetAccount(candidateAddress)
	depositAmountString := candidateAcc.GetCandidateState(types.CandidateKeyDepositAmount)
	depositAmount, ok := new(big.Int).SetString(depositAmountString, 10)
	if !ok {
		panic("Big.Int SetString function failed")
	}

	// 退还押金
	candidateDepositPoolAcc := am.GetAccount(params.DepositPoolAddress)
	if candidateDepositPoolAcc.GetBalance().Cmp(depositAmount) < 0 { // 判断押金池中的押金是否足够，如果不足直接panic
		panic("The balance of candidate deposit pool account is insufficient.")
	}
	// 减少押金池中的余额
	candidateDepositPoolAcc.SetBalance(new(big.Int).Sub(candidateDepositPoolAcc.GetBalance(), depositAmount))
	// 退还押金到取消的候选节点账户
	candidateAcc.SetBalance(new(big.Int).Add(candidateAcc.GetBalance(), depositAmount))
	// 设置候选节点info中的押金余额为nil
	candidateAcc.SetCandidateState(types.CandidateKeyDepositAmount, "")
}

// modifyCandidateInfo 修改candidate info 操作
func (c *CandidateVoteEnv) modifyCandidateInfo(amount *big.Int, senderAddr common.Address, txBuildProfile types.Profile) error {
	senderAcc := c.am.GetAccount(senderAddr)
	candidateProfile := senderAcc.GetCandidate()

	// 修改候选节点注册信息
	// 判断交易的amount字段的值是否大于0,大于0则为追加质押金额
	if amount.Cmp(big.NewInt(0)) > 0 {
		if c.CanTransfer(c.am, senderAddr, amount) {
			c.Transfer(c.am, senderAddr, params.DepositPoolAddress, amount)
		} else {
			return ErrInsufficientBalance
		}
		// 修改质押押金和对应的票数变化
		if depositAmount, ok := candidateProfile[types.CandidateKeyDepositAmount]; ok {
			oldDeposit, success := new(big.Int).SetString(depositAmount, 10)
			if !success {
				log.Errorf("Fatal error!!! Parse deposit balance failed. CandidateAddress: %s", senderAddr.String())
				return ErrParseDepositAmount
			}
			newDeposit := new(big.Int).Add(oldDeposit, amount)
			// 修改质押押金
			candidateProfile[types.CandidateKeyDepositAmount] = newDeposit.String()
			// 修改押金增加导致的票数的增加
			addDepositChangeVotes(oldDeposit, newDeposit, senderAcc)
		} else {
			log.Errorf("Failed to get deposit balance. CandidateAddress: %s", senderAddr.String())
			return ErrFailedGetDepositBalacne
		}
	}
	// nodeId和质押金额不能通过传入的参数修改，其他都可以修改
	for key, val := range txBuildProfile {
		if key != types.CandidateKeyNodeID && key != types.CandidateKeyDepositAmount {
			candidateProfile[key] = val
		}
	}
	senderAcc.SetCandidate(candidateProfile)
	return nil
}

// addDepositChangeVotes 押金变化导致的票数变化
func addDepositChangeVotes(oldDeposit, newDeposit *big.Int, senderAcc types.AccountAccessor) {
	// 新老质押的金额与75Lemo相除，把求的数比较如果增加了则增加相应的票数
	oldNum := new(big.Int).Div(oldDeposit, params.DepositExchangeRate)
	newNum := new(big.Int).Div(newDeposit, params.DepositExchangeRate)
	addVotes := new(big.Int).Sub(newNum, oldNum)
	if addVotes.Cmp(big.NewInt(0)) > 0 { // 达到增加vote的条件
		newVotes := new(big.Int).Add(senderAcc.GetVotes(), addVotes)
		senderAcc.SetVotes(newVotes)
	}
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
		if err := c.registerCandidate(tx.Amount(), senderAddr, txBuildProfile); err != nil {
			return err
		}
	} else { // 已经注册过候选节点的情况
		if candidateState == types.NotCandidateNode { // 如果此账户注册之后又注销了候选节点，则不能重新注册
			return ErrRegisterAgain
		} else if candidateState == types.IsCandidateNode { // 此账户已经是一个候选节点账户
			// 是否为注销 candidate 交易
			if c.unRegisterCandidate(senderAcc, txBuildProfile) {
				return nil
			}
			// 修改candidate info 交易
			if err := c.modifyCandidateInfo(tx.Amount(), senderAddr, txBuildProfile); err != nil {
				return err
			}
		} else {
			log.Errorf("Get an unexpected character [%s] when we call \"candidateProfile[types.CandidateKeyIsCandidate]\". Expected value: [%s] and [%s]", candidateState, types.NotCandidateNode, types.IsCandidateNode)
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
	if !ok || IsCandidate == types.NotCandidateNode || IsCandidate == "" {
		return ErrOfNotCandidateNode
	}

	voterAccount := c.am.GetAccount(voter)
	exchangeVotes := new(big.Int).Div(initialBalance, params.VoteExchangeRate)

	// 不能再次投同一个候选节点
	if voterAccount.GetVoteFor() == newCandidateAddr {
		return ErrAlreadyVoted
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
		if oldCandidateAccount.GetCandidateState(types.CandidateKeyIsCandidate) == types.IsCandidateNode {
			// reduce the number of votes for old candidate nodes
			oldNodeVoters := new(big.Int).Sub(oldCandidateAccount.GetVotes(), modifyVotes)
			oldCandidateAccount.SetVotes(oldNodeVoters)
		}
	}

	// 增加新节点的票数
	nodeVoters := new(big.Int).Add(newCandidateAccount.GetVotes(), modifyVotes)
	newCandidateAccount.SetVotes(nodeVoters)
}
