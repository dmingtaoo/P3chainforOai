package blockSync

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/validator"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"strconv"
	"testing"
	"time"
)

var listenAddr = []string{
	"127.0.0.1:20130",
	"127.0.0.1:20131",
	"127.0.0.1:20132",
	"127.0.0.1:20133",
}
var testNodes []mockNode //存储所有测试用模拟节点
type mockNode struct {
	prvKey       *ecdsa.PrivateKey
	listenAddr   string
	discoverNode *discover.Node
}

type Dper struct {
	selfKey   keys.Key
	p2pServer *server.Server
	blockSync *SyncProtocal
	nodeID    common.NodeID
}

type DpPer struct {
	peer     Dper
	peerchan chan bool
}

type waitChan chan bool

var waitChans = [3]waitChan{
	0: make(chan bool),
	1: make(chan bool),
	2: make(chan bool),
}

var DpPers = [4]DpPer{
	{
		peerchan: make(chan bool),
	},
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

type mockConfig struct {
	bootstrapNodes []*discover.Node
	listenAddr     string
	nat            nat.Interface
	isBooter       bool
}

// 区块验证测试用
const (
	UPNETID  string = "UPNET"
	LOWNETID string = "LOWNET"
)

type testNode struct {
	prvKey *ecdsa.PrivateKey
	nodeID common.NodeID
	addr   common.Address
}
type subnet struct {
	netID string
	nodes []*testNode
}

var leaders *subnet
var subnets []*subnet

// 创建一组测试用网络节点(discover.Node)
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

// 在本地创建一个P3-Chain网络节点(开放whisper客户端、separator客户端、服务器三大模块)
// 然后为此节点在本地维护一个P3-Chain网络
func mockDper(index int, mc *mockConfig) (*Dper, error) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	selfKey := keys.NewKeyFromECDSA(prvKey)
	tempName := "p3Chain:test"
	chain, _ := eles.InitBlockChain_Memory()
	sync := New(chain, dpnet.Node{}, nil) //创建区块同步对象
	srv := &server.Server{                //创建本节点的服务器
		PrivateKey:     prvKey,
		Discovery:      true,
		BootstrapNodes: mc.bootstrapNodes,
		MaxPeers:       10,
		Name:           tempName,
		Protocols:      []server.Protocol{sync.Protocol()},
		ListenAddr:     mc.listenAddr,
		NAT:            mc.nat,
	}

	//启动服务器模块
	srv.Start()

	dper := &Dper{
		selfKey:   *selfKey,
		p2pServer: srv, //本地p2p网络的控制对象
		blockSync: sync,
		nodeID:    crypto.KeytoNodeID(&prvKey.PublicKey),
	}
	return dper, nil
}

func SyncNode(index int, tempMockCfg *mockConfig) (*Dper, error) {
	testDper, err := mockDper(index, tempMockCfg) //按照配置信息，在本地创建一个运行sync协议的网络节点
	if err != nil {
		return nil, errors.New("fail in mockDper: " + fmt.Sprint(err))
	}
	fmt.Printf("第%d个节点,其NodeID为: %x\n", index, testDper.nodeID)
	DpPers[index].peer = *testDper
	DpPers[index].peerchan <- true
	return testDper, nil
}

func TestSynchronization(t *testing.T) {
	if err := initTest(); err != nil { //创造一组测试用Node节点
		t.Fatalf("fail in initTest: %v", err)
	}

	glog.SetToStderr(true)
	tempMockCfg := [4]mockConfig{
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[0].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
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
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[3].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
		},
	}
	tempMockCfg[1].bootstrapNodes = append(tempMockCfg[1].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[2].bootstrapNodes = append(tempMockCfg[2].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[3].bootstrapNodes = append(tempMockCfg[3].bootstrapNodes, testNodes[0].discoverNode)

	go SyncNode(0, &tempMockCfg[0])
	go SyncNode(1, &tempMockCfg[1])
	go SyncNode(2, &tempMockCfg[2])
	go SyncNode(3, &tempMockCfg[3])

	for i := 0; i < 4; i++ { //等待其他4个节点完成部署
		<-DpPers[i].peerchan
	}
	time.Sleep(3 * time.Second)

	chain0 := DpPers[0].peer.blockSync.ChainManager
	chain1 := DpPers[1].peer.blockSync.ChainManager
	chain2 := DpPers[2].peer.blockSync.ChainManager

	//1.必须在执行Validate()函数之前,完成整个dpNet网络的构建(即需要提前设置好SyncProtocal对象的CommonValidator)
	vm := InitValidateManager()

	DpPers[0].peer.blockSync.SetValidateManager(vm)
	DpPers[1].peer.blockSync.SetValidateManager(vm)
	DpPers[2].peer.blockSync.SetValidateManager(vm)

	tblock := &eles.Block{
		Subnet:       []byte(subnets[0].netID),
		Leader:       subnets[0].nodes[0].addr,
		Version:      common.StringToHash("test version"),
		Nonce:        uint8(0),
		Transactions: make([]eles.Transaction, 0),
		SubnetVotes:  make([]eles.SubNetSignature, 0),
	}
	tblock.ComputeBlockID()
	fmt.Printf("第一个产生的区块的ID: %x\n", tblock.BlockID)
	tblock1 := &eles.Block{
		Subnet:       []byte(subnets[1].netID),
		Leader:       subnets[1].nodes[0].addr,
		Version:      common.StringToHash("test version"),
		Nonce:        uint8(1),
		Transactions: make([]eles.Transaction, 0),
		SubnetVotes:  make([]eles.SubNetSignature, 0),
	}
	tblock1.PrevBlock = tblock.BlockID
	tblock1.ComputeBlockID()
	fmt.Printf("第二个产生的区块的ID: %x\n", tblock1.BlockID)
	tblock2 := &eles.Block{
		Subnet:       []byte(subnets[0].netID),
		Leader:       subnets[0].nodes[0].addr,
		Version:      common.StringToHash("test version"),
		Nonce:        uint8(2),
		Transactions: make([]eles.Transaction, 0),
		SubnetVotes:  make([]eles.SubNetSignature, 0),
	}
	tblock2.PrevBlock = tblock1.BlockID
	tblock2.ComputeBlockID()
	fmt.Printf("第三个产生的区块的ID: %x\n", tblock2.BlockID)
	tblock3 := &eles.Block{
		Subnet:       []byte(subnets[1].netID),
		Leader:       subnets[1].nodes[0].addr,
		Version:      common.StringToHash("test version"),
		Nonce:        uint8(3),
		Transactions: make([]eles.Transaction, 0),
		SubnetVotes:  make([]eles.SubNetSignature, 0),
	}
	tblock3.PrevBlock = tblock2.BlockID
	tblock3.ComputeBlockID()
	fmt.Printf("第四个产生的区块的ID: %x\n", tblock3.BlockID)
	wb := PacketBlock(tblock, 0)
	wb1 := PacketBlock(tblock1, 1)
	wb2 := PacketBlock(tblock2, 2)
	//wb3 := PacketBlock(tblock3, 3)

	blocks := []*eles.WrapBlock{wb, wb1, wb2}
	signBlocks := generateSignHash(blocks)

	chain0.InsertChain(signBlocks[0]) //将产生的区块1加入到本地0号节点区块链上
	chain1.InsertChain(signBlocks[0]) //将产生的区块1加入到本地1号节点区块链上

	chain0.InsertChain(signBlocks[1]) //将产生的区块2加入到本地0号节点区块链上
	chain1.InsertChain(signBlocks[1]) //将产生的区块2加入到本地0号节点区块链上

	//chain0.InsertChain(wb2)

	//DpPers[0].peer.blockSync.BroadcastBlock(wb1) //向所有已完成sync协议handshake的节点广播此区块

	time.Sleep(3 * time.Second)
	fmt.Println()
	fmt.Println("同步前：")
	height, head, genesis := chain0.GetChainStatus()
	fmt.Printf("chain0 height: %v ,head hash: %x ,genesis hash: %x\n", height, head, genesis)

	height1, head1, genesis1 := chain1.GetChainStatus()
	fmt.Printf("chain1 height: %v ,head hash: %x ,genesis hash: %x\n", height1, head1, genesis1)

	height2, head2, genesis2 := chain2.GetChainStatus()
	fmt.Printf("chain2 height: %v ,head hash: %x ,genesis hash: %x\n", height2, head2, genesis2)

	//3.运行各节点的Start()协程,进行随时同步
	go DpPers[0].peer.blockSync.Start()
	go DpPers[1].peer.blockSync.Start()
	go DpPers[2].peer.blockSync.Start()
	time.Sleep(10 * time.Second)
	chain2.InsertChain(signBlocks[2])

	time.Sleep(10 * time.Second)
	fmt.Println()
	fmt.Println("同步后：")
	height, head, genesis = chain0.GetChainStatus()
	fmt.Printf("chain0 height: %v ,head hash: %x ,genesis hash: %x\n", height, head, genesis)

	height1, head1, genesis1 = chain1.GetChainStatus()
	fmt.Printf("chain1 height: %v ,head hash: %x ,genesis hash: %x\n", height1, head1, genesis1)

	height2, head2, genesis2 = chain2.GetChainStatus()
	fmt.Printf("chain2 height: %v ,head hash: %x ,genesis hash: %x\n", height2, head2, genesis2)

}

func PacketBlock(block *eles.Block, index uint64) *eles.WrapBlock {
	wb := &eles.WrapBlock{
		RawBlock:   block,
		OriginPeer: common.NodeID(DpPers[0].peer.p2pServer.Self().ID),
		Number:     index,
		ReceivedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	return wb
}

func InitValidateManager() *validator.ValidateManager {
	testinit()
	tCommonValidator := &validator.CommonValidator{
		SubnetPool:   map[string]validator.SubnetVoters{},
		LeaderVoters: map[common.Address]bool{},
	}
	for _, leader := range leaders.nodes {
		tCommonValidator.LeaderVoters[leader.addr] = true
	}
	for _, subnet := range subnets {
		voters := make(validator.SubnetVoters)
		for _, follower := range subnet.nodes {
			voters[follower.addr] = true
		}
		tCommonValidator.SubnetPool[subnet.netID] = voters
	}
	vm := new(validator.ValidateManager)
	vm.Update(tCommonValidator)
	vm.SetDefaultValidThreshold()
	return vm
}

func testinit() {
	leaders = new(subnet)
	leaders.netID = UPNETID
	subnets = make([]*subnet, 0)

	leader0 := generateTestNode(testNodes[0].prvKey) //自己生成
	leaders.nodes = append(leaders.nodes, leader0)
	newSN0 := new(subnet)
	newSN0.netID = LOWNETID + strconv.Itoa(0)
	newSN0.nodes = append(newSN0.nodes, leader0)
	follower0 := generateTestNode(testNodes[1].prvKey) //自己生成
	newSN0.nodes = append(newSN0.nodes, follower0)
	subnets = append(subnets, newSN0)

	leader1 := generateTestNode(testNodes[2].prvKey) //自己生成
	leaders.nodes = append(leaders.nodes, leader1)
	newSN1 := new(subnet)
	newSN1.netID = LOWNETID + strconv.Itoa(1)
	newSN1.nodes = append(newSN1.nodes, leader1)
	follower1 := generateTestNode(testNodes[3].prvKey) //自己生成
	newSN1.nodes = append(newSN1.nodes, follower1)
	subnets = append(subnets, newSN1)

}

// 根据私钥生成一个测试用节点对象testNode
func generateTestNode(prvKey *ecdsa.PrivateKey) *testNode {
	nodeID := crypto.KeytoNodeID(&prvKey.PublicKey)
	addr, err := crypto.NodeIDtoAddress(nodeID)
	if err != nil {
		panic(err)
	}
	temp := &testNode{
		prvKey: prvKey,
		nodeID: nodeID,
		addr:   addr,
	}
	return temp
}

func generateSignHash(tblock []*eles.WrapBlock) []*eles.WrapBlock {
	lowerSignatures := []eles.SubNetSignature{}

	var testBlocks []*eles.WrapBlock

	rep := eles.BlockReceipt{
		TxReceipt: make([]eles.TransactionReceipt, 0),
		WriteSet:  make([]eles.WriteEle, 0),
	}

	for i := 0; i < len(tblock); i++ { //遍历每一个传入的区块,为每一个区块设置签名(tblock[i].RawBlock.SubnetVotes和tblock[i].RawBlock.LeaderVotes)
		upperSignatures := []eles.LeaderSignature{}
		tblock[i].RawBlock.Receipt = rep

		blockID, err := tblock[i].RawBlock.Hash() //获取该区块ID
		tblock[i].RawBlock.BlockID = blockID
		if err != nil {
			panic(err)
		}
		receiptID, err := tblock[i].RawBlock.BackReceiptID() //获取该区块的收据ID
		if err != nil {
			panic(err)
		}

		//1.进行下层共识签名
		for k := 0; k < len(subnets); k++ { //以子网为单位对该区块进行签名
			for j := 0; j < len(subnets[k].nodes); j++ {
				sig, err := crypto.SignHash(blockID, subnets[k].nodes[j].prvKey) //该子网的所有节点都需要对该区块的blockID进行签名
				//fmt.Printf("Sender--------------sig:%x\n", sig)
				if err != nil {
					panic(err)
				}
				lowerSignatures = append(lowerSignatures, sig)
			}
			tblock[i].RawBlock.SubnetVotes = lowerSignatures //结束后,收集本子网所有节点的数字签名-->当前区块
		}

		//2.进行上层共识签名
		for j := 0; j < len(leaders.nodes); j++ {
			sig, err := crypto.SignHash(receiptID, leaders.nodes[j].prvKey) //每一个Leader节点对该区块的receiptID进行签名
			//fmt.Printf("Sender-----------------------receiptID:%x , sig :%x\n", receiptID, sig)
			if err != nil {
				panic(err)
			}
			upperSignatures = append(upperSignatures, sig)
		}
		tblock[i].RawBlock.LeaderVotes = upperSignatures //结束后,收集所有Leader节点的数字签名-->当前区块
		//fmt.Printf("当前区块:%x ,获取的Leader节点投票:%x\n", tblock[i].RawBlock.BlockID, tblock[i].RawBlock.LeaderVotes)

		testBlocks = append(testBlocks, tblock[i]) //完成上下车共识的区块添加到testBlocks集合中

	}
	return testBlocks
}
