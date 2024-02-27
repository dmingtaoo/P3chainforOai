package consensus

import (
	"p3Chain/common"
	"p3Chain/core/separator"
)

// separator.PeerGroup is a LowerChannel  separator.PeerGroup 类实现了此接口
type LowerChannel interface {
	CurrentMsgs() []*separator.Message                                                         //返回当前节点组消息池内的separator协议消息
	MarkRetrievedMsgs(msgIDs []common.Hash)                                                    //给指定ID的msg消息标记已读标志位
	NewLowerConsensusMessage(selfNode common.NodeID, payload []byte) *separator.Message        //创建一条新的组内共识消息(基于separator协议)
	NewIntraWorldStateUpdateMessage(selfNode common.NodeID, payload []byte) *separator.Message //创建一条分片内滞后版世界状态更新消息
	NewControlMessage(selfNode common.NodeID, payload []byte) *separator.Message               // 创建一条控制类消息
	MsgBroadcast(msg *separator.Message)                                                       //组内广播指定的msg消息
	MsgSend(msg *separator.Message, receiver common.NodeID) error                              //单点发送msg消息
	BackMembers() []common.NodeID                                                              //返回当前节点组内所有成员的NodeID
}

// eparator.PeerGroup 类实现了此接口,所有的Leader节点组成一个upper子网
type UpperChannel interface {
	CurrentMsgs() []*separator.Message
	MarkRetrievedMsgs(msgIDs []common.Hash)
	NewUpperConsensusMessage(selfNode common.NodeID, payload []byte) *separator.Message
	MsgBroadcast(msg *separator.Message)
	MsgSend(msg *separator.Message, receiver common.NodeID) error
	BackMembers() []common.NodeID
}
