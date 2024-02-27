package router

import (
	"fmt"
	"net/http"
	"p3Chain/api"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/ginHttp/pkg/app"
	e "p3Chain/ginHttp/pkg/error"
	"p3Chain/ginHttp/pkg/msgtype"
	loglogrus "p3Chain/log_logrus"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
)

func DPNetworkFormat(dnw *api.DPNetwork) (*msgtype.DPNetWork, error) {

	if dnw == nil {
		return nil, fmt.Errorf("%s", "the DPNetwork is not exist!")
	}

	dnwS := new(msgtype.DPNetWork)

	for _, booter := range dnw.Booters {
		dnwS.Booters = append(dnwS.Booters, fmt.Sprintf("%x", booter))
	}

	for _, leader := range dnw.Leaders {
		dnwS.Leaders = append(dnwS.Leaders, fmt.Sprintf("%x", leader))
	}

	for _, subnet := range dnw.SubNets {
		var subnetS msgtype.DPSubNetwork
		subnetS.Leader = fmt.Sprintf("%x", subnet.Leader)
		subnetS.NetID = subnet.NetID
		for _, follower := range subnet.Followers {
			subnetS.Followers = append(subnetS.Followers, fmt.Sprintf("%x", follower))
		}
		dnwS.SubNets = append(dnwS.SubNets, subnetS)
	}

	return dnwS, nil
}

func NodeInfoFormat(ni *api.NodeInfo) (*msgtype.DPNode, error) {
	if ni == nil {
		return nil, fmt.Errorf("%s", "the DPNetwork is not exist!")
	}

	nodeInfoS := new(msgtype.DPNode)

	nodeInfoS.NodeID = fmt.Sprintf("%x", ni.NodeID)
	nodeInfoS.LeaderID = fmt.Sprintf("%x", ni.LeaderID)
	nodeInfoS.NetID = ni.NetID

	switch ni.Role {
	case dpnet.UnKnown:
		nodeInfoS.Role = "UnKnown"
	case dpnet.Booter:
		nodeInfoS.Role = "Booter"
	case dpnet.Follower:
		nodeInfoS.Role = "Follower"
	case dpnet.Leader:
		nodeInfoS.Role = "Leader"
	default:
		nodeInfoS.Role = "UnKnown"
	}

	return nodeInfoS, nil

}

func DPSubNetworkFormat(dsn *api.DPSubNetwork) (*msgtype.DPSubNetwork, error) {
	if dsn == nil {
		return nil, fmt.Errorf("%s", "the DPSubNetwork is not exist!\n")
	}
	subnetS := new(msgtype.DPSubNetwork)

	subnetS.Leader = fmt.Sprintf("%x", dsn.Leader)
	subnetS.NetID = dsn.NetID
	for _, follower := range dsn.Followers {
		subnetS.Followers = append(subnetS.Followers, fmt.Sprintf("%x", follower))
	}

	return subnetS, nil
}

func BackDPNetWork(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackDPNetWork\n")
		dpNetwork := ns.GetDpNetInfo()
		appG := app.Gin{c}

		data, err := DPNetworkFormat(dpNetwork)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackDPNetWork  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_DPNETWORK, nil)
			return
		}

		appG.Response(http.StatusOK, e.SUCCESS, data)
		loglogrus.Log.Infof("Http: Reply to user request -- BackDPNetWork  succeed!\n")
	}
}

func BackAllConsensusNode(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackAllConsensusNode\n")
		nodeList := ns.GetAllConsensusNode()
		appG := app.Gin{c}
		if nodeList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackAllConsensusNode  warn:the DPNetwork is not exist!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_DPNETWORK, nil)
			return
		}

		type consensusNodes struct {
			Nodes     []string
			NodeCount int
		}
		var data consensusNodes
		for _, node := range nodeList {
			data.Nodes = append(data.Nodes, fmt.Sprintf("%x", node))
		}
		data.NodeCount = len(data.Nodes)
		appG.Response(http.StatusOK, e.SUCCESS, data)
		loglogrus.Log.Infof("Http: Reply to user request -- BackAllConsensusNode  succeed!\n")
	}
}

func BackNodeInfoByNodeID(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		snodeID := c.PostForm("nodeID")

		valid := validation.Validation{}
		valid.Required(snodeID, "nodeID").Message("查询节点的ID值不能为空\n")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("BackNodeInfoByNodeID", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- BackNodeInfoByNodeID  NodeID:%s\n", snodeID)
		nodeID, err := common.HexToNodeID(snodeID)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackNodeInfoByNodeID  warn:%s\n", err)
			appG.Response(http.StatusOK, e.INVALID_NODE_ID, nil)
			return
		}

		node := ns.GetNodeInfoByNodeID(nodeID)

		data, err := NodeInfoFormat(node)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackNodeInfoByNodeID  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_DPNETWORK, nil)
			return
		}
		appG.Response(http.StatusOK, e.SUCCESS, data)

		loglogrus.Log.Infof("Http: Reply to user request -- BackNodeInfoByNodeID  succeed!\n")
	}
}

func BackNodeInfoSelf(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackNodeInfoSelf\n")
		node := ns.GetNodeInfoSelf()
		appG := app.Gin{c}
		data, err := NodeInfoFormat(node)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackNodeInfoSelf  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_DPNETWORK, nil)
			return
		}
		appG.Response(http.StatusOK, e.SUCCESS, data)
		loglogrus.Log.Infof("Http: Reply to user request -- BackNodeInfoSelf  succeed!\n")
	}
}

func BackGroupCount(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackGroupCount\n")
		appG := app.Gin{c}
		appG.Response(http.StatusOK, e.SUCCESS, ns.GetGroupCount())
		loglogrus.Log.Infof("Http: Reply to user request -- BackGroupCount  succeed!\n")
	}
}

func BackAllGroupName(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackAllGroupName\n")
		appG := app.Gin{c}
		appG.Response(http.StatusOK, e.SUCCESS, ns.GetAllGroupName())
		loglogrus.Log.Infof("Http: Reply to user request -- BackAllGroupName  succeed!\n")
	}
}

func BackUpperNetNodeList(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackUpperNetNodeList\n")
		appG := app.Gin{c}
		upperNode := ns.GetUpperNetNodeList()
		upperNodeS := make([]string, 0)

		for _, node := range upperNode {
			upperNodeS = append(upperNodeS, fmt.Sprintf("%x", node))
		}
		appG.Response(http.StatusOK, e.SUCCESS, upperNodeS)
		loglogrus.Log.Infof("Http: Reply to user request -- BackUpperNetNodeList  succeed!\n")
	}
}

func BackAllBooters(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackAllBooters\n")
		appG := app.Gin{c}

		booters := ns.GetAllBooters()
		bootersS := make([]string, 0)
		for _, booter := range booters {
			bootersS = append(bootersS, fmt.Sprintf("%x", booter))
		}
		appG.Response(http.StatusOK, e.SUCCESS, bootersS)
		loglogrus.Log.Infof("Http: Reply to user request -- BackAllBooters  succeed!\n")
	}
}

func BackAllLeaders(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackAllLeaders\n")
		appG := app.Gin{c}

		leaders := ns.GetAllLeaders()
		leadersS := make([]string, 0)
		for _, leader := range leaders {
			leadersS = append(leadersS, fmt.Sprintf("%x", leader))
		}
		appG.Response(http.StatusOK, e.SUCCESS, leadersS)
		loglogrus.Log.Infof("Http: Reply to user request -- BackAllLeaders  succeed!\n")
	}
}

func BackNodeListByGroupName(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		group := c.PostForm("group")
		appG := app.Gin{c}
		valid := validation.Validation{}

		valid.Required(group, "group").Message("分区名不能为空")

		if valid.HasErrors() {
			app.MarkErrors("BackNodeListByGroupName", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- BackNodeListByGroupName\n")
		nodeList := ns.GetNodeListByGroupName(group)
		if nodeList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackNodeListByGroupName  warn:the DPNetwork is not exist!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_DPNETWORK, nil)
			return
		}
		data := make([]string, 0)
		for _, node := range nodeList {
			data = append(data, fmt.Sprintf("%x", node))
		}
		appG.Response(http.StatusOK, e.SUCCESS, data)
		loglogrus.Log.Infof("Http: Reply to user request -- BackNodeListByGroupName  succeed!\n")
	}
}

func BackLeaderNodeIDByGroupName(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		group := c.PostForm("group")
		appG := app.Gin{c}
		valid := validation.Validation{}

		valid.Required(group, "group").Message("分区名不能为空")

		if valid.HasErrors() {
			app.MarkErrors("BackLeaderNodeIDByGroupName", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- BackLeaderNodeIDByGroupName\n")
		leader := ns.GetLeaderNodeIDByGroupName(group)
		appG.Response(http.StatusOK, e.SUCCESS, fmt.Sprintf("%x", leader))
		loglogrus.Log.Infof("Http: Reply to user request -- BackLeaderNodeIDByGroupName  succeed!\n")
	}
}

func BackSubNetByGroupName(ns *api.NetWorkService) func(*gin.Context) {
	return func(c *gin.Context) {
		group := c.PostForm("group")
		appG := app.Gin{c}
		valid := validation.Validation{}

		valid.Required(group, "group").Message("分区名不能为空")

		if valid.HasErrors() {
			app.MarkErrors("BackSubNetByGroupName", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- BackSubNetByGroupName\n")
		subNet := ns.GetSubNetByGroupName(group)

		data, err := DPSubNetworkFormat(subNet)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackSubNetByGroupName  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_DPSUBNET, nil)
			return
		}
		appG.Response(http.StatusOK, e.SUCCESS, data)
		loglogrus.Log.Infof("Http: Reply to user request -- BackSubNetByGroupName  succeed!\n")
	}
}
