// Package p2p implements the p3Chain p2p network protocols.
package server

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"p3Chain/common"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
)

const (
	defaultDialTimeout      = 15 * time.Second
	refreshPeersInterval    = 30 * time.Second
	staticPeerCheckInterval = 15 * time.Second

	// Maximum number of concurrently handshaking inbound connections.
	maxAcceptConns = 50 //允许的并发握手入站的连接的最大数量

	// Maximum number of concurrently dialing outbound connections.
	maxActiveDialTasks = 16

	// Maximum time allowed for reading a complete message.
	// This is effectively the amount of time a connection can be idle.
	frameReadTimeout = 30 * time.Second

	// Maximum amount of time allowed for writing a complete message.
	frameWriteTimeout = 20 * time.Second
)

var errServerStopped = errors.New("server stopped")

var srvjslog = logger.NewJsonLogger()

// Server manages all peer connections.
//
// The fields of Server are used as configuration parameters.
// You should set them before starting the Server. Fields may not be
// modified while the server is running.
// 服务器模块管理所有节点的连接,需要在服务器模块启动之前完成所有字段的配置,服务器启动之后不能进行配置
type Server struct {
	// This field must be set to a valid secp256k1 private key.
	PrivateKey *ecdsa.PrivateKey //本地节点的私钥

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int //允许的最大连接数

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int //handshake阶段连接等待队列的最大数,默认值为0

	// Discovery specifies whether the peer discovery mechanism should be started
	// or not. Disabling is usually useful for protocol debugging (manual topology).
	Discovery bool //表明节点的discovery机制是否被启动(用于发现其他节点)

	// Name sets the node name of this server.
	// Use common.MakeName to create a name that follows existing conventions.
	Name string //节点服务器模块的名称,使用common.MakeName来创建一个遵循现有公约的Name

	// Bootstrap nodes are used to establish connectivity
	// with the rest of the network.
	BootstrapNodes []*discover.Node //引导节点列表

	// Static nodes are used as pre-configured connections which are always
	// maintained and re-connected on disconnects.
	StaticNodes []*discover.Node //静态节点列表,静态连接时需要随时保持的连接(一旦断连需要重连)

	// Trusted nodes are used as pre-configured connections which are always
	// allowed to connect, even above the peer limit.
	TrustedNodes []*discover.Node //信任节点列表,始终允许与这些节点的连接

	// NodeDatabase is the path to the database containing the previously seen
	// live nodes in the network.
	NodeDatabase string //数据库路径,该数据库中包含了本节点先前遇到过的活动节点

	// Protocols should contain the protocols supported
	// by the server. Matching protocols are launched for
	// each peer.
	Protocols []Protocol //当前节点支持的所有协议

	// If ListenAddr is set to a non-nil address, the server
	// will listen for incoming connections.
	//
	// If the port is zero, the operating system will pick a port. The
	// ListenAddr field will be updated with the actual address when
	// the server is started.
	ListenAddr string //服务器模块的监听地址.如果端口号被设置为0，操作系统将会随机选择一个端口号

	// If set to a non-nil value, the given NAT port mapper
	// is used to make the listening port available to the
	// Internet.
	NAT nat.Interface //完成NAT穿越

	// If Dialer is set to a non-nil value, the given Dialer
	// is used to dial outbound peer connections.
	Dialer *net.Dialer //拨号器

	// If NoDial is true, the server will not dial any peers.
	NoDial bool //如果为true，服务器模块将不会与任何节点dial

	// Hooks for testing. These are useful because we can inhibit
	// the whole protocol stack.
	newTransport func(net.Conn) transport //用于测试的Hook函数
	newPeerHook  func(*Peer)

	lock    sync.Mutex // protects running
	running bool

	ntab         discoverTable
	listener     net.Listener //本地网络监听器接口
	ourHandshake *protoHandshake
	lastLookup   time.Time

	// These are for Peers, PeerCount (and nothing else).
	peerOp     chan peerOpFunc
	peerOpDone chan struct{}

	quit          chan struct{}
	addstatic     chan *discover.Node //传输需要 静态连接 的节点
	delstatic     chan *discover.Node //传输需要删除的静态节点
	posthandshake chan *conn
	addpeer       chan *conn
	delpeer       chan *Peer
	loopWG        sync.WaitGroup // loop, listenLoop
}

type peerOpFunc func(map[discover.NodeID]*Peer)

type connFlag int

// 连接类型
const (
	dynDialedConn connFlag = 1 << iota
	staticDialedConn
	inboundConn
	trustedConn
)

// conn wraps a network connection with information gathered
// during the two handshakes.
type conn struct {
	fd        net.Conn        //conn socket
	transport                 //此接口继承了MsgReadWriter接口
	flags     connFlag        //连接类型(静态 or 动态)
	cont      chan error      // The run loop uses cont to signal errors to setupConn.
	id        discover.NodeID // valid after the encryption handshake
	caps      []Cap           // valid after the protocol handshake
	name      string          // valid after the protocol handshake
}

type transport interface {
	// The two handshakes.
	doEncHandshake(prv *ecdsa.PrivateKey, dialDest *discover.Node) (discover.NodeID, error) //加密握手
	doProtoHandshake(our *protoHandshake) (*protoHandshake, error)                          //协议握手
	// The MsgReadWriter can only be used after the encryption
	// handshake has completed. The code uses conn.id to track this
	// by setting it to a non-nil value after the encryption handshake.
	MsgReadWriter //仅当加密握手完成之后才会被使用
	// transports must provide Close because we use MsgPipe in some of
	// the tests. Closing the actual network connection doesn't do
	// anything in those tests because MsgPipe doesn't use it.
	close(err error)
}

func (c *conn) String() string {
	s := c.flags.String() + " conn" //显示连接类型
	if (c.id != discover.NodeID{}) {
		s += fmt.Sprintf(" %x", c.id[:8]) //显示对端id的前八位
	}
	s += " " + c.fd.RemoteAddr().String() //显示对端地址
	return s
}

func (f connFlag) String() string {
	s := ""
	if f&trustedConn != 0 {
		s += " trusted"
	}
	if f&dynDialedConn != 0 {
		s += " dyn dial"
	}
	if f&staticDialedConn != 0 {
		s += " static dial"
	}
	if f&inboundConn != 0 {
		s += " inbound"
	}
	if s != "" {
		s = s[1:]
	}
	return s
}

// 判断是否为指定的连接类型
func (c *conn) is(f connFlag) bool {
	return c.flags&f != 0
}

// Peers returns all connected peers.
// 本方法返回当前所有与本服务器模块连接的peer
func (srv *Server) Peers() []*Peer {
	var ps []*Peer
	select {
	// Note: We'd love to put this function into a variable but
	// that seems to cause a weird compiler error in some
	// environments.
	// srv.peerOp 中存入一个匿名函数,该函数负责将形参peers的所有成员追加到ps切片中
	case srv.peerOp <- func(peers map[discover.NodeID]*Peer) {
		for _, p := range peers {
			ps = append(ps, p)
		}
	}:
		<-srv.peerOpDone
	case <-srv.quit:
	}
	return ps
}

// PeerCount returns the number of connected peers.
// 返回已连接的peer的数目
func (srv *Server) PeerCount() int {
	var count int
	select {
	//srv.peerOp中存入一个匿名函数,该函数负责计算形参ps的长度
	case srv.peerOp <- func(ps map[discover.NodeID]*Peer) { count = len(ps) }:
		<-srv.peerOpDone
	case <-srv.quit:
	}
	return count
}

// AddPeer connects to the given node and maintains the connection until the
// server is shut down. If the connection fails for any reason, the server will
// attempt to reconnect the peer.
// 本方法负责为服务器模块添加一个需要连接的 static peer
func (srv *Server) AddPeer(node *discover.Node) {
	select {
	case srv.addstatic <- node:
	case <-srv.quit:
	}
}

// DelPeer dels a node from static nodes, prevents the server from keeping trying to
// connect it.
// 本方法负责为服务器模块删除一个静态节点
func (srv *Server) DelPeer(node *discover.Node) {
	select {
	case srv.delstatic <- node:
	case <-srv.quit:
	}
}

// Self returns the local node's endpoint information.
// 返回本地节点的信息(Node结构体信息)
func (srv *Server) Self() *discover.Node {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	// If the server's not running, return an empty node
	// 如果服务器模块并没有运行
	if !srv.running {
		return &discover.Node{IP: net.ParseIP("0.0.0.0")}
	}
	// If the node is running but discovery is off, manually assemble the node infos
	// 服务器以及运行,但是没有开始discovery功能(没有Kad路由表),手动组装节点信息
	if srv.ntab == nil {
		// Inbound connections disabled, use zero address
		if srv.listener == nil { //本地没有监听器(不允许其他节点的连接)
			return &discover.Node{IP: net.ParseIP("0.0.0.0"), ID: discover.PubkeyID(&srv.PrivateKey.PublicKey)}
		}
		// Otherwise inject the listener address too
		addr := srv.listener.Addr().(*net.TCPAddr) //获取本地监听地址
		return &discover.Node{                     //返回本地节点的Node信息
			ID:  discover.PubkeyID(&srv.PrivateKey.PublicKey),
			IP:  addr.IP,
			TCP: uint16(addr.Port),
		}
	}
	// Otherwise return the live node infos
	return srv.ntab.Self() //返回本地节点的Node信息
}

// Stop terminates the server and all active peer connections.
// It blocks until all active connections have been closed.
// 终止服务器模块以及所有活跃节点连接,本方法将一直阻塞直到所有的连接都已经被关闭。
func (srv *Server) Stop() {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if !srv.running {
		return
	}
	srv.running = false
	if srv.listener != nil {
		// this unblocks listener Accept
		srv.listener.Close()
	}
	close(srv.quit)
	srv.loopWG.Wait()
}

// Start starts running the server.
// Servers can not be re-used after stopping.
// 本方法负责启动运行一个服务器模块(并在最后单独协程运行srv.run(dialer)),服务器模块在停止后不能被重新使用
func (srv *Server) Start() {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running { //服务器已在运行状态
		loglogrus.Log.Infof("P2P: Server already running\n")
		return
	}
	srv.running = true
	loglogrus.Log.Infof("P2P: Prepare for Starting Server...\n")

	// static fields
	if srv.PrivateKey == nil { //判断是否有私钥
		loglogrus.Log.Errorf("P2P: Server.PrivateKey hasn't been setted!\n")
		panic("P2P: Server.PrivateKey hasn't been setted!")
	}
	if srv.newTransport == nil { //判断是否有transport接口
		srv.newTransport = newRLPX
	}
	if srv.Dialer == nil { //判断是否有拨号器
		srv.Dialer = &net.Dialer{Timeout: defaultDialTimeout}
	}
	srv.quit = make(chan struct{})
	srv.addpeer = make(chan *conn)
	srv.delpeer = make(chan *Peer)
	srv.posthandshake = make(chan *conn)
	srv.addstatic = make(chan *discover.Node)
	srv.delstatic = make(chan *discover.Node)
	srv.peerOp = make(chan peerOpFunc)
	srv.peerOpDone = make(chan struct{})

	// node table
	if srv.Discovery { //允许使用discovery功能
		ntab, err := discover.ListenUDP(srv.PrivateKey, srv.ListenAddr, srv.NAT, srv.NodeDatabase) //在本地创建一个新的Kad路由表
		if err != nil {
			loglogrus.Log.Errorf("P2P: Create Kad discover.Table is failed, err:%v\n", err)
			panic("P2P: Create Kad discover.Table is failed,Unable to do node discovery!")
		}
		srv.ntab = ntab
	}

	dynPeers := srv.MaxPeers / 2 //动态节点数需为最大连接节点数的一半
	if !srv.Discovery {
		dynPeers = 0
	}
	dialer := newDialState(srv.StaticNodes, srv.ntab, dynPeers)

	// handshake
	srv.ourHandshake = &protoHandshake{Version: baseProtocolVersion, Name: srv.Name, ID: discover.PubkeyID(&srv.PrivateKey.PublicKey)}
	for _, p := range srv.Protocols { //遍历srv.Protocols,获取所有需要的协议,追加到srv.ourHandshake中
		srv.ourHandshake.Caps = append(srv.ourHandshake.Caps, p.cap())
	}
	// listen/dial
	if srv.ListenAddr != "" { //已开启本地节点的监听功能
		srv.startListening() //调用srv.startListening()进入监听状态
	}
	if srv.NoDial && srv.ListenAddr == "" { //未开启监听功能的情况下
		loglogrus.Log.Infof("I will be kind-of useless, neither dialing nor listening.\n")
	}
	srv.loopWG.Add(1)
	go srv.run(dialer)
	srv.running = true
}

func (srv *Server) startListening() {
	// Launch the TCP listener.
	listener, err := net.Listen("tcp", srv.ListenAddr) //监听tcp地址
	if err != nil {
		loglogrus.Log.Errorf("P2P: The server module cannot perform TCP Listening, err:%v\n", err)
		panic("P2P: The server module cannot perform TCP Listening")
	}
	laddr := listener.Addr().(*net.TCPAddr)
	srv.ListenAddr = laddr.String()
	srv.listener = listener
	srv.loopWG.Add(1)
	go srv.listenLoop() //for循环接收peer的accept,建立TCP连接,然后完成与这些peer的加密handshake与协议handshake
	// Map the TCP listening port if NAT is configured.
	if !laddr.IP.IsLoopback() && srv.NAT != nil { //本地监听地址非环回地址,同时存在NAT
		srv.loopWG.Add(1)
		go func() { //完成NAT映射(获取自身公网地址)
			nat.Map(srv.NAT, srv.quit, "tcp", laddr.Port, laddr.Port, "p3Chain p2p")
			srv.loopWG.Done()
		}()
	}
}

type dialer interface {
	newTasks(running int, peers map[discover.NodeID]*Peer, now time.Time) []task //根据所有已连接peer创建task任务列表
	taskDone(task, time.Time)
	addStatic(*discover.Node) //添加静态节点
	delStatic(*discover.Node) //删除静态节点
}

func (srv *Server) RemovePeer(peer *Peer) {
	srv.delpeer <- peer
}

func (srv *Server) run(dialstate dialer) {
	defer srv.loopWG.Done()
	var (
		peers   = make(map[discover.NodeID]*Peer)
		trusted = make(map[discover.NodeID]bool, len(srv.TrustedNodes))

		tasks        []task //存储所有已经经过Do函数完成的任务
		pendingTasks []task //存储所有正在等待处理的任务
		taskdone     = make(chan task, maxActiveDialTasks)
	)
	// Put trusted nodes into a map to speed up checks.
	// Trusted peers are loaded on startup and cannot be
	// modified while the server is running.
	for _, n := range srv.TrustedNodes { //Trusted peers 在startup阶段被装载,server运行节点是不可以进行修改的
		trusted[n.ID] = true
	}

	// Some task list helpers.
	delTask := func(t task) { // delTask 对应的匿名函数负责在task任务队列中删除形参t指定的任务
		for i := range tasks {
			if tasks[i] == t {
				tasks = append(tasks[:i], tasks[i+1:]...) //删除第i个任务
				break
			}
		}
	}
	scheduleTasks := func(new []task) { //负责将形参指定的任务添加到pengdingTasks队列等待处理,同时处理一部分pengdingTasks队列中的任务
		pt := append(pendingTasks, new...)       //将形参指定的任务队列追加到pendingTasks中
		start := maxActiveDialTasks - len(tasks) //start等于当前减去tasks任务队列任务数之后还剩的最多容许的任务数
		if len(pt) < start {                     //pt中存储的是等待处理的任务
			start = len(pt) //start等于上述两者最小的那一个
		}
		if start > 0 {
			tasks = append(tasks, pt[:start]...) //将等待任务队列中前start个任务追加到tasks任务队列中(已经完成的任务)
			for _, t := range pt[:start] {       //遍历这前start个等待处理的任务
				t := t
				glog.V(logger.Detail).Infoln("new task:", t)
				go func() { t.Do(srv); taskdone <- t }() //调用协程负责处理这些任务(根据任务类型执行不同的Do方法)
			}
			copy(pt, pt[start:])              //更新pt队列
			pendingTasks = pt[:len(pt)-start] //更新pengdingTasks队列
		}
	}

running:
	for {
		// Query the dialer for new tasks and launch them.
		now := time.Now()
		nt := dialstate.newTasks(len(pendingTasks)+len(tasks), peers, now) //创建一个任务队列,根据peers各节点的连接类型填充任务
		scheduleTasks(nt)                                                  //处理位于pengdingTasks队列正在等待处理的任务(一个任务一个协程)

		select {
		case <-srv.quit: //服务器结束运行(唯有这一种情况会跳出整个for循环)
			// The server was stopped. Run the cleanup logic.
			break running
		case n := <-srv.addstatic: //需要添加静态节点
			// This channel is used by AddPeer to add to the
			// ephemeral static peer list. Add it to the dialer,
			// it will keep the node connected.
			dialstate.addStatic(n)
		case n := <-srv.delstatic: //删除一个静态节点
			// This channel is used by DelPeer to delete a node from
			// static peer list.
			dialstate.delStatic(n)
		case op := <-srv.peerOp: //op获取到一个匿名函数
			// This channel is used by Peers and PeerCount.
			op(peers) //调用匿名函数(两种可能:1.将此处的peers追加到211行的ps切片中  2.计算peers的长度返回到231行的count中)
			srv.peerOpDone <- struct{}{}
		case t := <-taskdone: //经过Do函数完成了任务
			// A task got done. Tell dialstate about it so it
			// can update its state and remove it from the active
			// tasks list.
			dialstate.taskDone(t, now) //根据完成的结果更新dialstate对象
			delTask(t)                 //从tasks任务队列中删除任务t
		case c := <-srv.posthandshake: //已经完成了与对端peer的加密handshake
			// A connection has passed the encryption handshake so
			// the remote identity is known (but hasn't been verified yet).
			if trusted[c.id] { //认为该节点是可信任的
				// Ensure that the trusted flag is set before checking against MaxPeers.
				c.flags |= trustedConn //连接类型设置为可信连接
			}
			// TODO: track in-progress inbound node IDs (pre-Peer) to avoid dialing them.
			c.cont <- srv.encHandshakeChecks(peers, c) //验证加密handshake是否正确
		case c := <-srv.addpeer: //已经完成了与对端peer的协议handshake
			// At this point the connection is past the protocol handshake.
			// Its capabilities are known and the remote identity is verified.
			err := srv.protoHandshakeChecks(peers, c) //验证协议handshake是否正确
			if err != nil {
			} else { //只有当加密handshake与协议handshake都正确时,才会运行srv.runPeer(p)
				// The handshakes are done and it passed all checks.
				p := newPeer(c, srv.Protocols) //根据conn连接器和支持的协议簇创建一个Peer对象
				peers[c.id] = p                //在peers集合中记录该Peer对象
				go srv.runPeer(p)              //由一个单独的协程负责运行该Peer对象的run函数
			}
			// The dialer logic relies on the assumption that
			// dial tasks complete after the peer has been added or
			// discarded. Unblock the task last.
			c.cont <- err
		case p := <-srv.delpeer: //需要删除一个peer(peer对象p的run协程出现了错误)
			// A peer disconnected.
			delete(peers, p.ID())

		}
	}

	// Terminate discovery. If there is a running lookup it will terminate soon.
	//关闭Kad路由表,如果有正在执行的lookup操作那么将会在不久后被迫结束
	if srv.ntab != nil {
		srv.ntab.Close()
	}
	// Disconnect all peers.
	//负责终止本地服务器模块与所有peers集合中节点的连接
	for _, p := range peers {
		p.Disconnect(DiscQuitting)
	}
	// Wait for peers to shut down. Pending connections and tasks are
	// not handled here and will terminate soon-ish because srv.quit
	// is closed.
	glog.V(logger.Detail).Infof("ignoring %d pending tasks at spindown", len(tasks))
	for len(peers) > 0 { //循环等待所有的peer从peers中被删除
		p := <-srv.delpeer
		glog.V(logger.Detail).Infoln("<-delpeer (spindown):", p)
		delete(peers, p.ID())
	}
}

func (srv *Server) protoHandshakeChecks(peers map[discover.NodeID]*Peer, c *conn) error {
	// Drop connections with no matching protocols.
	if len(srv.Protocols) > 0 && countMatchingProtocols(srv.Protocols, c.caps) == 0 { //服务器不支持对端节点需要的协议
		return DiscUselessPeer
	}
	// Repeat the encryption handshake checks because the
	// peer set might have changed between the handshakes.
	return srv.encHandshakeChecks(peers, c) //验证加密handshake是否正确
}

// 加密handshake的检查(验证)
func (srv *Server) encHandshakeChecks(peers map[discover.NodeID]*Peer, c *conn) error {
	switch {
	case !c.is(trustedConn|staticDialedConn) && len(peers) >= srv.MaxPeers: //
		loglogrus.Log.Warnf("进行加密handshake: 连接节点数超过上限!\n")
		return DiscTooManyPeers
	case peers[c.id] != nil: //可信节点在peers集合中(表明可信节点先于peers集合中的节点与本机完成了连接)
		loglogrus.Log.Warnf("进行加密handshake: 节点已经与本机建立了连接\n")
		return DiscAlreadyConnected
	case c.id == srv.Self().ID: //可信节点是本地节点
		return DiscSelf
	default:
		return nil
	}
}

// listenLoop runs in its own goroutine and accepts
// inbound connections.
// listenLoop在单独的协程中运行,负责接受入站连接请求
func (srv *Server) listenLoop() {
	defer srv.loopWG.Done()
	loglogrus.Log.Infof("Listening on:%s", srv.listener.Addr().String())

	// This channel acts as a semaphore limiting
	// active inbound connections that are lingering pre-handshake.
	// If all slots are taken, no further connections are accepted.
	tokens := maxAcceptConns
	if srv.MaxPendingPeers > 0 {
		tokens = srv.MaxPendingPeers
	}
	slots := make(chan struct{}, tokens)
	for i := 0; i < tokens; i++ {
		slots <- struct{}{}
	}

	for { //循环accept新的TCP连接请求
		<-slots
		fd, err := srv.listener.Accept() //接受新节点的TCP连接
		if err != nil {
			loglogrus.Log.Warnf("Can't Accept TCP Connection Request,err:%v\n", err)
			//WARN: here might be a bug
			return
		}
		mfd := newMeteredConn(fd, true) //新节点的conn socket登记为入站socket

		go func() {
			srv.setupConn(mfd, inboundConn, nil) //用协程负责执行加密handshake与协议handshake
			slots <- struct{}{}
		}()
	}
}

// setupConn runs the handshakes and attempts to add the connection
// as a peer. It returns when the connection has been added as a peer
// or the handshakes have failed.
// 在TCP连接的基础之上，运行加密handshake与协议handshake
func (srv *Server) setupConn(fd net.Conn, flags connFlag, dialDest *discover.Node) {

	// Prevent leftover pending conns from entering the handshake.
	srv.lock.Lock()
	running := srv.running
	srv.lock.Unlock()
	//此时的 srv.newTransport == newRLPX , 因此 transport 类型为 rlpx
	c := &conn{fd: fd, transport: srv.newTransport(fd), flags: flags, cont: make(chan error)}
	//server不在运行状态,关闭
	if !running {
		c.close(errServerStopped)
		return
	}
	// Run the encryption handshake.
	//运行加密握手
	var err error
	if c.id, err = c.doEncHandshake(srv.PrivateKey, dialDest); err != nil {
		loglogrus.Log.Warnf("%v faild enc handshake: %v", c, err)
		c.close(err)
		return
	}
	// For dialed connections, check that the remote public key matches.
	if dialDest != nil && c.id != dialDest.ID {
		c.close(DiscUnexpectedIdentity)
		loglogrus.Log.Warnf("%v dialed identity mismatch, want %x", c, dialDest.ID[:8])
		return
	}
	//调用checkpoint方法,将conn对象c输入到srv.posthandshake管道中(表明我们已经完成了加密handshake)
	if err := srv.checkpoint(c, srv.posthandshake); err != nil {
		loglogrus.Log.Warnf("%v failed checkpoint posthandshake: %v", c, err)
		c.close(err)
		return
	}
	// Run the protocol handshake
	//运行协议的handshake
	phs, err := c.doProtoHandshake(srv.ourHandshake)
	if err != nil {
		loglogrus.Log.Warnf("%v failed proto handshake: %v", c, err)
		c.close(err)
		return
	}
	if phs.ID != c.id {
		loglogrus.Log.Warnf("%v wrong proto handshake identity: %x", c, phs.ID[:8])
		c.close(DiscUnexpectedIdentity)
		return
	}
	c.caps, c.name = phs.Caps, phs.Name
	//调用checkpoint方法,将conn对象c输入到srv.addpeer管道中(表明我们已经完成了协议handshake)
	if err := srv.checkpoint(c, srv.addpeer); err != nil {
		loglogrus.Log.Warnf("%v failed checkpoint addpeer: %v", c, err)
		c.close(err)
		return
	}
	// If the checks completed successfully, runPeer has now been
	// launched by run.
}

// checkpoint sends the conn to run, which performs the
// post-handshake checks for the stage (posthandshake, addpeer).
// 用于向管道中输入信号
func (srv *Server) checkpoint(c *conn, stage chan<- *conn) error {
	select {
	case stage <- c:
	case <-srv.quit:
		return errServerStopped
	}
	select {
	case err := <-c.cont:
		return err
	case <-srv.quit:
		return errServerStopped
	}
}

// runPeer runs in its own goroutine for each peer.
// it waits until the Peer logic returns and removes
// the peer.
// runPeer为peer对象提供一个单独的协议运行run函数
func (srv *Server) runPeer(p *Peer) {
	loglogrus.Log.Infof("Added Protocal peer (NodeID:%x) (srcIPAddr:%s) (destIPAddr:%s) \n", p.ID(), p.rw.fd.LocalAddr().String(),
		p.rw.fd.RemoteAddr().String())
	srvjslog.LogJson(&logger.P2PConnected{
		RemoteId:            p.ID().String(),         //远端peer的NodeID
		RemoteAddress:       p.RemoteAddr().String(), //远端peer的TCP地址
		RemoteVersionString: p.Name(),                //本地与远端peer连接器的名称
		NumConnections:      srv.PeerCount(),         //本地服务器连接的远端peer的数目
	})

	if srv.newPeerHook != nil { //测试用
		srv.newPeerHook(p)
	}
	discreason := p.run() //为此peer对象运行run函数
	// Note: run waits for existing peers to be sent on srv.delpeer
	// before returning, so this send should not select on srv.quit.

	//唯有run函数执行过程中出错才会执行以下内容
	srv.delpeer <- p //将此peer对象添加到待删除管道

	loglogrus.Log.Infof("Remove Protocal peer (NodeID:%x) (srcIPAddr:%s) (destIPAddr:%s) ,err:%v\n", p.ID(), p.rw.fd.LocalAddr().String(),
		p.rw.fd.RemoteAddr().String(), discreason)
	srvjslog.LogJson(&logger.P2PDisconnected{
		RemoteId:       p.ID().String(),
		NumConnections: srv.PeerCount(),
	})
}

// ReadRandomNodes could back num discover.Node urls which are nodes
// already in discover.Table, this func is probably help in test.
// 返回形参num指定个数的Node节点的字符串信息(在Kad路由表中随机获取的)
func (srv *Server) ReadRandomNodes(num int) ([]string, int) {
	nodes := make([]*discover.Node, num)
	readNum := srv.ntab.ReadRandomNodes(nodes) //返回num个Kad路由表中的随机节点,存入nodes集合中
	res := make([]string, 0)
	for _, n := range nodes { //遍历nodes集合
		if n == nil {
			continue
		}
		res = append(res, n.String()) //获取非空的Node节点的字符串表达形式
	}
	return res, readNum
}

// 根据形参指定的nodeID在本地Kad路由表中查询该Node
func (srv *Server) LookUpNode(nodeID common.NodeID) (*discover.Node, error) {
	if srv.ntab == nil {
		return nil, fmt.Errorf("discovery is not launched")
	}
	candidates := srv.ntab.Lookup(discover.NodeID(nodeID)) //在Kad路由表中查询距离目标节点最近的若干Node节点
	for _, node := range candidates {                      //遍历上述查询获得的Node节点集合
		if nodeID == common.NodeID(node.ID) { //存在节点与目标节点的NodeID相同(找到了目标节点)
			return node, nil //返回这个目标节点
		}
	}
	return nil, fmt.Errorf("%v: node not found in discovery", nodeID)
}
