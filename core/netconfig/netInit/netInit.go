package netInit

// const (
// 	protocolName           = "netInit"
// 	protocolVersion uint64 = 0x00 // ver 0.0

// 	statusCode = 0x00 // for handshake
// )

// type NetInit struct {
// 	SelfInfo dpnet.Node

// 	OthersInfo      []dpnet.Node // 同一分区的其他节点的NodeInfo(通过初始化通信得到)
// 	OthersInfoMutex sync.RWMutex

// 	LeadersInfo      []dpnet.Node // 网络中各分区Leader的NodeInfo
// 	LeadersInfoMutex sync.RWMutex

// 	BootersInfo      []dpnet.Node // 网络中所有Booter节点的NodeInfo
// 	BootersInfoMutex sync.RWMutex

// 	AllNodesInfo      []dpnet.Node // 存储收集到的所有节点的NodeInfo
// 	AllNodesInfoMutex sync.RWMutex

// 	protocol server.Protocol // 基于p2p server模块进行通信
// }

// func NewNetInfo(self dpnet.Node) *NetInit {
// 	netInit := &NetInit{
// 		SelfInfo:    self,
// 		OthersInfo:  make([]dpnet.Node, 0),
// 		LeadersInfo: make([]dpnet.Node, 0),
// 		BootersInfo: make([]dpnet.Node, 0),
// 	}

// 	netInit.protocol = server.Protocol{
// 		Name:    protocolName,
// 		Version: uint(protocolVersion),
// 		Length:  2,
// 		Run:     netInit.handlePeer,
// 	}

// 	return netInit
// }

// // 向这个p2p连接节点发送自己的 dpnet.Node, 同时接收对方节点的 dpnet.Node
// func (netInit *NetInit) handlePeer(peer *server.Peer, rw server.MsgReadWriter, ctx context.Context) error {
// 	netInitPeer := newPeer(netInit.SelfInfo, peer, rw)

// 	loglogrus.Log.Infof("[NetInit] 与节点 (%x) 进行 NetInit handshake!\n", netInitPeer.remoterNodeID)

// 	//向对方节点发送 NetInit 协议启动消息，并等待回复
// 	if err := netInitPeer.handshake(); err != nil {
// 		loglogrus.Log.Warnf("[NetInit] 协议handshake失败,err:%v\n", err)
// 		return err
// 	} else {
// 		otherNode := netInitPeer.remoteNodeInfo
// 		loglogrus.Log.Infof("[NetInit] 完成与节点(%x)的 handshake. 该节点的 NetID(%s) Role(%d)\n", otherNode.NodeID, otherNode.NetID, otherNode.Role)

// 		netInit.AllNodesInfoMutex.Lock()
// 		netInit.AllNodesInfo = append(netInit.AllNodesInfo, otherNode)
// 		netInit.AllNodesInfoMutex.Unlock()

// 		if otherNode.Role == dpnet.Booter {
// 			netInit.BootersInfoMutex.Lock()
// 			netInit.BootersInfo = append(netInit.BootersInfo, otherNode)
// 			netInit.BootersInfoMutex.Unlock()
// 		}

// 		if otherNode.Role == dpnet.Leader {
// 			netInit.LeadersInfoMutex.Lock()
// 			netInit.LeadersInfo = append(netInit.LeadersInfo, otherNode)
// 			netInit.LeadersInfoMutex.Unlock()
// 		}

// 		if otherNode.NetID == netInitPeer.selfNodeInfo.NetID {
// 			netInit.OthersInfoMutex.Lock()
// 			netInit.OthersInfo = append(netInit.OthersInfo, otherNode)
// 			netInit.OthersInfoMutex.Unlock()
// 		}
// 	}

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return nil
// 		default:
// 			time.Sleep(time.Second)
// 		}
// 	}
// }

// func (netInit *NetInit) Protocol() server.Protocol {
// 	return netInit.protocol
// }

// func (netInit *NetInit) Version() uint {
// 	return netInit.protocol.Version
// }

// func (netInit *NetInit) BackAllBooters() []dpnet.Node {
// 	res := make([]dpnet.Node, 0)

// 	netInit.BootersInfoMutex.RLock()
// 	defer netInit.BootersInfoMutex.RUnlock()

// 	res = append(res, netInit.BootersInfo...)

// 	return res
// }

// func (netInit *NetInit) BackAllLeaders() []dpnet.Node {
// 	res := make([]dpnet.Node, 0)

// 	netInit.LeadersInfoMutex.RLock()
// 	defer netInit.LeadersInfoMutex.RUnlock()

// 	res = append(res, netInit.LeadersInfo...)

// 	return res
// }

// func (netInit *NetInit) BackAllOthers() []dpnet.Node {
// 	res := make([]dpnet.Node, 0)

// 	netInit.OthersInfoMutex.RLock()
// 	defer netInit.OthersInfoMutex.RUnlock()

// 	res = append(res, netInit.OthersInfo...)

// 	return res
// }

// func (netInit *NetInit) BackAllNodes() []dpnet.Node {
// 	res := make([]dpnet.Node, 0)

// 	netInit.AllNodesInfoMutex.RLock()
// 	defer netInit.AllNodesInfoMutex.RUnlock()

// 	res = append(res, netInit.AllNodesInfo...)

// 	return res
// }
