package main

import (
	"encoding/json"
	"github.com/LemoFoundationLtd/lemochain-go/chain"
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

// initGenesis 初始化创始块action
func initGenesis(ctx *cli.Context) error {
	// init log
	log.Setup(log.LevelInfo, false, false)

	// open special genesis config file
	genesisFile := ctx.Args().First()
	if len(genesisFile) == 0 {
		node.Fatalf("Must supply genesis json file path")
	}
	file, err := os.Open(genesisFile)
	if err != nil {
		node.Fatalf("Failed to open genesis file:%v ", err)
	}
	defer file.Close()

	// decode genesis config file string
	genesis := new(chain.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		node.Fatalf("invalid genesis file: %v", err)
	}

	// setup genesis block
	dir := ctx.GlobalString(node.DataDirFlag.Name)
	dir = filepath.Join(dir, "chaindata")
	db, err := store.NewCacheChain(dir)
	hash, err := chain.SetupGenesisBlock(db, genesis)
	if err != nil {
		node.Fatalf(err.Error())
	}
	db.Close()
	log.Infof("init genesis succeed. hash: %s", hash.Hex())
	return nil
}
