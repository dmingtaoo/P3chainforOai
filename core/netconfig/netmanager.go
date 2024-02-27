package netconfig

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"p3Chain/blockSync"
	"p3Chain/common"
	"p3Chain/core/centre"
	"p3Chain/core/dpnet"
	"p3Chain/core/separator"
	"p3Chain/core/validator"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
	"reflect"
	"sync"
)

type NetManager struct {
	selfKey *ecdsa.PrivateKey    //自己的私钥
	srv     *server.Server       //当前节点的服务器模块对象
	shh     *whisper.Whisper     //当前节点的whisper协议客户端对象
	Spr     *separator.Separator //当前节点的separator协议客户端对象
	Cen     *centre.Centre       //当前节点的central协议对象
	Syn     *blockSync.SyncProtocal

	msgTTL   uint64      // message about net configuration lifetime
	SelfNode *dpnet.Node //记录本节点在P3-Chain网络中的节点ID/节点所属子网ID/节点在网络中的身份

	viewNet *dpnet.DpNet //当前节点所认知的整个P3-Chain网络的信息

	netMutex sync.RWMutex

	lowerChannel *separator.PeerGroup // channel for subnet peer group connections to run
	// lower-level consensus. nil for booter nodes.

	upperChannel *separator.PeerGroup // channel for upper-level consensus. nil for followers.

	configInitFilterID, configUpdateFilterID int
}

// return a new NetManager
// 为本地的节点创建一个P3-Chain网络(viewNet)
func NewNetManager(prvKey *ecdsa.PrivateKey, viewNet *dpnet.DpNet, srv *server.Server, shh *whisper.Whisper, Spr *separator.Separator, Cen *centre.Centre, Syn *blockSync.SyncProtocal, msgTTL uint64, node *dpnet.Node) *NetManager {
	nm := &NetManager{
		srv:      srv,
		selfKey:  prvKey,
		shh:      shh,
		Spr:      Spr,
		Cen:      Cen,
		Syn:      Syn,
		SelfNode: node,
		viewNet:  viewNet,
	}
	if msgTTL <= 0 {
		nm.msgTTL = defaultMsgLifeTime
	} else {
		nm.msgTTL = msgTTL
	}
	return nm
}

// func (nm *NetManager) InitViewNet() {

// 	loglogrus.Log.Infof("[NetManager] 基于 NetInit 协议进行 ViewNet 初始化.....\n")

// 	// booters := nm.nit.BackAllBooters()
// 	// loglogrus.Log.Infof("[NetManager] 基于 NetInit 协议进行 ViewNet 初始化, Booter 数量:%d\n", len(booters))
// 	// leaders := nm.nit.BackAllLeaders()
// 	// loglogrus.Log.Infof("[NetManager] 基于 NetInit 协议进行 ViewNet 初始化, Leader 数量:%d\n", len(leaders))
// 	// others := nm.nit.BackAllOthers()
// 	// loglogrus.Log.Infof("[NetManager] 基于 NetInit 协议进行 ViewNet 初始化, 本组内其它节点数量:%d\n", len(others))

// 	// for _, booter := range booters {
// 	// 	nm.viewNet.AddNode(booter)
// 	// }

// 	// for _, leader := range leaders {
// 	// 	nm.viewNet.AddNode(leader)
// 	// }

// 	// for _, other := range others {
// 	// 	nm.viewNet.AddNode(other)
// 	// }

// 	allNodes := nm.nit.BackAllNodes()
// 	for _, node := range allNodes {
// 		nm.viewNet.AddNode(node)
// 	}
// }

// 配置本节点在网络中的身份
// change the role of self node
func (nm *NetManager) SetSelfNodeRole(role dpnet.RoleType) {
	nm.SelfNode.Role = role
}

// 获取本节点在网络中的身份
func (nm *NetManager) GetSelfNodeRole() dpnet.RoleType {
	return nm.SelfNode.Role
}

// 配置本节点所处的子网(上层网、中层的各子网)
// change the netID of self node
func (nm *NetManager) SetSelfNodeNetID(netID string) {
	nm.SelfNode.NetID = netID
}

// 获取本节点所处的网络的ID
func (nm *NetManager) GetSelfNodeNetID() string {
	return nm.SelfNode.NetID
}

// 获取本节点存储的整个dpnet网络的各个子网网络信息
func (nm *NetManager) GetDPNetInfo() map[string]*dpnet.SubNet {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	return nm.viewNet.SubNets
}

func (nm *NetManager) GetDpViewNet() *dpnet.DpNet {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	return nm.viewNet
}

// 将节点自身添加到本地建立的P3-Chain网络中(根据节点自身的身份、要加入的子网进行合理加入)
// Add self node into the view net.
func (nm *NetManager) ViewNetAddSelf() error {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	err := nm.viewNet.AddNode(*nm.SelfNode)
	return err
}

// Add self as a booter when the node is a part-time booter
func (nm *NetManager) ViewNetAddSelfAsBooter() error {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	selfBooter := dpnet.NewNode(nm.SelfNode.NodeID, int(dpnet.Booter), dpnet.UpperNetID)
	err := nm.viewNet.AddNode(*selfBooter)
	return err
}

// 网络的启动配置器，负责为whisper客户端配置一个对接收消息的处理方法
// First start stage: inject topics about net configuration into whisper client
func (nm *NetManager) StartConfigChannel() error {
	nm.shh.InjectIdentity(nm.selfKey) //把自身获得的公私钥对注册到自身的whisper协议对象中
	var handler NetConfigHandler      //消息处理器(接口)
	switch nm.SelfNode.Role {         //根据本节点的身份选用合适的消息处理器(Common 或者 Booter)
	case dpnet.Leader, dpnet.Follower, dpnet.UnKnown:
		handler = &CommonHandler{nm}
	case dpnet.Booter:
		handler = &BooterHandler{nm}
	default:
		return fmt.Errorf("no such node role")
	}

	//为whisper客户端安装一个消息处理方法(Fn),当whisper客户端收到消息时调用此方法(Fn)
	nm.configInitFilterID = nm.shh.Watch(whisper.Filter{
		To:     nil,
		From:   nil,
		Topics: whisper.NewFilterTopicsFromStrings([]string{stringConfigInit}),
		Fn:     handler.DoInitMsg,
	})

	nm.configUpdateFilterID = nm.shh.Watch(whisper.Filter{
		To:     nil,
		From:   nil,
		Topics: whisper.NewFilterTopicsFromStrings([]string{stringConfigUpdate}),
		Fn:     handler.DoUpdateMsg,
	})
	return nil
}

// After construct the initial dpnet, use ConfigLowerChannel to add members of
// lower-level consensus into separator.
// 完成本地dpnet的创建之后，利用viewNet中包含的信息更新Spr字段的separator协议客户端对象(因为separator协议尚未运行,因此只能获取当前子网内的节点部署情况)
func (nm *NetManager) ConfigLowerChannel() error {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	//peerNodes := nm.viewNet.BackSubnetNodesID(nm.SelfNode.NodeID, nm.SelfNode.NetID) //返回当前节点所在子网的除自身外其他所有节点
	peerNodes := nm.viewNet.BackSubnetNodesID(common.NodeID{}, nm.SelfNode.NetID) //返回当前节点所在子网的除自身外其他所有节点
	//fmt.Println()
	for _, peerNode := range peerNodes { //这里的peerNodes都是与当前节点位于同一子网的节点
		if err := nm.Spr.SetNode(nm.SelfNode.NetID, peerNode); err != nil { //spr字段现阶段只能获取到本子网内的节点部署情况，对于其他网络节点组无法获知
			loglogrus.Log.Warnf("Separator Construction: Config LowerChannel is failed, Can't to join node to corresponding group (%s) !\n", nm.SelfNode.NetID)
			return fmt.Errorf("fall in config lower channel: %v", err)
		}
	}
	if pg, err := nm.Spr.BackPeerGroup(nm.SelfNode.NetID); err != nil {
		loglogrus.Log.Warnf("Separator Construction: Config LowerChannel is failed, Can't find corresponding group (%s) in separator network!\n", nm.SelfNode.NetID)
		return fmt.Errorf("fall in config lower channel: %v", err)
	} else {
		nm.lowerChannel = pg
	}

	return nil
}

// 热更新 lower channel 层（PeerGroup类没有对节点进行角色的划分,viewNet类才是对角色进行划分）
func (nm *NetManager) UpdateLowerChannel(node dpnet.Node, mode int) error {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()

	// if node.NetID != nm.SelfNode.NetID { //仅处理与自己位于同一个节点组的更新节点(因为只允许节点以follower身份插入网络,而当前子网实际上并不需要知道其他子网的follower信息)
	// 	return nil
	// }

	if mode == UpdateNodeState { //对于节点组内部成员身份的变化,不需要采取额外的措施。因为节点组内的成员节点并没有发生任何改变(没有新增节点也没有退出节点,因此不需要更新PeerGroup)，只是某一个节点的身份进行了改变(由upper channel层进行更新,需要更新PeerGroup)
		return nil
	}

	peerNodes := nm.viewNet.BackSubnetNodesID(common.NodeID{}, node.NetID) // 返回指定子网的全部节点的信息

	switch mode {
	case AddNewNode:
		// 1.必须保证新添加的节点原本并不存在于指定的PeerGroup中
		var isExist bool = false
		for i := 0; i < len(peerNodes); i++ {
			if peerNodes[i] == node.NodeID {
				isExist = true
				break
			} else {
				continue
			}
		}
		if isExist {
			return fmt.Errorf("fall in update lower channel~~~: Cannot add the same node repeatedly")
		} else {
			if err := nm.Spr.SetNode(node.NetID, node.NodeID); err != nil { //把新节点插入到相应的PeerGroup中,作为follower
				return fmt.Errorf("fall in update lower channel~~~: %v", err)
			}
		}
	case DelNode:
		var isExist bool = false
		for i := 0; i < len(peerNodes); i++ {
			if peerNodes[i] == node.NodeID {
				isExist = true
				break
			} else {
				continue
			}
		}
		// 2.如果节点确实存在与PeerGroup,将节点以及相关资源从PeerGroup中删除
		if isExist {
			if err := nm.Spr.RemoveNode(node.NetID, node.NodeID); err != nil {
				return fmt.Errorf("fall in update lower channel~~~: %v", err)
			}
		} else {
			return fmt.Errorf("fall in update lower channel~~~: The target node does not exist")
		}
	default:
		return fmt.Errorf("fall in update lower channel~~~: No corresponding update instruction exists")
	}
	return nil
}

// After construct the initial dpnet, use ConfigUpperChannel to add members of
// upper-level consensus into separator.
// TODO: At this time, the subnet ID for upper-level consensus is solid which is "upper".
// In the future, customized subnet ID should be realized.
// //完成本地dpnet的初始化之后，使用本方法将所有高共识层的成员节点(全部领导节点和引导节点)加入到本地维护的dpnet中(从viewnet到spr)
func (nm *NetManager) ConfigUpperChannel() error {
	//从本地P3-Chain的Upper网络中取出除了自身以外的其他全部领导节点和引导节点
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()

	// peerNodes := nm.viewNet.BackSubnetNodesID(nm.SelfNode.NodeID, dpnet.UpperNetID)
	peerNodes := nm.viewNet.BackSubnetNodesID(common.NodeID{}, dpnet.UpperNetID)

	for _, peerNode := range peerNodes {
		if err := nm.Spr.SetNode(dpnet.UpperNetID, peerNode); err != nil {
			loglogrus.Log.Warnf("Separator Construction: Config UpperChannel is failed, Can't to join node to corresponding group (%s) !\n", dpnet.UpperNetID)
			return fmt.Errorf("fall in config upper channel: %v", err)
		}
	}

	if pg, err := nm.Spr.BackPeerGroup(dpnet.UpperNetID); err != nil { //返回整个Upper网络(节点组)
		loglogrus.Log.Warnf("Separator Construction: Config UpperChannel is failed, Can't find corresponding group (%s) in separator network!\n", dpnet.UpperNetID)
		return fmt.Errorf("fall in config upper channel: %v", err)
	} else {
		nm.upperChannel = pg
	}
	return nil
}

// 热更新 upper channel 层
func (nm *NetManager) UpdateUpperChannel(node dpnet.Node, mode int) error {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()

	if mode == AddNewNode { // 不允许新节点以Leader或Booter身份加入到网络中(只能以follower身份加入,后续再通过update进行角色切换)
		return nil
	}

	// if nm.SelfNode.Role == dpnet.Follower { // 对于Follower节点来说，其本地没有存储upper PeerGroup
	// 	return nil
	// }

	peerNodes := nm.viewNet.BackSubnetNodesID(common.NodeID{}, dpnet.UpperNetID) //获取所有upper层的Node

	oldPeer := common.NodeID{} //如果新节点(或者是更新后)原本存在于upper层,则oldPeer会记录其NodeID
	for _, peerNode := range peerNodes {
		if peerNode == node.NodeID {
			oldPeer = node.NodeID
			break
		}
	}
	switch mode {
	case UpdateNodeState:
		// 1.更新前的节点不存在于upper层(是一个follower),但更新后变成了Leader或Booter。则需要将该节点添加到upper层PeerGroup
		if (reflect.DeepEqual(oldPeer, common.NodeID{})) && (node.Role == dpnet.Leader || node.Role == dpnet.Booter) {
			if err := nm.Spr.SetNode(dpnet.UpperNetID, node.NodeID); err != nil {
				return fmt.Errorf("fall in update upper channel~~~~: %v", err)
			} else {
				loglogrus.Log.Infof("Separator Update: 成功将节点(%x)加入到群组(%s)中\n", node.NodeID, dpnet.UpperNetID)
			}

		} else if (!reflect.DeepEqual(oldPeer, common.NodeID{})) && (node.Role == dpnet.Follower) { // 2.更新前的节点存在于upper层(是一个leader),但更新后变成了follower。则需要将该节点从upper层移除
			if err := nm.Spr.RemoveNode(dpnet.UpperNetID, node.NodeID); err != nil {
				return fmt.Errorf("fall in update upper channel~~~~: %v", err)
			} else {
				loglogrus.Log.Infof("Separator Update: 成功将节点(%x)从群组(%s)中删除\n", node.NodeID, dpnet.UpperNetID)
			}
		} else {
			loglogrus.Log.Infof("Separator Update: 无效的节点切换策略\n")
		}
	case DelNode:
		//必须保证被删除的节点原本存在于upper层(不能重复删除)
		if (reflect.DeepEqual(oldPeer, common.NodeID{})) {
			return fmt.Errorf("fall in update upper channel~~~~: The node is not exists in the upper channel")
		} else {
			if err := nm.Spr.RemoveNode(dpnet.UpperNetID, node.NodeID); err != nil {
				return fmt.Errorf("fall in update upper channel~~~~: %v", err)
			}
		}
	default:
		return fmt.Errorf("fall in update upper channel~~~: No corresponding update instruction exists")
	}

	return nil
}

// Second start stage: start the subnets after setting members of separator peer groups.
// Should be executed after channels has been configured.
// 启动当前节点的服务器模块，运行separator协议
func (nm *NetManager) StartSeparateChannels() error {
	nm.Spr.Start(nm.srv)
	return nil
}

// once dpnet has been constructed, the init topic could be removed
func (nm *NetManager) UninstallInitTopic() {
	nm.shh.Unwatch(nm.configInitFilterID)
}

// 调用whisper协议发送数据包
func (nm *NetManager) SendMsg(mc MsgConfiger) error {
	msg := whisper.NewMessage(mc.BackPayload())                    //根据消息的载荷先创建一条whisper协议Message数据包
	envelope, err := msg.Wrap(whisper.DefaultPoW, whisper.Options{ //将messages数据包打包成envelop信封
		From:   mc.BackFrom(),     //添加消息发送节点私钥
		To:     mc.BackTo(),       //添加消息目的节点公钥
		TTL:    mc.BackLifeTime(), //添加消息的TTL生存周期
		Topics: []whisper.Topic{mc.BackTopic()},
	})
	if err != nil {
		return err
	}
	err = nm.shh.Send(envelope) //采用whisper协议进行发送(From和To都是nil，意味着没有payload加密以及数字签名，因此只要接收方Topic表正确就能收到此消息)
	if err != nil {
		return err
	}
	return nil
}

// 将自身的NodeID/NetID/Role等信息发送给目标节点(用于构建separator网络)
// 发送目标是所有与自身存在whisper通信的节点
func (nm *NetManager) SendInitSelfNodeState() error {

	imc := InitMsgConfiger{
		MsgCode:  selfNodeState,
		LifeTime: nm.msgTTL,
		Sender:   nm.SelfNode.NodeID,
		NetID:    nm.SelfNode.NetID,
		NodeID:   nm.SelfNode.NodeID,
		Role:     nm.SelfNode.Role,
	}

	signature, err := crypto.Sign(imc.Hash(), nm.selfKey) //获取消息的数字签名
	if err != nil {
		loglogrus.Log.Warnf("Separator: Unable to generate digital signature for selfNodeState Msg!\n")
		return err
	}
	imc.Signature = signature

	return nm.SendMsg(&imc)
}

// send self information as a booter.
func (nm *NetManager) SendInitSelfBooterState() error {
	imc := InitMsgConfiger{
		MsgCode:  selfNodeState,
		LifeTime: nm.msgTTL,
		Sender:   nm.SelfNode.NodeID,
		NetID:    dpnet.UpperNetID,
		NodeID:   nm.SelfNode.NodeID,
		Role:     dpnet.Booter,
	}

	signature, err := crypto.Sign(imc.Hash(), nm.selfKey)
	if err != nil {
		glog.V(logger.Error).Infof("fail in SendInitSelfNodeState: %v", err)
		return err
	}
	imc.Signature = signature

	nm.SendMsg(&imc)
	return nm.SendMsg(&imc)
}

// 重连请求
func (nm *NetManager) SendReconnectState() error {
	umc := UpdateMsgConfiger{
		MsgCode:  ReconnectState,
		LifeTime: nm.msgTTL,
		Sender:   nm.SelfNode.NodeID,
		NetID:    nm.SelfNode.NetID,
		NodeID:   nm.SelfNode.NodeID,
		Role:     nm.SelfNode.Role,
	}
	signature, err := crypto.Sign(umc.Hash(), nm.selfKey)
	if err != nil {
		glog.V(logger.Error).Infof("fail in SendInitSelfNodeState: %v", err)
		return err
	}
	umc.Signature = signature

	return nm.SendMsg(&umc)
}

// Booter通过向断连节点发送此消息，使其重新获取整个分区网络的拓扑结构
func (nm *NetManager) SendDpNetInfo() error {
	umc := UpdateMsgConfiger{
		MsgCode:  DpNetInfo,
		LifeTime: nm.msgTTL,
		Sender:   nm.SelfNode.NodeID,
		NetID:    nm.SelfNode.NetID,
		NodeID:   nm.SelfNode.NodeID,
		Role:     nm.SelfNode.Role,
		NodeList: nm.BackAllDper(),
	}
	signature, err := crypto.Sign(umc.Hash(), nm.selfKey)
	if err != nil {
		loglogrus.Log.Warnf("fail in SendInitSelfNodeState: %v", err)
		return err
	}
	umc.Signature = signature
	for _, node := range nm.BackAllDper().NodeList {
		fmt.Printf("nodeID:%x , groupID: %s\n", node.NodeID, node.NetID)
	}

	return nm.SendMsg(&umc)
}

// 中心节点通过此方法发送dpnet网络配置消息(发送CentralConfigMsg消息)
func (nm *NetManager) SendCentralConfig() error {

	if nm.Cen == nil {
		fmt.Println("cen 是空的")
	}

	nm.Cen.BooterCentralInit() // Booter节点自己构建viewNet
	nm.CentralizedConf()

	// ccf := CentralConfigMsg{
	// 	MsgCode:  centralConfig,
	// 	LifeTime: nm.msgTTL,
	// 	Sender:   nm.SelfNode.NodeID,
	// 	DpNet:    nm.Cen.DpNet, //DpNet网络配置
	// 	NodeID:   nm.SelfNode.NodeID,
	// }

	// signature, err := crypto.Sign(ccf.Hash(), nm.selfKey) //获取消息的数字签名
	// if err != nil {
	// 	glog.V(logger.Error).Infof("fail in SendCentralConfig: %v", err)
	// 	return err
	// }
	// ccf.Signature = signature

	// nm.SendMsg(&ccf) //将添加数字签名后的消息进行发送
	// return nm.SendMsg(&ccf)
	return nil
}

// 所有节点根据central协议运行结果(dpNet字段)更新viewNet字段
func (nm *NetManager) CentralizedConf() {
	nm.viewNet = &nm.Cen.DpNet
}

// 所有节点根据central协议运行结果(dpNet字段)更新SelfNode字段
func (nm *NetManager) SelfNodeUpdate() error {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	for _, group := range nm.viewNet.SubNets {
		for _, node := range group.Nodes {
			if node.NodeID == nm.SelfNode.NodeID {
				nm.SelfNode.NetID = node.NetID
				nm.SelfNode.Role = node.Role
				return nil
			}
		}
	}

	return errors.New("couldn't find selfNode!")
}

func (nm *NetManager) BackUpperChannel() (*separator.PeerGroup, error) {
	return nm.Spr.BackPeerGroup(dpnet.UpperNetID)
}

func (nm *NetManager) BackLowerChannel() (*separator.PeerGroup, error) {
	return nm.Spr.BackPeerGroup(nm.SelfNode.NetID)
}

// ExportValidator exports a CommonValidator to help validate blocks.
// NOTE: The validator is built using the snapshot of the view net, so
// it should be updated everytime the view net is changed.
func (nm *NetManager) ExportValidator() (*validator.CommonValidator, error) {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()

	if nm.viewNet == nil {
		loglogrus.Log.Warnf("Separator Construct: Export Validator is failed, dpnet.DpNet is nil!\n")
		return nil, fmt.Errorf("no net is viewed")
	}

	cv := &validator.CommonValidator{
		SubnetPool:   map[string]validator.SubnetVoters{},
		LeaderVoters: map[common.Address]bool{},
	}

	for _, leader := range nm.viewNet.Leaders {
		addr, err := crypto.NodeIDtoAddress(leader)
		if err != nil {
			return nil, err
		}
		cv.LeaderVoters[addr] = true
	}
	for netID, subnet := range nm.viewNet.SubNets {
		voters := make(validator.SubnetVoters)
		for nodeID, _ := range subnet.Nodes {
			addr, err := crypto.NodeIDtoAddress(nodeID)
			if err != nil {
				return nil, err
			}
			voters[addr] = true
		}
		cv.SubnetPool[netID] = voters
	}
	return cv, nil
}

// BackBooters return the booters in the viewnet
func (nm *NetManager) BackBooters() []common.NodeID {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	return nm.viewNet.Booters
}

func (nm *NetManager) BackViewNetInfo() string {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	return nm.viewNet.String()
}

func (nm *NetManager) GetSubNetLeader(groupID string) common.NodeID {
	nm.netMutex.RLock()
	defer nm.netMutex.RUnlock()
	return nm.viewNet.BackSubNetLeader(groupID)
}

func (nm *NetManager) BackViewNet() *dpnet.DpNet {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	return nm.viewNet
}

func (nm *NetManager) BackAllDper() NodeListMsg {
	nm.netMutex.Lock()
	defer nm.netMutex.Unlock()
	dpNet := nm.viewNet
	nodeList := make([]dpnet.Node, 0)
	for _, net := range dpNet.SubNets {
		for _, node := range net.Nodes {
			nodeList = append(nodeList, *node)
		}
	}
	booterList := make([]dpnet.Node, 0)
	for _, booterID := range dpNet.Booters {
		booter := dpnet.Node{NetID: dpnet.UpperNetID, Role: 1, NodeID: booterID}
		booterList = append(booterList, booter)
	}

	nodeListMsg := NodeListMsg{
		NodeList:   nodeList,
		BooterList: booterList,
	}

	return nodeListMsg
}

func (nm *NetManager) Close() {
	nm.srv.Stop()
	if nm.Spr != nil {
		nm.Spr.Stop()
	}
	if nm.Cen != nil {
	}
	if nm.Syn != nil {
		nm.Syn.Stop()
	}
	if nm.shh != nil {
		nm.shh.Stop()
	}
}

func (nm *NetManager) BackSelfUrl() (string, error) {
	if nm.srv == nil {
		return "", fmt.Errorf("server is nil")
	}
	url := nm.srv.Self().String()
	return url, nil
}

// 某节点需要中途更新自己的Node Role 调用此方法
func (nm *NetManager) SendUpdateNodeState(role dpnet.RoleType) error {
	umc := UpdateMsgConfiger{
		MsgCode:  UpdateNodeState,
		LifeTime: nm.msgTTL,
		Sender:   nm.SelfNode.NodeID,
		NetID:    nm.SelfNode.NetID,
		NodeID:   nm.SelfNode.NodeID,
		Role:     role,
	}
	signature, err := crypto.Sign(umc.Hash(), nm.selfKey)
	if err != nil {
		loglogrus.Log.Warnf("fail in SendUpdateNodeState: %v", err)
		return err
	}
	umc.Signature = signature

	errr := nm.SendMsg(&umc)
	return errr
}

func (nm *NetManager) LeaderChangeValidata(NodeID common.NodeID, groupID string, lcm *LeaderChangeMsg) bool {
	// 1.获取指定节点组所有节点的 NodeID,将其转换为公钥.同时验证消息中携带的节点签名数是否足够
	nodeIDs := nm.viewNet.BackSubnetNodesID(common.NodeID{}, groupID)
	// 1.1 必须存在足够多的节点签名才可以进行Leader Change
	pbftFactor := (len(nodeIDs) - 1) / 3
	if len(lcm.Signatures) < 2*pbftFactor {
		return false
	}
	var nodeIsExist bool = false
	pubKeys := make([]*ecdsa.PublicKey, 0)
	for _, nodeID := range nodeIDs {
		if nodeID == NodeID {
			nodeIsExist = true
		}
		if pub, err := crypto.NodeIDtoKey(nodeID); err != nil {
			continue
		} else {
			pubKeys = append(pubKeys, pub) // 保留根据节点的NodeID获取的所有公钥
		}
	}
	// 1.2 必须保证被选举出的新节点存在于指定的节点组中
	if !nodeIsExist {
		fmt.Printf("新选举的Leader节点:%x 并不存在于指定的节点组: %s中\n", NodeID, groupID)
		return false
	}

	// 2.验证LeaderChangeMessage中包含的 2*f 条数字签名的正确性。如果验证不合格则直接退出,如果验证合格继续进行下一步
	sigPubKeys := make([][]byte, 0)
	for _, sig := range lcm.Signatures { //从lcm的数字签名和消息hash中获取出所有的公钥
		if sigPublicKey, err := crypto.Ecrecover(lcm.ViewChangeHash.Bytes(), sig); err != nil {
			fmt.Printf("解析数字签名公钥失败,err:%v\n", err)
		} else {
			sigPubKeys = append(sigPubKeys, sigPublicKey)
		}
	}
	// 2.1 必须保证解析出来公钥在第一步中转换得到的pubKeys中存在
	for i := 0; i < len(sigPubKeys); i++ {
		sigValid := false
		for j := 0; j < len(pubKeys); j++ {
			// fmt.Printf("正在匹配: 数字签名中获取的公钥:%x  本地保存的公钥:%x\n", sigPubKeys[i], crypto.FromECDSAPub(pubKeys[j]))
			sigValid = reflect.DeepEqual(sigPubKeys[i], crypto.FromECDSAPub(pubKeys[j]))
			if sigValid {
				break
			}
		}
		if !sigValid { //sigPubKeys 中一旦存在虚假签名,则判定为更换失败
			fmt.Println("存在虚假公钥,更换Leader节点失败...............")
			fmt.Println()
			return false
		}
	}
	fmt.Printf("当前节点:%x 完成签名验证\n", nm.SelfNode.NodeID)
	fmt.Println()
	return true
}

func (nm *NetManager) GetPrivateKey() *ecdsa.PrivateKey {
	nm.netMutex.RLock()
	defer nm.netMutex.RUnlock()
	return nm.selfKey
}

func (nm *NetManager) SendUpdateLeaderChange(msgHash common.Hash, sigs [][]byte) error {
	lcm := LeaderChangeMsg{
		ViewChangeHash: msgHash,
		Signatures:     sigs,
	}
	umc := UpdateMsgConfiger{
		MsgCode:      LeaderChange,
		LifeTime:     nm.msgTTL,
		Sender:       nm.SelfNode.NodeID,
		NetID:        nm.SelfNode.NetID, //给定子网netID
		NodeID:       nm.SelfNode.NodeID,
		Role:         dpnet.Leader, //自己为新的Leader
		LeaderChange: lcm,
	}
	signature, err := crypto.Sign(umc.Hash(), nm.selfKey)
	if err != nil {
		fmt.Printf("发送LeaderChange Msg失败: 生成数字签名失败,err:%v\n", err)
		return err
	}
	umc.Signature = signature

	errr := nm.SendMsg(&umc)
	return errr
}
