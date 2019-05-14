package main

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/main/console"
	"github.com/LemoFoundationLtd/lemochain-core/network/rpc"
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
		Flags:     attachFlags,
		Category:  "CONSOLE COMMANDS",
		Description: `
The Glemo console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Dapp JavaScript API.
This command allows to open a console on a running glemo node.`,
	}

	createaccountCommand = cli.Command{
		Action:    createAccount,
		Name:      "createaccount",
		Usage:     "createaccount",
		ArgsUsage: "[endpoint]",
		Flags:     createaccountFlags,
		Category:  "BLOCKCHAIN COMMANDS",
		Description: `
Create an account when we execute "./glemo createaccount".`,
	}
)

// createAccount
func createAccount(ctx *cli.Context) error {
	acc, err := crypto.GenerateAddress()
	if err == nil {
		fmt.Println("Please keep your account safe! \nPlease apply again if the private key is divulged!\n ")
		fmt.Printf("Private:\n%s\n", acc.Private)
		fmt.Printf("PubKey:\n%s\n", acc.Public)
		fmt.Printf("LemoAddress:\n%s", acc.Address.String())
		fmt.Println("\n")
		return nil
	} else {
		fmt.Println("Create account error:", err.Error())
		fmt.Println("Suggest to retry!!!")
		return nil
	}
}

func localConsole(ctx *cli.Context) error {

	n := makeFullNode(ctx)
	startNode(ctx, n)
	defer n.Stop()

	client, err := n.Attach()
	if err != nil {
		log.Critf("Failed to attach to the inproc glemo: %v", err)
	}
	startConsole(client, n.ChainID())
	return nil
}

func remoteConsole(ctx *cli.Context) error {
	// Attach to a remotely running glemo instance and start the JavaScript console
	endpoint := ctx.Args().First()
	if endpoint == "" {
		log.Critf("Unable to attach to remote glemo: no ipc path")
	}
	client, err := rpc.Dial(endpoint)
	if err != nil {
		log.Critf("Unable to attach to remote glemo: %v", err)
	}
	var chainID uint16
	if err := client.Call(&chainID, "chain_chainID"); err != nil {
		log.Critf("Unable to call remote glemo: %v", err)
	}
	startConsole(client, chainID)
	return nil
}

func startConsole(client *rpc.Client, chainID uint16) {
	config := console.Config{
		DocRoot: "scripts", // consoleObj.Execute("exec.js") will execute the file "js/exec.js"
		Client:  client,
		ChainID: chainID,
	}

	consoleObj, err := console.New(config)
	if err != nil {
		log.Critf("Failed to start the JavaScript console: %v", err)
	}
	defer consoleObj.Stop(false)
	// Otherwise print the welcome screen and enter interactive mode
	consoleObj.Welcome()
	consoleObj.Interactive()
}
