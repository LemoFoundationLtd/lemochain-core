package main

import (
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
	cfg := &glemoConfig{
		Lemo: node.DefaultConfig,
		Node: defaultNodeConfig(),
	}
	node.SetNodeConfig(ctx, &cfg.Node)
	n, err := node.New(&cfg.Lemo, &cfg.Node)
	if err != nil {
		node.Fatalf("Failed to create node: %v", err)
	}
	node.SetLemoConfig(ctx, &cfg.Lemo)
	return n, cfg
}

func makeFullNode(ctx *cli.Context) *node.Node {
	node, _ := makeConfigNode(ctx)
	// todo
	return node
}
