package types

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"io"
	"math/big"
	"sync/atomic"
)

//go:generate gencodec -type txdata --field-override txdataMarshaling -out gen_tx_json.go

var (
	DefaultTTTL   uint64 = 2 * 60 * 60 // Transaction Time To Live, 2hours
	ErrInvalidSig        = errors.New("invalid transaction v, r, s values")
	TxVersion     uint8  = 1 // current transaction version. should between 0 and 128
)

type Transactions []*Transaction

type Transaction struct {
	data txdata

	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	Recipient     *common.Address `json:"to" rlp:"nil"` // nil means contract creation
	RecipientName string          `json:"toName"`
	GasPrice      *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit      uint64          `json:"gasLimit" gencodec:"required"`
	Amount        *big.Int        `json:"amount" gencodec:"required"`
	Data          []byte          `json:"data" gencodec:"required"`
	Expiration    uint64          `json:"expirationTime" gencodec:"required"`
	Message       string          `json:"message"`

	// V is combined by these properties:
	//     type    version secp256k1.V  chainId
	// |----8----|----7----|----1----|----16----|
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

type txdataMarshaling struct {
	GasPrice   *hexutil.Big
	GasLimit   hexutil.Uint64
	Amount     *hexutil.Big
	Data       hexutil.Bytes
	Expiration hexutil.Uint64
	V          *hexutil.Big
	R          *hexutil.Big
	S          *hexutil.Big
}

func NewTransaction(to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, chainId uint16, expiration uint64, toName string, message string) *Transaction {
	return newTransaction(0, TxVersion, chainId, &to, amount, gasLimit, gasPrice, data, expiration, toName, message)
}

func NewContractCreation(amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, chainId uint16, expiration uint64, toName string, message string) *Transaction {
	return newTransaction(0, TxVersion, chainId, nil, amount, gasLimit, gasPrice, data, expiration, toName, message)
}

func newTransaction(txType uint8, version uint8, chainId uint16, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, expiration uint64, toName string, message string) *Transaction {
	if version >= 128 {
		panic(fmt.Sprintf("invalid transaction version %d, should < 128", version))
	}
	d := txdata{
		Recipient:     to,
		RecipientName: toName,
		GasPrice:      new(big.Int),
		GasLimit:      gasLimit,
		Amount:        new(big.Int),
		Data:          data,
		Expiration:    expiration,
		Message:       message,
		V:             CombineV(txType, version, chainId),
		R:             new(big.Int),
		S:             new(big.Int),
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
	_, version, V, _ := ParseV(dec.V)
	if version != TxVersion {
		return ErrInvalidSig
	}
	// should has R, S
	if !crypto.ValidateSignatureValues(V, dec.R, dec.S) {
		return ErrInvalidSig
	}
	*tx = Transaction{data: dec}
	return nil
}

func (tx *Transaction) Type() uint8        { txType, _, _, _ := ParseV(tx.data.V); return txType }
func (tx *Transaction) Version() uint8     { _, version, _, _ := ParseV(tx.data.V); return version }
func (tx *Transaction) ChainId() uint16    { _, _, _, chainId := ParseV(tx.data.V); return chainId }
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

func (tx *Transaction) From() (common.Address, error) {
	from := tx.from.Load()
	if from != nil {
		return from.(common.Address), nil
	}

	// parse type and create signer by self
	// now we have one signer only
	addr, err := MakeSigner().GetSender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(addr)
	return addr, nil
}

func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// WithSignature returns a new transaction with the given signature.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.ParseSignature(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.GasPrice, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *Transaction) Raw() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

func (tx *Transaction) String() string {
	var from, to string
	if tx.data.V != nil {
		if f, err := tx.From(); err != nil { // derive but don't cache
			from = "[invalid sender: invalid sig]"
		} else {
			from = fmt.Sprintf("%x", f[:])
		}
	} else {
		from = "[invalid sender: nil V field]"
	}

	if tx.data.Recipient == nil {
		to = "[contract creation]"
	} else {
		to = fmt.Sprintf("%x", tx.data.Recipient[:])
	}
	enc, _ := rlp.EncodeToBytes(&tx.data)
	return fmt.Sprintf(`
	TX(%x)
	Contract:   %v
	Type:       %d
	Version:    %d
	ChainId:    %d
	From:       %s
	To:         %s
	ToName:     %s
	GasPrice:   %#x
	GasLimit    %#x
	Amount:     %#x
	Data:       0x%x
	Expiration: %#x
	Message:    0x%x
	V:          %#x
	R:          %#x
	S:          %#x
	Hex:        %x
`,
		tx.Hash(),
		tx.data.Recipient == nil,
		tx.Type(),
		tx.Version(),
		tx.ChainId(),
		from,
		to,
		tx.data.RecipientName,
		tx.data.GasPrice,
		tx.data.GasLimit,
		tx.data.Amount,
		tx.data.Data,
		tx.data.Expiration,
		tx.data.Message,
		tx.data.V,
		tx.data.R,
		tx.data.S,
		enc,
	)
}

// SetSecp256k1V merge secp256k1.V into the result of CombineV function
func SetSecp256k1V(V *big.Int, secp256k1V byte) *big.Int {
	// V = V & ((sig[64] & 1) << 16)
	return new(big.Int).SetBit(V, 16, uint(secp256k1V&1))
}

// CombineV combines type, version, chainId together to get V (without secp256k1.V)
func CombineV(txType uint8, version uint8, chainId uint16) *big.Int {
	return new(big.Int).SetUint64((uint64(txType) << 24) + (uint64(version&0x7f) << 17) + uint64(chainId))
}

// ParseV split V to 4 parts
func ParseV(V *big.Int) (txType uint8, version uint8, secp256k1V uint8, chainId uint16) {
	uint64V := V.Uint64()
	txType = uint8((uint64V >> 24) & 0xff)
	version = uint8((uint64V >> 17) & 0x7f)
	secp256k1V = uint8((uint64V >> 16) & 1)
	chainId = uint16(uint64V & 0xffff)
	return
}
