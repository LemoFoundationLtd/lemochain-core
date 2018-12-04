package p2p

import (
	"errors"
)

var (
	ErrConnectSelf        = errors.New("can't connect yourself")
	ErrGenesisNotMatch    = errors.New("can't match genesis block")
	ErrBadRemoteID        = errors.New("bad remoteID")
	ErrUnavailablePackage = errors.New("unavailable net package")
	ErrBadPubKey          = errors.New("invalid public key")
)
