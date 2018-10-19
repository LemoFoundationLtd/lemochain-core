package main

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/main/console"
	"github.com/LemoFoundationLtd/lemochain-go/main/node"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"syscall"
)

var (
	app = node.NewApp("the lemochain-go command line interface")
	// flags to configure the node
	nodeFlags = []cli.Flag{
		node.DataDirFlag,
		node.NetworkIdFlag,
		node.MaxPeersFlag,
		node.ListenPortFlag,
		node.ExtraDataFlag,
		node.NodeKeyFileFlag,
		node.MiningEnabledFlag,
		node.JSpathFlag,
		node.DebugFlag,
		node.LogLevelFlag,
	}

	rpcFlags = []cli.Flag{
		node.RPCEnabledFlag,
		node.RPCListenAddrFlag,
		node.RPCPortFlag,
		node.RPCApiFlag,
		node.WSEnabledFlag,
		node.WSListenAddrFlag,
		node.WSPortFlag,
		node.WSApiFlag,
		node.WSAllowedOriginsFlag,
		node.IPCDisabledFlag,
		node.IPCPathFlag,
	}
)

func init() {
	app.Action = glemo
	app.HideVersion = true
	app.Copyright = "Copyright 2017-2018 The Lemochain-go Authors"
	app.Commands = []cli.Command{
		initCommand,
		consoleCommand,
		attachCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	app.Flags = append(app.Flags, nodeFlags...)

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		// if err := debug.Setup(ctx); err != nil {
		// 	return err
		// }
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		// debug.Exit()
		console.Stdin.Close()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func glemo(ctx *cli.Context) error {
	n := makeFullNode(ctx)
	startNode(ctx, n)
	n.Wait()
	return nil
}

func startNode(ctx *cli.Context, n *node.Node) {
	if err := n.Start(); err != nil {
		node.Fatalf("Error tarting node: %v", err)
	}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigCh)
		<-sigCh
		log.Info("Got interrupt, shutting down...")
		for i := 5; i > 0; i-- {
			<-sigCh
			if i > 1 {
				log.Warnf("Already shutting down, interupt more to panic. times: %d", i-1)
			}
		}
	}()

	go func() {
		// todo api处理
	}()

	if ctx.IsSet(node.MiningEnabledFlag.Name) {
		if err := n.StartMining(); err != nil {
			log.Errorf("start mining failed: %v", err)
		}
	}
}
