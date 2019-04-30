package node

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
)

// VerifyLemoAddress check lemoAddress
func VerifyLemoAddress(lemoAddress string) bool {
	return common.CheckLemoAddress(lemoAddress)
}

// VerifyNode check string node (node = nodeID@IP:Port)
func VerifyNode(node string) bool {
	nodeId, endpoint := p2p.ParseNodeString(node)
	if nodeId == nil || endpoint == "" {
		return false
	}
	return true
}

// VerifyTx transaction parameter verification
func VerifyTx(tx *types.Transaction) error {
	toNameLength := len(tx.ToName())
	if toNameLength > MaxTxToNameLength {

		log.Errorf("The length of toName field in transaction is out of max length limit. toName length = %d. max length limit = %d. ", toNameLength, MaxTxToNameLength)
		return ErrToName
	}
	txMessageLength := len(tx.Message())
	if txMessageLength > MaxTxMessageLength {
		log.Errorf("The length of message field in transaction is out of max length limit. message length = %d. max length limit = %d. ", txMessageLength, MaxTxMessageLength)
		return ErrTxMessage
	}
	switch tx.Type() {
	case params.OrdinaryTx:
		if tx.To() == nil {
			if len(tx.Data()) == 0 {
				return ErrCreateContract
			}
		}
	case params.VoteTx:
	case params.RegisterTx, params.CreateAssetTx, params.IssueAssetTx, params.ReplenishAssetTx, params.ModifyAssetTx, params.TransferAssetTx:
		if len(tx.Data()) == 0 {
			return ErrSpecialTx
		}
	default:
		log.Errorf("The transaction type does not exit . type = %v", tx.Type())
		return ErrTxType
	}
	return nil
}
