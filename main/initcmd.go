package main

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/main/node"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

var (
	initCommand = cli.Command{
		Action:    initGenesis,
		Name:      "init",
		Usage:     "Bootstrap and initialize a new genesis block",
		ArgsUsage: "<genesisPath>",
		Flags: []cli.Flag{
			node.DataDirFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The init command initializes a new genesis block.

It expects the genesis file as argument.`,
	}
)

var (
	ErrFileReadFailed     = errors.New("open genesis config file failed")
	ErrInvalidGenesisFile = errors.New("invalid genesis file")
	ErrEmptyDeputyNodes   = errors.New("deputy nodes is empty")
)

// initGenesis 初始化创始块action
func initGenesis(ctx *cli.Context) error {
	// init log
	log.Setup(log.LevelInfo, false, false)

	// open special genesis config file
	genesisFile := ctx.Args().First()
	dir := ctx.GlobalString(node.DataDirFlag.Name)
	if len(genesisFile) == 0 {
		log.Crit("Must supply genesis json file path")
	}

	hash, err := setupGenesisBlock(genesisFile, dir)
	if err != nil {
		log.Crit(err.Error())
	}
	log.Infof("init genesis succeed. hash: %s", hash.Hex())
	return nil
}

func setupGenesisBlock(genesisFile, datadir string) (common.Hash, error) {
	genesis, err := unmarshal(genesisFile)
	if err != nil {
		return common.Hash{}, err
	}
	return saveBlock(datadir, genesis)
}

// saveBlock save block to db
func saveBlock(datadir string, genesis *chain.Genesis) (common.Hash, error) {
	chaindata := filepath.Join(datadir, "chaindata")
	db := store.NewChainDataBase(chaindata, DRIVER_MYSQL, DNS_MYSQL)
	hash, err := chain.SetupGenesisBlock(db, genesis)
	if err != nil {
		return common.Hash{}, err
	}
	db.Close()
	// check deputy nodes
	if len(genesis.DeputyNodes) == 0 {
		return common.Hash{}, ErrEmptyDeputyNodes
	}
	return hash, nil
}

// unmarshal
func unmarshal(genesisFile string) (*chain.Genesis, error) {
	file, err := os.Open(genesisFile)
	if err != nil {
		log.Errorf("%v", err)
		return nil, ErrFileReadFailed
	}
	defer file.Close()

	// decode genesis config file string
	genesis := new(chain.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		log.Errorf("%v", err)
		return nil, ErrInvalidGenesisFile
	}

	return genesis, nil
}
