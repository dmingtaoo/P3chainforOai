package centre

import (
	"encoding/json"
	"io/ioutil"
	"p3Chain/common"
	"p3Chain/core/dpnet"
)

// central协议中心节点json配置
type CentralObj struct {
	Peers     []PeerObj //节点集合
	PeerCount int       //需配置的节点数
	NetCount  int       //所需子网的个数
}

// 单个节点的配置信息
type PeerObj struct {
	NodeID_S string
	NodeID   common.NodeID  //需配置的节点的NodeID
	Role     dpnet.RoleType //节点的Role  ( ==2 表示Leader节点 ; ==3 表示Follwer节点)
	NetID    string         //节点的子网ID
}

// 定义一个可供全局使用的CentralObj对象
var CentralObject *CentralObj

// 导入netconfig包时，自动调用init函数初始化CentralObject(从conf/central.json读取相关配置参数)
func init() {
	//默认参数
	CentralObject = &CentralObj{
		Peers:     make([]PeerObj, 0),
		PeerCount: 0,
		NetCount:  0,
	}

	//读取配置文件
	CentralObject.Reload()
}

// 读取配置文件
func (c *CentralObj) Reload() {
	data, err := ioutil.ReadFile("conf/central.json")
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, c); err != nil { //按照json格式解析，配置 CentralObject
		panic(err)
	}
	for _, v := range c.Peers {
		nodeID := common.Hex2Bytes(v.NodeID_S)
		copy(v.NodeID[:], nodeID)
	}
}
