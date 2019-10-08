package main

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/main/node"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"gopkg.in/urfave/cli.v1"
	"os"
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

	block := setupGenesisBlock(genesisFile, dir)
	log.Infof("init genesis succeed. hash: %s", block.Hash().Hex())
	return nil
}

func setupGenesisBlock(genesisFile, datadir string) *types.Block {
	genesis, err := loadGenesisFile(genesisFile)
	if err != nil {
		panic(err)
	}
	return saveBlock(datadir, genesis)
}

// saveBlock save block to db
func saveBlock(datadir string, genesis *chain.Genesis) *types.Block {
	chaindata := node.GetChainDataPath(datadir)
	db := store.NewChainDataBase(chaindata)
	defer func() {
		if err := db.Close(); err != nil {
			log.Errorf("close db failed. %v", err)
		}
	}()
	return chain.SetupGenesisBlock(db, genesis)
}

// loadGenesisFile
func loadGenesisFile(filePath string) (*chain.Genesis, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("%v", err)
		return nil, ErrFileReadFailed
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Errorf("close genesis file failed. %v", err)
		}
	}()

	// decode genesis config file string
	genesis := new(chain.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		log.Errorf("%v", err)
		return nil, ErrInvalidGenesisFile
	}

	return genesis, nil
}
