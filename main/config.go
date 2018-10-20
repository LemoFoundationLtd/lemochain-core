package main

import (
	"github.com/LemoFoundationLtd/lemochain-go/common/flag"
	"github.com/LemoFoundationLtd/lemochain-go/main/node"
	"gopkg.in/urfave/cli.v1"
)

type glemoConfig struct {
	Lemo node.LemoConfig
	Node node.NodeConfig
}

func defaultNodeConfig() node.NodeConfig {
	return node.NodeConfig{}
}

func makeConfigNode(ctx *cli.Context) (*node.Node, *glemoConfig) {
	totalFlags := append(nodeFlags, rpcFlags...)
	flags := flag.NewCmdFlags(ctx, totalFlags)
	cfg := &glemoConfig{
		Lemo: node.DefaultConfig,
		Node: defaultNodeConfig(),
	}
	node.SetNodeConfig(flags, &cfg.Node)
	n, err := node.New(&cfg.Lemo, &cfg.Node, flags)
	if err != nil {
		node.Fatalf("Failed to create node: %v", err)
	}
	node.SetLemoConfig(flags, &cfg.Lemo)
	return n, cfg
}

func makeFullNode(ctx *cli.Context) *node.Node {
	node, _ := makeConfigNode(ctx)
	// todo
	return node
}
