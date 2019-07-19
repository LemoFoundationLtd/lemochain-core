package log

import (
	"github.com/inconshreveable/log15"
	"io"
	"os"
)

var Hub ClosingHandler

type ClosingHandler struct {
	io.WriteCloser
	log15.Handler
}

func (h *ClosingHandler) Close() error {
	return h.WriteCloser.Close()
}

func fileHandler(path string, fmtr log15.Format) (log15.Handler, error) {
	if Hub.WriteCloser != nil {
		Hub.Close()
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	Hub = ClosingHandler{f, log15.StreamHandler(f, fmtr)}
	return Hub, nil
}

// FileHandler 写入文件handler
func FileHandler(path string, fmtr log15.Format) log15.Handler {
	h, err := fileHandler(path, fmtr)
	if err != nil {
		panic(err)
	}
	return h
}
