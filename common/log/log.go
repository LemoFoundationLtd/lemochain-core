package log

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/inconshreveable/log15/term"
	"github.com/mattn/go-colorable"
	"io"
	"os"
)

var srvLog = log15.New()

const (
	LevelCrit  = log15.LvlCrit
	LevelError = log15.LvlError
	LevelWarn  = log15.LvlWarn
	LevelInfo  = log15.LvlInfo
	LevelDebug = log15.LvlDebug
)

func init() {
	Setup(LevelInfo, false, false)
}

// Setup change the log config immediately
// The lv is higher the more logs would be visible
func Setup(lv log15.Lvl, toFile bool, showCodeLine bool) {
	outputLv := lv
	useColor := term.IsTty(os.Stdout.Fd()) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stderr)
	if useColor {
		output = colorable.NewColorableStderr()
	}
	handlers := []log15.Handler{
		log15.StreamHandler(output, TerminalFormat(useColor, showCodeLine)),
	}
	if toFile {
		handlers = append(handlers, FileHandler(logFilePath, log15.JsonFormat())) // 日志文件存储路径: ./log/glemo.log
	}
	handler := log15.MultiHandler(handlers...)
	handler = log15.LvlFilterHandler(outputLv, handler)
	srvLog.SetHandler(handler)
}

func Debug(msg string, ctx ...interface{}) {
	srvLog.Debug(msg, ctx...)
}

func Debugf(format string, values ...interface{}) {
	msg := fmt.Sprintf(format, values...)
	srvLog.Debug(msg)
}

func Info(msg string, ctx ...interface{}) {
	srvLog.Info(msg, ctx...)
}

func Infof(format string, values ...interface{}) {
	msg := fmt.Sprintf(format, values...)
	srvLog.Info(msg)
}

func Warn(msg string, ctx ...interface{}) {
	srvLog.Warn(msg, ctx...)
}

func Warnf(format string, values ...interface{}) {
	msg := fmt.Sprintf(format, values...)
	srvLog.Warn(msg)
}

func Error(msg string, ctx ...interface{}) {
	srvLog.Error(msg, ctx...)
}

func Errorf(format string, values ...interface{}) {
	msg := fmt.Sprintf(format, values...)
	srvLog.Error(msg)
}

func Crit(msg string, ctx ...interface{}) {
	srvLog.Crit(msg, ctx...)
	os.Exit(1)
}

func Critf(format string, values ...interface{}) {
	msg := fmt.Sprintf(format, values...)
	srvLog.Crit(msg)
	os.Exit(1)
}

// Lazy allows you to defer calculation of a logged value that is expensive
// to compute until it is certain that it must be evaluated with the given filters.
//
// Lazy may also be used in conjunction with a Logger's New() function
// to generate a child logger which always reports the current value of changing
// state.
//
// You may wrap any function which takes no arguments to Lazy. It may return any
// number of values of any type.
type Lazy = log15.Lazy
