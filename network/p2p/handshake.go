package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/ecies"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/secp256k1"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"io"
)

const (
	sskLen = 16
	shaLen = 32
	pubLen = 64
	sigLen = 65
)

var (
	PackagePrefix = []byte{0x5a, 0x48}     // package flag
	PackageLength = 4                      // package length bytes
	PackageMaxLen = 1 * 1024 * 1024 * 1024 // 1 Gb
)

// encHandshake object for handshake
type encHandshake struct {
	remoteID NodeID

	remotePub            *ecies.PublicKey
	initNonce, respNonce []byte
	randomPrvKey         *ecies.PrivateKey
	remoteRandomPubKey   *ecies.PublicKey
}

// String
func (h *encHandshake) String() string {
	return fmt.Sprintf("remoteID: %s; initNonce: %s; respNonce: %s; randomPrvKey: %s; remoteRandomPubKey: %s",
		common.ToHex(h.remoteID[:]), common.ToHex(h.initNonce), common.ToHex(h.respNonce),
		common.ToHex(crypto.FromECDSA(h.randomPrvKey.ExportECDSA())),
		common.ToHex(exportPubKey(h.remoteRandomPubKey)))
}

// newCliEncHandshake new instance for client
func newCliEncHandshake(remoteID *NodeID) (*encHandshake, error) {
	if remoteID == nil {
		return nil, ErrNilRemoteID
	}
	h := &encHandshake{remoteID: *remoteID}
	// generate InitNonce
	h.initNonce = make([]byte, shaLen)
	if _, err := rand.Read(h.initNonce); err != nil {
		return nil, err
	}
	// fetch remote public key
	rPub, err := h.remoteID.PubKey()
	if err != nil {
		return nil, ErrBadRemoteID
	}
	h.remotePub = ecies.ImportECDSAPublic(rPub)
	// generate random private key
	h.randomPrvKey, err = ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// newSrvEncHandshake new instance for server
func newSrvEncHandshake(reqMsg *authReqMsg, prv *ecdsa.PrivateKey) (h *encHandshake, err error) {
	h = new(encHandshake)
	h.initNonce = reqMsg.InitNonce[:]
	copy(h.remoteID[:], reqMsg.ClientPubKey[:])
	rPub, err := h.remoteID.PubKey()
	if err != nil {
		return nil, ErrBadRemoteID
	}
	h.remotePub = ecies.ImportECDSAPublic(rPub)
	h.randomPrvKey, err = ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
	if err != nil {
		return h, err
	}
	// shared token
	token, err := ecies.ImportECDSA(prv).GenerateShared(h.remotePub, 16, 16)
	signed := xor(token, h.initNonce)
	randomRemotePubKey, err := secp256k1.RecoverPubkey(signed, reqMsg.Signature[:])
	if err != nil {
		return h, ErrRecoveryFailed
	}
	h.remoteRandomPubKey, err = importPubKey(randomRemotePubKey)
	if err != nil {
		return h, err
	}
	// RespNonce
	h.respNonce = make([]byte, shaLen)
	if _, err := rand.Read(h.respNonce); err != nil {
		return h, err
	}
	return h, nil
}

// authReqMsg client request object
type authReqMsg struct {
	Signature    [sigLen]byte
	ClientPubKey [pubLen]byte
	InitNonce    [shaLen]byte
}

// authRespMsg server response object
type authRespMsg struct {
	RandomPubKey [pubLen]byte
	RespNonce    [shaLen]byte
}

// secrets mark aes for node
type secrets struct {
	RemoteID NodeID
	Aes      []byte
}

// xor bits operation
func xor(a, b []byte) []byte {
	res := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		res[i] = a[i] ^ b[i]
	}
	return res
}

// makeAuthReqMsg generate request message
func (h *encHandshake) makeAuthReqMsg(prv *ecdsa.PrivateKey) ([]byte, error) {
	// token
	token, err := ecies.ImportECDSA(prv).GenerateShared(h.remotePub, sskLen, sskLen)
	if err != nil {
		return nil, err
	}
	// xor
	signed := xor(token, h.initNonce)
	signature, err := crypto.Sign(signed, h.randomPrvKey.ExportECDSA())
	if err != nil {
		return nil, err
	}
	// msg
	msg := new(authReqMsg)
	copy(msg.InitNonce[:], h.initNonce)
	copy(msg.ClientPubKey[:], crypto.FromECDSAPub(&prv.PublicKey)[1:])
	copy(msg.Signature[:], signature)
	buf, err := rlp.EncodeToBytes(&msg)
	if err != nil {
		return nil, err
	}
	// ecies
	res, err := ecies.Encrypt(rand.Reader, h.remotePub, buf, nil, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// makeAuthRespMsg generate response message
func (h *encHandshake) makeAuthRespMsg(prv *ecdsa.PrivateKey) ([]byte, error) {
	// generate response message
	respMsg := new(authRespMsg)
	copy(respMsg.RandomPubKey[:], exportPubKey(&h.randomPrvKey.PublicKey))
	copy(respMsg.RespNonce[:], h.respNonce)

	// rlp encode
	buf := new(bytes.Buffer)
	if err := rlp.Encode(buf, respMsg); err != nil {
		return nil, err
	}

	// ecies encrypt
	encBuf, err := ecies.Encrypt(rand.Reader, h.remotePub, buf.Bytes(), nil, nil)
	if err != nil {
		return nil, err
	}
	return encBuf, nil
}

// clientEncHandshake initiate a network request as client
func clientEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, remoteID *NodeID) (s *secrets, err error) {
	// generate init object
	h, err := newCliEncHandshake(remoteID)
	if err != nil {
		return s, err
	}
	// generate encrypt data
	encBuf, err := h.makeAuthReqMsg(prv)
	if err != nil {
		return s, err
	}
	// send data to server
	if err = write(conn, encBuf); err != nil {
		return nil, err
	}

	// receive
	respMsg, err := readHandshakeRespMsg(conn, prv)
	if err != nil {
		return s, err
	}
	h.respNonce = make([]byte, shaLen)
	copy(h.respNonce, respMsg.RespNonce[:])
	h.remoteRandomPubKey, err = importPubKey(respMsg.RandomPubKey[:])
	// log.Debugf("client encHandshake obj: %s", h.String())
	if err != nil {
		return nil, err
	}
	// Aes
	token, err := h.randomPrvKey.GenerateShared(h.remoteRandomPubKey, sskLen, sskLen)
	hash := crypto.Keccak256(token, crypto.Keccak256(h.respNonce, h.initNonce))
	s = &secrets{
		RemoteID: *remoteID,
		Aes:      hash[:16],
	}
	return s, nil
}

// readHandshakeBuf read net stream
func readHandshakeBuf(conn io.ReadWriter, prv *ecdsa.PrivateKey) ([]byte, error) {
	// prefix
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	if bytes.Compare(PackagePrefix, buf) != 0 {
		return nil, ErrUnavailablePackage
	}
	// content length
	buf = make([]byte, PackageLength)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(buf)
	if length == 0 || length > uint32(PackageMaxLen) {
		return nil, ErrUnavailablePackage
	}
	// content
	buf = make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	// ecies decrypt
	buf, err := ecies.ImportECDSA(prv).Decrypt(buf, nil, nil)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// readHandshakeRespMsg read server's response message
func readHandshakeRespMsg(conn io.ReadWriter, prv *ecdsa.PrivateKey) (*authRespMsg, error) {
	// read handshake data
	buf, err := readHandshakeBuf(conn, prv)
	if err != nil {
		return nil, err
	}

	// rlp decode
	resp := new(authRespMsg)
	s := rlp.NewStream(bytes.NewReader(buf), 0)
	s.Decode(resp)
	return resp, nil
}

// readHandshakeReqMsg read client's request message
func readHandshakeReqMsg(conn io.ReadWriter, prv *ecdsa.PrivateKey) (*authReqMsg, error) {
	buf, err := readHandshakeBuf(conn, prv)
	if err != nil {
		return nil, err
	}

	// rlp decode
	req := new(authReqMsg)
	s := rlp.NewStream(bytes.NewReader(buf), 0)
	s.Decode(req)
	return req, nil
}

// serverEncHandshake accept a network request as server
func serverEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, callback func()) (s *secrets, err error) {
	// read request data
	reqMsg, err := readHandshakeReqMsg(conn, prv)
	if err != nil {
		return nil, err
	}
	// generate encHandshake
	h, err := newSrvEncHandshake(reqMsg, prv)
	if err != nil {
		return nil, err
	}
	// make response message
	encBuf, err := h.makeAuthRespMsg(prv)
	if err != nil {
		return nil, err
	}
	// just for test
	if callback != nil {
		callback()
	}
	// send data to client
	if err = write(conn, encBuf); err != nil {
		return nil, err
	}
	// aes
	token, err := h.randomPrvKey.GenerateShared(h.remoteRandomPubKey, 16, 16)
	if err != nil {
		return nil, err
	}
	hash := crypto.Keccak256(token, crypto.Keccak256(h.respNonce, h.initNonce))
	s = &secrets{
		Aes:      hash[:16],
		RemoteID: h.remoteID,
	}
	return s, nil
}

// importPubKey convert bytes to public key
func importPubKey(input []byte) (*ecies.PublicKey, error) {
	var pubKey65 []byte
	if len(input) == 64 {
		pubKey65 = append([]byte{0x04}, input...)
	} else if len(input) == 65 {
		pubKey65 = input
	} else {
		return nil, ErrBadRemoteID
	}
	pub := crypto.ToECDSAPub(pubKey65)
	if pub.X == nil {
		return nil, ErrBadPubKey
	}
	return ecies.ImportECDSAPublic(pub), nil
}

// exportPubKey export public key to bytes
func exportPubKey(pub *ecies.PublicKey) []byte {
	if pub == nil {
		panic("nil public key")
	}
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)[1:]
}

// write send message to remote
func write(conn io.ReadWriter, encBuf []byte) error {
	// length
	length := make([]byte, PackageLength)
	binary.BigEndian.PutUint32(length, uint32(len(encBuf)))
	// header
	buffer := append(PackagePrefix, length...)
	// header and body
	buffer = append(buffer, encBuf...)
	// write
	if _, err := conn.Write(buffer); err != nil {
		return err
	}
	return nil
}
