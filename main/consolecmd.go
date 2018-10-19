package main

import (
	"github.com/LemoFoundationLtd/lemochain-go/main/console"
	"github.com/LemoFoundationLtd/lemochain-go/main/node"
	"github.com/LemoFoundationLtd/lemochain-go/network/rpc"
	"gopkg.in/urfave/cli.v1"
)

var (
	consoleCommand = cli.Command{
		Action:   localConsole,
		Name:     "console",
		Usage:    "Start an interactive JavaScript environment",
		Flags:    append(nodeFlags, rpcFlags...),
		Category: "CONSOLE COMMANDS",
		Description: `
The Glemo console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Dapp JavaScript API.`,
	}

	attachCommand = cli.Command{
		Action:    remoteConsole,
		Name:      "attach",
		Usage:     "attach",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{node.DataDirFlag},
		Category:  "CONSOLE COMMANDS",
		Description: `
The Glemo console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Dapp JavaScript API.
This command allows to open a console on a running glemo node.`,
	}
)

func localConsole(ctx *cli.Context) error {
	n := makeFullNode(ctx)
	startNode(ctx, n)
	defer n.Stop()

	client, err := n.Attach()
	if err != nil {
		node.Fatalf("Failed to attach to the inproc glemo: %v", err)
	}
	startConsole(client)
	return nil
}

func remoteConsole(ctx *cli.Context) error {
	// Attach to a remotely running glemo instance and start the JavaScript console
	endpoint := ctx.Args().First()
	if endpoint == "" {
		node.Fatalf("Unable to attach to remote glemo: no ipc path")
	}
	client, err := rpc.Dial(endpoint)
	if err != nil {
		node.Fatalf("Unable to attach to remote glemo: %v", err)
	}
	startConsole(client)
	return nil
}

func startConsole(client *rpc.Client) {
	config := console.Config{
		DocRoot: "scripts", // consoleObj.Execute("exec.js") will execute the file "js/exec.js"
		Client:  client,
	}

	consoleObj, err := console.New(config)
	if err != nil {
		node.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer consoleObj.Stop(false)
	// Otherwise print the welcome screen and enter interactive mode
	consoleObj.Welcome()
	consoleObj.Interactive()
}
