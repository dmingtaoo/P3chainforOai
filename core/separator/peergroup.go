package separator

import (
	"fmt"
	"p3Chain/common"
	loglogrus "p3Chain/log_logrus"
	"sync"
	"time"
)

const (
	expireCycle = 1600 * time.Millisecond // could be larger

	// TODO: Customize the max life for msg stored in msg pool for a subnet
	defaultMsgPoolExpireTime = 10 * time.Second //子网消息池消息的存活时间
)

// multiple subnets use the same separator, so there are multiple peerGroups
// separator协议规定，一个节点组构成的一个子网
// 节点组对象
type PeerGroup struct {
	running bool                    //whether start   是否已经运行separator协议
	groupID string                  // groupID represents the subnet netID  节点组的ID，也就是子网ID
	members map[common.NodeID]*peer // expect peer connects   记录本节点组的所有成员节点

	messagePool       map[common.Hash]*msgState // cache for messages read    本节点的消息池
	msgPoolMu         sync.RWMutex              // mutex to store and load messages
	msgPoolExpireTime time.Duration             // the max stale time of msg stored in pool   消息在消息池中的存活时间

	groupMu sync.RWMutex // Mutex to sync the active peers in group

	quit chan struct{}
}

// 创建一个新的节点组(子网)
// Return a new peer group with the given subnet ID
func NewPeerGroup(groupID string) *PeerGroup {
	PeerGroup := &PeerGroup{
		running:           false,
		groupID:           groupID,
		members:           make(map[common.NodeID]*peer),
		messagePool:       make(map[common.Hash]*msgState),
		msgPoolExpireTime: defaultMsgPoolExpireTime,
		quit:              make(chan struct{}),
	}
	return PeerGroup
}

// 获取节点群组的网络ID
func (pg *PeerGroup) GetGroupID() string {
	return pg.groupID
}

// 返回当前群组中所有节点的NodeID
func (pg *PeerGroup) GetMemberID() []common.NodeID {

	var nodes = make([]common.NodeID, 0)

	for id, _ := range pg.members {
		nodes = append(nodes, id)
	}
	return nodes
}

// 往自身所在的separator节点组中添加一个新的节点ID(并没有添加节点实体)
func (pg *PeerGroup) addMember(memberID common.NodeID) error {
	pg.groupMu.Lock()
	defer pg.groupMu.Unlock()
	if _, ok := pg.members[memberID]; ok { //检查相同ID的节点是否已经存在于节点组中
		loglogrus.Log.Warnf("Separator Construction: member (%x) has already within this group (%s)\n", memberID, pg.groupID)
		return fmt.Errorf("member (%x) has already within this group (%s)", memberID, pg.groupID)
	}
	pg.members[memberID] = nil //在map中为指定的memberID空出一个新的节点空间
	return nil
}

// 添加一个节点实体
func (pg *PeerGroup) AddMemberEntity(memberID common.NodeID, entity *peer) error {
	pg.groupMu.Lock()
	defer pg.groupMu.Unlock()
	if value, ok := pg.members[memberID]; !ok { //检查此节点的ID是否已经存在于当前节点组
		return fmt.Errorf("member %x is not in the current group ", memberID)
	} else {
		_ = value
		//fmt.Printf("AddMemberEntity调用前,spr中对应nodeID的node实体:%v\n", value)
		pg.members[memberID] = entity //赋予该ID一个peer实体(只有当NodeID已经存在于members集合是才会被加入)
	}
	//fmt.Printf("AddMemberEntity调用后,spr中对应nodeID的node实体:%v\n", pg.members[memberID])
	return nil
}

// 在节点组中删除一个指定ID的成员节点
func (pg *PeerGroup) deleteMember(memberID common.NodeID) {
	pg.groupMu.Lock()
	defer pg.groupMu.Unlock()
	delete(pg.members, memberID)
}

// 让当前整个节点组开始运行separator协议(就是让节点组开始管理自身的消息池)
func (pg *PeerGroup) Start() {
	pg.running = true
	loglogrus.Log.Infof("Separator Construct: Peer Group (%s) is started!\n", pg.groupID)
	go pg.update() //在本地更新目标节点组消息池
}

// 停止运行节点组的separator协议
func (pg *PeerGroup) Stop() {
	close(pg.quit)
	pg.running = false
	loglogrus.Log.Infof("peer group: %s stopped", pg.groupID)
}

// 负责定时对节点组消息池中的过期消息进行销毁
func (pg *PeerGroup) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expireCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C:
			pg.msgExpire() //定期对消息池中过期的消息进行销毁

		case <-pg.quit: //节点组关闭信号
			return
		}
	}
}

// 负责清除消息池中的过期和已读消息
func (pg *PeerGroup) msgExpire() {
	now := time.Now()
	expiredMsgs := make([]common.Hash, 0) //等待存储所有失效的消息
	pg.msgPoolMu.RLock()
	for msgHash, msgState := range pg.messagePool { //从节点组的消息池中读取当前存在的所有消息
		if msgState.retrieved { //此消息已经被读过了，放入失效切片
			expiredMsgs = append(expiredMsgs, msgHash)
			continue
		}
		if msgState.expireTime.Before(now) { //此消息存活时间超时，放入失效切片
			expiredMsgs = append(expiredMsgs, msgHash)
			continue
		}
	}
	pg.msgPoolMu.RUnlock()
	if len(expiredMsgs) == 0 { //没有任何失效消息，直接返回
		return
	}

	pg.msgPoolMu.Lock()
	defer pg.msgPoolMu.Unlock()

	for _, msgHash := range expiredMsgs { //从消息池中删除所有失效消息
		delete(pg.messagePool, msgHash)
	}
}

// 负责把获取的msg消息扔进本节点组的消息池中
func (pg *PeerGroup) ReadMsg(msg *Message) error {
	pg.groupMu.RLock()
	defer pg.groupMu.RUnlock()

	//检测接收到的msg消息是否来自于本组内的节点，如果不是的话直接退出
	if _, ok := pg.members[msg.From]; !ok {
		err := fmt.Errorf("node %x not in net %s", msg.From, pg.groupID)
		return err
	}
	//让msg消息加入当前时间戳信息，打包成msgState对象
	state := &msgState{
		msg:        msg,
		expireTime: time.Now().Add(pg.msgPoolExpireTime),
		retrieved:  false,
	}

	pg.msgPoolMu.Lock()
	defer pg.msgPoolMu.Unlock()

	pg.messagePool[msg.CalculateHash()] = state //把消息扔进消息池中
	return nil
}

// back current messages stored in message pool
// 返回消息池中所有剩余的message消息()
func (pg *PeerGroup) CurrentMsgs() []*Message {
	pg.msgPoolMu.RLock()
	defer pg.msgPoolMu.RUnlock()

	msgs := make([]*Message, 0)
	for _, msgState := range pg.messagePool {
		if msgState.retrieved {
			continue
		}
		msgs = append(msgs, msgState.msg)
	}
	return msgs
}

// mark the retrieved messages to kick them out of the pool
// 为形参指定的消息标记已读标志( msgIDs 是消息的哈希值)
func (pg *PeerGroup) MarkRetrievedMsgs(msgIDs []common.Hash) {
	pg.msgPoolMu.Lock()
	defer pg.msgPoolMu.Unlock()
	for _, msgID := range msgIDs { //遍历所有的消息哈希
		if _, ok := pg.messagePool[msgID]; ok { //在消息池中检查此消息是否存在
			msgState := pg.messagePool[msgID]
			msgState.MarkRetrived() //打上已读标志
		}
	}
}

// 将形参指定的msg消息广播给当前节点组内所有成员节点
func (pg *PeerGroup) MsgBroadcast(msg *Message) {
	pg.groupMu.RLock()
	defer pg.groupMu.RUnlock()
	for _, peer := range pg.members { //遍历本节点组的所有成员节点
		if peer != nil {
			if err := peer.sendMsg(msg); err != nil { //向所有成员节点广播发送此msg消息
				loglogrus.Log.Errorf("Separator failed: The message cannot be broadcast within the peer group, err:%v\n", err)
			}
		}
	}
}

// 指定当前节点组内的某一个成员节点发送msg消息
func (pg *PeerGroup) MsgSend(msg *Message, reciever common.NodeID) error {
	peer, ok := pg.members[reciever] //查看目标节点是否存在于本节点组
	if !ok {                         //情况一：不存在
		return fmt.Errorf("subnet: %s has no node: %x", pg.groupID, reciever)
	}

	if peer == nil { //情况二：存在ID，但是还没有完成协议handshake(没有节点实体peer对象)
		return fmt.Errorf("reciever: %x has not been connected yet", reciever)
	}
	return peer.sendMsg(msg) //情况三：存在且完成连接，向本节点发送msg消息
}

func (pg *PeerGroup) NewLowerConsensusMessage(selfNode common.NodeID, payload []byte) *Message {
	msg := &Message{
		MsgCode: lowerConsensusCode,
		NetID:   pg.groupID,
		From:    selfNode,
		PayLoad: payload,
	}
	return msg
}

func (pg *PeerGroup) NewUpperConsensusMessage(selfNode common.NodeID, payload []byte) *Message {
	msg := &Message{
		MsgCode: upperConsensusCode,
		NetID:   pg.groupID,
		From:    selfNode,
		PayLoad: payload,
	}
	return msg
}

func (pg *PeerGroup) NewControlMessage(selfNode common.NodeID, payload []byte) *Message {
	msg := &Message{
		MsgCode: controlCode,
		NetID:   pg.groupID,
		From:    selfNode,
		PayLoad: payload,
	}
	return msg
}

func (pg *PeerGroup) NewIntraWorldStateUpdateMessage(selfNode common.NodeID, payload []byte) *Message {
	msg := &Message{
		MsgCode: intraWSUpdateCode,
		NetID:   pg.groupID,
		From:    selfNode,
		PayLoad: payload,
	}
	return msg
}

func (pg *PeerGroup) BackMembers() []common.NodeID {
	pg.groupMu.RLock()
	defer pg.groupMu.RUnlock()
	backMembers := make([]common.NodeID, len(pg.members))
	i := 0
	for nodeID, _ := range pg.members {
		backMembers[i] = nodeID
		i++
	}
	return backMembers
}

func (pg *PeerGroup) BackLiveMembers() []common.NodeID {
	pg.groupMu.RLock()
	defer pg.groupMu.RUnlock()
	backMembers := make([]common.NodeID, 0)
	for nodeID, pr := range pg.members {
		if pr == nil {
			continue
		}
		backMembers = append(backMembers, nodeID)
	}
	return backMembers
}
