package network

import "time"

type BlockSyncFlag struct {
	running    bool
	peer       *peer
	from       uint32
	to         uint32
	lastUpdate time.Duration
}

// NewBlockSync
func NewBlockSync() *BlockSyncFlag {
	return &BlockSyncFlag{}
}

// Init
func (s *BlockSyncFlag) Init(p *peer) {
	s.running = true
	s.peer = p
	s.lastUpdate = time.Duration(time.Now().Second())
}

// Finish
func (s *BlockSyncFlag) Finish() {
	s.running = false
	s.peer = nil
}

// Error
func (s *BlockSyncFlag) Error() {
	s.peer.SyncFailed()
	s.running = false
	s.peer = nil
}
