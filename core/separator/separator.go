package separator

import (
	"context"
	"fmt"
	"p3Chain/common"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/server"
	"sync"
	"time"
)

const (
	statusCode                = 0x00 // for handshake
	messageCode               = 0x01 // plain text use this code,
	MessageCode               = 0x01
	lowerConsensusCode        = 0x02 // code about lower consensus
	upperConsensusCode        = 0x03 // code about upper consensus
	controlCode               = 0x04 // control information, and maybe not necessary
	transactionCode           = 0x05 // transactions use this code
	TransactionCode           = 0x05
	transactionsCode          = 0x06 // series of transactions
	TransactionsCode          = 0x06 // series of transactions
	intraWSUpdateCode         = 0x07 // intra-shard world state update
	protocolVersion    uint64 = 0x00 // ver 0.0

	protocolName = "spr"

	bindCycle = 5 * time.Second // interval for binding known peers to subnet
)

// Separator is a protocol to construct subnets
// based on the P2P communication layer.
// Separator协议在whisper协议实现节点P2P通信的基础上，将P2P网络组织化，构成多个子网的形式
type Separator struct {
	protocol   server.Protocol       //继承Protocal类，需自行实现separator协议功能
	PeerGroups map[string]*PeerGroup //整个DpNet中所有的节点组组成的集合

	groupMu sync.Mutex // sync the peer groups

	addedPeers map[common.NodeID]*discover.Node // added to static nodes
	knownPeers map[common.NodeID]*peer          //保存所有与当前节点存在活跃separator连接的其他节点集合

	staticMu sync.Mutex // Mutex to sync the static nodes to be added
	peerMu   sync.Mutex // Mutex to sync the active known peers
	quit     chan struct{}
}

// 记录断连节点的信息
type UnKnownPeer struct {
	nodeID  common.NodeID
	groupID string
}

// Create a new separator through the base P2P network.
// 新建一个Separator对象,自主实现protocal类,协议Run函数为separator.handlePeer
func New() *Separator {
	separator := &Separator{
		PeerGroups: make(map[string]*PeerGroup),
		addedPeers: make(map[common.NodeID]*discover.Node),
		knownPeers: make(map[common.NodeID]*peer),
		quit:       make(chan struct{}),
	}

	separator.protocol = server.Protocol{
		Name:    protocolName,
		Version: uint(protocolVersion),
		Length:  8,
		Run:     separator.handlePeer,
	}

	return separator
}

// 获取当前separator对象支持的协议Protocal对象
func (spr *Separator) Protocol() server.Protocol {
	return spr.protocol
}

// 获取当前separator对象支持的协议版本
func (spr *Separator) Version() uint {
	return spr.protocol.Version
}

// 返回整个KnownPeers集合
func (spr *Separator) GetKnownPeers() map[common.NodeID]*peer {

	var peers map[common.NodeID]*peer = make(map[common.NodeID]*peer)
	for i, v := range spr.knownPeers {
		peers[i] = v
	}
	return peers
}

// 返回所有保存的PeerGroup的NetID
func (spr *Separator) GetPeerGroup() []string {
	var netID []string = make([]string, 0)

	for i, _ := range spr.PeerGroups {
		netID = append(netID, i)
	}
	return netID
}

// 参数分别是本节点与远端peer(已经建立了p2p连接)的连接控制器、读写器
func (spr *Separator) handlePeer(peer *server.Peer, rw server.MsgReadWriter, ctx context.Context) error {
	// Create new peer and add it to known list.
	separatorPeer := newPeer(spr, peer, rw) //在p2p连接的基础上与对方peer建立separator协议
	spr.peerMu.Lock()
	spr.knownPeers[separatorPeer.nodeID] = separatorPeer //在knownPeers记录本节点与对端节点的separator连接控制器
	spr.peerMu.Unlock()

	loglogrus.Log.Infof("与节点(%x)进行separator handshake!\n", separatorPeer.nodeID)

	defer func() {
		spr.peerMu.Lock()
		loglogrus.Log.Infof("与节点(%x)断开separator handshake!\n", separatorPeer.nodeID)
		delete(spr.knownPeers, separatorPeer.nodeID) //无法正常收到对端节点的消息时，将此对端节点从knownPeers切片中移除
		spr.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	//向对方节点发送separator协议启动消息，并等待回复
	if err := separatorPeer.handshake(); err != nil {
		loglogrus.Log.Warnf("separator: 协议handshake失败,err:%v\n", err)
		return err
	}

	// Read and process inbound messages directly to merge into attached group messages.
	//循环等待对端节点的消息
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Fetch the next packet and decode it.
			packet, err := rw.ReadMsg() //再次等待对端节点的消息，根据message包的消息码分类处理
			if err != nil {
				return err
			}
			switch packet.Code {
			case MessageCode: //表示收到了对方节点的普通message消息
				var msg Message
				if err := packet.Decode(&msg); err != nil { //对message包进行rlp解码
					glog.V(logger.Warn).Infof("%v: failed to decode msg: %v", peer, err)
					continue
				}
				//fmt.Printf("收到Message消息：separator协议:消息内容--发送方NodeID: %x\n", msg.From)
				//fmt.Println("收到Message消息：separator协议:消息内容--目标子网NetID: ", msg.NetID)
				if err := spr.ReadMsg(&msg); err != nil { //由指定的子网负责读取该msg消息
					glog.V(logger.Warn).Infof("%v: failed in add msg: %v", peer, err)
				}
			case TransactionCode:
				var msg Message
				if err := packet.Decode(&msg); err != nil { //对message包进行rlp解码
					glog.V(logger.Warn).Infof("%v: failed to decode msg: %v", peer, err)
					continue
				}
				//fmt.Printf("收到Transaction消息：separator协议:消息内容--发送方NodeID: %x\n", msg.From)
				//fmt.Println("收到Transaction消息：separator协议:消息内容--目标子网NetID: ", msg.NetID)
				if err := spr.ReadMsg(&msg); err != nil { //由指定的子网负责读取该msg消息
					glog.V(logger.Warn).Infof("%v: failed in add msg: %v", peer, err)
				}
			case TransactionsCode:
				var msg Message
				if err := packet.Decode(&msg); err != nil { //对message包进行rlp解码
					glog.V(logger.Warn).Infof("%v: failed to decode msg: %v", peer, err)
					continue
				}
				if err := spr.ReadMsg(&msg); err != nil { //由指定的子网负责读取该msg消息
					glog.V(logger.Warn).Infof("%v: failed in add msg: %v", peer, err)
				}

			case lowerConsensusCode: //表示收到了对方的低共识消息
				var msg Message
				if err := packet.Decode(&msg); err != nil {
					glog.V(logger.Warn).Infof("%v: failed to decode msg: %v", peer, err)
					continue
				}
				if msg.From != separatorPeer.nodeID {
					glog.V(logger.Warn).Infof("%v: msg from ID is incorrect: expect: %v, have: %v", peer, separatorPeer.nodeID, msg.From)

				}
				if err := spr.ReadMsg(&msg); err != nil {
					glog.V(logger.Warn).Infof("%v: failed in add msg: %v", peer, err)
				}
			case upperConsensusCode:
				var msg Message
				if err := packet.Decode(&msg); err != nil {
					glog.V(logger.Warn).Infof("%v: failed to decode msg: %v", peer, err)
					continue
				}
				if msg.From != separatorPeer.nodeID {
					glog.V(logger.Warn).Infof("%v: msg from ID is incorrect: expect: %v, have: %v", peer, separatorPeer.nodeID, msg.From)

				}
				if err := spr.ReadMsg(&msg); err != nil {
					glog.V(logger.Warn).Infof("%v: failed in add msg: %v", peer, err)
				}
			case controlCode:
				var msg Message
				if err := packet.Decode(&msg); err != nil {
					loglogrus.Log.Warnf("%v: failed to decode msg: %v", peer, err)
					continue
				}
				if msg.From != separatorPeer.nodeID {
					loglogrus.Log.Warnf("%v: msg from ID is incorrect: expect: %v, have: %v", peer, separatorPeer.nodeID, msg.From)

				}
				if err := spr.ReadMsg(&msg); err != nil {
					loglogrus.Log.Warnf("%v: failed in add msg: %v", peer, err)
				}
			case intraWSUpdateCode:
				var msg Message
				if err := packet.Decode(&msg); err != nil {
					glog.V(logger.Warn).Infof("%v: failed to decode msg: %v", peer, err)
					continue
				}
				if msg.From != separatorPeer.nodeID {
					glog.V(logger.Warn).Infof("%v: msg from ID is incorrect: expect: %v, have: %v", peer, separatorPeer.nodeID, msg.From)

				}
				if err := spr.ReadMsg(&msg); err != nil {
					glog.V(logger.Warn).Infof("%v: failed in add msg: %v", peer, err)
				}
			default:
				glog.V(logger.Warn).Infof("%v: unknown msgcode: %v", peer, packet.Code)

			}
		}
	}

}

// 让本节点本地保存的所有节点组运行separator协议
func (spr *Separator) Start(srv *server.Server) {
	spr.MakeDialBook(srv, nil) // 此时separator协议尚未开始运行，此时调用spr.MakeDialBook(srv)只能使当前节点与自身所属子网内所有节点实现静态连接
	go spr.update(srv)         //让本节点随时更新整个separator网络的状态(或者说是每一个网络节点的子网归属状态)

	for _, group := range spr.PeerGroups { //遍历所有的子网节点组
		group.Start() //让本地记录的每一个子网开始运行separator协议(开始运行消息池的定时销毁功能)
	}

}

// 让所有尚未运行separator协议的节点组开始运行separator协议
func (spr *Separator) Update(srv *server.Server, unknownPeers []*UnKnownPeer) {
	spr.MakeDialBook(srv, unknownPeers) // TODO: at this time, there could be zombie nodes when nodes change their IP address
	for _, known := range unknownPeers {
		spr.SetNode(known.groupID, known.nodeID)
		spr.BackPeerGroup(known.groupID)
	}
	for _, group := range spr.PeerGroups {
		if !group.running {
			group.Start()
		}
	}
}

// 让本地所有的节点组停止运行separator协议
func (spr *Separator) Stop() {
	for _, group := range spr.PeerGroups {
		group.Stop()
	}
	close(spr.quit) //关闭本地的separator对象
	glog.V(logger.Info).Infoln("Separator stopped")
}

// try to update binding members of subnets.
// 定时调用spr.BindPeer()方法
func (spr *Separator) update(srv *server.Server) {
	bind := time.NewTicker(bindCycle)
	spr.DoBind(srv)

	// Repeat updates until termination is requested
	for {
		select {
		case <-bind.C:
			spr.DoBind(srv)

		case <-spr.quit: //收到本地的separator对象的关闭信号
			return
		}
	}
}

func (spr *Separator) DoBind(srv *server.Server) {
	unBindNum, unknownPeers := spr.BindPeer() //完成节点本地保存的separator网络的更新(也就是各节点的子网归属问题),同时返回异常节点个数
	if unBindNum > 0 {
		loglogrus.Log.Warnf("Separator Construct: (%d) nodes unable to establish P2P communication due to exceptions", unBindNum)
		for _, unknownPeer := range unknownPeers {
			spr.RemoveNode(unknownPeer.groupID, unknownPeer.nodeID) // 将断连节点从spr.PeerGroup中删除
			loglogrus.Log.Infof("查询到的断连节点: NodeID (%x) , NetID (%s)\n", unknownPeer.nodeID, unknownPeer.groupID)
		}
		spr.TidyDialBook(srv) // 将断连节点的静态P2P信息在P2P模块中删除

		spr.Update(srv, unknownPeers) // TODO: 可能需要尝试多次重连
	}
}

// Bind the known peers to the groups, return the number of unmatched
// 根据knownPeers集合更新spr.PeerGroups字段，即更新每一个节点组的Node成员信息(从而完成了对整个dpNet网络的更新)
func (spr *Separator) BindPeer() (int, []*UnKnownPeer) {
	spr.peerMu.Lock()
	defer spr.peerMu.Unlock()

	unMatched := 0
	unknownPeers := make([]*UnKnownPeer, 0)

	for _, group := range spr.PeerGroups { //遍历所有的子网
		group.groupMu.Lock()
		for nodeID, _ := range group.members { //遍历每个子网的所有成员节点
			if p, ok := spr.knownPeers[nodeID]; ok { //查看这些节点对本节点来说是否与本节点处于separator通信状态
				group.members[nodeID] = p //如果是的话，完成本地的目标节点的子网归属问题
			} else {
				unMatched += 1 //目标子网节点不存在于knownPeers中，说明本节点与目标节点的separator通信存在异常
				unKnownPeer := &UnKnownPeer{nodeID: nodeID, groupID: group.groupID}
				unknownPeers = append(unknownPeers, unKnownPeer)
			}
		}
		group.groupMu.Unlock()
	}

	return unMatched, unknownPeers
}

// 返回整个separator网络中尚未记录在addedPeers(未与本节点静态连接的节点)中的剩余节点的ID
func (spr *Separator) unAddedNodes() map[common.NodeID]struct{} {

	unAdded := make(map[common.NodeID]struct{})
	for _, group := range spr.PeerGroups { //遍历separator协议层的所有子网节点组
		group.groupMu.RLock()
		for memID, _ := range group.members { //遍历每个子网中的所有成员节点
			if _, ok := spr.addedPeers[memID]; ok { //检查这些节点是否已经存在于addedPeers中
				continue
			}
			unAdded[memID] = struct{}{} //将所有尚未加入到addedPeers中的所有节点的ID记录下来，并返回
		}
		group.groupMu.RUnlock()
	}
	return unAdded
}

// add members of peer groups into added list, then bottom server would try to connect them
// TODO: at this time, there could be zombie nodes when nodes change their IP address
// 让当前节点的server模块与当前separator网络内的自身已知的全部节点的客户端实现 静态连接
func (spr *Separator) MakeDialBook(srv *server.Server, reconnectPeers []*UnKnownPeer) {
	spr.staticMu.Lock()
	defer spr.staticMu.Unlock()

	unAdded := spr.unAddedNodes() //获取所有尚未记录在addedPeers(未与本节点静态连接的节点)中的剩余节点的ID

	if reconnectPeers != nil {
		for _, reconnectPeer := range reconnectPeers {
			unAdded[reconnectPeer.nodeID] = struct{}{}
		}
	}

	if len(unAdded) == 0 {
		fmt.Println("separator层: unAdded集合中没有任何节点")
		loglogrus.Log.Infof("separator层: unAdded集合中没有任何节点\n")
	}

	for nodeID, _ := range unAdded { //遍历这些节点
		loglogrus.Log.Infof("separator层: 位于unAdded集合中的节点nodeID: %x\n", nodeID)
		node, err := srv.LookUpNode(nodeID) //在现存的Kad网络中查找这些节点(每一个nodeID)是否还存在于p2p网络中
		if err != nil {
			loglogrus.Log.Infof("Separator Construct: Can't find node(%x) in p2p network, err: %v", err)
			continue
		}
		srv.AddPeer(node)             //让此node节点作为本地节点的 静态节点(server模块与自身所有已连接的客户端节点完成静态连接)
		spr.addedPeers[nodeID] = node //标志他们已经与本节点建立 静态连接
		loglogrus.Log.Infof("separator层: 成功与nodeID: %x建立静态链接, 节点IP:%v , 端口号:%v\n", nodeID, node.IP, node.TCP)
	}
}

// kick nodes that are not a member of a group anymore but still in added list.
// 删除addedPeers集合中所有过期的静态连接(连接已经不存在与dpNet网络中，但仍然记录在addedPeers中)
func (spr *Separator) TidyDialBook(srv *server.Server) {
	alive := make(map[common.NodeID]struct{}) //存储所有当前存活的节点的id列表(本地separator网络保存的节点)
	for _, group := range spr.PeerGroups {    //遍历所有节点组
		group.groupMu.RLock()
		for memID, _ := range group.members { //遍历所有成员节点
			alive[memID] = struct{}{}
		}
		group.groupMu.RUnlock()
	}

	spr.staticMu.Lock()
	defer spr.staticMu.Unlock()

	deleteNodes := make([]common.NodeID, 0)
	for node, _ := range spr.addedPeers { //遍历所有静态连接节点
		if _, ok := alive[node]; !ok { //查看这些静态节点是否存在于alive中
			deleteNodes = append(deleteNodes, node) //如果不存在与alive中，将其添加到待删除切片中(已经过期的静态连接信息)
		}
	}
	for _, node := range deleteNodes { //断开与这些节点的P2P连接
		srv.DelPeer(spr.addedPeers[node])
		delete(spr.addedPeers, node)
		loglogrus.Log.Infof("删除与节点 (%x) 的静态连接 \n", node)
	}
}

// 向指定ID的子网节点组中添加一个指定的节点ID.更新separator对象的PeerGroups集合(只更新子网名和NodeID，具体的Peer对象需要在协议handshake的Run函数中完成更新)
func (spr *Separator) SetNode(netID string, nodeID common.NodeID) error {
	spr.groupMu.Lock()
	defer spr.groupMu.Unlock()
	var group *PeerGroup
	if _, ok := spr.PeerGroups[netID]; !ok { //在本地搜索对应ID的子网
		group = NewPeerGroup(netID)   //如果不存在，则创建一个新的子网
		spr.PeerGroups[netID] = group //在节点本地的separator对象的PeerGroups集合中存储这个新的子网
	} else {
		group = spr.PeerGroups[netID]
	}
	return group.addMember(nodeID) //将新的节点的NodeID添加到指定子网中(只是添加了一个ID，没有添加节点实体，还需进行separator的协议handshake)
}

func (spr *Separator) RemoveNode(netID string, nodeID common.NodeID) error {
	spr.groupMu.Lock()
	defer spr.groupMu.Unlock()
	var group *PeerGroup
	if _, ok := spr.PeerGroups[netID]; !ok { //在本地搜索对应ID的子网,必须保证指定的子网是存在的
		return fmt.Errorf("Failed to update the node identity,because the the target subnet is not exist")
	} else {
		group = spr.PeerGroups[netID]
	}
	group.deleteMember(nodeID)
	return nil
}

// 由本地存储的对应的子网(每一条message消息都会指定自身的目标子网)负责读取形参msg消息(将本msg消息扔进此目标子网的消息池中)
func (spr *Separator) ReadMsg(msg *Message) error {

	group, ok := spr.PeerGroups[msg.NetID] //在整个separator协议层查询ID=msg.NetID的子网
	if !ok {
		err := fmt.Errorf("netID not found: %v", msg.NetID)
		return err
	}

	return group.ReadMsg(msg) //由该子网的节点组负责读取此消息(扔进此节点组的消息池中)
}

// 在本地separator对象的PeerGroups集合中查询指定ID的子网并返回PeerGroup对象
func (spr *Separator) BackPeerGroup(netID string) (*PeerGroup, error) {
	spr.groupMu.Lock()
	defer spr.groupMu.Unlock()

	if group, ok := spr.PeerGroups[netID]; !ok {
		loglogrus.Log.Warnf("Separator Construct: Can't find corresponding group (%s) in separator network!", netID)
		return nil, fmt.Errorf("can't find corresponding group (%s) in separator network!", netID)
	} else {
		// TODO:后续删除
		// for nodeID, node := range group.members {
		// 	loglogrus.Log.Infof("子网:%s --- 节点:%x , 内容(%v)\n", group.groupID, nodeID, node)
		// }

		return group, nil
	}
}
