package types

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto/sha3"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"math/big"
)

//go:generate gencodec -type Header -field-override headerMarshaling -out gen_header_json.go

type Header struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	LemoBase    common.Address `json:"miner"            gencodec:"required"`
	VersionRoot common.Hash    `json:"versionRoot"      gencodec:"required"`
	TxRoot      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	LogsRoot    common.Hash    `json:"changeLogRoot"    gencodec:"required"`
	EventRoot   common.Hash    `json:"eventRoot"        gencodec:"required"`
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	Height      uint32         `json:"height"           gencodec:"required"`
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        *big.Int       `json:"timestamp"        gencodec:"required"`
	SignData    []byte         `json:"signData"         gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"` // 最大256byte
}

type headerMarshaling struct {
	GasLimit hexutil.Uint64
	GasUsed  hexutil.Uint64
	Time     *hexutil.Big
	SignData hexutil.Bytes
	Extra    hexutil.Bytes
	Hash     common.Hash `json:"hash"`
}

// 签名信息
type SignData [65]byte

// Block
type Block struct {
	Header         *Header
	Txs            []*Transaction
	ChangeLog      []*ChangeLog // todo
	Events         []*Event
	ConfirmPackage []SignData
}

func NewBlock(header *Header, txs []*Transaction, changeLog []*ChangeLog, events []*Event, confirmPackage []SignData) *Block {
	return &Block{
		Header:         header,
		Txs:            txs,
		ChangeLog:      changeLog,
		Events:         events,
		ConfirmPackage: confirmPackage,
	}
}

type Blocks []*Block

// Hash 块hash 排除 SignData字段
func (h *Header) Hash() common.Hash {
	return rlpHash([]interface{}{
		h.ParentHash,
		h.LemoBase,
		h.VersionRoot,
		h.TxRoot,
		h.LogsRoot,
		h.Bloom,
		h.Height,
		h.GasLimit,
		h.GasUsed,
		h.Time,
		h.Extra,
	})
}

// Copy 拷贝一份头
func (h *Header) Copy() *Header {
	cpy := *h
	if cpy.Time = new(big.Int); h.Time != nil {
		cpy.Time.Set(h.Time)
	}
	return &cpy
}

// rlpHash 数据rlp编码后求hash
func rlpHash(data interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, data)
	hw.Sum(h[:0])
	return h
}

// func (b *Block) Header() *Header { return b.Header }

func (b *Block) Hash() common.Hash        { return b.Header.Hash() }
func (b *Block) Height() uint32           { return b.Header.Height }
func (b *Block) ParentHash() common.Hash  { return b.Header.ParentHash }
func (b *Block) LemoBase() common.Address { return b.Header.LemoBase }
func (b *Block) VersionRoot() common.Hash { return b.Header.VersionRoot }
func (b *Block) TxHash() common.Hash      { return b.Header.TxRoot }
func (b *Block) LogsHash() common.Hash    { return b.Header.LogsRoot }
func (b *Block) EventRoot() common.Hash   { return b.Header.EventRoot }
func (b *Block) Bloom() Bloom             { return b.Header.Bloom }
func (b *Block) GasLimit() uint64         { return b.Header.GasLimit }
func (b *Block) GasUsed() uint64          { return b.Header.GasUsed }
func (b *Block) Time() *big.Int           { return new(big.Int).Set(b.Header.Time) }
func (b *Block) SignData() []byte         { return b.Header.SignData }
func (b *Block) Extra() []byte            { return b.Header.Extra }

// func (b *Block) Txs() []*Transaction               { return b.Txs }
// func (b *Block) ChangeLog() []*ChangeLog           { return b.ChangeLog }
// func (b *Block) Events() []*Event                  { return b.Events }
// func (b *Block) ConfirmPackage() []SignData        { return b.ConfirmPackage }
func (b *Block) SetHeader(header *Header)          { b.Header = header }
func (b *Block) SetTxs(txs []*Transaction)         { b.Txs = txs }
func (b *Block) SetConfirmPackage(pack []SignData) { b.ConfirmPackage = pack }
func (b *Block) SetChangeLog(log []*ChangeLog)     { b.ChangeLog = log }
func (b *Block) SetEvents(events []*Event)         { b.Events = events }
