package main

import (
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/main/console"
	"github.com/LemoFoundationLtd/lemochain-go/main/node"
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
	config := console.Config{
		DocRoot: "scripts", // consoleObj.Execute("exec.js") will execute the file "js/exec.js"
		Client:  client,
	}
	consoleObj, err := console.New(config)
	if err != nil {
		log.Errorf("Failed to start the JavaScript console: %v", err)
	}
	defer consoleObj.Stop(false)
	// Otherwise print the welcome screen and enter interactive mode
	consoleObj.Welcome()
	consoleObj.Interactive()
	return nil
}

func remoteConsole(ctx *cli.Context) error {
	// todo
	return nil
}
