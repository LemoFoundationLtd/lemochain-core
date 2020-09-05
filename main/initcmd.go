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
	"path/filepath"
)

const (
	genesisConfigName = "genesis.json"
)

var (
	initCommand = cli.Command{
		Action: initGenesis,
		Name:   "init",
		Usage:  "Bootstrap and initialize a new genesis block",
		Flags: []cli.Flag{
			node.DataDirFlag,
		},
		Category:    "BLOCKCHAIN COMMANDS",
		Description: `The init command initializes a new genesis block.`,
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
	dir := ctx.GlobalString(node.DataDirFlag.Name)

	block := setupGenesisBlock(dir)
	log.Infof("init genesis succeed. hash: %s", block.Hash().Hex())
	return nil
}

func setupGenesisBlock(dataDir string) *types.Block {
	genesis, err := loadGenesisFile(dataDir)
	if err != nil {
		panic(err)
	}
	return saveBlock(dataDir, genesis)
}

// saveBlock save block to db
func saveBlock(dataDir string, genesis *chain.Genesis) *types.Block {
	chaindata := node.GetChainDataPath(dataDir)
	db := store.NewChainDataBase(chaindata)
	defer func() {
		if err := db.Close(); err != nil {
			log.Errorf("close db failed. %v", err)
		}
	}()
	genesisBlock, err := db.GetBlockByHeight(0)
	if err == nil && genesisBlock != nil {
		log.Errorf("Genesis block is existed. Please clean the folder \"%s\" first", chaindata)
		panic(chain.ErrGenesisExist)
	}
	return chain.SetupGenesisBlock(db, genesis)
}

// loadGenesisFile
func loadGenesisFile(dataDir string) (*chain.Genesis, error) {
	filePath := filepath.Join(dataDir, genesisConfigName)
	if _, err := os.Stat(filePath); err != nil {
		// Try to read from relative path
		filePath = genesisConfigName
	}
	log.Infof("Load genesis config file: %s", filePath)
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
