package netconfig

const (
	stringConfig       = "NETCONFIG"
	stringConfigInit   = stringConfig + ":" + "INIT"
	stringConfigUpdate = stringConfig + ":" + "UPDATE"
)

const (
	defaultMsgLifeTime = 60 // 60s,1min
)

const (
	// topic configinit: message codes
	selfNodeState = 0x00 // broadcast the self node state including NodeID, subnet ID and role.
	confirmLeader = 0x01 // confirm the leader of the node's subnet.
	setOrder      = 0x02 // only for leaders, set a booter as order to help upper-level consensus.
	setRole       = 0x03 // force nodes to set their subnet ID and role.
	centralConfig = 0x04 // 中心节点发送的中心化配置消息

	ReconnectState = 0x05 // 断连节点的重连请求消息
	DpNetInfo      = 0x06 // Booter向断连节点回复整个分区网络的拓扑结构

	UpdateNodeState = 0x07 //更新节点组内一个节点的状态(通常是更新节点的Role)
	AddNewNode      = 0x08 //向节点组中添加一个新的节点
	DelNode         = 0x09 //从节点组中删除一个节点

	LeaderChange = 0x10
)
