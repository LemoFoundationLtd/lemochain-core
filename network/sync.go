package network

import "time"

type BlockSyncFlag struct {
	running    bool
	peer       *peer
	from       uint32
	to         uint32
	lastUpdate int64
}

// NewBlockSync
func NewBlockSync() *BlockSyncFlag {
	return &BlockSyncFlag{}
}

// Init
func (s *BlockSyncFlag) Init(p *peer) {
	s.running = true
	s.peer = p
	s.lastUpdate = time.Now().Unix()
}

// Finish
func (s *BlockSyncFlag) Finish() {
	s.running = false
	s.peer = nil
}

// Error
func (s *BlockSyncFlag) Error() {
	if s.peer != nil {
		s.peer.SyncFailed()
		s.peer = nil
	}
	s.running = false
}
