package netInit

// type peer struct {
// 	selfNodeInfo dpnet.Node

// 	remoterNodeID  common.NodeID        //对方节点的nodeID
// 	remoteNodeInfo dpnet.Node           // 需要从对方的握手消息中获取
// 	peer           *server.Peer         //与对方节点的连接控制器
// 	ws             server.MsgReadWriter //与对方节点的读写器

// 	lastHeartBeat time.Time //与对方节点的上次心跳包通信时间

// 	quit chan struct{}
// }

// func newPeer(self dpnet.Node, remote *server.Peer, rw server.MsgReadWriter) *peer {
// 	return &peer{
// 		selfNodeInfo:  self,
// 		remoterNodeID: common.NodeID(remote.ID()),
// 		peer:          remote,
// 		ws:            rw,
// 		lastHeartBeat: time.Now(),
// 		quit:          make(chan struct{}),
// 	}
// }

// func (p *peer) handshake() error {
// 	// Send the handshake status message asynchronously
// 	errc := make(chan error, 1)
// 	go func() {
// 		errc <- server.SendItems(p.ws, statusCode, p.selfNodeInfo) //向对方节点发送协议启动消息（包括自己的节点信息）
// 	}()
// 	// Fetch the remote status packet and verify protocol match
// 	packet, err := p.ws.ReadMsg() //获取远端主机节点回复的状态包(同样也是server模块)
// 	if err != nil {
// 		return err
// 	}
// 	if packet.Code != statusCode {
// 		loglogrus.Log.Errorf("[NetInit] NetInit handshake failed, packet.Code(%d) != statusCode(%d)\n", packet.Code, statusCode)
// 		return fmt.Errorf("peer sent %x before status packet", packet.Code)
// 	}
// 	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
// 	if _, err := s.List(); err != nil {
// 		loglogrus.Log.Errorf("[NetInit] NetInit handshake failed, rlp解析失败, err:%v\n", err)
// 		return fmt.Errorf("bad status message: %v", err)
// 	}
// 	otherNode := dpnet.Node{}

// 	if err = s.Decode(&otherNode); err != nil { // 从握手消息中获得对方节点的Node信息
// 		loglogrus.Log.Errorf("[NetInit] NetInit handshake failed, rlp解码失败, err:%v\n", err)
// 		return fmt.Errorf("bad status message: %v", err)
// 	}

// 	p.remoteNodeInfo = otherNode

// 	// Wait until out own status is consumed too
// 	if err := <-errc; err != nil {
// 		return fmt.Errorf("failed to send status packet: %v", err)
// 	}
// 	return nil
// }
