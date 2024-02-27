package separatortest

import (
	"fmt"
	"p3Chain/core/dpnet"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/rlp"
	"testing"
	"time"
)

func TestNodeUpdate(t *testing.T) {
	if err := initTest(); err != nil { //创造一组测试用Node节点
		loglogrus.Log.Errorf("fail in initTest: %v", err)
	}

	tempMockCfg := [5]mockConfig{
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[0].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[1].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[2].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet2",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[3].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet2",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[4].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
	}
	tempMockCfg[1].bootstrapNodes = append(tempMockCfg[1].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[2].bootstrapNodes = append(tempMockCfg[2].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[3].bootstrapNodes = append(tempMockCfg[3].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[4].bootstrapNodes = append(tempMockCfg[4].bootstrapNodes, testNodes[0].discoverNode)

	go dpnetNode(0, &tempMockCfg[0], "testnet1", dpnet.Leader)
	go dpnetNode(1, &tempMockCfg[1], "testnet1", dpnet.Follower)
	go dpnetNode(2, &tempMockCfg[2], "testnet2", dpnet.Leader)
	go dpnetNode(3, &tempMockCfg[3], "testnet2", dpnet.Follower)

	for i := 0; i < 4; i++ {
		<-DpPers[i].peerchan
	}

	fmt.Println()
	// 1.让testnet2的Leader节点打印upper PeerGroup的节点分布情况
	upperGroup, err := DpPers[2].peer.netManager.BackUpperChannel()
	if err != nil {
		fmt.Printf("获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		for _, nodeID := range upperNodes {
			fmt.Printf("更新前: upper层节点 -- %x\n", nodeID)
		}
	}

	// 2.更新testnet1的Leader节点，让其变为Follower. 接着再次让testnet2的Leader打印Upper PeerGroup的分布情况
	DpPers[0].peer.netManager.SendUpdateNodeState(dpnet.Follower)
	time.Sleep(3 * time.Second)
	upperGroup, err = DpPers[2].peer.netManager.BackUpperChannel()
	if err != nil {
		fmt.Printf("获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("更新后: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("更新后: upper层节点 -- %x\n", nodeID)
		}
	}

	// 3.更新testnet1的Leader节点，让其变为Leader. 接着再次让testnet2的Leader打印Upper PeerGroup的分布情况
	DpPers[1].peer.netManager.SendUpdateNodeState(dpnet.Leader)
	time.Sleep(3 * time.Second)

	upperGroup, err = DpPers[2].peer.netManager.BackUpperChannel()
	if err != nil {
		fmt.Printf("获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("更新后: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("更新后: upper层节点 -- %x\n", nodeID)
		}
	}

}

func TestLeaderChangeMsg(t *testing.T) {
	if err := initTest(); err != nil { //创造一组测试用Node节点

		t.Fatalf("fail in initTest: %v", err)
	}
	glog.SetToStderr(true)
	tempMockCfg := [6]mockConfig{
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[0].listenAddr,
			nat:            nat.Any(),
			isBooter:       true,
			netID:          "upperNet",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[1].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[2].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[3].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet2",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[4].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet2",
		},
	}
	tempMockCfg[1].bootstrapNodes = append(tempMockCfg[1].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[2].bootstrapNodes = append(tempMockCfg[2].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[3].bootstrapNodes = append(tempMockCfg[3].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[4].bootstrapNodes = append(tempMockCfg[4].bootstrapNodes, testNodes[0].discoverNode)

	go dpnetNode(0, &tempMockCfg[0], "upperNet", dpnet.Booter)
	go dpnetNode(1, &tempMockCfg[1], "testnet1", dpnet.Leader)
	go dpnetNode(2, &tempMockCfg[2], "testnet1", dpnet.Follower)
	go dpnetNode(3, &tempMockCfg[3], "testnet2", dpnet.Leader)
	go dpnetNode(4, &tempMockCfg[4], "testnet2", dpnet.Follower)

	for i := 0; i < 5; i++ {
		<-DpPers[i].peerchan
	}

	// 1.让子网testnet2的Leader节点打印upper层结构
	upperGroup, err := DpPers[3].peer.netManager.BackUpperChannel()
	if err != nil {
		fmt.Printf("[testnet2 Leader] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Leader] 更新前: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Leader] 更新前: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}
	// 1.让子网testnet2的Follower节点打印upper层结构
	upperGroup, err = DpPers[4].peer.netManager.BackUpperChannel()
	if err != nil {
		fmt.Printf("[testnet2 Follower] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Follower] 更新前: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Follower] 更新前: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}

	// 2. 让testnet1的follower节点发送LeaderChange Msg
	sigs := make([][]byte, 0)
	// 2.1 使用当前新Leader的NodeID作为Hash
	serialized, err := rlp.EncodeToBytes(DpPers[2].peer.netManager.SelfNode.NodeID)
	if err != nil {
		fmt.Printf("[testnet1 Follower] 发送Leader Change Msg出错,err:=%v\n", err)
	}
	hash := crypto.Sha3Hash(serialized)

	// 2.2 模拟搜集组内其他节点对上述Hash的数字签名(必须用私钥进行签名),其中要包含自己对上述Hash的数字签名
	signature, err := crypto.Sign(hash.Bytes(), DpPers[2].peer.netManager.GetPrivateKey())
	if err != nil {
		fmt.Printf("[testnet1 Follower] 节点1 进行数字签名失败\n")
	} else {
		sigs = append(sigs, signature)
	}

	// 2.3 发送搜集消息的hash值和所有节点的数字签名
	DpPers[2].peer.netManager.SendUpdateLeaderChange(hash, sigs)
	time.Sleep(5 * time.Second)

	// 3. 让让子网testnet2的Leader/Follower节点再次打印upper层结构
	fmt.Println()
	upperGroup, err = DpPers[3].peer.netManager.BackUpperChannel()
	if err != nil {
		fmt.Printf("[testnet2 Leader] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Leader] 更新后: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Leader] 更新后: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}
	upperGroup, err = DpPers[4].peer.netManager.BackUpperChannel()
	if err != nil {
		fmt.Printf("[testnet2 Follower] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Follower] 更新后: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Follower] 更新后: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}

}
