package dpnet

import (
	"fmt"
	"p3Chain/common"
	"sync"
)

type RoleType uint8

const (
	UnKnown RoleType = iota
	Booter
	Leader
	Follower
)
const (
	UpperNetID   = "upper"
	InitialNetID = "chaos"
)

type DpNet struct {
	SubNets map[string]*SubNet
	Leaders map[string]common.NodeID
	Booters []common.NodeID

	dpNetMutex sync.RWMutex
}

type SubNet struct {
	NetID   string
	Leaders []common.NodeID
	Nodes   map[common.NodeID]*Node
	// BlockCount uint64 //当前peer节点本地区块链高度
}

// A higher level node from the sight of application layer.
type Node struct {
	NetID  string
	Role   RoleType
	NodeID common.NodeID // the node's public key (ECC public key marshalled)
}

// 中心节点部署网络使用的结构体+
type DpNetCentral struct { //整个DpNet网络的结构
	SubNets []SubNetCentral //子网集合
	Leaders []common.NodeID //记录所有Leader节点的NodeID
	Booters []common.NodeID //记录所有Booter节点的NodeID
}

// 中心节点部署网络使用的结构体
type SubNetCentral struct { //单个子网的结构
	NetID string
	Nodes []Node
}

func NewNode(nodeID common.NodeID, role int, netID string) *Node {
	node := &Node{
		NetID:  netID,
		NodeID: nodeID,
		Role:   RoleType(role),
	}
	return node
}

func NewSubNet(netID string) *SubNet {
	return &SubNet{
		NetID:   netID,
		Leaders: make([]common.NodeID, 0),
		Nodes:   make(map[common.NodeID]*Node),
	}
}

func NewDpNet() *DpNet {
	dn := new(DpNet)
	dn.SubNets = make(map[string]*SubNet)
	dn.Leaders = make(map[string]common.NodeID)
	return dn
}

// Add Booter, Leader and Follower node to a subnet. For a node has been in a subnet
// this function would do nothing. Please use UpdateNode to update the node of dpnet.
// 根据节点的身份,将其添加到指定的子网中
func (dpn *DpNet) AddNode(node Node) error {
	dpn.dpNetMutex.Lock()
	defer dpn.dpNetMutex.Unlock()

	if node.Role == Booter { //新添加Node是Booter,查询其是否位于dpn.Booters集合中，不在的话将其加入dpn.Booters
		var flag bool
		for _, booter := range dpn.Booters {
			if booter == node.NodeID {
				flag = true
				break
			}
		}
		if flag { //如果已经存在于dpn.Booters集合中，直接退出
			return nil
		}
		dpn.Booters = append(dpn.Booters, node.NodeID) //不在dpn.Booters集合中,将其添加
		return nil
	}
	if node.NetID == "" {
		return fmt.Errorf("node is not booter but with nil NetID")
	}
	subNet, ok := dpn.SubNets[node.NetID] //在本地查询节点要加入的子网
	if ok {                               //ok==true  表示存在对于子网
		if _, exist := subNet.Nodes[node.NodeID]; exist { //对于子网是否已经存在此节点
			return nil //存在,则退出
		} else {
			subNet.Nodes[node.NodeID] = &node //不存在,需要将该节点加入到子网中
			if node.Role == Leader {          //判断是否为领导节点
				dpn.Leaders[node.NetID] = node.NodeID
				subNet.Leaders = append(subNet.Leaders, node.NodeID) //需要额外将其添加到领导节点subNet.Leaders集合中
			}
			return nil
		}
	} else { //不存在对应子网
		newSubNet := NewSubNet(node.NetID)   //创建一个新的对应子网
		newSubNet.Nodes[node.NodeID] = &node //向新子网中添加该节点
		if node.Role == Leader {             //判断是否为领导节点
			dpn.Leaders[node.NetID] = node.NodeID
			newSubNet.Leaders = append(newSubNet.Leaders, node.NodeID)
		}
		dpn.SubNets[node.NetID] = newSubNet //将此子网更新到dpn网络中
		return nil
	}
}

// 本方法只用于更新已有节点在当前节点组内的身份(不能将节点从一个节点组移动到另一个节点组)
// 本方法不能用于Booter节点降级为其他类型节点的情况,因为这意味着需要进行跨组更新(因为需要为其提供新的子网ID)。因此对于Booter节点的role变更,需要结合使用DelNode()和AddNode()
func (dpn *DpNet) UpdateNode(node Node) error {
	dpn.dpNetMutex.Lock()
	defer dpn.dpNetMutex.Unlock()

	newNode := new(Node)
	newNode.NetID = node.NetID
	newNode.NodeID = node.NodeID
	newNode.Role = node.Role

	//1.需要查看节点是否存在于指定的节点组
	subNet, ok := dpn.SubNets[newNode.NetID] //先查看指定的子网是否存在
	if !ok {
		return fmt.Errorf("the given subnet does not exist")
	}
	oldNode, exist := subNet.Nodes[newNode.NodeID] //在查看指定的子网内是否有此节点
	if !exist {
		return fmt.Errorf("node is not exist in given subnet")
	}

	if oldNode.Role == Follower && (newNode.Role == Booter || newNode.Role == Leader) { // 1.此节点是从Follower升级为Leader或Booter
		if newNode.Role == Booter { //将节点更新为Booter节点,需要更新dpn的Booters队列,同时将其从原来的节点组删除
			dpn.Booters = append(dpn.Booters, newNode.NodeID)
			delete(subNet.Nodes, newNode.NodeID)
		} else { //将节点新增为Leader节点,还需要更新subNet的Leaders队列
			subNet.Nodes[newNode.NodeID].Role = newNode.Role
			dpn.Leaders[newNode.NetID] = newNode.NodeID
			subNet.Leaders = make([]common.NodeID, 0) // 清空是为了保证：一个节点组只能有一个Leader节点
			subNet.Leaders = append(subNet.Leaders, newNode.NodeID)
		}
	} else if oldNode.Role == Leader && newNode.Role == Follower { // 2.此节点是从Leader降级为Follower
		subNet.Nodes[newNode.NodeID].Role = newNode.Role
		delete(dpn.Leaders, subNet.NetID)
		subNet.Leaders = make([]common.NodeID, 0) // 该子网组暂时失去了Leader节点

	} else if oldNode.Role == Leader && newNode.Role == Booter { // 3.此节点是从Leader升级为Booter
		dpn.Booters = append(dpn.Booters, newNode.NodeID)
		delete(subNet.Nodes, newNode.NodeID)
		subNet.Leaders = make([]common.NodeID, 0) // 暂时失去了Leader节点
		delete(dpn.Leaders, newNode.NetID)
	} else {
		return fmt.Errorf("setting the node's identity as the default is not allowed")
	}
	return nil
}

func (dpn *DpNet) DelNode(node Node) error {
	dpn.dpNetMutex.Lock()
	defer dpn.dpNetMutex.Unlock()

	// 1.被删除节点为Booter
	if node.Role == Booter {
		for index, nodeID := range dpn.Booters {
			if nodeID == node.NodeID {
				dpn.Booters = append(dpn.Booters[:index], dpn.Booters[index+1:]...)
				break
			}
		}
		return nil
	}

	newNode := new(Node)
	newNode.NetID = node.NetID
	newNode.NodeID = node.NodeID
	newNode.Role = node.Role

	// 需要查看节点是否存在于指定的节点组
	subNet, ok := dpn.SubNets[newNode.NetID] //先查看指定的子网是否存在
	if !ok {
		return fmt.Errorf("the given subnet does not exist")
	}
	oldNode, exist := subNet.Nodes[newNode.NodeID] //在查看指定的子网内是否有此节点
	if !exist {
		return fmt.Errorf("node is not exist in given subnet")
	}

	if oldNode.Role == Leader { // 如果删除的是Leader节点，还需要进行一些额外的操作
		subNet.Leaders = make([]common.NodeID, 0)
		delete(dpn.Leaders, newNode.NetID)
	}

	delete(subNet.Nodes, newNode.NodeID)

	return nil

}

func (dpn *DpNet) BackSubNetLeader(groupID string) common.NodeID {
	dpn.dpNetMutex.Lock()
	defer dpn.dpNetMutex.Unlock()

	leaders := dpn.SubNets[groupID].Leaders
	if len(leaders) == 0 {
		return common.NodeID{}
	}

	return leaders[0]

}

// return subnet nodes ID list without the self one
// 返回节点所在网络中除自身之外的其余所有节点的ID
func (dpn *DpNet) BackSubnetNodesID(selfNode common.NodeID, netID string) []common.NodeID {
	dpn.dpNetMutex.RLock()
	defer dpn.dpNetMutex.RUnlock()

	nodesID := make([]common.NodeID, 0)
	if netID == UpperNetID { //节点位于Upper网络中,返回的NodeID集合中包含除自己之外的所有网络Leader节点和Booter节点的ID
		for _, id := range dpn.Leaders { //获取所有Leader节点ID
			if id != selfNode {
				nodesID = append(nodesID, id)
			}
		}
	outer:
		for _, id := range dpn.Booters { //获取所有Booter节点ID
			if id != selfNode {
				for i := 0; i < len(nodesID); i++ {
					if id == nodesID[i] { // in case that part-time booter is allowed
						continue outer
					}
				}
				nodesID = append(nodesID, id)
			}
		}
		return nodesID
	}
	if subnet, ok := dpn.SubNets[netID]; ok { //节点位于下层的某一个子网中，返回该子网中除自身之外的其他所有节点的ID
		for _, node := range subnet.Nodes {
			if node.NodeID != selfNode {
				if !InSlice(nodesID, node.NodeID) { //如果已经存在与nodesID集合中,就不需要重复添加了
					nodesID = append(nodesID, node.NodeID)
				}
			}
		}
		return nodesID
	} else {
		return nodesID
	}
}

// 查询某一NodeID是否已经在items集合中
func InSlice(items []common.NodeID, item common.NodeID) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func (node *Node) String() string {
	res := fmt.Sprintf("NodeID: %x, NetID: %s, Role: %d", node.NodeID, node.NetID, node.Role)
	return res
}

func (dn *DpNet) String() string {
	res := fmt.Sprintf("Subnets number: %d, leader number: %d, booter number: %d\n", len(dn.SubNets), len(dn.Leaders), len(dn.Booters))
	res += "Subnets are:\n"
	for netID, subnet := range dn.SubNets {
		res += fmt.Sprintf("NetID: %s, member number: %d\n", netID, len(subnet.Nodes))
	}
	return res
}
