package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

var (
	prvCli    *ecdsa.PrivateKey
	pubCli    *ecdsa.PublicKey
	nodeIDCli NodeID

	prvSrv    *ecdsa.PrivateKey
	pubSrv    *ecdsa.PublicKey
	nodeIDSrv NodeID
)

func init() {
	prvCli, _ = crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
	pubCli = &prvCli.PublicKey
	nodeIDCli = PubKeyToNodeID(pubCli)

	prvSrv, _ = crypto.ToECDSA(common.FromHex("0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"))
	pubSrv = &prvSrv.PublicKey
	nodeIDSrv = PubKeyToNodeID(pubSrv)
}

func newListener() *net.Conn {
	endpoint := "0.0.0.0:8001"
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		return nil
	}
	conn, err := listener.Accept()
	if err != nil {
		return nil
	}
	return &conn
}

func newClient(t *testing.T, cliPeerCh chan *Peer) {
	endpoint := "127.0.0.1:8001"
	conn, err := net.DialTimeout("tcp", endpoint, 3*time.Second)
	if err != nil {
		t.Fatalf("dial failed")
	}
	peer := NewPeer(conn)
	srvNodeID := PubKeyToNodeID(pubSrv)
	if err = peer.DoHandshake(prvCli, &srvNodeID); err != nil {
		t.Fatalf("client handshake failed: %v", err)
	}
	p := peer.(*Peer)
	fmt.Printf("client: aes=%s\r\n", common.ToHex(p.aes))
	cliPeerCh <- p
}

func Test_doHandshake(t *testing.T) {
	cliPeerCh := make(chan *Peer)
	go newClient(t, cliPeerCh)
	conn := newListener()
	if conn == nil {
		t.Fatalf("new server failed")
	}
	peer := NewPeer(*conn)
	if err := peer.DoHandshake(prvSrv, nil); err != nil {
		t.Fatalf("server handshake failed: %v", err)
	}
	p := peer.(*Peer)
	fmt.Printf("server: aes=%s\r\n", common.ToHex(p.aes))
	cliPeer := <-cliPeerCh
	if bytes.Compare(cliPeer.aes, p.aes) != 0 {
		t.Fatalf("aes not match")
	}
}

func newPeers(t *testing.T) (pCli, pSrv IPeer) {
	connCli, connSrv := net.Pipe()
	pCli = NewPeer(connCli)
	pSrv = NewPeer(connSrv)
	errSrvCh := make(chan error)
	go func() {
		errSrvCh <- pSrv.DoHandshake(prvSrv, nil)
	}()
	err := pCli.DoHandshake(prvCli, &nodeIDSrv)
	errSrv := <-errSrvCh
	assert.Nil(t, err)
	assert.Nil(t, errSrv)

	pC := pCli.(*Peer)
	pS := pSrv.(*Peer)
	if bytes.Compare(pC.aes, pS.aes) != 0 {
		t.Error("AES not match")
	}
	go pCli.Run()
	go pSrv.Run()
	return
}

func Test_totalLogic(t *testing.T) {
	pCli, pSrv := newPeers(t)

	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	// client
	go func() {
		err := pCli.WriteMsg(uint32(2), buf)
		assert.Nil(t, err)
	}()

	// server
	msg, err := pSrv.ReadMsg()
	assert.Nil(t, err)
	if bytes.Compare(buf, msg.Content) != 0 {
		t.Error("message not match")
	}

	pCli.Close()
	pSrv.Close()
}

func Test_Chan(t *testing.T) {
	c := make(chan struct{})
	// go func() {
	// 	// c <- struct{}{}
	// 	close(c)
	// }()

	select {
	case _, ok := <-c:
		if ok {
			close(c)
			fmt.Println("ok")
		} else {
			fmt.Println("not ok")
		}
		break
	default:
		close(c)
		fmt.Println("ok")
	}
}
