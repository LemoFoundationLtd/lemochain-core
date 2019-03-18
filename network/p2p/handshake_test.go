package p2p

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"testing"
)

func Test_newCliEncHandshake_error(t *testing.T) {
	var remoteID *NodeID = nil
	_, err := newCliEncHandshake(remoteID)
	assert.Equal(t, ErrNilRemoteID, err)

	remoteID = new(NodeID)
	_, err = newCliEncHandshake(remoteID)
	assert.Equal(t, ErrBadRemoteID, err)
}

func Test_newSrvEncHandshake_error(t *testing.T) {
	reqMsg := new(authReqMsg)
	prv, _ := crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))

	_, err := newSrvEncHandshake(reqMsg, prv)
	assert.Equal(t, ErrBadRemoteID, err)

	nodeID := common.FromHex("0xddb5fc36c415799e4c0cf7046ddde04aad6de8395d777db4f46ebdf258e55ee1d698fdd6f81a950f00b78bb0ea562e4f7de38cb0adf475c5026bb885ce74afb0")
	copy(reqMsg.ClientPubKey[:], nodeID)
	_, err = newSrvEncHandshake(reqMsg, prv)
	assert.Equal(t, ErrRecoveryFailed, err)
}

func Test_clientEncHandshake_ok(t *testing.T) {
	connCli, connSrv := net.Pipe()
	prvCli, _ := crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
	prvSrv, _ := crypto.ToECDSA(common.FromHex("0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"))

	nodeIDSrv := PubKeyToNodeID(&prvSrv.PublicKey)

	srvSCh := make(chan *secrets)
	go func() {
		srvS, err := serverEncHandshake(connSrv, prvSrv, nil)
		assert.NoError(t, err)
		srvSCh <- srvS
	}()

	cliS, err := clientEncHandshake(connCli, prvCli, &nodeIDSrv)
	assert.NoError(t, err)
	srvS := <-srvSCh
	if bytes.Compare(cliS.Aes, srvS.Aes) != 0 {
		t.Fatalf("Aes not match")
	}
}

func Test_clientEncHandshake_error(t *testing.T) {
	connCli, connSrv := net.Pipe()
	prvCli, _ := crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
	prvSrv, _ := crypto.ToECDSA(common.FromHex("0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"))

	nodeIDSrv := PubKeyToNodeID(&prvSrv.PublicKey)

	srvSCh := make(chan *secrets)
	go func() {
		srvS, err := serverEncHandshake(connSrv, prvSrv, func() {
			connSrv.Close()
		})
		assert.Equal(t, "io: read/write on closed pipe", err.Error())
		srvSCh <- srvS
	}()

	cliS, err := clientEncHandshake(connCli, prvCli, &nodeIDSrv)
	assert.Equal(t, io.EOF, err)
	srvS := <-srvSCh
	assert.Nil(t, srvS)
	assert.Nil(t, cliS)
}
