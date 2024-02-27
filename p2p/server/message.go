package server

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"p3Chain/rlp"
)

// Msg defines the structure of a p2p message.
//
// Note that a Msg can only be sent once since the Payload reader is
// consumed during sending. It is not possible to create a Msg and
// send it any number of times. If you want to reuse an encoded
// structure, encode the payload into a byte array and create a
// separate Msg with a bytes.Reader as Payload for each send.
// Msg结构体定义了节点之间P2P message的格式
// 注意: 由于消息的发送与读取是直接以RLP字节流方式,因此一旦消息被读取就不可再次重复读取,必须要发送端重新对消息编码之后再发送
type Msg struct {
	Code       uint64    //Msg的类型
	Size       uint32    //Msg消息有效载荷的字节长度
	Payload    io.Reader //从哪里读取Payload载荷(结合上述Size字段进行定长字节读取)
	ReceivedAt time.Time //接收时刻
}

// Decode parses the RLP content of a message into
// the given value, which must be a pointer.
//
// For the decoding rules, please see package rlp.
// Decode方法负责解析RLP流,将其转化并直接写入到形参val结构体各字段中,因此val必须是一个结构体地址
func (msg Msg) Decode(val interface{}) error {
	s := rlp.NewStream(msg.Payload, uint64(msg.Size)) //从msg.Payload读取msg.Size长度的字节流
	if err := s.Decode(val); err != nil {             //将字节流消息解析传输到val中
		return newPeerError(errInvalidMsg, "(code %x) (size %d) %v", msg.Code, msg.Size, err)
	}
	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("msg #%v (%v bytes)", msg.Code, msg.Size)
}

// Discard reads any remaining payload data into a black hole.
// Discard方法负责从msg.Payload指定的io.Reader的缓冲区中读取所有剩余内容,将其消耗掉(扔到ioutil.Discard黑洞中)
func (msg Msg) Discard() error {
	_, err := io.Copy(ioutil.Discard, msg.Payload)
	return err
}

// 负责接收P2P节点之间通信的Msg消息
type MsgReader interface {
	ReadMsg() (Msg, error)
}

// 负责在P2P节点之间发送Msg消息
type MsgWriter interface {
	// WriteMsg sends a message. It will block until the message's
	// Payload has been consumed by the other end.
	//
	// Note that messages can be sent only once because their
	// payload reader is drained.

	// WriteMsg函数在发送Msg消息后会一直阻塞,直到对端读取了PayLoad
	// 需注意:每条Msg消息只能发送一次
	WriteMsg(Msg) error
}

// MsgReadWriter provides reading and writing of encoded messages.
// Implementations should ensure that ReadMsg and WriteMsg can be
// called simultaneously from multiple goroutines.
type MsgReadWriter interface {
	MsgReader //继承
	MsgWriter //继承
}

// Send writes an RLP-encoded message with the given code.
// data should encode as an RLP list.
// Send函数将对data数据进行rlp编码,然后将其与Msg Code 、Msg Size 合并为RLP list后交给 w.WriteMsg 完成发送
func Send(w MsgWriter, msgcode uint64, data interface{}) error {
	size, r, err := rlp.EncodeToReader(data) //将给的的data数据进行rlp编码,然后返回一个io.Reader可以读取编码后的数据
	if err != nil {
		return err
	}
	return w.WriteMsg(Msg{Code: msgcode, Size: uint32(size), Payload: r})
}

// SendItems writes an RLP with the given code and data elements.
// For a call such as:
//
//    SendItems(w, code, e1, e2, e3)
//
// the message payload will be an RLP list containing the items:
//
//    [e1, e2, e3]

// 调用Send函数完成对elems 若干对数据的发送
func SendItems(w MsgWriter, msgcode uint64, elems ...interface{}) error {
	return Send(w, msgcode, elems)
}

// netWrapper wraps a MsgReadWriter with locks around
// ReadMsg/WriteMsg and applies read/write deadlines.
// 为MsgReadWriter读写器的读和写分别绑定保护锁和等待截止时间
type netWrapper struct {
	rmu, wmu sync.Mutex

	rtimeout, wtimeout time.Duration //发送/接收消息时的timeout时间
	conn               net.Conn      //conn socket
	wrapped            MsgReadWriter //读写器接口(外界需要实现这个接口)
}

// 实现ReadMsg()
func (rw *netWrapper) ReadMsg() (Msg, error) {
	rw.rmu.Lock()
	defer rw.rmu.Unlock()
	rw.conn.SetReadDeadline(time.Now().Add(rw.rtimeout)) //设置接收等待截止时间
	return rw.wrapped.ReadMsg()                          //等待读取msg消息
}

// 实现WriteMsg
func (rw *netWrapper) WriteMsg(msg Msg) error {
	rw.wmu.Lock()
	defer rw.wmu.Unlock()
	rw.conn.SetWriteDeadline(time.Now().Add(rw.wtimeout)) //设置发送等待截止时间
	return rw.wrapped.WriteMsg(msg)                       //发送msg消息
}

// eofSignal wraps a reader with eof signaling. the eof channel is
// closed when the wrapped reader returns an error or when count bytes
// have been read.

// eofSignal 用eof信号包装一个 io.Reader。
// eof管道会在io.Reader读取发生错误或者是当已经读取了指定count的字节数(count==0)时会被关闭
type eofSignal struct {
	wrapped io.Reader
	count   uint32          // number of bytes left  剩余还未读取的字节数
	eof     chan<- struct{} //单向管道(只写)
}

// note: when using eofSignal to detect whether a message payload
// has been read, Read might not be called for zero sized messages.
// Read方法可能无法检测到长度为0的消息的读取情况
func (r *eofSignal) Read(buf []byte) (int, error) {
	if r.count == 0 { //r.count==0,说明已经读取完毕,那么本次没有读取任何字节(返回0),返回io.EOF作为文末标志
		if r.eof != nil {
			r.eof <- struct{}{}
			r.eof = nil //相当于关闭了eof管道
		}
		return 0, io.EOF
	}

	max := len(buf)              //可能需要读取的最大字节数(就是传入缓冲区buf的长度,但读取的消息不一定填满整个buf缓冲区)
	if int(r.count) < len(buf) { //仅读取指定count字节数的字节流
		max = int(r.count)
	}
	n, err := r.wrapped.Read(buf[:max])               //仅读取指定count字节数的字节流
	r.count -= uint32(n)                              //r.count需要减去读取成功的字节数
	if (err != nil || r.count == 0) && r.eof != nil { //读取过程中发生了错误或者r.count == 0时,需要关闭eof管道(结束本次读取)
		r.eof <- struct{}{} // tell Peer that msg has been consumed  管道写入消息的意义是:通知节点当前msg消息被认为已被完成读取
		r.eof = nil
	}
	return n, err
}

// MsgPipe creates a message pipe. Reads on one end are matched
// with writes on the other. The pipe is full-duplex, both ends
// implement MsgReadWriter.
// MsgPipe负责创建一对message pipe ,每个pipe是双工的(一对单向管道组成)
func MsgPipe() (*MsgPipeRW, *MsgPipeRW) {
	var (
		c1, c2  = make(chan Msg), make(chan Msg)      //两个Msg消息管道,一个用于只写,一个用于只读
		closing = make(chan struct{})                 //用于通知关闭的管道
		closed  = new(int32)                          //是否已经关闭(标志位)
		rw1     = &MsgPipeRW{c1, c2, closing, closed} //Msg Pipe
		rw2     = &MsgPipeRW{c2, c1, closing, closed} //Msg Pipe
	)
	return rw1, rw2
}

// ErrPipeClosed is returned from pipe operations after the
// pipe has been closed.

// ErrPipeClosed 在pipe关闭后负责作为提示消息
var ErrPipeClosed = errors.New("p2p: read or write on closed message pipe")

// MsgPipeRW is an endpoint of a MsgReadWriter pipe.
type MsgPipeRW struct {
	w       chan<- Msg    //Msg消息的写入管道
	r       <-chan Msg    //Msg消息的读取管道
	closing chan struct{} //pipe关闭管道
	closed  *int32        //标志位(是否已关闭)
}

// WriteMsg sends a messsage on the pipe.
// It blocks until the receiver has consumed the message payload.

// 向MsgPipe的写端写入msg消息(只写入payload载荷)，然后阻塞等待收端的consumed信号
func (p *MsgPipeRW) WriteMsg(msg Msg) error {
	if atomic.LoadInt32(p.closed) == 0 { //p.closed == 0 表示pipe尚未被关闭
		consumed := make(chan struct{}, 1)                        //双向带缓冲管道
		msg.Payload = &eofSignal{msg.Payload, msg.Size, consumed} //任何实现了Read方法的对象都可以作为io.Reader使用
		select {
		case p.w <- msg: //将msg消息写入pipe的写端
			if msg.Size > 0 {
				// wait for payload read or discard
				select {
				case <-consumed: //等待consumed管道产生信号,也就是eofSignal的Read方法读取指定长度字节流或读取出错时
				case <-p.closing: //pipe被关闭了
				}
			}
			return nil //此时意味着:1.要么是consumed成功(也就是完成了读写) 2.要么是中途pipe被关闭了
		case <-p.closing:
		}
	}
	return ErrPipeClosed //p.closed != 0 表示pipe已经被关闭,需要产生错误提示
}

// ReadMsg returns a message sent on the other end of the pipe.
// 在Msg Pipe 的读端等待读取msg消息
func (p *MsgPipeRW) ReadMsg() (Msg, error) {
	if atomic.LoadInt32(p.closed) == 0 { //确保pipe未被关闭
		select {
		case msg := <-p.r: //等待pipe读端的消息
			return msg, nil //将消息返回
		case <-p.closing:
		}
	}
	return Msg{}, ErrPipeClosed
}

// Close unblocks any pending ReadMsg and WriteMsg calls on both ends
// of the pipe. They will return ErrPipeClosed. Close also
// interrupts any reads from a message payload.
func (p *MsgPipeRW) Close() error {
	if atomic.AddInt32(p.closed, 1) != 1 {
		// someone else is already closing
		atomic.StoreInt32(p.closed, 1) // avoid overflow
		return nil
	}
	close(p.closing) //关闭pipe管道
	return nil
}

// ExpectMsg reads a message from r and verifies that its
// code and encoded RLP content match the provided values.
// If content is nil, the payload is discarded and not verified.

// ExpectMsg负责从r读取消息,然后根据给定的code和content确定读取的msg是否和预期的相同
func ExpectMsg(r MsgReader, code uint64, content interface{}) error {
	msg, err := r.ReadMsg() //读取msg消息
	if err != nil {
		return err
	}
	if msg.Code != code { //检验msg消息的消息码是否正确
		return fmt.Errorf("message code mismatch: got %d, expected %d", msg.Code, code)
	}
	if content == nil { //如果认为此次为无效读取,则丢弃之后所有的缓冲数据
		return msg.Discard()
	} else { //如果认为此次为有效读取
		contentEnc, err := rlp.EncodeToBytes(content) //1.首先将传入的content编码为rlp字节流
		if err != nil {
			panic("content encode error: " + err.Error())
		}
		if int(msg.Size) != len(contentEnc) { //检验数据流字节长度是否正确
			return fmt.Errorf("message size mismatch: got %d, want %d", msg.Size, len(contentEnc))
		}
		actualContent, err := ioutil.ReadAll(msg.Payload) //从msg.Payload指定的io.Reader中读取全部字节流
		if err != nil {
			return err
		}
		if !bytes.Equal(actualContent, contentEnc) { //检验msg.Payload中读取的数据与rlp编码后的content是否完全相等
			return fmt.Errorf("message payload mismatch:\ngot:  %x\nwant: %x", actualContent, contentEnc)
		}
	}
	return nil //所有检测无误后,返回nil
}

// for test
func unhex(str string) []byte {
	b, err := hex.DecodeString(strings.Replace(str, "\n", "", -1))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}
