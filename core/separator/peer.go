package separator

import (
	"fmt"
	"p3Chain/common"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/p2p/server"
	"p3Chain/rlp"
	"time"
)

// peer represents a separator protocol peer connection.
type peer struct {
	nodeID common.NodeID        //对方节点的nodeID
	host   *Separator           //本机的separator协议对象，包含本节点保存的整个separator网络的信息
	peer   *server.Peer         //与对方节点的连接控制器
	ws     server.MsgReadWriter //与对方节点的读写器

	lastHeartBeat time.Time //与对方节点的上次心跳包通信时间

	quit chan struct{}
}

// 在与对方节点的P2P连接的基础上，构建与对象的separator协议通信对象
func newPeer(host *Separator, remote *server.Peer, rw server.MsgReadWriter) *peer {
	return &peer{
		nodeID:        common.NodeID(remote.ID()),
		host:          host,
		peer:          remote,
		ws:            rw,
		lastHeartBeat: time.Now(),
		quit:          make(chan struct{}),
	}
}

// hand shake send the protocol initiation status message to the remote peer and
// verifies the remote status too.
// 通过handshake操作向对方节点发送separator协议启动消息，然后等待接收并验证对方节点状态是否正常。
func (p *peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		errc <- server.SendItems(p.ws, statusCode, protocolVersion) //向对方节点发送协议启动消息(用server模块发送)
	}()
	// Fetch the remote status packet and verify protocol match
	packet, err := p.ws.ReadMsg() //获取远端主机节点回复的状态包(同样也是server模块)
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("peer sent %x before status packet", packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	if _, err := s.List(); err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	if peerVersion != protocolVersion {
		return fmt.Errorf("protocol version mismatch %d != %d", peerVersion, protocolVersion)
	}
	// Wait until out own status is consumed too
	if err := <-errc; err != nil {
		return fmt.Errorf("failed to send status packet: %v", err)
	}
	return nil
}

// 调用节点p2p的server模块向着与对方节点的读写器写入msg消息(message包形式)
func (p *peer) sendMsg(msg *Message) error {
	if err := server.Send(p.ws, msg.MsgCode, msg); err != nil {
		return err
	}
	loglogrus.Log.Infof("Separator: Send msg to peer: %x, msgcode: %v\n", p.nodeID, msg.MsgCode)
	return nil
}
