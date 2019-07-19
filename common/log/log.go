package log

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/inconshreveable/log15/term"
	"github.com/mattn/go-colorable"
	"io"
	"os"
	"time"
)

const (
	DoRotateTimeInterval = 10 * time.Second
	fPrefix              = "glemo"
	logFileName          = "glemo.log"
	RotateLogSize        = 64 * 1024 * 1024 // 64M
	BackUp_Count         = 20               // 滚动日志文件数

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
	handler := log15.StreamHandler(output, TerminalFormat(useColor, showCodeLine))
	if toFile {
		handler = log15.MultiHandler(
			handler,
			FileHandler(logFileName, log15.JsonFormat()),
		)
	}
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

// DoRotate 滚动日志
func DoRotate(lv log15.Lvl, toFile bool, showCodeLine bool) {
	if !toFile {
		return
	}
	ticker := time.NewTicker(DoRotateTimeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if info, err := os.Stat(logFileName); err == nil {
				if info.Size() > RotateLogSize {
					Hub.Close() // 关闭日志文件句柄
					// 开始滚动日志
					for j := BackUp_Count; j >= 1; j-- {
						curFileName := fmt.Sprintf("%s_%d.log", fPrefix, j)
						k := j - 1
						preFileName := fmt.Sprintf("%s_%d.log", fPrefix, k)

						if k == 0 {
							preFileName = logFileName
						}
						if _, err := os.Stat(curFileName); err == nil {
							os.Remove(curFileName)
						}
						if _, err := os.Stat(preFileName); err == nil {
							os.Rename(preFileName, curFileName)
						}
					}
					// 重新setup
					Setup(lv, true, showCodeLine)
				}
			} else {
				Errorf("Open %s error: %v", logFileName, err)
			}
		}
	}
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
