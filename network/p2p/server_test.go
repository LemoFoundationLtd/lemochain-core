package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"io"
	"net"
	"testing"
	"time"
)

const (
	prvKeyS = "058ac392b8254f1188f2609759dcd89d551249673db9515f034ac7c55ba93376"
	pubKeyS = "c7dd92e3553585cccd6bc00011ddc54c5bb1233daa2f4b9257927db812afad36d6999e389a79e2c380b5c3b8037af23e7b1da3ee7c2a07cb31e3ca92beafc311"

	prvKeyC = "6559e412543a6ac50d641873f24ecc387a5e0c509a5c824c9395889f1aec67b0"
	pubKeyC = "3c8f19bfbd593adc5ee48d1d96ce9fcfafaf30b5c4739add2aaa2d90d6966d52986f8dab5e3b9fdb59bb2fded683fe4df73ecdfcbb743d538eb5668a0c82ca07"
)

func init() {
	//log.Init(log.LevelDebug, false)
}

func startListenTCP(t *testing.T) *Server {
	prvKey, _ := crypto.ToECDSA(common.FromHex(prvKeyS))
	c := Config{
		PrivateKey:        prvKey,
		MaxPeerNum:        100,
		MaxPendingPeerNum: 10,
		Name:              "glemo",
		NodeDatabase:      "./node/nodelist",
		ListenAddr:        "127.0.0.1:6000",
	}
	srv := &Server{
		Config: c,
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Could not start server: %v", err)
	}
	return srv
}

func TestListenTCP(t *testing.T) {
	startListenTCP(t)
	time.Sleep(1 * time.Second)
}

func TestTCPServer(t *testing.T) {
	srv := startListenTCP(t)
	time.Sleep(1 * time.Second)

	conn, err := net.DialTimeout("tcp", srv.ListenAddr, 3*time.Second)
	if err != nil {
		t.Fatal("dial failed")
	}
	buf := make([]byte, 4+64)
	// 发送自己的NodeID
	v := make([]byte, 4)
	binary.BigEndian.PutUint32(v, uint32(1))
	copy(buf[:4], v)
	copy(buf[4:], common.FromHex(pubKeyC))
	conn.Write(buf)
	// 读取对方的NodeID
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal(err)
	}
	version := binary.BigEndian.Uint32(buf[:4])
	if version != 1 {
		t.Fatal("version not match")
	}
	log.Info(fmt.Sprintf("connect peer:%s success", common.ToHex(buf[4:])))
}

func TestDial(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:6001")
	if err != nil {
		t.Fatalf("can't setup listener:%v", err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Error("accept err:", err)
			return
		}
		accepted <- conn
	}()

	srv := startListenTCP(t)
	connected := make(chan *Peer)
	srv.PeerEvent = func(peer *Peer, flag PeerEventFlag) error {
		connected <- peer
		return nil
	}
	srv.AddStaticPeer("127.0.0.1:6001")
	select {
	case conn := <-accepted:
		defer conn.Close()
		buf := make([]byte, 68)
		v := make([]byte, 4)
		binary.BigEndian.PutUint32(v, uint32(1))
		copy(buf[:4], v)
		copy(buf[4:], common.FromHex(pubKeyC))
		conn.Write(buf)
		fmt.Println(common.ToHex(buf))
		conn.Read(buf)
		fmt.Println(common.ToHex(buf))
		select {
		case <-connected:
			fmt.Printf("dial success. node:%s", common.ToHex(buf[4:20]))
		case <-time.After(1 * time.Second):
			t.Error("timeout for dial")
		}
	case <-time.After(1 * time.Second):
		t.Error("server did not connect within one second")
	}
	time.Sleep(1 * time.Second)
}
