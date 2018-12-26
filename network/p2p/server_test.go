package p2p

import (
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func initServer(port int) *Server {
	prvCli, _ = crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
	config := Config{
		PrivateKey: prvCli,
		Port:       port,
	}
	discover := newDiscover()
	server := NewServer(config, discover)
	return server
}

func Test_Listen_failed(t *testing.T) {
	server := initServer(70707)
	assert.Panics(t, func() {
		server.Start()
	})
}
