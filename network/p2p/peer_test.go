package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"net"
	"testing"
	"time"
)

var (
	prvCli *ecdsa.PrivateKey
	pubCli *ecdsa.PublicKey

	prvSrv *ecdsa.PrivateKey
	pubSrv *ecdsa.PublicKey
)

func init() {
	prvCli, _ = crypto.ToECDSA(common.FromHex("0xc21b6b2fbf230f665b936194d14da67187732bf9d28768aef1a3cbb26608f8aa"))
	pubCli = &prvCli.PublicKey

	prvSrv, _ = crypto.ToECDSA(common.FromHex("0x9c3c4a327ce214f0a1bf9cfa756fbf74f1c7322399ffff925efd8c15c49953eb"))
	pubSrv = &prvSrv.PublicKey
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
	peer := newPeer(conn)
	srvNodeID := PubKeyToNodeID(pubSrv)
	if err = peer.doHandshake(prvCli, &srvNodeID); err != nil {
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
	peer := newPeer(*conn)
	if err := peer.doHandshake(prvSrv, nil); err != nil {
		t.Fatalf("server handshake failed: %v", err)
	}
	p := peer.(*Peer)
	fmt.Printf("server: aes=%s\r\n", common.ToHex(p.aes))
	cliPeer := <-cliPeerCh
	if bytes.Compare(cliPeer.aes, p.aes) != 0 {
		t.Fatalf("aes not match")
	}
}

type MyStruct struct {
	C int32
	S string
	B []byte
}

type Tar struct {
	Str MyStruct
	// Arr []*MyStruct
}

func Test_size(t *testing.T) {
	contains := make([]string, 0, 10)
	contains = append(contains, "11111111")
	contains = append(contains, "22222222")
	contains = append(contains, "33333333")
	contains = append(contains, "44444444")
	contains = append(contains, "55555555")
	for i, _ := range contains {
		if i > 2 {
			contains = append(contains[0:i], contains[i+1:]...)
		}
	}
	fmt.Println(contains)
}
