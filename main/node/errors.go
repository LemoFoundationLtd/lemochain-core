package node

import "errors"

var (
	ErrAlreadyRunning    = errors.New("already running")
	ErrOpenFileFailed    = errors.New("open file datadir failed")
	ErrServerStartFailed = errors.New("start p2p server failed")
	ErrRpcStartFailed    = errors.New("start rpc failed")
)
