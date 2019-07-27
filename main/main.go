package main

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/flag"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/main/console"
	"github.com/LemoFoundationLtd/lemochain-core/main/node"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"github.com/LemoFoundationLtd/lemochain-core/network/ntp"
	"github.com/inconshreveable/log15"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"syscall"
	"time"
)

var (
	app = node.NewApp("the lemochain-core command line interface")
	// flags to configure the node
	nodeFlags = []cli.Flag{
		node.DataDirFlag,
		node.MaxPeersFlag,
		node.ListenPortFlag,
		node.ExtraDataFlag,
		node.AutoMineFlag,
		node.JSpathFlag,
		node.DebugFlag,
		node.LogLevelFlag,
	}

	rpcFlags = []cli.Flag{
		node.RPCEnabledFlag,
		node.RPCListenAddrFlag,
		node.RPCPortFlag,
		node.RPCCORSDomainFlag,
		node.RPCVirtualHostsFlag,
		node.WSEnabledFlag,
		node.WSListenAddrFlag,
		node.WSPortFlag,
		node.WSAllowedOriginsFlag,
		node.IPCDisabledFlag,
		node.IPCPathFlag,
	}

	attachFlags        = make([]cli.Flag, 0)
	createaccountFlags = make([]cli.Flag, 0)
	createnodekeyFlags = make([]cli.Flag, 0)
)

func init() {
	app.Action = glemo
	app.HideVersion = true
	app.Copyright = "Copyright 2017-2018 The lemochain-core Authors"
	app.Commands = []cli.Command{
		initCommand,
		consoleCommand,
		attachCommand,
		createaccountCommand,  // create an account when run "./glemo createaccount"
		createanodekeyCommand, // create nodekey and nodeID when run "./glemo createnodekey"
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)

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
		// recover panic and add panic info to log.txt
		if e := recover(); e != nil {
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			log.Error("==>", string(buf[:n]))
			os.Exit(1)
		}
		os.Exit(1)
	}
}

// initLog init log config
func initLog(ctx *cli.Context) {
	// flag in command is in range 1~5
	logFlag := -1
	if ctx.GlobalIsSet(common.LogLevel) {
		logFlag = ctx.GlobalInt(common.LogLevel) - 1
	} else if ctx.IsSet(common.LogLevel) {
		logFlag = ctx.Int(common.LogLevel) - 1
	}
	// logLevel is in range 0~4
	logLevel := log15.Lvl(logFlag)
	// default level
	if logLevel < 0 || logLevel > 4 {
		logLevel = log.LevelError // 1
	}
	showCodeLine := logLevel >= 3 // LevelInfo, LevelDebug
	log.Setup(logLevel, true, showCodeLine)
}

func makeFullNode(ctx *cli.Context) *node.Node {
	initLog(ctx)
	// 校对系统时间,目前只针对linux系统
	if err := ntp.TimeProof(); err != nil {
		log.Errorf("Time proof failed. Error details:%vPlease synchronize the system time correctly before starting the glemo", err)
		if runtime.GOOS == "linux" {
			panic(err)
		}
	}
	// process flags
	totalFlags := append(nodeFlags, rpcFlags...)
	flags := flag.NewCmdFlags(ctx, totalFlags)
	// new node
	return node.New(flags)
}

func glemo(ctx *cli.Context) error {
	n := makeFullNode(ctx)
	startNode(ctx, n)
	n.Wait()
	return nil
}

func interrupt(wait func() error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	<-sigCh
	log.Info("Got interrupt, shutting down...")
	go wait()
	for i := 5; i > 0; i-- {
		<-sigCh
		if i > 1 {
			log.Warnf("Already shutting down, interrupt more to panic. times: %d", i-1)
		}
	}
	panic("boom")
}

func startNode(ctx *cli.Context, n *node.Node) {
	if err := n.Start(); err != nil {
		log.Critf("Error starting node: %v", err)
	}

	// go metrics.PointMetricsLog() // 打印出系统内存和磁盘的占用情况
	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
	// 启动告警系统
	go metrics.NewAlarmManager().Start()

	go interrupt(n.Stop)

	if ctx.IsSet(node.AutoMineFlag.Name) {
		if err := n.StartMining(); err != nil {
			log.Errorf("Start mining failed: %v", err)
		}
	}
}
