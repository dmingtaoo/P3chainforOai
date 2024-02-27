package server

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"sync"
	"time"

	"p3Chain/crypto"
	"p3Chain/crypto/ecies"
	"p3Chain/crypto/secp256k1"
	"p3Chain/crypto/sha3"
	"p3Chain/p2p/discover"
	"p3Chain/rlp"
)

const (
	maxUint24 = ^uint32(0) >> 8

	sskLen = 16 // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen = 65 // elliptic S256
	pubLen = 64 // 512 bit pubkey in uncompressed representation without format byte
	shaLen = 32 // hash length (for nonce etc)

	authMsgLen  = sigLen + shaLen + pubLen + shaLen + 1
	authRespLen = pubLen + shaLen + 1

	eciesBytes     = 65 + 16 + 32
	encAuthMsgLen  = authMsgLen + eciesBytes  // size of the final ECIES payload sent as initiator's handshake
	encAuthRespLen = authRespLen + eciesBytes // size of the final ECIES payload sent as receiver's handshake

	// total timeout for encryption handshake and protocol
	// handshake in both directions.
	handshakeTimeout = 5 * time.Second

	// This is the timeout for sending the disconnect reason.
	// This is shorter than the usual timeout because we don't want
	// to wait if the connection is known to be bad anyway.
	discWriteTimeout = 1 * time.Second
)

// rlpx is the transport protocol used by actual (non-test) connections.
// It wraps the frame encoder with locks and read/write deadlines.
type rlpx struct {
	fd net.Conn

	rmu, wmu sync.Mutex
	rw       *rlpxFrameRW
}

// 根据conn socket 返回一个transport接口
func newRLPX(fd net.Conn) transport {
	fd.SetDeadline(time.Now().Add(handshakeTimeout)) //设置handshake的超时时间
	return &rlpx{fd: fd}                             //rlpx类型实现了transport接口
}

// rlpx实现transport接口的ReadMsg()
func (t *rlpx) ReadMsg() (Msg, error) {
	t.rmu.Lock()
	defer t.rmu.Unlock()
	t.fd.SetReadDeadline(time.Now().Add(frameReadTimeout)) //设置帧读取超时时间
	return t.rw.ReadMsg()                                  //底层用rlpxFrameRW结构体实现ReadMsg()
}

// rlpx实现transport接口的WriteMsg()
func (t *rlpx) WriteMsg(msg Msg) error {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	t.fd.SetWriteDeadline(time.Now().Add(frameWriteTimeout)) //设置帧发送超时时间
	return t.rw.WriteMsg(msg)                                //底层用rlpxFrameRW结构体实现WriteMsg()
}

// rlpx实现transport接口的Close()
func (t *rlpx) close(err error) {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	// Tell the remote end why we're disconnecting if possible.
	// 向远端peer发送断开连接的原因
	if t.rw != nil {
		if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
			t.fd.SetWriteDeadline(time.Now().Add(discWriteTimeout))
			SendItems(t.rw, discMsg, r) //消息类型为discMsg ,内容为r
		}
	}
	t.fd.Close() //关闭连接socket
}

// doProtoHandshake完成与对端节点的handshake操作
func (t *rlpx) doProtoHandshake(our *protoHandshake) (their *protoHandshake, err error) {
	// Writing our handshake happens concurrently, we prefer
	// returning the handshake read error. If the remote side
	// disconnects us early with a valid reason, we should return it
	// as the error so it can be tracked elsewhere.
	werr := make(chan error, 1)
	go func() { werr <- Send(t.rw, handshakeMsg, our) }()          //协程负责发送handshake消息
	if their, err = readProtocolHandshake(t.rw, our); err != nil { //等待读取对方的handshake回复
		<-werr // make sure the write terminates too  确保写协程结束
		return nil, err
	}
	if err := <-werr; err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}
	return their, nil
}

// 读取对端节点handshake回复包，同时检验其是否符合标准(是否完成handshake操作)
func readProtocolHandshake(rw MsgReader, our *protoHandshake) (*protoHandshake, error) {
	msg, err := rw.ReadMsg() //读取对方的回复
	if err != nil {          //读取错误
		return nil, err
	}
	if msg.Size > baseProtocolMaxMsgSize { //msg消息过大
		return nil, fmt.Errorf("message too big")
	}
	if msg.Code == discMsg { //msg类型为discMsg，返回错误原因reason
		// Disconnect before protocol handshake is valid according to the
		// spec and we send it ourself if the posthanshake checks fail.
		// We can't return the reason directly, though, because it is echoed
		// back otherwise. Wrap it in a string instead.
		var reason [1]DiscReason
		rlp.Decode(msg.Payload, &reason)
		return nil, reason[0]
	}
	if msg.Code != handshakeMsg { //msg类型不为handshakeMsg,返回错误提示
		return nil, fmt.Errorf("expected handshake, got %x", msg.Code)
	}
	//以下为收到msg消息类型为handshakeMsg的处理情况
	var hs protoHandshake
	if err := msg.Decode(&hs); err != nil { //将rlp流格式的msg解析到hs结构体中
		return nil, err
	}
	// validate handshake info
	if hs.Version != our.Version { //判断协议类型是否正确
		return nil, DiscIncompatibleVersion
	}
	if (hs.ID == discover.NodeID{}) { //判断节点ID是否为空
		return nil, DiscInvalidIdentity
	}
	return &hs, nil
}

// doEncHandshake runs the protocol handshake using authenticated
// messages. the protocol handshake is the first authenticated message
// and also verifies whether the encryption handshake 'worked' and the
// remote side actually provided the right public key.
// prv为本地节点的私钥, dial为对端节点的Node结构体
func (t *rlpx) doEncHandshake(prv *ecdsa.PrivateKey, dial *discover.Node) (discover.NodeID, error) {
	var (
		sec secrets
		err error
	)
	if dial == nil { //对端节点Node结构体为空
		sec, err = receiverEncHandshake(t.fd, prv, nil) //等待接收加密的handshake消息
	} else { //Node结构体不为空,约定加密通信需要使用的密钥,建立加密信道
		sec, err = initiatorEncHandshake(t.fd, prv, dial.ID, nil)
	}
	if err != nil {
		return discover.NodeID{}, err
	}
	t.wmu.Lock()
	t.rw = newRLPXFrameRW(t.fd, sec)
	t.wmu.Unlock()
	return sec.RemoteID, nil //返回对端节点的NodeID
}

// encHandshake contains the state of the encryption handshake.
type encHandshake struct {
	initiator bool
	remoteID  discover.NodeID

	remotePub            *ecies.PublicKey  // remote-pubk
	initNonce, respNonce []byte            // nonce
	randomPrivKey        *ecies.PrivateKey // ecdhe-random
	remoteRandomPub      *ecies.PublicKey  // ecdhe-random-pubk
}

// secrets represents the connection secrets
// which are negotiated during the encryption handshake.
type secrets struct {
	RemoteID              discover.NodeID
	AES, MAC              []byte
	EgressMAC, IngressMAC hash.Hash
	Token                 []byte
}

// secrets is called after the handshake is completed.
// It extracts the connection secrets from the handshake values.
func (h *encHandshake) secrets(auth, authResp []byte) (secrets, error) {
	ecdheSecret, err := h.randomPrivKey.GenerateShared(h.remoteRandomPub, sskLen, sskLen)
	if err != nil {
		return secrets{}, err
	}

	// derive base secrets from ephemeral key agreement
	sharedSecret := crypto.Sha3(ecdheSecret, crypto.Sha3(h.respNonce, h.initNonce))
	aesSecret := crypto.Sha3(ecdheSecret, sharedSecret)
	s := secrets{
		RemoteID: h.remoteID,
		AES:      aesSecret,
		MAC:      crypto.Sha3(ecdheSecret, aesSecret),
		Token:    crypto.Sha3(sharedSecret),
	}

	// setup sha3 instances for the MACs
	mac1 := sha3.NewKeccak256()
	mac1.Write(xor(s.MAC, h.respNonce))
	mac1.Write(auth)
	mac2 := sha3.NewKeccak256()
	mac2.Write(xor(s.MAC, h.initNonce))
	mac2.Write(authResp)
	if h.initiator {
		s.EgressMAC, s.IngressMAC = mac1, mac2
	} else {
		s.EgressMAC, s.IngressMAC = mac2, mac1
	}

	return s, nil
}

// 由本地私钥和远端节点的公钥生成共享密钥
func (h *encHandshake) ecdhShared(prv *ecdsa.PrivateKey) ([]byte, error) {
	return ecies.ImportECDSA(prv).GenerateShared(h.remotePub, sskLen, sskLen)
}

// initiatorEncHandshake negotiates a session token on conn.
// it should be called on the dialing side of the connection.
//
// prv is the local client's private key.
// token is the token from a previous session with this node.

// initiatorEncHandshake 负责在两个节点的conn上协商会话token,该token将会在连接的拨号端被使用
// prv是本地客户端的私钥
// token来自于本节点的上一个会话
func initiatorEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, remoteID discover.NodeID, token []byte) (s secrets, err error) {
	h, err := newInitiatorHandshake(remoteID)
	if err != nil {
		return s, err
	}
	auth, err := h.authMsg(prv, token)
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(auth); err != nil {
		return s, err
	}

	response := make([]byte, encAuthRespLen)
	if _, err = io.ReadFull(conn, response); err != nil {
		return s, err
	}
	if err := h.decodeAuthResp(response, prv); err != nil {
		return s, err
	}
	return h.secrets(auth, response)
}

func newInitiatorHandshake(remoteID discover.NodeID) (*encHandshake, error) {
	// generate random initiator nonce
	n := make([]byte, shaLen)
	if _, err := rand.Read(n); err != nil {
		return nil, err
	}
	// generate random keypair to use for signing
	randpriv, err := ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
	if err != nil {
		return nil, err
	}
	rpub, err := remoteID.Pubkey()
	if err != nil {
		return nil, fmt.Errorf("bad remoteID: %v", err)
	}
	h := &encHandshake{
		initiator:     true,
		remoteID:      remoteID,
		remotePub:     ecies.ImportECDSAPublic(rpub),
		initNonce:     n,
		randomPrivKey: randpriv,
	}
	return h, nil
}

// authMsg creates an encrypted initiator handshake message.
// 创建一个加密的初始化加密消息
func (h *encHandshake) authMsg(prv *ecdsa.PrivateKey, token []byte) ([]byte, error) {
	var tokenFlag byte
	if token == nil {
		// no session token found means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers
		// generate shared key from prv and remote pubkey
		//未找到会话token意味着我们需要生成共享密钥。由这个共享密钥作为对等方的初始会话令牌
		var err error
		if token, err = h.ecdhShared(prv); err != nil { //token存储共享密钥
			return nil, err
		}
	} else { //存在token
		// for known peers, we use stored token from the previous session
		tokenFlag = 0x01
	}

	// sign known message:
	//   ecdh-shared-secret^nonce for new peers
	//   token^nonce for old peers
	signed := xor(token, h.initNonce)
	signature, err := crypto.Sign(signed, h.randomPrivKey.ExportECDSA()) //私钥加密获取数字签名
	if err != nil {
		return nil, err
	}

	// encode auth message
	// signature || sha3(ecdhe-random-pubk) || pubk || nonce || token-flag
	msg := make([]byte, authMsgLen)
	n := copy(msg, signature)
	n += copy(msg[n:], crypto.Sha3(exportPubkey(&h.randomPrivKey.PublicKey)))
	n += copy(msg[n:], crypto.FromECDSAPub(&prv.PublicKey)[1:])
	n += copy(msg[n:], h.initNonce)
	msg[n] = tokenFlag

	// encrypt auth message using remote-pubk
	return ecies.Encrypt(rand.Reader, h.remotePub, msg, nil, nil)
}

// decodeAuthResp decode an encrypted authentication response message.
func (h *encHandshake) decodeAuthResp(auth []byte, prv *ecdsa.PrivateKey) error {
	msg, err := crypto.Decrypt(prv, auth)
	if err != nil {
		return fmt.Errorf("could not decrypt auth response (%v)", err)
	}
	h.respNonce = msg[pubLen : pubLen+shaLen]
	h.remoteRandomPub, err = importPublicKey(msg[:pubLen])
	if err != nil {
		return err
	}
	// ignore token flag for now
	return nil
}

// receiverEncHandshake negotiates a session token on conn.
// it should be called on the listening side of the connection.
//
// prv is the local client's private key.   prv是本地客户端的私钥
// token is the token from a previous session with this node.  token来自于此节点的上一个会话
func receiverEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, token []byte) (s secrets, err error) {
	// read remote auth sent by initiator.
	auth := make([]byte, encAuthMsgLen)
	if _, err := io.ReadFull(conn, auth); err != nil { //读取授权消息
		return s, err
	}
	h, err := decodeAuthMsg(prv, token, auth)
	if err != nil {
		return s, err
	}

	// send auth response
	resp, err := h.authResp(prv, token)
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(resp); err != nil {
		return s, err
	}

	return h.secrets(auth, resp)
}

func decodeAuthMsg(prv *ecdsa.PrivateKey, token []byte, auth []byte) (*encHandshake, error) {
	var err error
	h := new(encHandshake)
	// generate random keypair for session
	h.randomPrivKey, err = ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
	if err != nil {
		return nil, err
	}
	// generate random nonce
	h.respNonce = make([]byte, shaLen)
	if _, err = rand.Read(h.respNonce); err != nil {
		return nil, err
	}

	msg, err := crypto.Decrypt(prv, auth)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt auth message (%v)", err)
	}

	// decode message parameters
	// signature || sha3(ecdhe-random-pubk) || pubk || nonce || token-flag
	h.initNonce = msg[authMsgLen-shaLen-1 : authMsgLen-1]
	copy(h.remoteID[:], msg[sigLen+shaLen:sigLen+shaLen+pubLen])
	rpub, err := h.remoteID.Pubkey()
	if err != nil {
		return nil, fmt.Errorf("bad remoteID: %#v", err)
	}
	h.remotePub = ecies.ImportECDSAPublic(rpub)

	// recover remote random pubkey from signed message.
	if token == nil {
		// TODO: it is an error if the initiator has a token and we don't. check that.

		// no session token means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers.
		// generate shared key from prv and remote pubkey.
		if token, err = h.ecdhShared(prv); err != nil {
			return nil, err
		}
	}
	signedMsg := xor(token, h.initNonce)
	remoteRandomPub, err := secp256k1.RecoverPubkey(signedMsg, msg[:sigLen])
	if err != nil {
		return nil, err
	}
	h.remoteRandomPub, _ = importPublicKey(remoteRandomPub)
	return h, nil
}

// authResp generates the encrypted authentication response message.
func (h *encHandshake) authResp(prv *ecdsa.PrivateKey, token []byte) ([]byte, error) {
	// responder auth message
	// E(remote-pubk, ecdhe-random-pubk || nonce || 0x0)
	resp := make([]byte, authRespLen)
	n := copy(resp, exportPubkey(&h.randomPrivKey.PublicKey))
	n += copy(resp[n:], h.respNonce)
	if token == nil {
		resp[n] = 0
	} else {
		resp[n] = 1
	}
	// encrypt using remote-pubk
	return ecies.Encrypt(rand.Reader, h.remotePub, resp, nil, nil)
}

// importPublicKey unmarshals 512 bit public keys.
func importPublicKey(pubKey []byte) (*ecies.PublicKey, error) {
	var pubKey65 []byte
	switch len(pubKey) {
	case 64:
		// add 'uncompressed key' flag
		pubKey65 = append([]byte{0x04}, pubKey...)
	case 65:
		pubKey65 = pubKey
	default:
		return nil, fmt.Errorf("invalid public key length %v (expect 64/65)", len(pubKey))
	}
	// TODO: fewer pointless conversions
	return ecies.ImportECDSAPublic(crypto.ToECDSAPub(pubKey65)), nil
}

func exportPubkey(pub *ecies.PublicKey) []byte {
	if pub == nil {
		panic("nil pubkey")
	}
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)[1:]
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}

var (
	// this is used in place of actual frame header data.
	// TODO: replace this when Msg contains the protocol type code.
	zeroHeader = []byte{0xC2, 0x80, 0x80}
	// sixteen zero bytes
	zero16 = make([]byte, 16)
)

// rlpxFrameRW implements a simplified version of RLPx framing.
// chunked messages are not supported and all headers are equal to
// zeroHeader.
//
// rlpxFrameRW is not safe for concurrent use from multiple goroutines.
type rlpxFrameRW struct {
	conn io.ReadWriter //读写器接口
	enc  cipher.Stream
	dec  cipher.Stream

	macCipher  cipher.Block
	egressMAC  hash.Hash
	ingressMAC hash.Hash
}

func newRLPXFrameRW(conn io.ReadWriter, s secrets) *rlpxFrameRW {
	macc, err := aes.NewCipher(s.MAC)
	if err != nil {
		panic("invalid MAC secret: " + err.Error())
	}
	encc, err := aes.NewCipher(s.AES)
	if err != nil {
		panic("invalid AES secret: " + err.Error())
	}
	// we use an all-zeroes IV for AES because the key used
	// for encryption is ephemeral.
	iv := make([]byte, encc.BlockSize())
	return &rlpxFrameRW{
		conn:       conn,
		enc:        cipher.NewCTR(encc, iv),
		dec:        cipher.NewCTR(encc, iv),
		macCipher:  macc,
		egressMAC:  s.EgressMAC,
		ingressMAC: s.IngressMAC,
	}
}

// 发送Msg消息
func (rw *rlpxFrameRW) WriteMsg(msg Msg) error {
	ptype, _ := rlp.EncodeToBytes(msg.Code) //msg.Code进行rlp编码

	// write header
	headbuf := make([]byte, 32)
	fsize := uint32(len(ptype)) + msg.Size
	if fsize > maxUint24 {
		return errors.New("message size overflows uint24")
	}
	putInt24(fsize, headbuf) // TODO: check overflow
	copy(headbuf[3:], zeroHeader)
	rw.enc.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now encrypted

	// write header MAC
	copy(headbuf[16:], updateMAC(rw.egressMAC, rw.macCipher, headbuf[:16]))
	if _, err := rw.conn.Write(headbuf); err != nil {
		return err
	}

	// write encrypted frame, updating the egress MAC hash with
	// the data written to conn.
	tee := cipher.StreamWriter{S: rw.enc, W: io.MultiWriter(rw.conn, rw.egressMAC)}
	if _, err := tee.Write(ptype); err != nil {
		return err
	}
	if _, err := io.Copy(tee, msg.Payload); err != nil {
		return err
	}
	if padding := fsize % 16; padding > 0 {
		if _, err := tee.Write(zero16[:16-padding]); err != nil {
			return err
		}
	}

	// write frame MAC. egress MAC hash is up to date because
	// frame content was written to it as well.
	fmacseed := rw.egressMAC.Sum(nil)
	mac := updateMAC(rw.egressMAC, rw.macCipher, fmacseed)
	_, err := rw.conn.Write(mac)
	return err
}

// 接收Msg消息
func (rw *rlpxFrameRW) ReadMsg() (msg Msg, err error) {
	// read the header
	headbuf := make([]byte, 32)
	if _, err := io.ReadFull(rw.conn, headbuf); err != nil {
		return msg, err
	}
	// verify header mac
	shouldMAC := updateMAC(rw.ingressMAC, rw.macCipher, headbuf[:16])
	if !hmac.Equal(shouldMAC, headbuf[16:]) {
		return msg, errors.New("bad header MAC")
	}
	rw.dec.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now decrypted
	fsize := readInt24(headbuf)
	// ignore protocol type for now

	// read the frame content
	var rsize = fsize // frame size rounded up to 16 byte boundary
	if padding := fsize % 16; padding > 0 {
		rsize += 16 - padding
	}
	framebuf := make([]byte, rsize)
	if _, err := io.ReadFull(rw.conn, framebuf); err != nil {
		return msg, err
	}

	// read and validate frame MAC. we can re-use headbuf for that.
	rw.ingressMAC.Write(framebuf)
	fmacseed := rw.ingressMAC.Sum(nil)
	if _, err := io.ReadFull(rw.conn, headbuf[:16]); err != nil {
		return msg, err
	}
	shouldMAC = updateMAC(rw.ingressMAC, rw.macCipher, fmacseed)
	if !hmac.Equal(shouldMAC, headbuf[:16]) {
		return msg, errors.New("bad frame MAC")
	}

	// decrypt frame content
	rw.dec.XORKeyStream(framebuf, framebuf)

	// decode message code
	content := bytes.NewReader(framebuf[:fsize])
	if err := rlp.Decode(content, &msg.Code); err != nil {
		return msg, err
	}
	msg.Size = uint32(content.Len())
	msg.Payload = content
	return msg, nil
}

// updateMAC reseeds the given hash with encrypted seed.
// it returns the first 16 bytes of the hash sum after seeding.
func updateMAC(mac hash.Hash, block cipher.Block, seed []byte) []byte {
	aesbuf := make([]byte, aes.BlockSize)
	block.Encrypt(aesbuf, mac.Sum(nil))
	for i := range aesbuf {
		aesbuf[i] ^= seed[i]
	}
	mac.Write(aesbuf)
	return mac.Sum(nil)[:16]
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putInt24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}
