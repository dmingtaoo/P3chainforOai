package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	loglogrus "p3Chain/log_logrus"
	"p3Chain/p2p/discover"
	"p3Chain/rlp"
)

const (
	baseProtocolVersion    = 4
	baseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 2 * 1024

	pingInterval = 15 * time.Second
)

// message类型
const (
	// devp2p message codes
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
	getPeersMsg  = 0x04
	peersMsg     = 0x05
)

// protoHandshake is the RLP structure of the protocol handshake.
// 协议握手包message 格式
type protoHandshake struct {
	Version    uint64
	Name       string
	Caps       []Cap           //存放所有使用到的协议的Cap结构体(包括协议名/协议版本)
	ListenPort uint64          //服务器监听端口号
	ID         discover.NodeID //自身节点ID
}

// Peer represents a connected remote node.
// 记录已连接的远端节点
type Peer struct {
	rw      *conn               //连接器
	running map[string]*protoRW //协议消息读写器(协议名为键值,协议读写器protoRW为value值)，protoRW结构体实现了MsgWriter和MsgReader接口

	wg       sync.WaitGroup
	protoErr chan error
	closed   chan struct{}
	disc     chan DiscReason //错误通道
}

// NewPeer returns a peer for testing purposes.
// 根据节点ID,自定义连接名称,协议信息构建一个Peer对象
func NewPeer(id discover.NodeID, name string, caps []Cap) *Peer {
	pipe, _ := net.Pipe()                                                   //创建一个双向的Pipe管道,两端在通信时都相当于一个conn socket,直接将数据在两端之间作拷贝,没有内部缓冲。
	conn := &conn{fd: pipe, transport: nil, id: id, caps: caps, name: name} //创建一个conn连接器
	peer := newPeer(conn, nil)                                              //根据conn连接器创建一个peer对象
	close(peer.closed)                                                      // 初始化时，先关闭peer.closed管道，确保Disconnect方法不会开始就被阻塞
	return peer
}

// ID returns the node's public key.
// 从连接器获取对方节点的NodeID
func (p *Peer) ID() discover.NodeID {
	return p.rw.id
}

// Name returns the node name that the remote node advertised.
// 返回连接器的Name名称
func (p *Peer) Name() string {
	return p.rw.name
}

// Caps returns the capabilities (supported subprotocols) of the remote peer.
// 返回与远端节点的连接支持的协议的Cap信息(可能有多个协议)
func (p *Peer) Caps() []Cap {
	// TODO: maybe return copy
	return p.rw.caps
}

// RemoteAddr returns the remote address of the network connection.
// 返回远端节点的网络地址
func (p *Peer) RemoteAddr() net.Addr {
	return p.rw.fd.RemoteAddr()
}

// LocalAddr returns the local address of the network connection.
// 获取本地节点的网络地址
func (p *Peer) LocalAddr() net.Addr {
	return p.rw.fd.LocalAddr()
}

// Disconnect terminates the peer connection with the given reason.
// It returns immediately and does not wait until the connection is closed.
// 本方法负责使用给定的原因终止与远端节点的连接(向错误管道p.disc发送reason)
// 又或者是接收到Peer结构体需要关闭的信号(p.closed收到信号)即会退出当前函数
func (p *Peer) Disconnect(reason DiscReason) {
	select {
	case p.disc <- reason:
	case <-p.closed:
	}
}

// String implements fmt.Stringer.
func (p *Peer) String() string {
	return fmt.Sprintf("Peer %x %v", p.rw.id[:8], p.RemoteAddr())
}

// 根据conn socket和所使用的协议群(可以不止一个协议)创建一个Peer对象
func newPeer(conn *conn, protocols []Protocol) *Peer {
	protomap := matchProtocols(protocols, conn.caps, conn) //protomap是一个键值为协议名,value为协议读写器的map集合
	p := &Peer{
		rw:       conn,
		running:  protomap,
		disc:     make(chan DiscReason),
		protoErr: make(chan error, len(protomap)+1), // protocols + pingLoop  每一个协议读写器有一个自己的错误通道,pingLoop还有一个错误通道
		closed:   make(chan struct{}),
	}
	return p
}

// 让Peer对象开始工作: 1.开启peer的读协程  2.开启peer的ping协程  3.开启若干协程分别执行peer支持的各个协议
// 返回值为任何一种导致当前run()发生错误的错误原因
func (p *Peer) run() DiscReason {
	var (
		writeStart = make(chan struct{}, 1) //用于通知开始写操作
		writeErr   = make(chan error, 1)    //写入错误通道
		readErr    = make(chan error, 1)    //读取错误通道
		reason     DiscReason               //关闭连接时使用的reason信号
		requested  bool
	)
	p.wg.Add(2)
	go p.readLoop(readErr) //启动节点的读协程
	go p.pingLoop()        //启动节点的ping协程

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Start all protocol handlers.
	// 启动所有P2P层之上协议的handshake(即在P2P层之上让节点之间开始建立起这些协议)
	writeStart <- struct{}{}                    //当传入p.startProtocols后会变成一个只读管道(在startProtocols函数中作为一个只读管道)
	p.startProtocols(writeStart, writeErr, ctx) //开启若干协程,负责调用各协议的Run函数(相当于节点已经开始运行这些协议)

	// Wait for an error or disconnect.
	//循环等待错误或者节点断连
loop:
	for {
		select {
		case err := <-writeErr: //协议执行过程中发生了错误
			// A write finished. Allow the next write to start if
			// there was no error.
			if err != nil { //错误原因不为nil,直接退出循环(找到了错误原因)
				loglogrus.Log.Warnf("%v: write error: %v\n", p, err)
				reason = DiscNetworkError
				break loop
			}
			writeStart <- struct{}{} //错误原因为nil,重新启动协议的Run函数
		case err := <-readErr: //节点的读协程发生错误
			if r, ok := err.(DiscReason); ok { //与远端节点因连接断开导致的错误
				loglogrus.Log.Warnf("%v: remote requested disconnect: %v\n", p, r)
				requested = true
				reason = r
			} else { //其他读取错误
				reason = DiscNetworkError
			}

			break loop //找到错误原因,退出循环
		case err := <-p.protoErr: //发送ping消息时出现错误
			reason = discReasonForError(err)
			loglogrus.Log.Warnf("%v: protocol error: %v (%v)\n", p, err, reason)
			break loop
		case reason = <-p.disc: //本地主动与对端peer结束了连接
			loglogrus.Log.Warnf("%v: locally requested disconnect: %v\n", p, reason)
			break loop
		}
	}
	close(p.closed)
	p.rw.close(reason)
	p.wg.Wait()
	if requested {
		reason = DiscRequested
	}
	return reason
}

// 负责定时循环发送ping消息给对方Peer
func (p *Peer) pingLoop() {
	ping := time.NewTicker(pingInterval) //P2P节点之间心跳包发送间隔:15s
	defer p.wg.Done()
	defer ping.Stop()
	for {
		select {
		case <-ping.C: //定时发送pingMsg消息(只有msgcode,没有实际的data)
			if err := SendItems(p.rw, pingMsg); err != nil {
				loglogrus.Log.Warnf("heart Beate, ping协程错误:%v\n", err)
				p.protoErr <- err //发送时产生错误,将错误提示传入protoErr管道,同时结束ping协程

				return
			}
		case <-p.closed: //closed管道被关闭时，退出当前ping协程
			loglogrus.Log.Warnf("heart beate, ping协程退出\n")
			return
		}
	}
}

// 负责从对端peer循环读取Msg消息,然后调用handle进行处理
func (p *Peer) readLoop(errc chan<- error) {
	defer p.wg.Done()
	for { //循环等待读取
		msg, err := p.rw.ReadMsg() //调用ReadMsg()读取消息(conn类并未找到对ReadMsg方法的实现？？？)
		if err != nil {
			errc <- err
			return
		}
		msg.ReceivedAt = time.Now()          //更新Msg消息的接收时刻
		if err = p.handle(msg); err != nil { //对收到的Msg消息进行处理
			errc <- err
			return
		}
	}
}

// 根据收到的Msg消息的类型进行分类处理(确定如何进行回复)
func (p *Peer) handle(msg Msg) error {
	switch {
	case msg.Code == pingMsg: //收到的是ping消息
		msg.Discard()               //清空读取缓冲区
		go SendItems(p.rw, pongMsg) //新的协程负责通过conn socket向对方回复pong消息
	case msg.Code == discMsg: //discMsg类型消息用于作为通信结束的标志,因此不需要msg.Discard()和检查错误,收到此消息之后conn连接将会断开
		var reason [1]DiscReason
		// This is the last message. We don't need to discard or
		// check errors because, the connection will be closed after it.
		rlp.Decode(msg.Payload, &reason) //读取结束原因，存入reason
		return reason[0]
	case msg.Code < baseProtocolLength: //如果收到其他基类协议消息则忽略,清空读取缓冲区
		// ignore other base protocol messages
		return msg.Discard()
	default: //不符合上述三种情况，表示消息是子类协议的消息
		// it's a subprotocol message
		proto, err := p.getProto(msg.Code) //根据msg.Code获取其对应的子类协议读写器
		if err != nil {
			return fmt.Errorf("msg code out of range: %v", msg.Code)
		}
		select {
		case proto.in <- msg: //将要发送的msg消息输入到协议读写器的in管道
			return nil
		case <-p.closed:
			return io.EOF
		}
	}
	return nil
}

// 检查protocols和caps相吻合的元素数目
func countMatchingProtocols(protocols []Protocol, caps []Cap) int {
	n := 0
	for _, cap := range caps {
		for _, proto := range protocols {
			if proto.Name == cap.Name && proto.Version == cap.Version {
				n++
			}
		}
	}
	return n
}

// matchProtocols creates structures for matching named subprotocols.
// 将协议的cap与对应的协议读写器相绑定,最后返回一个 协议名--协议读写器 的map集合
func matchProtocols(protocols []Protocol, caps []Cap, rw MsgReadWriter) map[string]*protoRW {
	sort.Sort(capsByNameAndVersion(caps)) //对caps内存储的所有协议的cap进行排序
	offset := baseProtocolLength          //子类协议的偏移量等于基类协议总数(这里为16)
	result := make(map[string]*protoRW)   //用于存储所有协议的协议读写器

outer:
	for _, cap := range caps { //遍历传入形参caps(每个cap只有协议名跟协议版本号)
		for _, proto := range protocols { //遍历传入形参protocols(有协议名、协议版本号、length、Run函数)
			if proto.Name == cap.Name && proto.Version == cap.Version {
				// If an old protocol version matched, revert it
				if old := result[cap.Name]; old != nil {
					offset -= old.Length
				}
				// Assign the new match
				result[cap.Name] = &protoRW{Protocol: proto, offset: offset, in: make(chan Msg), w: rw}
				offset += proto.Length

				continue outer
			}
		}
	}
	return result
}

// 为所有的节点支持的协议开启单独的协程,让这些协程负责执行协议自行实现的Run函数(whisper协议和separator协议都存在自行实现的Run)
func (p *Peer) startProtocols(writeStart <-chan struct{}, writeErr chan<- error, ctx context.Context) {
	p.wg.Add(len(p.running))          //等待组添加的个数==协议的个数(即是说我们需要等待全部协议读写器完成Run操作之后才行)
	for _, proto := range p.running { //遍历所有协议读写器
		proto := proto
		proto.closed = p.closed
		proto.wstart = writeStart
		proto.werr = writeErr
		go func() { //采用协程各自执行协议的Run函数
			p.wg.Done()                     //每成功执行一个协议的Run,等待组-1
			err := proto.Run(p, proto, ctx) //调用对应协议自行实现的Run函数 (protoRW实现了MsgReadWriter接口，因此可以作为实参传入)
			if err == nil {
				err = errors.New("protocol returned")
			} else if err != io.EOF {
				loglogrus.Log.Warnf("%v: Protocol %s/%d error: %v\n", p, proto.Name, proto.Version, err)
			}
			loglogrus.Log.Warnf("协议handshake, 协议Run函数错误,err:%v\n", err)
			p.protoErr <- err

		}()
	}
}

// getProto finds the protocol responsible for handling
// the given message code.
// 负责查找处理给定message code的协议(返回对应的协议读写器)
func (p *Peer) getProto(code uint64) (*protoRW, error) {
	for _, proto := range p.running { //遍历当前节点所有支持的协议读写器
		if code >= proto.offset && code < proto.offset+proto.Length { //查找符合code条件的协议读写器
			return proto, nil
		}
	}
	return nil, newPeerError(errInvalidMsgCode, "%d", code)
}

// 协议的读写器
type protoRW struct {
	Protocol                 //继承Protocol类
	in       chan Msg        // receices read messages   用于输入Msg消息的管道(从handle函数的default情况中获取)
	closed   <-chan struct{} // receives when peer is shutting down   当peer节点关闭时会收到信号
	wstart   <-chan struct{} // receives when write may start   一旦从管道收到信号,就需要发送Msg消息
	werr     chan<- error    // for write results     用于输入发送Msg消息时发生的错误
	offset   uint64          //相对于基类协议的偏移量
	w        MsgWriter       //用于发送Msg消息
}

// 协议读写器实现的 WriteMsg 函数(实现了MsgWriter接口)
func (rw *protoRW) WriteMsg(msg Msg) (err error) {
	if msg.Code >= rw.Length { //要发送的Msg消息的Code不合法(超过当前协议能表示的范围)
		return newPeerError(errInvalidMsgCode, "not handled")
	}
	msg.Code += rw.offset //msg消息的Code需要在基类协议code的基础上进行偏移
	select {
	case <-rw.wstart: //收到启动信号,发送Msg消息
		err = rw.w.WriteMsg(msg)
		// Report write status back to Peer.run. It will initiate
		// shutdown if the error is non-nil and unblock the next write
		// otherwise. The calling protocol code should exit for errors
		// as well but we don't want to rely on that.
		rw.werr <- err //返回错误消息
	case <-rw.closed:
		err = fmt.Errorf("shutting down")
	}
	return err
}

// 协议读写器实现的 ReadMsg 函数(实现了MsgReader接口)
func (rw *protoRW) ReadMsg() (Msg, error) {
	select {
	case msg := <-rw.in: //从in管道读取Msg消息
		msg.Code -= rw.offset //回复msg.Code
		return msg, nil       //返回读取的数据
	case <-rw.closed:
		return Msg{}, io.EOF
	}
}
