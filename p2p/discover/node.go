//build the dpnode in P2P

package discover

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"p3Chain/common"
	"p3Chain/crypto"
	"strconv"
	"strings"

	"p3Chain/crypto/secp256k1"
)

// Node is the basic element in the p2p network.
type Node struct {
	IP  net.IP // len 4 for IPv4 of 16 for IPv6
	UDP uint16 // UDP port number
	TCP uint16 // TCP port number
	ID  NodeID // the node's public key (ECC public key marshalled)

	// This is a cached copy of sha3(ID) which is used for node
	// distance calculations. This is part of Node in order to make it
	// possible to write tests that need a node at a certain distance.
	// In those tests, the content of sha will not actually correspond
	// with ID.
	sha common.Hash //NodeID的哈希值
}

type NodeID common.NodeID

// NodeID prints as a long hexadecimal number.
func (n NodeID) String() string {
	return fmt.Sprintf("%x", n[:]) //以字符串形式返回NodeID
}

// 创建一个新的Kad节点
func newNode(id NodeID, ip net.IP, udpPort, tcpPort uint16) *Node {
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4 //将IP地址转化为点分十进制
	}
	selfSha := crypto.Sha3Hash(id[:]) //计算NodeID的哈希值(SHA3)

	return &Node{
		IP:  ip,
		UDP: udpPort,
		TCP: tcpPort,
		ID:  id,
		sha: selfSha,
	}
}

// 计算并赋值该NodeID的哈希值
func (n *Node) Sha() {
	n.sha = crypto.Sha3Hash(n.ID[:])
}

// 返回UDP端口+IP地址
func (n *Node) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.IP, Port: int(n.UDP)}
}

// The string of Node back the url of a node in string type.
// 返回该Node的字符串形式url ：
//
//	scheme://userinfo@host/path?query#fragment
//
// -->  dpnode://User@Host?discport=n.UDP
func (n *Node) String() string {
	addr := net.TCPAddr{IP: n.IP, Port: int(n.TCP)} //获取Node的TCP地址
	u := url.URL{
		Scheme: "dpnode",
		User:   url.User(fmt.Sprintf("%x", n.ID[:])), //NodeID
		Host:   addr.String(),                        //TCP地址
	}
	if n.UDP != n.TCP {
		u.RawQuery = "discport=" + strconv.Itoa(int(n.UDP)) //discport means discovery port
	}
	return u.String()
}

// ParseNode parsed a node URL and back a node.
// The scheme of a node URL is "dpnode"
//
// The hexadecimal node ID is encoded in the username portion of the
// URL, separated from the host by an @ sign. The hostname can only be
// given as an IP address, DNS domain names are not allowed. The port
// in the host name section is the TCP listening port. If the TCP and
// UDP (discovery) ports differ, the UDP port is specified as query
// parameter "discport".
//
//如果TCP和UDP端口不一致，UDP端口被指定为查询参数“discport”

// In the following example, the node URL describes
// a node with IP address 10.3.58.6, TCP listening port 30303
// and UDP discovery port 30301.
//
//    dpnode://<hex node id>@10.3.58.6:30303?discport=30301

// 对字符串形式的url进行解析(取出各部分：NodeID , IP , UDP Port , TCP Port)
func ParseNode(rawurl string) (*Node, error) {
	var (
		id      NodeID
		ip      net.IP
		tcpPort uint64
		udpPort uint64
	)
	u, err := url.Parse(rawurl) //u结构体即为对字符串url解析后的结果
	if err != nil {
		return nil, err
	}

	if u.Scheme != "dpnode" {
		return nil, errors.New("invalid URL scheme, want \"dpnode\"")
	}
	// Parse the Node ID from the user portion.
	if u.User == nil {
		return nil, errors.New("does not contain node ID")
	}
	if id, err = HexID(u.User.String()); err != nil {
		return nil, fmt.Errorf("invalid node ID (%v)", err)
	}
	// Parse the IP address.
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %v", err)
	}
	if ip = net.ParseIP(host); ip == nil {
		return nil, errors.New("invalid IP address")
	}
	// Ensure the IP is 4 bytes long for IPv4 addresses.
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	// Parse the port numbers.
	if tcpPort, err = strconv.ParseUint(port, 10, 16); err != nil {
		return nil, errors.New("invalid port")
	}
	udpPort = tcpPort
	qv := u.Query()               //获取并解析出u的RawQuery字段
	if qv.Get("discport") != "" { //获取discport之后的UDP端口号
		udpPort, err = strconv.ParseUint(qv.Get("discport"), 10, 16)
		if err != nil {
			return nil, errors.New("invalid discport in query")
		}
	}
	return newNode(id, ip, uint16(udpPort), uint16(tcpPort)), nil

}

// MustParseNode parses a node URL. It panics if the URL is not valid.
func MustParseNode(rawurl string) *Node {
	n, err := ParseNode(rawurl)
	if err != nil {
		panic("invalid node URL: " + err.Error()) //当解析出错时，发出panic警报
	}
	return n
}

// HexID converts a hex string to a NodeID.
// The string may be prefixed with 0x.

// 将一个十六进制字符串转换为NodeID
func HexID(in string) (NodeID, error) {
	if strings.HasPrefix(in, "0x") {
		in = in[2:]
	}
	var id NodeID
	b, err := hex.DecodeString(in) //将字符串按照十六进制格式进行解析
	if err != nil {
		return id, err
	} else if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, need %d hex bytes", len(id))
	}
	copy(id[:], b) //将转换后的b复制到id中，作为返回值
	return id, nil
}

// MustHexID converts a hex string to a NodeID.
// It panics if the string is not a valid NodeID.
func MustHexID(in string) NodeID {
	id, err := HexID(in)
	if err != nil {
		panic(err)
	}
	return id
}

// PubkeyID returns a marshaled representation of the given public key.

// 传入参数是椭圆曲线签名算法加密的公钥
func PubkeyID(pub *ecdsa.PublicKey) NodeID {
	var id NodeID
	pbytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	if len(pbytes)-1 != len(id) {
		panic(fmt.Errorf("need %d bit pubkey, got %d bits", (len(id)+1)*8, len(pbytes)))
	}
	copy(id[:], pbytes[1:]) //将序列化的pub作为NodeID
	return id
}

// Pubkey returns the public key represented by the node ID.
// It returns an error if the ID is not a point on the curve.

// 用NodeID创建一个椭圆曲线结构体公钥
func (id NodeID) Pubkey() (*ecdsa.PublicKey, error) {
	return crypto.NodeIDtoKey(common.NodeID(id))
}

// recoverNodeID computes the public key used to sign the
// given hash from the signature.
// 通过数字签名与消息的哈希计算出相应的公钥
//
// right now it is not important!
func recoverNodeID(hash, sig []byte) (id NodeID, err error) {
	pubkey, err := secp256k1.RecoverPubkey(hash, sig) //RecoverPubkey返回签名者的公钥
	if err != nil {
		return id, err
	}
	if len(pubkey)-1 != len(id) {
		return id, fmt.Errorf("recovered pubkey has %d bits, want %d bits", len(pubkey)*8, (len(id)+1)*8)
	}
	for i := range id { //公钥作为查找的NodeID
		id[i] = pubkey[i+1]
	}
	return id, nil
}

// distcmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.

// 分别比较目标target与a和b的距离
// 如果a距离目标更近就返回-1  如果b距离目标更近就返回1  如果距离相等就返回0
// 距离的计算是通过异或比较哈希值
// common.Hash是byte数组
func distcmp(target, a, b common.Hash) int {
	for i := range target {
		da := a[i] ^ target[i] //不是bit进行异或，而是byte之间进行异或
		db := b[i] ^ target[i]
		if da > db { //第一次出现不相等时就会退出
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

// table of leading zero counts for bytes [0..255]
var lzcount = [256]int{
	8, 7, 6, 6, 5, 5, 5, 5,
	4, 4, 4, 4, 4, 4, 4, 4,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

// logdist returns the logarithmic distance between a and b, log2(a ^ b).
// the distance is the number of prefix zero after XOR.

// 返回a与b的对数距离 --> log2(a ^ b)
// 这个距离是指异或后的前缀0的个数
func logdist(a, b common.Hash) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 { //1Byte的a[i]与b[i]相等
			lz += 8 //8bit相等，即是说此次异或结果中有8个前缀0
		} else {
			lz += lzcount[x] //根据异或的结果将此次结果的前缀0个数追加到lz上 (异或结果越大，前缀0数目越少)
			//lzcount[0] = 8  -->意味着异或结果为0 ,存在8个前缀0
			break //一旦a与b出现不等就会退出循环
		}
	}
	return len(a)*8 - lz //返回a与b不相等的bit位数
}

// hashAtDistance returns a random hash such that logdist(a, b) == n
// 本函数返回一个满足 logdist(a, b) == n 的随机哈希值 b (哈希值a与距离n是已知的)
func hashAtDistance(a common.Hash, n int) (b common.Hash) {
	//距离为0，b就是a
	if n == 0 {
		return a
	}
	// flip bit at position n, fill the rest with random bits
	b = a
	pos := len(a) - n/8 - 1 //确定出a与b是在哪一byte上是不等的
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit // TODO: randomize end bits
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}
