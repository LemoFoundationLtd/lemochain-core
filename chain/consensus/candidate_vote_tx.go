package consensus

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-core/common"
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

type CandidateVoteTx struct {
	am          *account.Manager
	CanTransfer func(vm.AccountManager, common.Address, *big.Int) bool
	Transfer    func(vm.AccountManager, common.Address, common.Address, *big.Int)
}

func NewCandidateVoteTx(am *account.Manager) *CandidateVoteTx {
	return &CandidateVoteTx{
		am:          am,
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
	}
}

// RegisterOrUpdateToCandidate candidate node account transaction call
func (c *CandidateVoteTx) RegisterOrUpdateToCandidate(candidateAddress, to common.Address, candiNode types.Profile, initialSenderBalance *big.Int) error {
	// Candidate node information
	newIsCandidate, ok := candiNode[types.CandidateKeyIsCandidate]
	if !ok {
		newIsCandidate = params.IsCandidateNode
	}
	minerAddress, ok := candiNode[types.CandidateKeyIncomeAddress]
	if !ok {
		minerAddress = candidateAddress.String()
	}
	nodeID, ok := candiNode[types.CandidateKeyNodeID]
	if !ok {
		return ErrOfRegisterNodeID
	}
	host, ok := candiNode[types.CandidateKeyHost]
	if !ok {
		return ErrOfRegisterHost
	}
	port, ok := candiNode[types.CandidateKeyPort]
	if !ok {
		return ErrOfRegisterPort
	}
	// Checking the balance is not enough
	if !c.CanTransfer(c.am, candidateAddress, params.RegisterCandidateNodeFees) {
		return ErrInsufficientBalance
	}
	// var snapshot = evm.am.Snapshot()

	// Register as a candidate node account
	nodeAccount := c.am.GetAccount(candidateAddress)
	// Check if the application address is already a candidate proxy node.
	profile := nodeAccount.GetCandidate()
	IsCandidate, ok := profile[types.CandidateKeyIsCandidate]
	// Set candidate node information if it is already a candidate node account
	if ok && IsCandidate == params.IsCandidateNode {
		// Determine whether to disqualify a candidate node
		if newIsCandidate == params.NotCandidateNode {
			profile[types.CandidateKeyIsCandidate] = params.NotCandidateNode
			nodeAccount.SetCandidate(profile)
			// Set the number of votes to 0
			nodeAccount.SetVotes(big.NewInt(0))
			// Transaction costs
			c.Transfer(c.am, candidateAddress, to, params.RegisterCandidateNodeFees)
			return nil
		}

		profile[types.CandidateKeyIncomeAddress] = minerAddress
		profile[types.CandidateKeyHost] = host
		profile[types.CandidateKeyPort] = port
		nodeAccount.SetCandidate(profile)
	} else if ok && IsCandidate == params.NotCandidateNode {
		return ErrAgainRegister
	} else {
		// Register candidate nodes
		newProfile := make(map[string]string, 5)
		newProfile[types.CandidateKeyIsCandidate] = params.IsCandidateNode
		newProfile[types.CandidateKeyIncomeAddress] = minerAddress
		newProfile[types.CandidateKeyNodeID] = nodeID
		newProfile[types.CandidateKeyHost] = host
		newProfile[types.CandidateKeyPort] = port
		nodeAccount.SetCandidate(newProfile)

		oldNodeAddress := nodeAccount.GetVoteFor()

		if (oldNodeAddress != common.Address{}) {
			oldNodeAccount := c.am.GetAccount(oldNodeAddress)
			oldNodeVoters := new(big.Int).Sub(oldNodeAccount.GetVotes(), initialSenderBalance)
			oldNodeAccount.SetVotes(oldNodeVoters)
		}
		nodeAccount.SetVoteFor(candidateAddress)
		nodeAccount.SetVotes(initialSenderBalance)

	}
	c.Transfer(c.am, candidateAddress, to, params.RegisterCandidateNodeFees)
	return nil
}

// CallVoteTx voting transaction call
func (c *CandidateVoteTx) CallVoteTx(voter, node common.Address, initialBalance *big.Int) error {
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
