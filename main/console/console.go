// Copyright 2016 The lemochain-go Authors
// This file is part of the lemochain-go library.
//
// The lemochain-go library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lemochain-go library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lemochain-go library. If not, see <http://www.gnu.org/licenses/>.

package console

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/LemoFoundationLtd/lemochain-go/main/jsre"
	"github.com/LemoFoundationLtd/lemochain-go/network/rpc"
	"github.com/mattn/go-colorable"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
)

var (
	passwordRegexp = regexp.MustCompile(`account.[nus]`)
	exit           = regexp.MustCompile(`^\s*exit\s*;*\s*$`)
)

// Config is the collection of configurations to fine tune the behavior of the
// JavaScript console.
type Config struct {
	DocRoot  string       // Filesystem path from where to load JavaScript files from
	Client   *rpc.Client  // RPC client to execute Lemochain requests through
	Prompter UserPrompter // Input prompter to allow interactive user feedback (defaults to TerminalPrompter)
	Printer  io.Writer    // Output writer to serialize any display strings to (defaults to os.Stdout)
	ChainID  uint16       // Id of LemoChain network
}

// Console is a JavaScript interpreted runtime environment. It is a fully fleged
// JavaScript console attached to a running node via an external or in-process RPC
// client.
type Console struct {
	client   *rpc.Client  // RPC client to execute Lemochain requests through
	jsre     *jsre.JSRE   // JavaScript runtime environment running the interpreter
	prompt   string       // Input prompt prefix string
	prompter UserPrompter // Input prompter to allow interactive user feedback
	printer  io.Writer    // Output writer to serialize any display strings to
}

func New(config Config) (*Console, error) {
	// Handle unset config values gracefully
	if config.Prompter == nil {
		config.Prompter = Stdin
	}
	if config.Printer == nil {
		config.Printer = colorable.NewColorableStdout()
	}
	// Initialize the console and return
	console := &Console{
		client:   config.Client,
		jsre:     jsre.New(config.DocRoot, config.Printer),
		prompt:   "> ",
		prompter: config.Prompter,
		printer:  config.Printer,
	}
	if err := console.init(config.ChainID); err != nil {
		return nil, err
	}
	return console, nil
}

// init retrieves the available APIs from the remote RPC provider and initializes
// the console's JavaScript namespaces based on the exposed modules.
func (c *Console) init(chainID uint16) error {
	// Initialize the JavaScript <-> Go RPC bridge
	bridge := newBridge(c.client, c.prompter, c.printer)
	c.jsre.Set("lemoConfig", struct{}{})

	lemoConfigObj, _ := c.jsre.Get("lemoConfig")
	lemoConfigObj.Object().Set("send", bridge.Send)
	lemoConfigObj.Object().Set("chainID", chainID)

	consoleObj, _ := c.jsre.Get("console")
	consoleObj.Object().Set("log", c.consoleOutput)
	consoleObj.Object().Set("error", c.consoleOutput)

	// Load all the internal utility JavaScript libraries
	if err := c.jsre.Compile("babel-polyfill.js", jsre.BabelPolyfillJS); err != nil {
		return fmt.Errorf("babel-polyfill.js: %v", err)
	}
	if err := c.jsre.Compile("lemo-client.js", jsre.LemoClientJS); err != nil {
		return fmt.Errorf("lemo-client.js: %v", err)
	}
	if _, err := c.jsre.Run("var lemo = new LemoClient(lemoConfig);"); err != nil {
		return fmt.Errorf("lemo lemoConfig: %v", err)
	}
	if _, err := c.jsre.Run("BigNumber = lemo.BigNumber;"); err != nil {
		return fmt.Errorf("expose BigNumber: %v", err)
	}
	// Load our extension for the module.
	if err := c.jsre.Compile("lemo-node-admin.js", jsre.LemoNodeAdminJS); err != nil {
		return fmt.Errorf("lemo-node-admin.js: %v", err)
	}
	return nil
}

// consoleOutput is an override for the console.log and console.error methods to
// stream the output into the configured output stream instead of stdout.
func (c *Console) consoleOutput(call otto.FunctionCall) otto.Value {
	output := []string{}
	for _, argument := range call.ArgumentList {
		output = append(output, fmt.Sprintf("%v", argument))
	}
	fmt.Fprintln(c.printer, strings.Join(output, " "))
	return otto.Value{}
}

// Welcome show summary of current Glemo instance and some metadata about the
// console's available modules.
func (c *Console) Welcome() {
	// Print some generic Glemo metadata
	fmt.Fprintf(c.printer, "Welcome to the lemo JavaScript console!\n")
	c.jsre.Run(`Promise.all([
		lemo.getNodeVersion(),
		lemo.mine.getMiner(),
		lemo.getCurrentBlock(false, false),
		lemo.getCurrentBlock(true, false)
	]).then(function(results) {
		var msg = [
			"node: v" + results[0],
			"sdk: v" + lemo.SDK_VERSION,
			"minerAddress: " + results[1],
			"current block: " + results[2].header.height + " " + results[2].header.hash + " (" + new Date(1000 * results[2].header.timestamp).toLocaleString() + ")",
			"latest stable block: " + results[3].header.height + " " + results[3].header.hash + " (" + new Date(1000 * results[3].header.timestamp).toLocaleString() + ")"
		].join("\n") + "\n";
		console.log(msg)
	});`)
	// List all the supported modules for the user to call
	if modules, err := c.client.SupportedModules(); err == nil {
		sort.Strings(modules)
		fmt.Fprintln(c.printer, "server side modules:", strings.Join(modules, ", "))
	}
	fmt.Fprintln(c.printer)
}

// Evaluate executes code and pretty prints the result to the specified output
// stream.
func (c *Console) Evaluate(statement string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(c.printer, "[native] error: %v\n", r)
		}
	}()
	return c.jsre.Evaluate(statement, c.printer)
}

// Interactive starts an interactive user session, where input is propted from
// the configured user prompter.
func (c *Console) Interactive() {
	var scheduler = make(chan string) // Channel to send the next prompt on and receive the input
	// Start a goroutine to listen for promt requests and send back inputs
	go func() {
		for {
			// Read the next user input
			line, err := c.prompter.PromptInput(<-scheduler)
			if err != nil {
				// In case of an error, either clear the prompt or fail
				if err == liner.ErrPromptAborted { // ctrl-C
					scheduler <- ""
					continue
				}
				close(scheduler)
				return
			}
			// User input retrieved, send for interpretation and loop
			scheduler <- line
		}
	}()
	// Monitor Ctrl-C too in case the input is empty and we need to bail
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, syscall.SIGINT, syscall.SIGTERM)

	// Start sending prompts to the user and reading back inputs
	for {
		// Send the next prompt, triggering an input read and process the result
		scheduler <- c.prompt
		select {
		case <-abort:
			// User forcefully quite the console
			fmt.Fprintln(c.printer, "caught interrupt, exiting")
			return

		case line, ok := <-scheduler:
			// User input was returned by the prompter, handle special cases
			if !ok || (exit.MatchString(line)) {
				return
			}
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			if !passwordRegexp.MatchString(line) && c.prompter != nil {
				c.prompter.AppendHistory(line)
			}
			c.Evaluate(line)
		}
	}
}

// Execute runs the JavaScript file specified as the argument.
func (c *Console) Execute(path string) error {
	return c.jsre.Exec(path)
}

// Stop cleans up the console and terminates the runtime environment.
func (c *Console) Stop(graceful bool) error {
	c.jsre.Stop(graceful)
	return nil
}
