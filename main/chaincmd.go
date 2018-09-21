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
		Action:    node.MigrateFlags(initGenesis),
		Name:      "init",
		Usage:     "Bootstrap and initialize a new genesis block",
		ArgsUsage: "<genesisPath>",
		Flags: []cli.Flag{
			node.DataDirFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The init command initializes a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.

It expects the genesis file as argument.`,
	}
)

// initGenesis 初始化创始块action
func initGenesis(ctx *cli.Context) error {
	genesisFile := ctx.Args().First()
	if len(genesisFile) == 0 {
		node.Fatalf("Must supply genesis json file path")
	}
	file, err := os.Open(genesisFile)
	if err != nil {
		node.Fatalf("Failed to open genesis file:%v ", err)
	}
	defer file.Close()
	genesis := new(chain.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		node.Fatalf("invalid genesis file: %v", err)
	}
	dir := ctx.GlobalString(node.DataDirFlag.Name)
	dir = filepath.Join(dir, "chaindata")
	db, err := store.NewCacheChain(dir)
	hash, err := chain.SetupGenesisBlock(db, genesis)
	if err != nil {
		node.Fatalf("Failed to init genesis: %v", err)
	}
	db.Close()
	log.Infof("init genesis succeed. hash: %s", hash.Hex())
	return nil
}
