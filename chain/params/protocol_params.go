package params

import (
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"math/big"
)

var (
	TargetGasLimit uint64 = GenesisGasLimit // The artificial target
)

const (
	GasLimitBoundDivisor uint64 = 1024      // The bound divisor of the gas limit, used in update calculations.
	MinGasLimit          uint64 = 200000    // Minimum the gas limit may ever be.
	GenesisGasLimit      uint64 = 105000000 // Gas limit of the Genesis block.

	CallValueTransferGas uint64 = 9000  // Paid for CALL when the value transfer is non-zero.
	CallNewAccountGas    uint64 = 25000 // Paid for CALL when the destination address didn't exist prior.

	OrdinaryTxGas         uint64 = 21000 // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	TxGasContractCreation uint64 = 53000 // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.
	VoteTxGas             uint64 = 35000 // 投票交易固定gas消耗
	RegisterTxGas         uint64 = 92000 // 注册候选节点固定gas消耗
	CreateAssetTxGas      uint64 = 67000 // 创建资产固定gas消耗
	IssueAssetTxGas       uint64 = 55000 // 发行资产固定gas消耗
	ReplenishAssetTxGas   uint64 = 25000 // 增发资产固定gas消耗
	ModifyAssetTxGas      uint64 = 35000 // 修改资产info固定gas消耗
	TransferAssetTxGas    uint64 = 30000 // 交易资产固定gas消耗
	ModifySigsTxGas       uint64 = 67000 // 设置多重签名账户交易固定gas消耗
	BoxTxGas              uint64 = 40000 // 设置箱子交易固定gas消耗

	TxMessageGas  uint64 = 68    // 交易中的message字段消耗gas
	TxDataZeroGas uint64 = 4     // Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions.
	QuadCoeffDiv  uint64 = 512   // Divisor for the quadratic particle of the memory cost equation.
	SstoreSetGas  uint64 = 20000 // Once per SLOAD operation.
	EventDataGas  uint64 = 8     // Per byte in a LOG* operation's data.
	CallStipend   uint64 = 2300  // Free gas given at beginning of call.

	Sha3Gas          uint64 = 30    // Once per SHA3 operation.
	Sha3WordGas      uint64 = 6     // Once per word of the SHA3 operation's data.
	SstoreResetGas   uint64 = 5000  // Once per SSTORE operation if the zeroness changes from zero.
	JumpdestGas      uint64 = 1     // Refunded gas, once per SSTORE operation if the zeroness changes to zero.
	CreateDataGas    uint64 = 200   //
	CallCreateDepth  uint64 = 1024  // Maximum depth of call/create stack.
	EventGas         uint64 = 375   // Per LOG* operation.
	CopyGas          uint64 = 3     //
	StackLimit       uint64 = 1024  // Maximum size of VM stack allowed.
	EventTopicGas    uint64 = 375   // Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas.
	CreateGas        uint64 = 32000 // Once per CREATE operation & contract-creation transaction.
	MemoryGas        uint64 = 3     // Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.
	TxDataNonZeroGas uint64 = 68    // Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions.

	MaxCodeSize = 24576 // Maximum bytecode to permit for a contract

	// Precompiled contract gas prices

	EcrecoverGas            uint64 = 3000   // Elliptic curve sender recovery gas price
	Sha256BaseGas           uint64 = 60     // Base price for a SHA256 operation
	Sha256PerWordGas        uint64 = 12     // Per-word price for a SHA256 operation
	Ripemd160BaseGas        uint64 = 600    // Base price for a RIPEMD160 operation
	Ripemd160PerWordGas     uint64 = 120    // Per-word price for a RIPEMD160 operation
	IdentityBaseGas         uint64 = 15     // Base price for a data copy operation
	IdentityPerWordGas      uint64 = 3      // Per-work price for a data copy operation
	ModExpQuadCoeffDiv      uint64 = 20     // Divisor for the quadratic particle of the big int modular exponentiation
	Bn256AddGas             uint64 = 500    // Gas needed for an elliptic curve addition
	Bn256ScalarMulGas       uint64 = 40000  // Gas needed for an elliptic curve scalar multiplication
	Bn256PairingBaseGas     uint64 = 100000 // Base price for an elliptic curve pairing check
	Bn256PairingPerPointGas uint64 = 80000  // Per-point price for an elliptic curve pairing check
)

var (
	TermDuration            uint32 = 1000000                       // 每届间隔
	InterimDuration         uint32 = 1000                          // 过渡期
	RewardCheckHeight       uint32 = 100000                        // The height to check if miners' reward is set
	ReleaseEvilNodeDuration uint32 = 30000                         // 释放作恶节点的过度期高度30000个区块，按照3秒出块大概24h
	MinGasPrice                    = big.NewInt(1000000000)        // 默认的最低gas price 为1G mo
	MinCandidateDeposit            = common.Lemo2Mo("5000000")     // 注册成为候选节点的质押金额最小值
	DepositPoolAddress             = common.HexToAddress("0x1001") // 设置接收注册候选节点押金费用1000LEMO的地址
	DepositExchangeRate            = common.Lemo2Mo("100")         // 质押金额兑换票数兑换率 100LEMO换1票
	VoteExchangeRate               = common.Lemo2Mo("200")         // 投票票数兑换率 200LEMO换1票

	MaxPackageLength uint32 = 100 * 1024 * 1024 // 100M
	MaxTxsForMiner   int    = 10000             // max transactions when mining a block

	TermRewardPoolTotal = common.Lemo2Mo("900000000") // 奖励池总量
	TermRewardContract  = common.HexToAddress("0x09") // 换届奖励的预编译合约地址
	MinRewardPrecision  = common.Lemo2Mo("1")         // 1 LEMO
)

//go:generate gencodec -type Reward --field-override RewardMarshaling -out gen_Reward_json.go
// The map to store the miner's reward
type RewardsMap map[uint32]*Reward
type Reward struct {
	Term  uint32   `json:"term" gencodec:"required"`
	Value *big.Int `json:"value" gencodec:"required"`
	Times uint32   `json:"times" gencodec:"required"`
}

type RewardMarshaling struct {
	Term  hexutil.Uint32
	Value *hexutil.Big10
	Times hexutil.Uint32
}

//go:generate gencodec -type RewardJson --field-override RewardJsonMarshaling -out gen_RewardJson_json.go
type RewardJson struct {
	Term  uint32   `json:"term" gencodec:"required"`
	Value *big.Int `json:"value" gencodec:"required"`
}

type RewardJsonMarshaling struct {
	Term  hexutil.Uint32
	Value *hexutil.Big10
}
