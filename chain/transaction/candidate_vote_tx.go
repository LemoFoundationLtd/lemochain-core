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
	ErrOfAgainVote         = errors.New("already voted the same as candidate node")
	ErrOfNotCandidateNode  = errors.New("node address is not candidate account")
	ErrOfRegisterNodeID    = errors.New("can't get nodeId of RegisterInfo")
	ErrOfRegisterHost      = errors.New("can't get host of RegisterInfo")
	ErrOfRegisterPort      = errors.New("can't get port of RegisterInfo")
	ErrAgainRegister       = errors.New("cannot register again after unregistering")
	ErrInsufficientBalance = errors.New("insufficient balance for transfer")
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
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	// Set candidate node information if it is already a candidate node account
	if ok && IsCandidate == params.IsCandidateNode {
		// Determine whether to disqualify a candidate node
		if newProfile[types.CandidateKeyIsCandidate] == params.NotCandidateNode {
			profile[types.CandidateKeyIsCandidate] = params.NotCandidateNode
			nodeAccount.SetCandidate(profile)
			// Set the number of votes to 0
			nodeAccount.SetVotes(big.NewInt(0))
			// deposit refund
			c.Transfer(c.am, params.FeeReceiveAddress, candidateAddress, params.RegisterCandidateNodeFees)
			return nil
		}

		profile[types.CandidateKeyIncomeAddress] = newProfile[types.CandidateKeyIncomeAddress]
		profile[types.CandidateKeyHost] = newProfile[types.CandidateKeyHost]
		profile[types.CandidateKeyPort] = newProfile[types.CandidateKeyPort]
		nodeAccount.SetCandidate(profile)
	} else if ok && IsCandidate == params.NotCandidateNode {
		return ErrAgainRegister
	} else {
		// Register candidate nodes
		// Checking the balance is not enough
		if !c.CanTransfer(c.am, candidateAddress, params.RegisterCandidateNodeFees) {
			return ErrInsufficientBalance
		}

		endProfile := make(map[string]string, 5)
		endProfile[types.CandidateKeyIsCandidate] = params.IsCandidateNode
		endProfile[types.CandidateKeyIncomeAddress] = newProfile[types.CandidateKeyIncomeAddress]
		endProfile[types.CandidateKeyNodeID] = newProfile[types.CandidateKeyNodeID]
		endProfile[types.CandidateKeyHost] = newProfile[types.CandidateKeyHost]
		endProfile[types.CandidateKeyPort] = newProfile[types.CandidateKeyPort]
		nodeAccount.SetCandidate(endProfile)

		oldNodeAddress := nodeAccount.GetVoteFor()

		if (oldNodeAddress != common.Address{}) {
			oldNodeAccount := c.am.GetAccount(oldNodeAddress)
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialSenderBalance)
			oldNodeAccount.SetVotes(oldNodeVoters)
		}
		nodeAccount.SetVoteFor(candidateAddress)
		nodeAccount.SetVotes(initialSenderBalance)
		// cash pledge
		c.Transfer(c.am, candidateAddress, params.FeeReceiveAddress, params.RegisterCandidateNodeFees)
	}

	return nil
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
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialBalance)
			oldNodeAccount.SetVotes(oldNodeVoters)
			// Increase the number of votes for new candidate nodes
			newNodeVoters := new(big.Int).Add(newNodeAccount.GetVotes(), initialBalance)
			newNodeAccount.SetVotes(newNodeVoters)
		}
	} else { // First vote
		// Increase the number of votes for candidate nodes
		nodeVoters := new(big.Int).Add(nodeAccount.GetVotes(), initialBalance)
		nodeAccount.SetVotes(nodeVoters)
	}
	// Set up voter account
	voterAccount.SetVoteFor(node)

	return nil
}
