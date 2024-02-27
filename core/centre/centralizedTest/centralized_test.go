package netconfig

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"p3Chain/accounts"
	"p3Chain/common"
	"p3Chain/core/centre"
	"p3Chain/core/dpnet"
	"p3Chain/core/netconfig"
	"p3Chain/core/separator"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	"p3Chain/database"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
	"p3Chain/utils"
	"testing"
	"time"
)

// 模拟用本地监听地址
var listenAddr = []string{
	"127.0.0.1:20130",
	"127.0.0.1:20131",
	"127.0.0.1:20132",
}

var (
	groupCount    int = 2
	groupLeader   int = 2
	groupFollower int = 2
)

var testNodes []mockNode //存储所有测试用模拟节点

type mockNode struct {
	prvKey       *ecdsa.PrivateKey
	listenAddr   string
	discoverNode *discover.Node
}

type DpPer struct {
	peer     Dper
	peerchan chan bool
}

var DpPers = [3]DpPer{
	{
		peerchan: make(chan bool),
	},
	{
		peerchan: make(chan bool),
	},
	{
		peerchan: make(chan bool),
	},
}

// 创建一组测试用网络节点(discover.Node) -- 存储于全局数组testNode中
func initTest() error {
	for i := 0; i < len(listenAddr); i++ {
		prv, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		discoverNode, err := discover.ParseUDP(prv, listenAddr[i]) //根据给定的私钥和监听地址返回一个node
		if err != nil {
			return err
		}
		mn := mockNode{
			prvKey:       prv,           //节点的私钥
			listenAddr:   listenAddr[i], //节点的监听地址
			discoverNode: discoverNode,  //本地节点
		}
		testNodes = append(testNodes, mn)
	}
	return nil
}

type mockConfig struct {
	bootstrapNodes []*discover.Node
	listenAddr     string
	nat            nat.Interface
	isBooter       bool
	//netID          string  不再需要
}
type Dper struct {
	selfKey        keys.Key
	blockDB        database.Database //本地维护一个区块数据库
	storageDB      database.Database //本地维护一个存储数据库
	accountManager *accounts.Manager

	netManager *netconfig.NetManager //本地维护一个P3-Chain网络
	p2pServer  *server.Server
}

func mockDper(index int, mc *mockConfig) (*Dper, error) {
	loadFile := "conf/key" + fmt.Sprint(index)
	prvKey, err := utils.LoadKey(loadFile) //先尝试Load私钥
	if err != nil {
		lprvk, err := crypto.GenerateKey()
		if err == nil {
			utils.SaveKey(*lprvk, loadFile)
		}
	}
	selfKey := keys.NewKeyFromECDSA(&prvKey)
	selfNodeID := crypto.KeytoNodeID(&prvKey.PublicKey)            //当前节点的NodeID(role和netID由中心节点配置)
	selfNode := dpnet.NewNode(selfNodeID, int(dpnet.Follower), "") //默认角色为Follwer

	tempName := "p3Chain:test"
	shh := whisper.New()                                                        //创建本节点的whisper客户端
	spr := separator.New()                                                      //创建本节点的separator客户端
	cen := centre.CentralNew(*selfNode, groupCount, groupLeader, groupFollower) //创建本节点的central客户端
	srv := &server.Server{                                                      //创建本节点的服务器
		PrivateKey:     &prvKey,
		Discovery:      true,
		BootstrapNodes: mc.bootstrapNodes,
		MaxPeers:       10,
		Name:           tempName,
		Protocols:      []server.Protocol{shh.Protocol(), spr.Protocol(), cen.Protocol()}, //支持三项协议
		ListenAddr:     mc.listenAddr,
		NAT:            mc.nat,
	}

	//启动服务器模块
	srv.Start()
	dpNet := dpnet.NewDpNet()
	netManager := netconfig.NewNetManager(&prvKey, dpNet, srv, shh, spr, cen, nil, 0, selfNode) //为本地节点维护一个P3-Chain网络

	dper := &Dper{
		selfKey:    *selfKey,
		netManager: netManager, //本地P3-Chain网络的控制对象(或者说工具)
		p2pServer:  srv,        //本地p2p网络的控制对象
	}
	return dper, nil
}

func TestMultiNodeWorkFlow(t *testing.T) {
	if err := initTest(); err != nil { //创造一组测试用Node节点
		t.Fatalf("fail in initTest: %v", err)
	}
	glog.SetToStderr(true)
	tempMockCfg := [3]mockConfig{
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[0].listenAddr,
			nat:            nat.Any(),
			isBooter:       true,
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[1].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[2].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
		},
	}
	tempMockCfg[1].bootstrapNodes = append(tempMockCfg[1].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[2].bootstrapNodes = append(tempMockCfg[2].bootstrapNodes, testNodes[0].discoverNode)

	go dpnetNode(0, &tempMockCfg[0]) //0号节点,作为中心节点,随机加入某一个子网做Follwer
	go dpnetNode(1, &tempMockCfg[1])
	go dpnetNode(2, &tempMockCfg[2])

	//等待以上三个协程执行完毕
	for i := 0; i < 3; i++ {
		<-DpPers[i].peerchan
	}
	//调用separator协议完成消息的发送
	msgcode := separator.MessageCode
	var payload []byte
	tempMsg := [2]separator.Message{
		{ //消息一: 1号节点在其所在节点组内的广播消息
			MsgCode: uint64(msgcode),
			NetID:   DpPers[1].peer.netManager.SelfNode.NetID,
			From:    DpPers[1].peer.netManager.SelfNode.NodeID,
			PayLoad: payload,
		},
		{ //消息二：2号节点在其所在节点组内的广播消息
			MsgCode: uint64(msgcode),
			NetID:   DpPers[2].peer.netManager.SelfNode.NetID,
			From:    DpPers[2].peer.netManager.SelfNode.NodeID,
			PayLoad: payload,
		},
	}
	for i := 0; i < 2; i++ { //计算Msg哈希值
		tempMsg[i].Hash = tempMsg[i].CalculateHash()
		//fmt.Printf("tempMsg[%d].Hash: %v\n", i, tempMsg[i].Hash)
	}

	//1. 1号节点广播给节点组内所有成员节点
	if testPg, err := DpPers[1].peer.netManager.Spr.BackPeerGroup(tempMsg[0].NetID); err != nil { //返回子网ID对应的 peergroup
		t.Logf("fail in Back PeerGroup: %s", err)
	} else {
		tempMsg[0].Broadcast(testPg)
		time.Sleep(2 * time.Second)
	}

	//2. 2号节点广播给节点组内所有成员节点
	if testPg, err := DpPers[2].peer.netManager.Spr.BackPeerGroup(tempMsg[1].NetID); err != nil { //返回子网ID对应的 peergroup
		t.Logf("fail in Back PeerGroup: %s", err)
	} else {
		tempMsg[1].Broadcast(testPg)
		time.Sleep(2 * time.Second)
	}

	//3. 0号节点负责接收消息
	var msgIndex int
	msgIndex = separator.Recieve(DpPers[0].peer.netManager.Spr.PeerGroups[DpPers[0].peer.netManager.SelfNode.NetID])
	fmt.Println("收到的message消息数:", msgIndex)
	time.Sleep(3 * time.Second)
	fmt.Println("--------------------------------------------------------------------------")

}

func dpnetNode(index int, tempMockCfg *mockConfig) (*Dper, error) {
	testDper, err := mockDper(index, tempMockCfg) //按照配置信息，在本地创建一个P3-Chain网络节点，并维护一个P3-Chain网络
	if err != nil {
		return nil, errors.New("fail in mockDper: " + fmt.Sprint(err))
	}
	time.Sleep(3 * time.Second) //等待server模块执行完central协议的Run函数

	//打印节点NodeID
	fmt.Printf("%d号节点NodeID: %x", index, testDper.netManager.SelfNode.NodeID)
	fmt.Println()

	//方法二使用
	if err := testDper.netManager.StartConfigChannel(); err != nil { //为whisper客户端配置一个对接收消息的处理方法
		return nil, errors.New("fail in start config channel: " + fmt.Sprint(err))
	}
	time.Sleep(3 * time.Second)

	//central协议的Run函数已经启动,中心节点负责对网络进行配置,然后广播发送给其他所有节点
	if tempMockCfg.isBooter == true { //这里选择Booter节点作为中心节点,因此此节点与其他所有节点都有静态连接
		//1.根据central协议cen对象获取其他节点的NodeID(从knownPeers集合中获取)
		//idArray := testDper.netManager.Cen.GetKnowsPeerNodeID()
		//idArray = append(idArray, testDper.netManager.SelfNode.NodeID) //把自己的NodeID也加入
		//2.进行dpnet网络配置
		//Node := make([]dpnet.Node, 0) //存储Node集合
		//for i := 0; i < len(idArray); i++ {
		//	node := dpnet.NewNode(idArray[i], int(dpnet.Follower), "") //根据NodeID集合创建Node集合
		//	Node = append(Node, *node)

		//}
		/*groupNum := 2               //分为两组进行配置
		Node[0].Role = dpnet.Leader //前两个节点分别作为两个子网的Leader节点(最先完成handshake的),其余节点都作为Follwer
		Node[0].NetID = "testnet1"
		Node[1].Role = dpnet.Leader
		Node[1].NetID = "testnet2"*/
		//1.从json对象中获取配置信息:子网数,节点数
		//groupNum := netconfig.CentralObject.NetCount
		peerNum := centre.CentralObject.PeerCount
		Node := make([]dpnet.Node, peerNum) //存储Node集合
		//2.根据json对象配置好各所有结点
		for i := 0; i < peerNum; i++ {
			//nodeID := common.Hex2Bytes(netconfig.CentralObject.Peers[i].NodeID_S)
			//copy(Node[i].NodeID[:], nodeID)
			common.Hex2NodeID(centre.CentralObject.Peers[i].NodeID_S, &Node[i].NodeID)
			Node[i].Role = centre.CentralObject.Peers[i].Role
			Node[i].NetID = centre.CentralObject.Peers[i].NetID
		}
		/*if peerNum < len(idArray) { //json对象配置的节点数不足,采集随机策略部署剩余节点
			rand.Seed(time.Now().UnixNano()) //产生一个随机数,随机分配节点属于某一个子网
			for i := peerNum; i < len(idArray); i++ {
				num := rand.Intn(100)
				if num < num/groupNum {
					Node[i].NetID = "testnet1"
				} else {
					Node[i].NetID = "testnet2"
				}
			}
		}*/
		// rand.Seed(time.Now().UnixNano()) //产生一个随机数,随机分配节点属于两个子网中的一个
		// for i := 2; i < len(idArray); i++ {
		// 	num := rand.Intn(100)
		// 	//
		// 	if num < num/groupNum {
		// 		Node[i].NetID = "testnet1"
		// 	} else {
		// 		Node[i].NetID = "testnet2"
		// 	}
		// }
		for i := 0; i < peerNum; i++ {
			fmt.Printf("正在加入cen对象-- NodeID: %x , NetID : %s\n", Node[i].NodeID, Node[i].NetID)
			fmt.Println()
			testDper.netManager.Cen.DpNet.AddNode(Node[i]) //将整个网络的配置情况更新到central协议对象的DpNet字段中
		}
		//查看中心节点是否完成网络部署
		/*viewNet := make(map[string]*dpnet.SubNet)
		viewNet = testDper.netManager.Cen.GetDPNetInfo()
		for netID, _ := range viewNet {
			fmt.Println("当前网络ID: ", viewNet[netID].NetID)
			fmt.Println("当前网络的领导节点哈希：", viewNet[netID].Leaders)
			fmt.Println("当前子网下网络节点个数:", len(viewNet[netID].Nodes))
			fmt.Println("当前子网下网络节点个数:", viewNet[netID].Nodes)
			fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~子网分割线")
		}
		fmt.Println("-------------------------------------------------------------节点分割线")
		fmt.Println("---------------------------以上为中心节点的配置内容---------------------------------------")*/
		//3.将配置信息广播给其他所有节点
		//方法一:调用server模块完成消息发送
		msg := centre.NewMessage(centre.ConfigureCode, &testDper.netManager.Cen.DpNet, testDper.netManager.SelfNode.NodeID)

		msg.Hash = msg.CalculateHash()

		peers := testDper.netManager.Cen.GetKnowsPeer() //获取所有节点的Peer对象

		for i := 0; i < len(peers); i++ { //向这些节点广播msg配置消息
			//fmt.Printf("第 %d 个节点的NodeID: %v\n", i+1, peers[i].GetNodeID())
			peers[i].SendMsg(msg)
		}
		//方法二:调用whisper模块完成消息发送
		/*if err := testDper.netManager.SendCentralConfig(); err != nil { //将dpNet网络配置信息发送给其他节点
			return nil, errors.New("fail in send self node state: " + fmt.Sprint(err))
		}*/
	}
	time.Sleep(3 * time.Second) //其他节点需要等待中心节点完成配置消息的发送

	testDper.netManager.CentralizedConf() //根据central协议结果(DpNet字段)更新viewNet字段
	//testDper.netManager.SelfNodeUpdate()  //更新netManager的SelfNode信息(有问题)

	//节点们显示网络部署配置情况(检查netManager.viewNet是否正确)
	viewNet := make(map[string]*dpnet.SubNet)
	viewNet = testDper.netManager.GetDPNetInfo()
	for netID, _ := range viewNet {
		//fmt.Println(index, "号节点--", "当前网络ID: ", viewNet[netID].NetID)
		//fmt.Println(index, "号节点--", "当前网络的领导节点哈希：", viewNet[netID].Leaders)
		//fmt.Println(index, "号节点--", "当前子网下网络节点个数:", len(viewNet[netID].Nodes))
		//fmt.Println(index, "号节点--", "当前子网下网络节点:", viewNet[netID].Nodes)
		for NodeID, Node := range viewNet[netID].Nodes { //更新netManager的selfNode
			//fmt.Println("当前节点NodeID: ", testDper.netManager.SelfNode.NodeID, "当前正在查询的NodeID : ", NodeID)
			if NodeID == testDper.netManager.SelfNode.NodeID {
				testDper.netManager.SelfNode.NetID = Node.NetID
				testDper.netManager.SelfNode.Role = Node.Role
				testDper.netManager.Spr.SetNode(Node.NetID, NodeID) //把自身的部署情况更新到spr字段
				break
			}
		}

		//fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~子网分割线")
	}
	//fmt.Println("-------------------------------------------------------------节点分割线")

	if err := testDper.netManager.ConfigLowerChannel(); err != nil { //将netManager字段中获取网络信息同步到spr字段中.最后赋给netManager的lowerChannel字段
		fmt.Println("节点", index, "spr同步失败", fmt.Sprint(err))
		return nil, errors.New("fail in config lower channel: " + fmt.Sprint(err))
	}
	//fmt.Println("节点", index, " spr同步结束。。。。。。。。。。")

	time.Sleep(2 * time.Second)
	if err := testDper.netManager.StartSeparateChannels(); err != nil { //让本地节点运行separator协议(获取peer对象实体)
		return nil, errors.New("fail in start separate channels: " + fmt.Sprint(err))
	}
	time.Sleep(5 * time.Second)
	//fmt.Println("separator运行成功....................")

	DpPers[index].peer = *testDper
	DpPers[index].peerchan <- true

	return testDper, nil
}
