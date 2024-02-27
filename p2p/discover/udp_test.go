package discover

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	logpkg "log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"p3Chain/crypto"
	"p3Chain/logger"
	"p3Chain/p2p/nat"
)

func init() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, logpkg.LstdFlags, logger.ErrorLevel))
}

// shared test variables
var (
	futureExp          = uint64(time.Now().Add(10 * time.Hour).Unix())
	testTarget         = NodeID{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1}
	testRemote         = rpcEndpoint{IP: net.ParseIP("1.1.1.1").To4(), UDP: 1, TCP: 2}
	testLocalAnnounced = rpcEndpoint{IP: net.ParseIP("2.2.2.2").To4(), UDP: 3, TCP: 4}
	testLocal          = rpcEndpoint{IP: net.ParseIP("3.3.3.3").To4(), UDP: 5, TCP: 6}
)

type udpTest struct {
	t                   *testing.T
	pipe                *dgramPipe
	table               *Table
	udp                 *udp
	sent                [][]byte
	localkey, remotekey *ecdsa.PrivateKey
	remoteaddr          *net.UDPAddr
}

func newUDPTest(t *testing.T) *udpTest {
	test := &udpTest{
		t:          t,
		pipe:       newpipe(),
		localkey:   newkey(),
		remotekey:  newkey(),
		remoteaddr: &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 30303},
	}
	test.table, test.udp = newUDP(test.localkey, test.pipe, nil, "")
	return test
}

// handles a packet as if it had been sent to the transport.
func (test *udpTest) packetIn(wantError error, ptype byte, data packet) error {
	enc, err := encodePacket(test.remotekey, ptype, data)
	if err != nil {
		// return test.errorf("packet (%d) encode error: %v", err)
		return test.errorf("packet (%v) encode error: %v", ptype, err)
	}
	test.sent = append(test.sent, enc)
	if err = test.udp.handlePacket(test.remoteaddr, enc); err != wantError {
		return test.errorf("error mismatch: got %q, want %q", err, wantError)
	}
	return nil
}

// waits for a packet to be sent by the transport.
// validate should have type func(*udpTest, X) error, where X is a packet type.
func (test *udpTest) waitPacketOut(validate interface{}) error {
	dgram := test.pipe.waitPacketOut()
	p, _, _, err := decodePacket(dgram)
	if err != nil {
		return test.errorf("sent packet decode error: %v", err)
	}
	fn := reflect.ValueOf(validate)
	exptype := fn.Type().In(0)
	if reflect.TypeOf(p) != exptype {
		return test.errorf("sent packet type mismatch, got: %v, want: %v", reflect.TypeOf(p), exptype)
	}
	fn.Call([]reflect.Value{reflect.ValueOf(p)})
	return nil
}

func (test *udpTest) errorf(format string, args ...interface{}) error {
	_, file, line, ok := runtime.Caller(2) // errorf + waitPacketOut
	if ok {
		file = filepath.Base(file)
	} else {
		file = "???"
		line = 1
	}
	err := fmt.Errorf(format, args...)
	fmt.Printf("\t%s:%d: %v\n", file, line, err)
	test.t.Fail()
	return err
}

func TestUDP_packetErrors(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	test.packetIn(errExpired, pingPacket, &ping{From: testRemote, To: testLocalAnnounced, Version: Version})
	test.packetIn(errBadVersion, pingPacket, &ping{From: testRemote, To: testLocalAnnounced, Version: 99, Expiration: futureExp})
	test.packetIn(errUnsolicitedReply, pongPacket, &pong{ReplyTok: []byte{}, Expiration: futureExp})
	test.packetIn(errUnknownNode, findnodePacket, &findnode{Expiration: futureExp})
	test.packetIn(errUnsolicitedReply, neighborsPacket, &neighbors{Expiration: futureExp})
}

func TestUDP_pingTimeout(t *testing.T) {
	t.Parallel()
	test := newUDPTest(t)
	defer test.table.Close()

	toaddr := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 2222}
	toid := NodeID{1, 2, 3, 4}
	if err := test.udp.ping(toid, toaddr); err != errTimeout {
		t.Error("expected timeout error, got", err)
	}
}

func TestUDP_findnodeTimeout(t *testing.T) {
	t.Parallel()
	test := newUDPTest(t)
	defer test.table.Close()

	toaddr := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 2222}
	toid := NodeID{1, 2, 3, 4}
	target := NodeID{4, 5, 6, 7}
	result, err := test.udp.findnode(toid, toaddr, target)
	if err != errTimeout {
		t.Error("expected timeout error, got", err)
	}
	if len(result) > 0 {
		t.Error("expected empty result, got", result)
	}
}

func TestUDP_findnode(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	// put a few nodes into the table. their exact
	// distribution shouldn't matter much, altough we need to
	// take care not to overflow any bucket.
	targetHash := crypto.Sha3Hash(testTarget[:])
	nodes := &nodesByDistance{target: targetHash}
	for i := 0; i < bucketSize; i++ {
		nodes.push(nodeAtDistance(test.table.self.sha, i+2), bucketSize)
	}
	test.table.add(nodes.entries)

	// ensure there's a bond with the test node,
	// findnode won't be accepted otherwise.
	test.table.db.updateNode(newNode(
		PubkeyID(&test.remotekey.PublicKey),
		test.remoteaddr.IP,
		uint16(test.remoteaddr.Port),
		99,
	))
	// check that closest neighbors are returned.
	test.packetIn(nil, findnodePacket, &findnode{Target: testTarget, Expiration: futureExp})
	expected := test.table.closest(targetHash, bucketSize)

	waitNeighbors := func(want []*Node) {
		test.waitPacketOut(func(p *neighbors) {
			if len(p.Nodes) != len(want) {
				t.Errorf("wrong number of results: got %d, want %d", len(p.Nodes), bucketSize)
			}
			for i := range p.Nodes {
				if p.Nodes[i].ID != want[i].ID {
					t.Errorf("result mismatch at %d:\n  got:  %v\n  want: %v", i, p.Nodes[i], expected.entries[i])
				}
			}
		})
	}
	waitNeighbors(expected.entries[:maxNeighbors])
	waitNeighbors(expected.entries[maxNeighbors:])
}

func TestUDP_findnodeMultiReply(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	// queue a pending findnode request
	resultc, errc := make(chan []*Node), make(chan error)
	go func() {
		rid := PubkeyID(&test.remotekey.PublicKey)
		ns, err := test.udp.findnode(rid, test.remoteaddr, testTarget)
		if err != nil && len(ns) == 0 {
			errc <- err
		} else {
			resultc <- ns
		}
	}()

	// wait for the findnode to be sent.
	// after it is sent, the transport is waiting for a reply
	test.waitPacketOut(func(p *findnode) {
		if p.Target != testTarget {
			t.Errorf("wrong target: got %v, want %v", p.Target, testTarget)
		}
	})

	// send the reply as two packets.
	list := []*Node{
		MustParseNode("dpnode://ba85011c70bcc5c04d8607d3a0ed29aa6179c092cbdda10d5d32684fb33ed01bd94f588ca8f91ac48318087dcb02eaf36773a7a453f0eedd6742af668097b29c@10.0.1.16:30303?discport=30304"),
		MustParseNode("dpnode://81fa361d25f157cd421c60dcc28d8dac5ef6a89476633339c5df30287474520caca09627da18543d9079b5b288698b542d56167aa5c09111e55acdbbdf2ef799@10.0.1.16:30303"),
		MustParseNode("dpnode://9bffefd833d53fac8e652415f4973bee289e8b1a5c6c4cbe70abf817ce8a64cee11b823b66a987f51aaa9fba0d6a91b3e6bf0d5a5d1042de8e9eeea057b217f8@10.0.1.36:30301?discport=17"),
		MustParseNode("dpnode://1b5b4aa662d7cb44a7221bfba67302590b643028197a7d5214790f3bac7aaa4a3241be9e83c09cf1f6c69d007c634faae3dc1b1221793e8446c0b3a09de65960@10.0.1.16:30303"),
	}
	rpclist := make([]rpcNode, len(list))
	for i := range list {
		rpclist[i] = nodeToRPC(list[i])
	}
	test.packetIn(nil, neighborsPacket, &neighbors{Expiration: futureExp, Nodes: rpclist[:2]})
	test.packetIn(nil, neighborsPacket, &neighbors{Expiration: futureExp, Nodes: rpclist[2:]})

	// check that the sent neighbors are all returned by findnode
	select {
	case result := <-resultc:
		if !reflect.DeepEqual(result, list) {
			t.Errorf("neighbors mismatch:\n  got:  %v\n  want: %v", result, list)
		}
	case err := <-errc:
		t.Errorf("findnode error: %v", err)
	case <-time.After(5 * time.Second):
		t.Error("findnode did not return within 5 seconds")
	}
}

func TestUDP_successfulPing(t *testing.T) {
	test := newUDPTest(t)
	added := make(chan *Node, 1)
	test.table.nodeAddedHook = func(n *Node) { added <- n }
	defer test.table.Close()

	// The remote side sends a ping packet to initiate the exchange.
	go test.packetIn(nil, pingPacket, &ping{From: testRemote, To: testLocalAnnounced, Version: Version, Expiration: futureExp})

	// the ping is replied to.
	test.waitPacketOut(func(p *pong) {
		pinghash := test.sent[0][:macSize]
		if !bytes.Equal(p.ReplyTok, pinghash) {
			t.Errorf("got pong.ReplyTok %x, want %x", p.ReplyTok, pinghash)
		}
		wantTo := rpcEndpoint{
			// The mirrored UDP address is the UDP packet sender
			IP: test.remoteaddr.IP, UDP: uint16(test.remoteaddr.Port),
			// The mirrored TCP port is the one from the ping packet
			TCP: testRemote.TCP,
		}
		if !reflect.DeepEqual(p.To, wantTo) {
			t.Errorf("got pong.To %v, want %v", p.To, wantTo)
		}
	})

	// remote is unknown, the table pings back.
	test.waitPacketOut(func(p *ping) error {
		if !reflect.DeepEqual(p.From, test.udp.ourEndpoint) {
			t.Errorf("got ping.From %v, want %v", p.From, test.udp.ourEndpoint)
		}
		wantTo := rpcEndpoint{
			// The mirrored UDP address is the UDP packet sender.
			IP: test.remoteaddr.IP, UDP: uint16(test.remoteaddr.Port),
			TCP: 0,
		}
		if !reflect.DeepEqual(p.To, wantTo) {
			t.Errorf("got ping.To %v, want %v", p.To, wantTo)
		}
		return nil
	})
	test.packetIn(nil, pongPacket, &pong{Expiration: futureExp})

	// the node should be added to the table shortly after getting the
	// pong packet.
	select {
	case n := <-added:
		rid := PubkeyID(&test.remotekey.PublicKey)
		if n.ID != rid {
			t.Errorf("node has wrong ID: got %v, want %v", n.ID, rid)
		}
		if !bytes.Equal(n.IP, test.remoteaddr.IP) {
			t.Errorf("node has wrong IP: got %v, want: %v", n.IP, test.remoteaddr.IP)
		}
		if int(n.UDP) != test.remoteaddr.Port {
			t.Errorf("node has wrong UDP port: got %v, want: %v", n.UDP, test.remoteaddr.Port)
		}
		if n.TCP != testRemote.TCP {
			t.Errorf("node has wrong TCP port: got %v, want: %v", n.TCP, testRemote.TCP)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("node was not added within 2 seconds")
	}
}

// dgramPipe is a fake UDP socket. It queues all sent datagrams.
type dgramPipe struct {
	mu      *sync.Mutex //互斥锁
	cond    *sync.Cond  //条件变量
	closing chan struct{}
	closed  bool
	queue   [][]byte
}

func newpipe() *dgramPipe {
	mu := new(sync.Mutex)
	return &dgramPipe{
		closing: make(chan struct{}), //传输结构体的管道
		cond:    &sync.Cond{L: mu},   //条件变量
		mu:      mu,                  //互斥锁
	}
}

// WriteToUDP queues a datagram.
func (c *dgramPipe) WriteToUDP(b []byte, to *net.UDPAddr) (n int, err error) {
	msg := make([]byte, len(b))
	copy(msg, b)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, errors.New("closed")
	}
	c.queue = append(c.queue, msg)
	c.cond.Signal()
	return len(b), nil
}

// ReadFromUDP just hangs until the pipe is closed.
func (c *dgramPipe) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	<-c.closing
	return 0, nil, io.EOF
}

func (c *dgramPipe) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		close(c.closing)
		c.closed = true
	}
	return nil
}

func (c *dgramPipe) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: testLocal.IP, Port: int(testLocal.UDP)}
}

func (c *dgramPipe) waitPacketOut() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	for len(c.queue) == 0 {
		c.cond.Wait()
	}
	p := c.queue[0]
	copy(c.queue, c.queue[1:])
	c.queue = c.queue[:len(c.queue)-1]
	return p
}

type MockNatHoleServer struct {
	addr       net.IP //本内网主机的内网IP地址
	port       int    //本地端口号
	backExtIP  net.IP
	serverIP   net.IP //第三方服务器的IP地址
	serverPort int    //第三方服务器开放的端口号
}

func (mshs *MockNatHoleServer) Run() {

	go udpHoleServer(mshs)
	go tcpHoleServer(mshs)
	for {
		time.Sleep(1 * time.Second)
	}
}

func udpHoleServer(mshs *MockNatHoleServer) {

	//监听指定端口的连接请求
	listen, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   mshs.addr,
		Port: mshs.port,
	})
	if err != nil {
		fmt.Println("Listen failed, err: ", err)
		return
	}
	defer listen.Close()
	//循环接收数据
	for {
		var data [1024]byte
		n, addr, err := listen.ReadFromUDP(data[:]) // 接收数据
		if err != nil {
			fmt.Println("read udp failed, err: ", err)
			continue
		}
		strArr := strings.Split(string(data[:n]), ":")
		//显示读取的客户端发送数据，客户端公网地址，接收消息字节数
		fmt.Printf("server: order:%v client_addr:%v byte_count:%v\n", strArr[0], addr, n)

		//1.指令为获取自身公网IP:PORT地址
		if 0 == strings.Compare("GET ExIP", strArr[0]) {
			_, err = listen.WriteToUDP([]byte(addr.String()), addr) // 向客户端发送其公网UDP地址
			if err != nil {
				fmt.Println(err)
			}
		}

	}
}

func handleConn(conn net.Conn) {

	addr := conn.RemoteAddr().String() //获取客户端的IP:PORT(公网地址)字符串形式

	buf := make([]byte, 1024) //读写缓冲区

	defer conn.Close()

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		//显示读取的客户端发送数据，客户端公网地址，接收消息字节数
		fmt.Printf("server: order:%v client_addr:%v byte_count:%v\n", string(buf[:n]), addr, n)

		//1.指令为获取自身公网IP:PORT地址
		if 0 == strings.Compare("TCP GET ExIP:", string(buf[:n])) {
			//向客户端发送其对外的TCP地址
			conn.Write([]byte(addr))
		}

	}

}

func tcpHoleServer(mshs *MockNatHoleServer) {

	//监听指定端口的连接请求
	listen, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   mshs.addr,
		Port: mshs.port,
	})
	if err != nil {
		return
	}
	defer listen.Close()

	//循环等待客户端的Conn请求，并且由协程负责处理
	for {
		conn, err1 := listen.Accept()
		if err1 != nil {
			return
		}
		go handleConn(conn)
	}
}

func TestNatHole(t *testing.T) {
	test := &udpTest{
		t:          t,
		pipe:       newpipe(),
		localkey:   newkey(),
		remotekey:  newkey(),
		remoteaddr: &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 30303},
	}

	mockServer := MockNatHoleServer{
		//1.内网环境下测试
		addr:      net.IP{127, 0, 0, 1}, //客户端bind时的本地地址(默认就行)
		port:      50001,                //客户端bind时的本地端口
		backExtIP: net.IP{127, 0, 0, 1},
		//2.采用第三方服务器测试
		//backExtIP:  net.IP{123, 235, 115, 192}, //预计的主机的对外公网IP地址
		serverIP:   net.IP{117, 50, 178, 228}, //第三方服务器的公网IP地址
		serverPort: 8080,                      //第三方服务器开放的访问端口
	}
	go mockServer.Run()

	natm := nat.NatHole(mockServer.addr, mockServer.port)
	addr := realAddr(test.localkey, test.pipe, natm, "")   //返回从第三方服务器获取的自身公网UDP地址
	if !reflect.DeepEqual(mockServer.backExtIP, addr.IP) { //判断从服务器获取的自身公网IP地址与预计的backExtIP是否相等
		t.Fatalf("not right ExtIP!")
	}
}

func TestStaticReflect(t *testing.T) {
	test := &udpTest{
		t:          t,
		pipe:       newpipe(),
		localkey:   newkey(),
		remotekey:  newkey(),
		remoteaddr: &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 30303},
	}
	extIP := net.IP{101, 43, 246, 68}
	filePath := "./ExIP.txt" //记录自身公网IP地址的文件的路径
	natm := nat.StaticReflect(filePath)
	addr := realAddr(test.localkey, test.pipe, natm, "")
	if !reflect.DeepEqual(extIP, addr.IP) {
		t.Fatalf("not right ExtIP!")
	}
}

func realAddr(priv *ecdsa.PrivateKey, c conn, natm nat.Interface, nodeDBPath string) *net.UDPAddr {
	tab, _ := newUDP(priv, c, natm, nodeDBPath)
	return tab.self.addr()
}
