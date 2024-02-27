package centre

import (
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/crypto"
	"p3Chain/rlp"
)

type Message struct {
	MsgCode uint64             //消息码
	Payload dpnet.DpNetCentral //整个DpNet网络的部署情况(用于中心节点发送给其他节点)
	From    common.NodeID      //消息包中包含发送者的nodeID
	Hash    common.Hash        //消息的哈希值
}

// 创建一条新的msg消息(根据dpnet.DpNet对象实现dpnet.DpNetCentral对象)
func NewMessage(msgCode uint64, payload *dpnet.DpNet, sender common.NodeID) *Message {
	var msg Message
	msg.MsgCode = msgCode
	msg.From = sender

	msg.Payload.SubNets = make([]dpnet.SubNetCentral, 0) //先创建子网群组
	msg.Payload.Booters = make([]common.NodeID, 0)       //Booter列表
	msg.Payload.Leaders = make([]common.NodeID, 0)       //Leader列表

	msg.Payload.Booters = append(msg.Payload.Booters, payload.Booters...)

	for groupID, group := range payload.SubNets {
		//1.根据子网id挨个创建subNet对象
		subNet := dpnet.SubNetCentral{
			NetID: groupID,
			Nodes: make([]dpnet.Node, 0),
		}
		//2.填充Node集合(同一个子网下的放到一个subNet集合中)
		for _, node := range group.Nodes {
			subNet.Nodes = append(subNet.Nodes, *node)
			if node.Role == dpnet.Booter {
				msg.Payload.Booters = append(msg.Payload.Booters, node.NodeID)
			} else if node.Role == dpnet.Leader {
				msg.Payload.Leaders = append(msg.Payload.Leaders, node.NodeID)
			}
		}
		//3.新建的subNet对象追加到SubNets中
		msg.Payload.SubNets = append(msg.Payload.SubNets, subNet)
	}
	return &msg
}

// 给出消息先rlp编码后SHA-3 的哈希值
func (m *Message) CalculateHash() common.Hash {
	if (m.Hash == common.Hash{}) {
		enc, _ := rlp.EncodeToBytes(m) //rlp编码
		m.Hash = crypto.Sha3Hash(enc)  //计算SHA-3 后的哈希值
	}
	return m.Hash
}
