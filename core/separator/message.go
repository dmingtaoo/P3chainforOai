package separator

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/rlp"
	"time"
)

type Message struct {
	MsgCode uint64        //消息码
	NetID   string        //指定此消息要发送给separator协议层的哪一个子网
	From    common.NodeID //消息包中包含发送者的nodeID
	PayLoad []byte        //消息的有效载荷

	Hash common.Hash // Cached Hash of the envelope to avoid reHashing every time   消息的哈希值
}

type msgState struct {
	msg        *Message  //Message消息包实体
	expireTime time.Time //消息包的过期时间
	retrieved  bool      //记录本消息是否已经被读取过了
}

// 给消息设置已读标志
func (ms *msgState) MarkRetrived() {
	ms.retrieved = true
}

// 给出消息先rlp编码后SHA-3 的哈希值
func (m *Message) CalculateHash() common.Hash {
	if (m.Hash == common.Hash{}) {
		enc, _ := rlp.EncodeToBytes(m) //rlp编码
		m.Hash = crypto.Sha3Hash(enc)  //计算SHA-3 后的哈希值
	}
	return m.Hash
}

func (msg *Message) Broadcast(pg *PeerGroup) {
	pg.MsgBroadcast(msg) //将msg广播给节点组内所有成员节点
}

func (msg *Message) Send(pg *PeerGroup, reciever common.NodeID) {
	if err := pg.MsgSend(msg, reciever); err != nil { //指定节点组内某一成员节点发送msg
		glog.V(logger.Info).Infoln("fail in send Message: %v", err)
		fmt.Println("[message.go] Msg Send is error:", err)
	}
}

func Recieve(pg *PeerGroup) int {
	msgIDs := make([]common.Hash, 0) //消息的哈希值
	msgIndex := 0
	for _, Message := range pg.CurrentMsgs() { //返回消息池中剩余msg消息
		msgIndex++
		fmt.Println("当前节点子网群组:", pg.groupID, " 中消息池中读取的message消息:", Message)
		msgIDs = append(msgIDs, Message.Hash)
	}
	// fmt.Println("收到的message消息数:", msgIndex)
	pg.MarkRetrievedMsgs(msgIDs)
	return msgIndex
}

// 修改PayLoad载荷
func (msg *Message) SetPayLoad(payload []byte) {
	msg.PayLoad = payload
}

// 修改目标子网
func (msg *Message) SetNetID(netID string) {
	msg.NetID = netID
}

// 获取目标子网ID
func (msg *Message) GetNetID() string {
	return msg.NetID
}

// 获取发送者NodeID
func (msg *Message) GetNodeID() common.NodeID {
	return msg.From
}

// 获取载荷
func (msg *Message) GetPayLoad() []byte {
	return msg.PayLoad
}
