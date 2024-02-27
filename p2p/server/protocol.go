package server

import (
	"context"
	"fmt"
)

// Protocol represents a P2P subprotocol implementation.
type Protocol struct {
	// Name should contain the official protocol name,
	// often a three-letter word.
	Name string //P2P之上的协议名称

	// Version should contain the version number of the protocol.
	Version uint //协议的版本号

	// Length should contain the number of message codes used
	// by the protocol.
	Length uint64 //包含协议使用的message code数目

	// Run is called in a new groutine when the protocol has been
	// negotiated with a peer. It should read and write messages from
	// rw. The Payload for each message must be fully consumed.
	//
	// The peer connection is closed when Start returns. It should return
	// any protocol-level error (such as an I/O error) that is
	// encountered.
	//Run函数当与某一节点实现协议时会在一个新的协程中被调用，它应当从rw读写器中读取和写入message
	//当Start函数返回时与对方节点的连接将被关闭，此时Run函数应该返回一个协议层面的错误提示
	Run func(peer *Peer, rw MsgReadWriter, ctx context.Context) error //这个函数需要由对应的协议完成(whisper协议和separator都各自对其进行了实现)
}

// 返回当前协议的名称、版本号
func (p Protocol) cap() Cap {
	return Cap{p.Name, p.Version}
}

// Cap is the structure of a peer capability.
type Cap struct {
	Name    string
	Version uint
}

// 返回当前协议的名称、版本号(以空接口切片方式)
func (cap Cap) RlpData() interface{} {
	return []interface{}{cap.Name, cap.Version}
}

func (cap Cap) String() string {
	return fmt.Sprintf("%s/%d", cap.Name, cap.Version)
}

// Cap切片,存储当前所有P2P层之上运行的协议信息
type capsByNameAndVersion []Cap

// 下面三个方法是为了实现sort.Sort()方法而实现的
func (cs capsByNameAndVersion) Len() int      { return len(cs) }
func (cs capsByNameAndVersion) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs capsByNameAndVersion) Less(i, j int) bool {
	return cs[i].Name < cs[j].Name || (cs[i].Name == cs[j].Name && cs[i].Version < cs[j].Version)
}
