// realize the kad protocol on udp
package discover

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"p3Chain/crypto"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/nat"
	"p3Chain/rlp"
	"time"
)

const Version = 4

var PUBLIC_IP net.IP //新增,专门存储外部环境获取的己方设备的公网IP地址

// Errors
var (
	errPacketTooSmall   = errors.New("too small")
	errBadHash          = errors.New("bad hash")
	errExpired          = errors.New("expired")
	errBadVersion       = errors.New("version mismatch")
	errUnsolicitedReply = errors.New("unsolicited reply")
	errUnknownNode      = errors.New("unknown node")
	errTimeout          = errors.New("RPC timeout")
	errClosed           = errors.New("socket closed")
)

// Timeouts
const (
	respTimeout = 500 * time.Millisecond //回复超时时间为500ms
	sendTimeout = 500 * time.Millisecond //发送超时时间为500ms
	expiration  = 20 * time.Second       //截止时间为20s

	refreshInterval = 1 * time.Hour //刷新时间为1h
)

// RPC packet types  四类操作
const (
	pingPacket = iota + 1 // zero is 'reserved'
	pongPacket
	findnodePacket
	neighborsPacket
)

// RPC request structures
type (
	ping struct {
		Version    uint
		From, To   rpcEndpoint //ping操作需要同时包含源端与目的端地址
		Expiration uint64      //此消息的存活时间
	}

	// pong is the reply to ping.
	pong struct {
		// This field should mirror the UDP envelope address
		// of the ping packet, which provides a way to discover the
		// the external address (after NAT).
		To rpcEndpoint

		ReplyTok   []byte // This contains the hash of the ping packet.  ping消息的哈希值
		Expiration uint64 // Absolute timestamp at which the packet becomes invalid.
	}

	// findnode is a query for nodes close to the given target.
	findnode struct {
		Target     NodeID // doesn't need to be an actual public key
		Expiration uint64
	}

	// reply to findnode
	neighbors struct {
		Nodes      []rpcNode
		Expiration uint64
	}

	rpcNode struct {
		IP  net.IP // len 4 for IPv4 or 16 for IPv6
		UDP uint16 // for discovery protocoll'l'l'l'l'l'l'l'l'l'l'l'l'l'l'l'l'l'l
		TCP uint16 // for RLPx protocol
		ID  NodeID
	}

	rpcEndpoint struct {
		IP  net.IP // len 4 for IPv4 or 16 for IPv6
		UDP uint16 // for discovery protocol
		TCP uint16 // for RLPx protocol
	}
)

// 利用UDP地址、TCP端口号创建一个新的rpcEndpoint结构体
func makeEndpoint(addr *net.UDPAddr, tcpPort uint16) rpcEndpoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return rpcEndpoint{IP: ip, UDP: uint16(addr.Port), TCP: tcpPort}
}

// 判断传入的节点是否为有效节点(检查IP地址是否有效)
// rpcNode类型转换为Node类型
func nodeFromRPC(rn rpcNode) (n *Node, valid bool) {
	// TODO: don't accept localhost, LAN addresses from internet hosts
	// TODO: check public key is on secp256k1 curve
	if rn.IP.IsMulticast() || rn.IP.IsUnspecified() || rn.UDP == 0 {
		return nil, false
	}
	return newNode(rn.ID, rn.IP, rn.UDP, rn.TCP), true
}

// Node类型转化为rpcNode类型
func nodeToRPC(n *Node) rpcNode {
	return rpcNode{ID: n.ID, IP: n.IP, UDP: n.UDP, TCP: n.TCP}
}

type packet interface {
	handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error
}

type conn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

// udp implements the RPC protocol.
type udp struct {
	conn        conn              //接收udp数据包的IP:PORT对应的socket
	priv        *ecdsa.PrivateKey //私钥
	ourEndpoint rpcEndpoint       //本地端点地址

	addpending chan *pending //等待处理管道
	gotreply   chan reply    //相应管道

	closing chan struct{}
	nat     nat.Interface

	*Table
}

// pending represents a pending reply.
//
// some implementations of the protocol wish to send more than one
// reply packet to findnode. in general, any neighbors packet cannot
// be matched up with a specific findnode packet.
//该协议的一些实现希望向findnode发送多个应答包。
//通常，任何邻居数据包都无法与特定findnode数据包匹配。
//
// our implementation handles this by storing a callback function for
// each pending reply. incoming packets from a node are dispatched
// to all the callback functions for that node.
//通过为每个挂起的回复存储回调函数来处理这个问题。
//来自节点的传入数据包被调度到该节点的所有回调函数。

type pending struct {
	// these fields must match in the reply.
	from  NodeID
	ptype byte

	// time when the request must complete
	//在截止时间到达之间，必须完成回复
	deadline time.Time

	// callback is called when a matching reply arrives. if it returns
	// true, the callback is removed from the pending reply queue.
	// if it returns false, the reply is considered incomplete and
	// the callback will be invoked again for the next matching reply.
	//当相对应的reply到达时，它会被挂起并添加一个回调函数负责处理
	//当回调函数返回true时，回调函数被清除出挂起回复队列
	//当返回false时，这个reply会被认为是未完成的而且会被重新唤醒等待下一个匹配的reply
	callback func(resp interface{}) (done bool)

	// errc receives nil when the callback indicates completion or an
	// error if no further reply is received within the timeout.
	//单向管道，只能写入error类型的数据
	//此管道若收到nil,表明callback完成
	//若收到error,表明在超时时间内没有收到reply
	errc chan<- error //only write channel
}

type reply struct {
	from  NodeID
	ptype byte
	data  interface{}
	// loop indicates whether there was
	// a matching request by sending on this channel.
	matched chan<- bool
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
// ListenUDP  根据 配置文件返回一个新 table，该table 在laddr上侦听UDP数据包
// 根据传入的UDP地址创建一个新的Kad路由表，包括本节点的UDP通信socket、私钥、nat、数据库路径
func ListenUDP(priv *ecdsa.PrivateKey, laddr string, natm nat.Interface, nodeDBPath string) (*Table, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr) //返回一个UDP网络地址
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr) //监听addr上的连接请求，并返回conn (并非建立了连接，而是为了接受消息时读取方便才产生一个conn)
	if err != nil {
		return nil, err
	}
	//将接收消息的conn与私钥、nat映射地址、数据库地址关联
	tab, _ := newUDP(priv, conn, natm, nodeDBPath)
	return tab, nil
}

// Giving the listening address and private key, back a parsed node
// 根据给定的监听地址和私钥，返回一个新的Node
func ParseUDP(priv *ecdsa.PrivateKey, laddr string) (*Node, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr) //返回一个UDP地址
	if err != nil {
		return nil, err
	}
	nodeID := PubkeyID(&priv.PublicKey)
	node := newNode(nodeID, addr.IP, uint16(addr.Port), uint16(addr.Port))
	return node, nil
}

// 重点是获取NAT映射的公网地址
func newUDP(priv *ecdsa.PrivateKey, c conn, natm nat.Interface, nodeDBPath string) (*Table, *udp) {
	udp := &udp{
		conn:       c,
		priv:       priv,
		closing:    make(chan struct{}),
		gotreply:   make(chan reply),
		addpending: make(chan *pending),
	}
	realaddr := c.LocalAddr().(*net.UDPAddr) //绑定的socket conn对应的本地UDP地址
	if natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(natm, udp.closing, "udp", realaddr.Port, realaddr.Port, "ethereum discovery")
		}
		//time.Sleep(100 * time.Millisecond)
		// TODO: react to external IP changes over time.
		if ext, err := natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}
	// TODO: separate TCP port
	udp.ourEndpoint = makeEndpoint(realaddr, uint16(realaddr.Port))
	udp.Table = newTable(udp, PubkeyID(&priv.PublicKey), realaddr, nodeDBPath)
	go udp.loop()
	go udp.readLoop()
	return udp.Table, udp
}

func (t *udp) close() {
	close(t.closing)
	t.conn.Close()
	// TODO: wait for the loops to end.
}

// ping sends a ping message to the given node and waits for a reply.
func (t *udp) ping(toid NodeID, toaddr *net.UDPAddr) error {
	// TODO: maybe check for ReplyTo field in callback to measure RTT
	errc := t.pending(toid, pongPacket, func(interface{}) bool { return true })
	t.send(toaddr, pingPacket, ping{
		Version:    Version,
		From:       t.ourEndpoint,
		To:         makeEndpoint(toaddr, 0), // TODO: maybe use known TCP port from DB
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	return <-errc
}

func (t *udp) waitping(from NodeID) error {
	return <-t.pending(from, pingPacket, func(interface{}) bool { return true })
}

// findnode sends a findnode request to the given node and waits until
// the node has sent up to k neighbors.
// findnode方法发送findnode请求给指定的节点，然后等待此节点将它的k个邻居发送完成
func (t *udp) findnode(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	//创建一个bucketSize大小的切片
	nodes := make([]*Node, 0, bucketSize)
	nreceived := 0
	errc := t.pending(toid, neighborsPacket, func(r interface{}) bool {
		reply := r.(*neighbors) //获取指定节点toid的邻居节点
		for _, rn := range reply.Nodes {
			nreceived++
			if n, valid := nodeFromRPC(rn); valid {
				nodes = append(nodes, n)
			}
		}
		return nreceived >= bucketSize
	})
	t.send(toaddr, findnodePacket, findnode{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	err := <-errc
	return nodes, err
}

// pending adds a reply callback to the pending reply queue.
// see the documentation of type pending for a detailed explanation.
// pengding方法为每一个等待处理的事件添加一个回调函数
// 该函数返回值是一个管道，且是一个只读管道
func (t *udp) pending(id NodeID, ptype byte, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: id, ptype: ptype, callback: callback, errc: ch}
	//非循环
	select {
	case t.addpending <- p: //将p写入appending管道后，返回ch，因为没有写入会导致读取该管道的函数被阻塞
		// loop will handle it
	case <-t.closing: //当closing管道被关闭时，此case会被执行，向ch中写入相应错误信息
		ch <- errClosed
	}
	return ch
}

// 收到了对方的消息，创建一个reply加入到gotreply管道中
func (t *udp) handleReply(from NodeID, ptype byte, req packet) bool {
	matched := make(chan bool)
	select {
	case t.gotreply <- reply{from, ptype, req, matched}:
		// loop will handle it
		return <-matched
	case <-t.closing:
		return false
	}
}

// loop runs in its own goroutin. it keeps track of
// the refresh timer and the pending reply queue.
func (t *udp) loop() {
	var (
		pending      []*pending
		nextDeadline time.Time
		timeout      = time.NewTimer(0) //time.NewTimer返回一个channel，在到达规定时间后向channel写入信号
		refresh      = time.NewTicker(refreshInterval)
	)
	<-timeout.C          // ignore first timeout 忽略第一次超时信号
	defer refresh.Stop() //关闭定时器，但是并不关闭管道
	defer timeout.Stop()

	//函数变量rearmTimeout
	rearmTimeout := func() {
		now := time.Now()
		//判断挂起的消息数目是否为0，或者对于此消息是否已经超时
		if len(pending) == 0 || now.Before(nextDeadline) {
			return
		}
		nextDeadline = pending[0].deadline
		timeout.Reset(nextDeadline.Sub(now)) //重新调整timeout对应的计时器计时时间
	}

	for {
		select {
		case <-refresh.C: //每1h refresh.C管道中会产生信号
			go t.refresh()

		case <-t.closing: //意味着udp结构体的socket被关闭
			for _, p := range pending { //遍历pengding中剩余的所有挂起事件
				p.errc <- errClosed //向他们的errc管道中写入错误消息
			}
			pending = nil //清空挂起队列
			return

		case p := <-t.addpending: //取出udp结构体挂起等待处理的事件
			p.deadline = time.Now().Add(respTimeout) //为此事件设置处理截止时间--500ms
			pending = append(pending, p)             //将此事件加入到本方法的挂起处理队列
			rearmTimeout()                           //刷新本方法的计时器

		case r := <-t.gotreply: //收到了远端节点的消息
			var matched bool
			for i := 0; i < len(pending); i++ { //遍历整个挂起队列，找出队列中与接收到的消息相匹配的挂起事件
				if p := pending[i]; p.from == r.from && p.ptype == r.ptype {
					matched = true //说明此挂起事件被成功处理
					if p.callback(r.data) {
						// callback indicates the request is done, remove it.
						//回调函数返回true，意味着本节点向远端节点的请求被处理，因此需要将此挂起事件从pengding队列清除
						p.errc <- nil //写入nil，表明回调函数被执行完毕
						copy(pending[i:], pending[i+1:])
						pending = pending[:len(pending)-1]
						i--
					}
				}
			}
			r.matched <- matched //对应handleReply方法

		case now := <-timeout.C: //发生了超时事件
			// notify and remove callbacks whose deadline is in the past.
			i := 0
			//遍历所有pengding队列中超时的事件
			for ; i < len(pending) && now.After(pending[i].deadline); i++ {
				pending[i].errc <- errTimeout //向errc管道写入超时信息
			}
			//清除pengding队列中所有超时的事件
			if i > 0 {
				copy(pending, pending[i:])
				pending = pending[:len(pending)-i]
			}
			rearmTimeout() //刷新本方法的计时器
		}
	}
}

const (
	macSize  = 256 / 8
	sigSize  = 520 / 8
	headSize = macSize + sigSize // space of packet frame data
)

var (
	headSpace = make([]byte, headSize)

	// Neighbors responses are sent across multiple packets to
	// stay below the 1280 byte limit. We compute the maximum number
	// of entries by stuffing a packet until it grows too large.
	maxNeighbors int
)

func init() {
	p := neighbors{Expiration: ^uint64(0)}
	maxSizeNode := rpcNode{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		p.Nodes = append(p.Nodes, maxSizeNode)
		size, _, err := rlp.EncodeToReader(p)
		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("cannot encode: " + err.Error())
		}
		if headSize+size+1 >= 1280 {
			maxNeighbors = n
			break
		}
	}
}

func (t *udp) send(toaddr *net.UDPAddr, ptype byte, req interface{}) error {
	packet, err := encodePacket(t.priv, ptype, req)
	if err != nil {
		return err
	}
	glog.V(logger.Detail).Infof(">>> %v %T\n", toaddr, req)
	if _, err = t.conn.WriteToUDP(packet, toaddr); err != nil {
		glog.V(logger.Detail).Infoln("UDP send failed:", err)
	}
	return err
}

func encodePacket(priv *ecdsa.PrivateKey, ptype byte, req interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	b.Write(headSpace) //将headSpace切片写入Buffer的底部
	b.WriteByte(ptype) //将字符ptype写入Buffer的底部
	if err := rlp.Encode(b, req); err != nil {
		glog.V(logger.Error).Infoln("error encoding packet:", err)
		return nil, err
	}
	packet := b.Bytes()
	sig, err := crypto.Sign(crypto.Sha3(packet[headSize:]), priv)
	if err != nil {
		glog.V(logger.Error).Infoln("could not sign packet:", err)
		return nil, err
	}
	copy(packet[macSize:], sig)
	// TODO: add the hash to the front. Note: this doesn't protect the
	// packet in any way. Our public key will be part of this hash in
	// The future.
	copy(packet, crypto.Sha3(packet[macSize:]))
	return packet, nil
}

// readLoop runs in its own goroutine. it handles incoming UDP packets.
func (t *udp) readLoop() {
	defer t.conn.Close()
	// Discovery packets are defined to be no larger than 1280 bytes.
	// Packets larger than this size will be cut at the end and treated
	// as invalid because their hash won't match.
	buf := make([]byte, 1280)
	for {
		nbytes, from, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		t.handlePacket(from, buf[:nbytes])
	}
}

func (t *udp) handlePacket(from *net.UDPAddr, buf []byte) error {
	packet, fromID, hash, err := decodePacket(buf)
	if err != nil {
		glog.V(logger.Debug).Infof("Bad packet from %v: %v\n", from, err)
		return err
	}
	status := "ok"
	if err = packet.handle(t, from, fromID, hash); err != nil {
		status = err.Error()
	}
	glog.V(logger.Detail).Infof("<<< %v %T: %s\n", from, packet, status)
	return err
}

func decodePacket(buf []byte) (packet, NodeID, []byte, error) {
	if len(buf) < headSize+1 {
		return nil, NodeID{}, nil, errPacketTooSmall
	}
	hash, sig, sigdata := buf[:macSize], buf[macSize:headSize], buf[headSize:]
	shouldhash := crypto.Sha3(buf[macSize:])
	if !bytes.Equal(hash, shouldhash) {
		return nil, NodeID{}, nil, errBadHash
	}
	fromID, err := recoverNodeID(crypto.Sha3(buf[headSize:]), sig)
	if err != nil {
		return nil, NodeID{}, hash, err
	}
	var req packet
	switch ptype := sigdata[0]; ptype {
	case pingPacket:
		req = new(ping)
	case pongPacket:
		req = new(pong)
	case findnodePacket:
		req = new(findnode)
	case neighborsPacket:
		req = new(neighbors)
	default:
		return nil, fromID, hash, fmt.Errorf("unknown type: %d", ptype)
	}
	err = rlp.DecodeBytes(sigdata[1:], req)
	return req, fromID, hash, err
}

func (req *ping) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if req.Version != Version {
		return errBadVersion
	}
	t.send(from, pongPacket, pong{
		To:         makeEndpoint(from, req.From.TCP),
		ReplyTok:   mac,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	if !t.handleReply(fromID, pingPacket, req) {
		// Note: we're ignoring the provided IP address right now
		go t.bond(true, fromID, from, req.From.TCP)
	}
	return nil
}

func (req *pong) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !t.handleReply(fromID, pongPacket, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (req *findnode) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if t.db.node(fromID) == nil {
		// No bond exists, we don't process the packet. This prevents
		// an attack vector where the discovery protocol could be used
		// to amplify traffic in a DDOS attack. A malicious actor
		// would send a findnode request with the IP address and UDP
		// port of the target as the source address. The recipient of
		// the findnode packet would then send a neighbors packet
		// (which is a much bigger packet than findnode) to the victim.
		return errUnknownNode
	}
	target := crypto.Sha3Hash(req.Target[:])
	t.mutex.Lock()
	closest := t.closest(target, bucketSize).entries
	t.mutex.Unlock()

	p := neighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}
	// Send neighbors in chunks with at most maxNeighbors per packet
	// to stay below the 1280 byte limit.
	for i, n := range closest {
		p.Nodes = append(p.Nodes, nodeToRPC(n))
		if len(p.Nodes) == maxNeighbors || i == len(closest)-1 {
			t.send(from, neighborsPacket, p)
			p.Nodes = p.Nodes[:0]
		}
	}
	return nil
}

func (req *neighbors) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !t.handleReply(fromID, neighborsPacket, req) {
		return errUnsolicitedReply
	}
	return nil
}

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}
