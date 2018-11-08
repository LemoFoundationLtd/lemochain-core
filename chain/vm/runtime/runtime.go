package runtime

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"math"
	"math/big"
	"time"

	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/chain/vm"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
)

// Config is a basic type specifying certain configuration flags for running
// the EVM.
type Config struct {
	ChainConfig *params.ChainConfig
	Origin      common.Address
	Coinbase    common.Address
	BlockHeight uint32
	Time        uint32
	GasLimit    uint64
	GasPrice    *big.Int
	Value       *big.Int
	Debug       bool
	EVMConfig   vm.Config

	AccountManager *account.Manager
	GetHashFn      func(n uint64) common.Hash
}

// sets defaults on the config
func setDefaults(cfg *Config) {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainID: 1,
		}
	}

	if cfg.Time == 0 {
		cfg.Time = uint32(time.Now().Unix())
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
}

// Execute executes the code using the input as call data during the execution.
// It returns the EVM's return value, the new state and an error if it failed.
//
// Executes sets up a in memory, temporarily, environment for the execution of
// the given code. It makes sure that it's restored to it's original state afterwards.
func Execute(code, input []byte, cfg *Config) ([]byte, error) {
	if cfg == nil {
		cfg = new(Config)
	}
	setDefaults(cfg)

	if cfg.AccountManager == nil {
		db, _ := store.NewCacheChain("../../../db")
		cfg.AccountManager = account.NewManager(common.Hash{}, db)
	}
	var (
		address = common.BytesToAddress([]byte("contract"))
		vmenv   = NewEnv(cfg)
		sender  = vm.AccountRef(cfg.Origin)
	)
	contractAccount := cfg.AccountManager.GetAccount(address)
	// set the receiver's (the executing contract) code for execution.
	contractAccount.SetCode(code)
	// Call the code with the given configuration.
	ret, _, err := vmenv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)

	return ret, err
}

// Create executes the code using the EVM create method
func Create(input []byte, cfg *Config) ([]byte, common.Address, uint64, error) {
	if cfg == nil {
		cfg = new(Config)
	}
	setDefaults(cfg)

	if cfg.AccountManager == nil {
		db, _ := store.NewCacheChain("../../../db")
		cfg.AccountManager = account.NewManager(common.Hash{}, db)
	}
	var (
		vmenv  = NewEnv(cfg)
		sender = vm.AccountRef(cfg.Origin)
	)

	// Call the code with the given configuration.
	code, address, leftOverGas, err := vmenv.Create(
		sender,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	return code, address, leftOverGas, err
}

// Call executes the code given by the contract's address. It will return the
// EVM's return value or an error if it failed.
//
// Call, unlike Execute, requires a config and also requires the am field to
// be set.
func Call(address common.Address, input []byte, cfg *Config) ([]byte, error) {
	setDefaults(cfg)

	vmenv := NewEnv(cfg)

	sender := cfg.AccountManager.GetAccount(cfg.Origin)
	// Call the code with the given configuration.
	ret, _, err := vmenv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)

	return ret, err
}
