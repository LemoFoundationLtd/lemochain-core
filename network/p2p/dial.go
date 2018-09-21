package p2p

import (
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"net"
	"time"
)

const (
	dialTimeout = 3 * time.Second
)

type dialTask struct {
	remoteNode string
	srv        *Server
}

func newDialTask(node string, srv *Server) *dialTask {
	return &dialTask{
		remoteNode: node,
		srv:        srv,
	}
}

func (d *dialTask) Run() error {
	conn, err := net.DialTimeout("tcp", d.remoteNode, dialTimeout)
	if err != nil {
		return err
	}
	if err = d.srv.HandleConn(conn, false); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}
