package network

import "errors"

var (
	ErrInvalidCode = errors.New("invalid code about net message")
	ErrReadTimeout = errors.New("protocol handshake timeout")
	ErrReadMsg     = errors.New("protocol handshake failed: read remote message failed")
)
