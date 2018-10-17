package p2p

import "fmt"

var (
	ErrConnectSelf     = fmt.Errorf("can't connect yourself")
	ErrGenesisNotMatch = fmt.Errorf("can't match genesis block")
)
