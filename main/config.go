package main

import (
	"github.com/LemoFoundationLtd/lemochain-go/main/node"
	"gopkg.in/urfave/cli.v1"
	"strconv"
)

type glemoConfig struct {
	Lemo node.LemoConfig
	Node node.NodeConfig
}

func defaultNodeConfig() node.NodeConfig {
	return node.NodeConfig{}
}

// loadTotalFlags load all flags to blockchain object
func loadTotalFlags(ctx *cli.Context) map[string]string {
	flags := make(map[string]string, len(nodeFlags)+len(rpcFlags))
	totalFlags := append(nodeFlags, rpcFlags...)
	for _, f := range totalFlags {
		switch f.(type) {
		case cli.StringFlag:
			if ctx.GlobalIsSet(f.GetName()) {
				flags[f.GetName()] = ctx.GlobalString(f.GetName())
			} else {
				flags[f.GetName()] = ctx.String(f.GetName())
			}
			break
		case cli.IntFlag:
			if ctx.GlobalIsSet(f.GetName()) {
				flags[f.GetName()] = strconv.Itoa(ctx.GlobalInt(f.GetName()))
			} else {
				flags[f.GetName()] = strconv.Itoa(ctx.Int(f.GetName()))
			}
			break
		case cli.BoolFlag:
			var r bool
			if ctx.GlobalIsSet(f.GetName()) {
				r = ctx.GlobalBool(f.GetName())
			} else {
				r = ctx.Bool(f.GetName())
			}
			if r {
				flags[f.GetName()] = "true"
			} else {
				flags[f.GetName()] = "false"
			}
		default:

		}
	}
	return flags
}

func makeConfigNode(ctx *cli.Context) (*node.Node, *glemoConfig) {
	flags := loadTotalFlags(ctx)
	cfg := &glemoConfig{
		Lemo: node.DefaultConfig,
		Node: defaultNodeConfig(),
	}
	node.SetNodeConfig(ctx, &cfg.Node)
	n, err := node.New(&cfg.Lemo, &cfg.Node, flags)
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
