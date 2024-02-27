package api

import (
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/netconfig"
	"reflect"
)

// 提供关于p3Chain网络相关的API，包括查看网络结构、节点组成等一系列信息
type NetWorkService struct {
	netManager *netconfig.NetManager
}

type NodeInfo struct { // TODO:需要为每一个共识节点准备这样一个信息结构体
	NodeID   common.NodeID
	Role     dpnet.RoleType
	NetID    string
	State    bool
	LeaderID common.NodeID
}

type DPNetwork struct {
	Booters []common.NodeID
	Leaders []common.NodeID

	SubNets []DPSubNetwork
}

type DPSubNetwork struct {
	Leader    common.NodeID
	NetID     string
	Followers []common.NodeID
}

func NewNetWorkService(netManager *netconfig.NetManager) *NetWorkService {
	return &NetWorkService{
		netManager: netManager,
	}
}

func BackDPNetworkByDPNet(dpNet *dpnet.DpNet) *DPNetwork {
	dnw := new(DPNetwork)

	dnw.Booters = append(dnw.Booters, dpNet.Booters...)

	for _, leader := range dpNet.Leaders {
		dnw.Leaders = append(dnw.Leaders, leader)
	}

	var dsnw DPSubNetwork
	for _, subNet := range dpNet.SubNets {
		dsnw.NetID = subNet.NetID
		if subNet.Leaders != nil {
			dsnw.Leader = subNet.Leaders[0]
		}

		dsnw.Followers = dpNet.BackSubnetNodesID(dsnw.Leader, dsnw.NetID)

		dnw.SubNets = append(dnw.SubNets, dsnw)

	}

	return dnw
}

// 返回当前节点保存的整个P3-Chain网络的节点分布情况
func (nws *NetWorkService) GetDpNetInfo() *DPNetwork {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}

	return BackDPNetworkByDPNet(dpNet)
}

// 返回所有共识节点的NodeID
func (nws *NetWorkService) GetAllConsensusNode() []common.NodeID {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}
	nodeList := make([]common.NodeID, 0)
	nodeList = append(nodeList, dpNet.Booters...)

	for groupID, _ := range dpNet.SubNets {
		nodeList = append(nodeList, dpNet.BackSubnetNodesID(common.NodeID{}, groupID)...)
	}

	return nodeList
}

// 返回共识节点的数量
func (nws *NetWorkService) GetConsensusNodeCount() int {
	return len(nws.GetAllConsensusNode())
}

// 根据节点的NodeID返回节点的具体信息(节点的身份/所属的子网名/子网的Leader节点/节点的IP地址？/节点的运行状态/节点的当前区块高度)
func (nws *NetWorkService) GetNodeInfoByNodeID(nodeID common.NodeID) *NodeInfo {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}
	nodeInfo := new(NodeInfo)

	for _, booter := range dpNet.Booters {
		if reflect.DeepEqual(booter, nodeID) {
			nodeInfo.NodeID = nodeID
			nodeInfo.NetID = dpnet.UpperNetID
			nodeInfo.Role = dpnet.Booter
			nodeInfo.LeaderID = common.NodeID{}

			// TODO:节点是否在正常工作需要ping该节点 , IP地址可能需要后续为节点添加新字段
			return nodeInfo
		}
	}

	for _, group := range dpNet.SubNets {
		if len(group.Leaders) != 0 && reflect.DeepEqual(group.Leaders[0], nodeID) {
			nodeInfo.NodeID = nodeID
			nodeInfo.LeaderID = group.Leaders[0]
			nodeInfo.NetID = group.NetID
			nodeInfo.Role = dpnet.Leader

			// TODO:ping该节点，查看是否在正常工作
			return nodeInfo
		}

		// TODO:检查是否是follower
		for follower, _ := range group.Nodes {
			if reflect.DeepEqual(follower, nodeID) {
				nodeInfo.NodeID = nodeID
				nodeInfo.LeaderID = group.Leaders[0]
				nodeInfo.NetID = group.NetID
				nodeInfo.Role = dpnet.Follower

				// TODO:ping该节点，查看是否在正常工作
				return nodeInfo
			}
		}
	}

	return nil
}

// 返回当前节点的具体信息
func (nws *NetWorkService) GetNodeInfoSelf() *NodeInfo {
	nodeSelf := nws.netManager.SelfNode

	return nws.GetNodeInfoByNodeID(nodeSelf.NodeID)

}

// 返回当前P3-Chain网络中分区的数量
func (nws *NetWorkService) GetGroupCount() int {

	dpNet := nws.netManager.GetDpViewNet()

	if dpNet == nil || dpNet.SubNets == nil {
		return 0
	} else {
		return len(dpNet.SubNets)
	}

}

// 返回P3-Chain网络中所有分区的分区组名
func (nws *NetWorkService) GetAllGroupName() []string {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}
	groupNames := make([]string, 0)

	if dpNet.SubNets != nil {
		for group, _ := range dpNet.SubNets {
			groupNames = append(groupNames, group)
		}
		return groupNames
	}

	return nil
}

// 返回P3-Chain网络中UpperNet中所有节点的NodeID
func (nws *NetWorkService) GetUpperNetNodeList() []common.NodeID {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}

	upperNodes := make([]common.NodeID, 0)

	upperNodes = append(upperNodes, dpNet.Booters...)

	for _, leader := range dpNet.Leaders {
		upperNodes = append(upperNodes, leader)
	}

	return upperNodes
}

// 返回P3-Chain网络中所有Booter节点的NodeID
func (nws *NetWorkService) GetAllBooters() []common.NodeID {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}

	return dpNet.Booters
}

// 返回P3-Chain网络中所有Leader节点的NodeID
func (nws *NetWorkService) GetAllLeaders() []common.NodeID {
	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}

	leaderNodes := make([]common.NodeID, 0)
	for _, leader := range dpNet.Leaders {
		leaderNodes = append(leaderNodes, leader)
	}

	return leaderNodes
}

// 返回P3-Chain网络中指定分区中所有节点的NodeID
func (nws *NetWorkService) GetNodeListByGroupName(groupName string) []common.NodeID {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}
	groupNodeList := make([]common.NodeID, 0)

	subNet := dpNet.SubNets[groupName]

	for follower, _ := range subNet.Nodes {
		groupNodeList = append(groupNodeList, follower)
	}

	return groupNodeList
}

// 返回指定分区的Leader节点(仅NodeID)
func (nws *NetWorkService) GetLeaderNodeIDByGroupName(groupName string) common.NodeID {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return common.NodeID{}
	}

	return dpNet.Leaders[groupName]

}

// 返回指定分区的详细信息
func (nws *NetWorkService) GetSubNetByGroupName(groupName string) *DPSubNetwork {

	dpNet := nws.netManager.GetDpViewNet()
	if dpNet == nil {
		return nil
	}

	subNet, ok := dpNet.SubNets[groupName]
	if !ok {
		return nil
	}

	dpSubNet := new(DPSubNetwork)
	dpSubNet.NetID = subNet.NetID
	if len(subNet.Leaders) != 0 {
		dpSubNet.Leader = subNet.Leaders[0]
	}

	dpSubNet.Followers = dpNet.BackSubnetNodesID(dpSubNet.Leader, dpSubNet.NetID)

	return dpSubNet
}
