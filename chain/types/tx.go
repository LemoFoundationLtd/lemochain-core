package types

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"io"
	"math/big"
	"strings"
	"sync/atomic"
)

const (
	TxSigLength        = 65
	MaxTxToNameLength  = 100
	MaxTxMessageLength = 1024
)

//go:generate gencodec -type txdata --field-override txdataMarshaling -out gen_tx_json.go

var (
	DefaultTTTL uint64 = 2 * 60 * 60 // Transaction Time To Live, 2hours
	TxVersion   uint8  = 1           // current transaction version. should between 0 and 128
)

type Transactions []*Transaction

// MerkleRootSha compute the root hash of transaction merkle trie
func (ts Transactions) MerkleRootSha() common.Hash {
	leaves := make([]common.Hash, len(ts))
	for i, item := range ts {
		leaves[i] = item.Hash()
	}
	return merkle.New(leaves).Root()
}

type Transaction struct {
	data txdata
	hash atomic.Value
	size atomic.Value
}

type txdata struct {
	Type          uint16          `json:"type" gencodec:"required"`
	Version       uint8           `json:"version" gencodec:"required"`
	ChainID       uint16          `json:"chainID" gencodec:"required"`
	From          common.Address  `json:"from" gencodec:"required"`
	GasPayer      *common.Address `json:"gasPayer" rlp:"nil"`
	Recipient     *common.Address `json:"to" rlp:"nil"` // nil means contract creation
	RecipientName string          `json:"toName"`
	GasPrice      *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit      uint64          `json:"gasLimit" gencodec:"required"`
	Amount        *big.Int        `json:"amount" gencodec:"required"`
	Data          []byte          `json:"data"`
	Expiration    uint64          `json:"expirationTime" gencodec:"required"`
	Message       string          `json:"message"`
	Sigs          [][]byte        `json:"sigs" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
	// gas payer signature
	GasPayerSigs [][]byte `json:"gasPayerSigs"`
}

type txdataMarshaling struct {
	Type         hexutil.Uint16
	Version      hexutil.Uint8
	ChainID      hexutil.Uint16
	GasPrice     *hexutil.Big10
	GasLimit     hexutil.Uint64
	Amount       *hexutil.Big10
	Data         hexutil.Bytes
	Expiration   hexutil.Uint64
	Sigs         []hexutil.Bytes
	GasPayerSigs []hexutil.Bytes
}

// NewReimbursementTransaction new instead of paying gas transaction
func NewReimbursementTransaction(from common.Address, to, gasPayer common.Address, amount *big.Int, data []byte, TxType uint16, chainID uint16, expiration uint64, toName string, message string) *Transaction {
	tx := newTransaction(from, TxType, TxVersion, chainID, &gasPayer, &to, amount, 0, nil, data, expiration, toName, message)
	return tx
}

// NewReimbursementContractCreation
func NewReimbursementContractCreation(from common.Address, gasPayer common.Address, amount *big.Int, data []byte, TxType uint16, chainID uint16, expiration uint64, toName string, message string) *Transaction {
	tx := newTransaction(from, TxType, TxVersion, chainID, &gasPayer, nil, amount, 0, nil, data, expiration, toName, message)
	return tx
}

// GasPayerSignatureTx
func GasPayerSignatureTx(tx *Transaction, gasPrice *big.Int, gasLimit uint64) *Transaction {
	tx.data.GasPrice = gasPrice
	tx.data.GasLimit = gasLimit
	return tx
}

// 注：TxType：0为普通交易，1为节点投票交易，2为注册成为代理节点交易
func NewTransaction(from common.Address, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, TxType uint16, chainID uint16, expiration uint64, toName string, message string) *Transaction {
	return newTransaction(from, TxType, TxVersion, chainID, nil, &to, amount, gasLimit, gasPrice, data, expiration, toName, message)
}

// 创建智能合约交易
func NewContractCreation(from common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, TxType uint16, chainID uint16, expiration uint64, toName string, message string) *Transaction {
	return newTransaction(from, TxType, TxVersion, chainID, nil, nil, amount, gasLimit, gasPrice, data, expiration, toName, message)
}

// 实例化一个to == nil的交易
func NoReceiverTransaction(from common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, TxType uint16, chainID uint16, expiration uint64, toName string, message string) *Transaction {
	return newTransaction(from, TxType, TxVersion, chainID, nil, nil, amount, gasLimit, gasPrice, data, expiration, toName, message)
}

func newTransaction(from common.Address, txType uint16, version uint8, chainID uint16, gasPayer, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, expiration uint64, toName string, message string) *Transaction {
	if version >= 128 {
		panic(fmt.Sprintf("invalid transaction version %d, should < 128", version))
	}
	if gasPayer == nil {
		gasPayer = &from
	}
	d := txdata{
		Type:          txType,
		Version:       version,
		ChainID:       chainID,
		From:          from,
		GasPayer:      gasPayer,
		Recipient:     to,
		RecipientName: toName,
		GasPrice:      new(big.Int),
		GasLimit:      gasLimit,
		Amount:        new(big.Int),
		Data:          data,
		Expiration:    expiration,
		Message:       message,
		Sigs:          make([][]byte, 0),
		Hash:          nil,
		GasPayerSigs:  make([][]byte, 0),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.GasPrice.Set(gasPrice)
	}
	return &Transaction{data: d}
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

// MarshalJSON encodes the lemoClient RPC transaction format.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the lemoClient RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txdata
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	version := dec.Version
	if version != TxVersion {
		return ErrInvalidVersion
	}

	for _, sig := range dec.Sigs {
		if len(sig) != TxSigLength {
			return ErrInvalidSig
		}
	}

	*tx = Transaction{data: dec}
	return nil
}

func (tx *Transaction) Type() uint16       { return tx.data.Type }
func (tx *Transaction) Version() uint8     { return tx.data.Version }
func (tx *Transaction) ChainID() uint16    { return tx.data.ChainID }
func (tx *Transaction) Data() []byte       { return common.CopyBytes(tx.data.Data) }
func (tx *Transaction) GasLimit() uint64   { return tx.data.GasLimit }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.GasPrice) }
func (tx *Transaction) Amount() *big.Int   { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Expiration() uint64 { return tx.data.Expiration }
func (tx *Transaction) ToName() string     { return tx.data.RecipientName }
func (tx *Transaction) Message() string    { return tx.data.Message }
func (tx *Transaction) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	}
	to := *tx.data.Recipient
	return &to
}

func (tx *Transaction) Sigs() [][]byte {
	return tx.data.Sigs
}

func (tx *Transaction) GasPayerSigs() [][]byte {
	return tx.data.GasPayerSigs
}

func (tx *Transaction) From() common.Address {
	return tx.data.From
}

// GetSigners returns address of instead of pay transaction gas.
func (tx *Transaction) GasPayer() common.Address {
	return *tx.data.GasPayer
}

func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.GasPrice, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *Transaction) String() string {
	var from, to string
	if len(tx.data.Sigs) > 0 {
		from = tx.From().String()
	} else {
		from = "[invalid sender: nil Sigs field]"
	}

	if tx.data.Recipient == nil {
		to = "[contract creation]"
	} else {
		to = tx.data.Recipient.String()
	}

	set := []string{
		fmt.Sprintf("Hash: %s", tx.Hash().Hex()),
		fmt.Sprintf("CreateContract: %v", tx.data.Recipient == nil),
		fmt.Sprintf("Type: %d", tx.Type()),
		fmt.Sprintf("Version: %d", tx.Version()),
		fmt.Sprintf("ChainID: %d", tx.ChainID()),
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("GasPayer: %s", tx.GasPayer().String()),
	}
	if len(tx.data.RecipientName) > 0 {
		set = append(set, fmt.Sprintf("ToName: %s", tx.data.RecipientName))
	}
	set = append(set, fmt.Sprintf("GasPrice: %v", tx.data.GasPrice))
	set = append(set, fmt.Sprintf("GasLimit: %v", tx.data.GasLimit))
	set = append(set, fmt.Sprintf("Amount: %v", tx.data.Amount))
	if len(tx.data.Data) > 0 {
		set = append(set, fmt.Sprintf("Data: %#x", tx.data.Data))
	}
	set = append(set, fmt.Sprintf("Expiration: %v", tx.data.Expiration))
	if len(tx.data.Message) > 0 {
		set = append(set, fmt.Sprintf("Message: %s", tx.data.Message))
	}
	if len(tx.Sigs()) > 0 {
		str := make([]string, 0)
		for _, sig := range tx.Sigs() {
			str = append(str, common.ToHex(sig))
		}
		sigs := strings.Join(str, ", ")
		set = append(set, fmt.Sprintf("Sigs: {%s}", sigs))
	}
	if len(tx.GasPayerSigs()) > 0 {
		str := make([]string, 0)
		for _, sig := range tx.GasPayerSigs() {
			str = append(str, common.ToHex(sig))
		}
		sigs := strings.Join(str, ", ")
		set = append(set, fmt.Sprintf("GasPayerSigs: {%s}", sigs))
	}
	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
}

// Clone deep copy transaction
func (tx *Transaction) Clone() *Transaction {
	cpy := *tx
	// Clear old hash. So we can change any field in the new tx. It will be created again when Hash() is called
	cpy.hash = atomic.Value{}

	if tx.data.Recipient != nil {
		*cpy.data.Recipient = *tx.data.Recipient
	}
	*cpy.data.GasPayer = *tx.data.GasPayer

	if tx.data.Sigs != nil {
		cpy.data.Sigs = make([][]byte, len(tx.data.Sigs), len(tx.data.Sigs))
		copy(cpy.data.Sigs, tx.data.Sigs)
	}
	if tx.data.GasPayerSigs != nil {
		cpy.data.GasPayerSigs = make([][]byte, len(tx.data.GasPayerSigs), len(tx.data.GasPayerSigs))
		copy(cpy.data.GasPayerSigs, tx.data.GasPayerSigs)
	}
	if tx.data.Data != nil {
		cpy.data.Data = make([]byte, len(tx.data.Data), len(tx.data.Data))
		copy(cpy.data.Data, tx.data.Data)
	}
	if tx.data.Hash != nil {
		*cpy.data.Hash = *tx.data.Hash
	}
	if tx.data.GasPrice != nil {
		cpy.data.GasPrice = new(big.Int).Set(tx.data.GasPrice)
	}
	if tx.data.Amount != nil {
		cpy.data.Amount = new(big.Int).Set(tx.data.Amount)
	}
	return &cpy
}

// VerifyTx transaction parameter verification
func (tx *Transaction) VerifyTx(chainID uint16, timeStamp uint64) error {
	// verify time
	if tx.Expiration() < timeStamp {
		log.Warnf("Tx out of date. tx expiration:%d, timeStamp:%d", tx.Expiration(), timeStamp)
		return ErrTxExpiration
	}
	// verify chainID
	if tx.ChainID() != chainID {
		log.Warnf("Tx chainID is incorrect. txChainId:%d, chainID:%d", tx.ChainID(), chainID)
		return ErrTxChainID
	}
	if tx.Amount().Sign() < 0 {
		return ErrNegativeValue
	}

	toNameLength := len(tx.ToName())
	if toNameLength > MaxTxToNameLength {
		log.Warnf("The length of toName field in transaction is out of max length limit. toName length = %d. max length limit = %d. ", toNameLength, MaxTxToNameLength)
		return ErrToName
	}
	txMessageLength := len(tx.Message())
	if txMessageLength > MaxTxMessageLength {
		log.Warnf("The length of message field in transaction is out of max length limit. message length = %d. max length limit = %d. ", txMessageLength, MaxTxMessageLength)
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
		log.Warnf("The transaction type does not exit . type = %v", tx.Type())
		return ErrTxType
	}
	return nil
}
