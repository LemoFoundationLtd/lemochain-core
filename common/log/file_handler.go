package log

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"os"
	"path/filepath"
)

const (
	logDir        = "log"       // 日志存储目录
	logFileName   = "glemo.log" // 最新日志存储文件
	fPrefix       = "glemo"
	RotateLogSize = 64 * 1024 * 1024 // 64M
	BackUpCount   = 19               // 滚动日志文件数
)

var (
	logFilePath = filepath.Join(logDir, logFileName) // 日志文件路径
)

// fileExist 查看log目录是否存在
func fileExist(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, nil
	}
}

// openLogFile
func openLogFile(logFilePath string) (*os.File, error) {
	// 查看log目录是否存在
	logDir := filepath.Dir(logFilePath)
	isExist, err := fileExist(logDir)
	if err != nil {
		panic(err)
	}
	if !isExist {
		// 创建log目录
		if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
			panic(err)
		}
	}
	// 创建或者打开./log/glemo.log 文件
	return os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

// FileHandler 写入文件handler
func FileHandler(logFilePath string, fmtr log15.Format) log15.Handler {
	h, err := fileHandler(logFilePath, fmtr)
	if err != nil {
		panic(err)
	}
	return h
}

func fileHandler(logFilePath string, fmtr log15.Format) (log15.Handler, error) {
	f, err := openLogFile(logFilePath)
	if err != nil {
		return nil, err
	}
	return WriteFileHandler(logFilePath, f, fmtr), nil
}

// listenAndRotateLog
func listenAndRotateLog(logFilePath string, f *os.File) *os.File {
	// 判断log文件是否需要滚动
	if info, err := f.Stat(); err == nil {
		if info.Size() >= RotateLogSize {
			f.Close()
			rotateLogFile(filepath.Dir(logFilePath)) // 滚动日志
			// 重新打开log文件句柄
			f, err = openLogFile(logFilePath)
			if err != nil {
				panic(err)
			}
			return f
		}
	}
	return f
}

func WriteFileHandler(logFilePath string, f *os.File, fmtr log15.Format) log15.Handler {
	h := log15.FuncHandler(func(r *log15.Record) error {
		f = listenAndRotateLog(logFilePath, f)
		_, err := f.Write(fmtr.Format(r))
		return err
	})
	return log15.LazyHandler(log15.BufferedHandler(20480, h)) // 缓存20k
}

// rotateLogFile 滚动日志文件
func rotateLogFile(logDir string) {
	// 开始滚动日志文件
	for j := BackUpCount; j >= 1; j-- {
		curFileName := filepath.Join(logDir, fmt.Sprintf("%s_%d.log", fPrefix, j))
		k := j - 1
		preFileName := filepath.Join(logDir, fmt.Sprintf("%s_%d.log", fPrefix, k))

		if k == 0 {
			preFileName = logFilePath
		}
		if _, err := os.Stat(curFileName); err == nil {
			os.Remove(curFileName)
		}
		if _, err := os.Stat(preFileName); err == nil {
			os.Rename(preFileName, curFileName)
		}
	}
}
