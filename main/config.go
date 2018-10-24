package main

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/main/node"
	"github.com/inconshreveable/log15"
	"gopkg.in/urfave/cli.v1"
)

func makeFullNode(ctx *cli.Context) *node.Node {
	// init log config
	var logLevel = 2
	if ctx.GlobalIsSet(common.LogLevel) {
		logLevel = ctx.GlobalInt(common.LogLevel)
	} else if ctx.IsSet(common.LogLevel) {
		logLevel = ctx.Int(common.LogLevel)
	}
	logLevel -= 1
	if logLevel < 0 || logLevel > 4 {
		logLevel = 2
	}
	log.Setup(log15.Lvl(logLevel), true, true)

	// process flags
	totalFlags := append(nodeFlags, rpcFlags...)
	flags := flag.NewCmdFlags(ctx, totalFlags)

	// new node
	n, err := node.New(flags)
	if err != nil {
		node.Fatalf("Failed to create node: %v", err)
	}
	return n
}
