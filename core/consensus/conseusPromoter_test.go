package consensus

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"p3Chain/common"
	"p3Chain/core/centre"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/netconfig"
	"p3Chain/core/separator"
	"p3Chain/core/worldstate"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
	"p3Chain/rlp"
	"sync"
	"testing"
	"time"
)

var (
	TEST_VERSION  = common.StringToHash("TEST VERSION")
	TEST_CONTRACT = common.StringToAddress("TEST CONTRACT")
	TEST_FUNCTION = common.StringToAddress("TEST FUNCTION")
	TEST_NETID    = "TESTNET"
)

var (
	groupCount    int = 2
	groupLeader   int = 2
	groupFollower int = 2
)

// 缓存区，保存上一个区块
type mockBlockCache struct {
	lastBlock *eles.Block
}

func (mbc *mockBlockCache) MarkLocalBlock(blockID common.Hash) {
	//TODO implement me
	panic("implement me")
}

func (mbc *mockBlockCache) BackUnmarkedLocalBlock() []*eles.Block {
	//TODO implement me
	panic("implement me")
}

func (mbc *mockBlockCache) Start() {
	//TODO implement me
	panic("implement me")
}

func (mbc *mockBlockCache) Stop() {
	//TODO implement me
	panic("implement me")
}

func (mbc *mockBlockCache) AddRemoteBlock(block *eles.Block) {
	panic("To Implement")
}

func (mbc *mockBlockCache) RemoveBlock(blockID common.Hash) {
	panic("To Implement")
}

func (mbc *mockBlockCache) IsExist(blockID common.Hash) bool {
	panic("To Implement")
}

// 模拟区块已经提交,因此当前区块成为上一区块(lastBlock)
func (mbc *mockBlockCache) AddLocalBlock(block *eles.Block) {
	fmt.Println("AddBlock is called")
	mbc.lastBlock = block
}

func (mbc *mockBlockCache) RetrieveBlock(blockID common.Hash) (*eles.Block, error) {
	panic("To Implement")
}

func (mbc *mockBlockCache) BackLocalBlocks() []*eles.Block {
	panic("To Implement")
}

// 模拟一个节点
type mockPeerNode struct {
	prevKey *ecdsa.PrivateKey //私钥
	nodeID  common.NodeID     //节点的NodeID
}

// 模拟一个节点组(mockLowerChannel实现了LowerChannel接口,在本测试中代替PeerGroup类)
type mockLowerChannel struct {
	mockGroupPeers []*mockPeerNode //节点组的所有成员节点
	msgPoolMux     sync.Mutex
	messagePool    map[common.Hash]*separator.Message //消息池(依靠separator协议)
	lastMsg        *separator.Message                 //上一条Msg
}

// 向节点组消息池中添加需要处理的msg消息
func (mlc *mockLowerChannel) AddMsgs(waitAddMsgs []*separator.Message) {
	mlc.msgPoolMux.Lock()
	defer mlc.msgPoolMux.Unlock()

	for _, msg := range waitAddMsgs {
		mlc.messagePool[msg.Hash] = msg //msg的哈希为key , key 为消息对象本身
	}
}

// 获取当前节点组消息池中的所有消息
func (mlc *mockLowerChannel) CurrentMsgs() []*separator.Message {
	mlc.msgPoolMux.Lock()
	defer mlc.msgPoolMux.Unlock()
	msgs := make([]*separator.Message, 0)
	for _, msg := range mlc.messagePool {
		msgs = append(msgs, msg)
	}
	return msgs
}

// 根据形参提供的key(msgID)集合，在消息池中找出对应的消息进行删除(视为已读)
func (mlc *mockLowerChannel) MarkRetrievedMsgs(msgIDs []common.Hash) {
	mlc.msgPoolMux.Lock()
	defer mlc.msgPoolMux.Unlock()

	for _, msgID := range msgIDs {
		delete(mlc.messagePool, msgID)
	}
}

// 创建一条新的共识消息(依托separator协议进行创建)
func (mlc *mockLowerChannel) NewLowerConsensusMessage(selfNode common.NodeID, payload []byte) *separator.Message {
	msg := &separator.Message{
		MsgCode: 2,          //消息类型为lowerConsensusCode
		NetID:   TEST_NETID, //目标子网NetID
		From:    selfNode,
		PayLoad: payload, //共识消息主体(pre-prepare/prepare/commit类型消息)
	}
	return msg
}

func (mlc *mockLowerChannel) NewIntraWorldStateUpdateMessage(selfNode common.NodeID, payload []byte) *separator.Message {
	msg := &separator.Message{
		MsgCode: 7,          //消息类型为intraWSUpdateCode
		NetID:   TEST_NETID, //目标子网NetID
		From:    selfNode,
		PayLoad: payload, //片内世界状态更新版本
	}
	return msg
}

func (mlc *mockLowerChannel) NewControlMessage(selfNode common.NodeID, payload []byte) *separator.Message {
	msg := &separator.Message{
		MsgCode: 4,          //消息类型为lowerConsensusCode
		NetID:   TEST_NETID, //目标子网NetID
		From:    selfNode,
		PayLoad: payload, //共识消息主体(pre-prepare/prepare/commit类型消息)
	}
	return msg
}

// 对形参传入的separator协议消息进行解包，解包为WrappedLowerConsensusMessage共识消息,最后从共识消息中获取相应信息
func (mlc *mockLowerChannel) MsgBroadcast(msg *separator.Message) {
	mlc.lastMsg = msg //当前msg消息更新到lastMsg字段
	fmt.Printf("MsgBroadcast is called, MsgCode: %v, NetID: %v, From: %x \n", msg.MsgCode, msg.NetID, msg.From)
	if msg.IsLowerConsensus() { //检查msg消息的类型是否为lowerConsensusCode
		wlcm, err := DeserializeLowerConsensusMsg(msg.PayLoad) //从msg消息的payload中解包出WrappedLowerConsensusMessage统一格式共识消息
		if err != nil {
			panic(err)
		}
		fmt.Printf("Consensus: %s, RoundID: %x, TypeCode: %v, Sender: %x \n", wlcm.Head.Consensus, wlcm.Head.RoundID, wlcm.Head.TypeCode, wlcm.Head.Sender)
	}
}

// 功能同上
func (mlc *mockLowerChannel) MsgSend(msg *separator.Message, receiver common.NodeID) error {
	mlc.lastMsg = msg
	fmt.Printf("MsgSend is called, MsgCode: %v, NetID: %v, From: %x, To: %x \n", msg.MsgCode, msg.NetID, msg.From, receiver)
	if msg.IsLowerConsensus() {
		wlcm, err := DeserializeLowerConsensusMsg(msg.PayLoad)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Consensus: %s, RoundID: %x, TypeCode: %v, Sender: %x \n", wlcm.Head.Consensus, wlcm.Head.RoundID, wlcm.Head.TypeCode, wlcm.Head.Sender)
	}
	return nil
}

// 返回当前节点组内保存的所有成员节点的NodeID
func (mlc *mockLowerChannel) BackMembers() []common.NodeID {
	mockPeers := make([]common.NodeID, len(mlc.mockGroupPeers))
	for i := 0; i < len(mlc.mockGroupPeers); i++ {
		mockPeers[i] = mlc.mockGroupPeers[i].nodeID
	}
	return mockPeers
}

// create a mock lower channel with the given peer number
// 创建一个新的MockLowerChannel对象(根据组内节点数目peerNum)
func newMockLowerChannel(peerNum int) *mockLowerChannel {
	mlc := &mockLowerChannel{
		mockGroupPeers: make([]*mockPeerNode, peerNum),           //PeerGroup
		messagePool:    make(map[common.Hash]*separator.Message), //消息池
	}
	for i := 0; i < peerNum; i++ {
		mlc.mockGroupPeers[i] = newMockPeerNode()
	}
	return mlc
}

// 创建一个新的mockPeerNode对象(私钥和NodeID)
func newMockPeerNode() *mockPeerNode {
	prv, err := crypto.GenerateKey() //私钥
	if err != nil {
		panic(err)
	}
	mpn := &mockPeerNode{
		prevKey: prv,
		nodeID:  crypto.KeytoNodeID(&prv.PublicKey), //从公钥获取NodeID
	}
	return mpn
}

// 创建一条新的交易(Transaction对象)
func generateUselessTX() eles.Transaction {
	senderPrvKey, err := crypto.GenerateKey() //交易发起者使用的私钥
	if err != nil {
		panic(err)
	}
	//创建一条新的交易对象
	tx := eles.Transaction{
		Sender:   crypto.PubkeyToAddress(senderPrvKey.PublicKey), //根据公钥生成的Address
		Nonce:    0,
		Version:  TEST_VERSION,
		LifeTime: 0,

		Contract:  TEST_CONTRACT,
		Function:  TEST_FUNCTION,
		Args:      make([][]byte, 0),
		CheckList: make([]eles.CheckElement, 0),
	}
	tx.SetTxID()                                             //设置交易的交易ID
	signature, err := crypto.SignHash(tx.TxID, senderPrvKey) //获取交易发起者的数字签名
	if err != nil {
		panic(err)
	}
	tx.Signature = signature //签名更新到交易对象中
	return tx
}

// 将交易封包成separator协议消息
func WrapTxMsg(from common.NodeID, tx *eles.Transaction) *separator.Message {
	payload, err := rlp.EncodeToBytes(tx) //编码交易为msg消息的payload
	if err != nil {
		panic(err)
	}
	msgcode := separator.TransactionCode
	msg := &separator.Message{
		MsgCode: uint64(msgcode), //消息类型为transactionCode
		NetID:   "testnet1",      //目标子网
		From:    from,            //来源,也就是当前节点的NodeID
		PayLoad: payload,         //交易
	}
	msg.CalculateHash()
	return msg
}

var listenAddr = []string{
	"127.0.0.1:20130",
	"127.0.0.1:20131",
	"127.0.0.1:20132",
	"127.0.0.1:20133",
	"127.0.0.1:20134",
	"127.0.0.1:20135",
}
var testNodes []mockNode //存储所有测试用模拟节点
type mockNode struct {
	prvKey       *ecdsa.PrivateKey
	listenAddr   string
	discoverNode *discover.Node
}

type Dper struct {
	selfKey    keys.Key
	netManager *netconfig.NetManager //本地维护一个P3-Chain网络
	p2pServer  *server.Server
}
type DpPer struct {
	peer     Dper
	peerchan chan bool
}
type mockConfig struct {
	bootstrapNodes []*discover.Node
	listenAddr     string
	nat            nat.Interface
	isBooter       bool
	netID          string
}

var DpPers = [6]DpPer{
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
	{
		peerchan: make(chan bool),
	},
	{
		peerchan: make(chan bool),
	},
}

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
func mockDper(mc *mockConfig) (*Dper, error) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	selfKey := keys.NewKeyFromECDSA(prvKey)
	var role int
	if mc.isBooter {
		role = int(dpnet.Booter)
	} else {
		role = int(dpnet.UnKnown)
	}

	var netID string
	if mc.netID == "" {
		netID = dpnet.InitialNetID
	} else {
		netID = mc.netID
	}

	selfNodeID := crypto.KeytoNodeID(&prvKey.PublicKey)
	selfNode := dpnet.NewNode(selfNodeID, role, netID) //根据NodeID/netID/role 创建一个P3-Chain 本地网络节点

	tempName := "p3Chain:test"
	shh := whisper.New()                                                        //创建本节点的whisper客户端
	spr := separator.New()                                                      //创建本节点的separator客户端
	cen := centre.CentralNew(*selfNode, groupCount, groupLeader, groupFollower) //创建本节点的central客户端

	srv := &server.Server{ //创建本节点的服务器
		PrivateKey:     prvKey,
		Discovery:      true,
		BootstrapNodes: mc.bootstrapNodes,
		MaxPeers:       10,
		Name:           tempName,
		Protocols:      []server.Protocol{shh.Protocol(), spr.Protocol()},
		ListenAddr:     mc.listenAddr,
		NAT:            mc.nat,
	}

	//启动服务器模块
	srv.Start()
	dpNet := dpnet.NewDpNet()
	netManager := netconfig.NewNetManager(prvKey, dpNet, srv, shh, spr, cen, nil, 0, selfNode) //为本地节点维护一个P3-Chain网络

	dper := &Dper{
		selfKey:    *selfKey,
		netManager: netManager, //本地P3-Chain网络的控制对象(或者说工具)
		p2pServer:  srv,        //本地p2p网络的控制对象
	}
	return dper, nil
}
func dpnetNode(index int, tempMockCfg *mockConfig, netID string, role dpnet.RoleType) (*Dper, error) {
	testDper, err := mockDper(tempMockCfg) //按照配置信息，在本地创建一个P3-Chain网络节点，并维护一个P3-Chain网络
	if err != nil {
		return nil, errors.New("fail in mockDper: " + fmt.Sprint(err))
	}

	testDper.netManager.SetSelfNodeNetID(netID) //设置本地节点的网络名(归属于哪一个网络)
	testDper.netManager.SetSelfNodeRole(role)   //将本地节点设置为本地P3-Chain网络的领导者

	if err := testDper.netManager.ViewNetAddSelf(); err != nil { //将本地节点按照上述两步配置加入到本地P3-Chain网络
		return nil, errors.New("fail in init view net: " + fmt.Sprint(err))
	}
	//打印netManager字段的selfNode字段中保存的dp-net网络中本地节点的NodeID 和 netID
	fmt.Println("dper netID: ", testDper.netManager.GetSelfNodeNetID(), " Role: ", testDper.netManager.GetSelfNodeRole())

	if err := testDper.netManager.StartConfigChannel(); err != nil { //为whisper客户端配置一个对接收消息的处理方法
		return nil, errors.New("fail in start config channel: " + fmt.Sprint(err))
	}
	time.Sleep(3 * time.Second)
	if err := testDper.netManager.SendInitSelfNodeState(); err != nil { //将自身的NodeID/NetID/Role等信息whisper广播发送给目标节点(用于构建separator网络)
		return nil, errors.New("fail in send self node state: " + fmt.Sprint(err))
	}
	time.Sleep(2 * time.Second)

	if err := testDper.netManager.ConfigLowerChannel(); err != nil { //将netManager字段中获取网络信息同步到spr字段中(只把当前自己所在的子网的部署情况进行同步)，最后赋给netManager的lowerChannel字段
		return nil, errors.New("fail in config lower channel: " + fmt.Sprint(err))
	}

	time.Sleep(2 * time.Second)
	if err := testDper.netManager.StartSeparateChannels(); err != nil { //让本地节点运行separator协议
		return nil, errors.New("fail in start separate channels: " + fmt.Sprint(err))
	}

	time.Sleep(5 * time.Second)

	DpPers[index].peer = *testDper
	DpPers[index].peerchan <- true
	return testDper, nil

}

// 根据配置信息，网络ID，节点身份 让本地节点加入到dp-net中
func (cp *ConsensusPromoter) addTransations(txs []*eles.Transaction) {
	cp.poolMu.RLock()
	defer cp.poolMu.RUnlock()
	for _, tx := range txs { //遍历形参指定的交易集合
		for _, txpool := range cp.transactionPools { //遍历本地所有的交易池
			if txpool.Match(tx) { //查询与当前交易tx匹配的交易池
				if err := txpool.Add(tx); err != nil { //将交易tx加入到与其匹配的交易池中
					glog.V(logger.Error).Infof("transaction pool: %v fail in add transaction: %v, %v", txpool.PoolID, tx.TxID, err)
				}
			}
		}
	}
}
func TestConsensusPromoter(t *testing.T) {
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
			netID:          "testnet1",
		},
		{ //设定配置信息
			bootstrapNodes: make([]*discover.Node, 0),
			listenAddr:     testNodes[3].listenAddr,
			nat:            nat.Any(),
			isBooter:       false,
			netID:          "testnet1",
		},
	}
	tempMockCfg[1].bootstrapNodes = append(tempMockCfg[1].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[2].bootstrapNodes = append(tempMockCfg[2].bootstrapNodes, testNodes[0].discoverNode)
	tempMockCfg[3].bootstrapNodes = append(tempMockCfg[3].bootstrapNodes, testNodes[0].discoverNode)
	go dpnetNode(0, &tempMockCfg[0], "testnet1", dpnet.Leader)
	go dpnetNode(1, &tempMockCfg[1], "testnet1", dpnet.Follower)
	go dpnetNode(2, &tempMockCfg[2], "testnet1", dpnet.Follower)
	go dpnetNode(3, &tempMockCfg[3], "testnet1", dpnet.Follower)

	for i := 0; i < 4; i++ {
		<-DpPers[i].peerchan
	}
	mockSM := worldstate.MockStateManager{}
	mockIntraSM := worldstate.MockStateManager{}
	mockNetManager := netconfig.NetManager{}
	mockCP := make([]*ConsensusPromoter, 4)
	for i := 0; i < 4; i++ {
		mockCP[i] = NewConsensusPromoter(DpPers[i].peer.netManager.SelfNode, DpPers[i].peer.selfKey.PrivateKey, &mockSM, &mockIntraSM, &mockNetManager)
		mockCP[i].SetLowerChannel(DpPers[i].peer.netManager.Spr.PeerGroups["testnet1"]) //mockLC实现LowerChannel接口
		mockBC := &mockBlockCache{}
		mockCP[i].SetBlockCache(mockBC)
		mockCP[i].Start() //启动Promoter对象
	}
	mockCtpbft := make([]*Ctpbft, 4)
	for i := 0; i < 4; i++ {
		mockCtpbft[i] = NewCtpbft()
		mockCtpbft[i].Install(mockCP[i])
	}
	mockTx := make([]*eles.Transaction, 0)
	tempTx := generateUselessTX()
	mockTx = append(mockTx, &tempTx)
	mockTxMsgs := WrapTxMsg(DpPers[0].peer.netManager.SelfNode.NodeID, &tempTx)

	//打印产生的交易的TxID和数字签名
	fmt.Println()
	fmt.Printf("生成交易：交易TxID --- : %x\n", tempTx.TxID)
	fmt.Printf("生成交易：交易发起者ID --- : %x\n", tempTx.Sender)
	fmt.Printf("生成交易：交易签名 --- : %x\n", tempTx.Signature)
	fmt.Println()

	//广播交易信息的节点需要将产生的交易自行加入自身交易池中
	mockCP[0].addTransations(mockTx)

	if testPg, err := DpPers[0].peer.netManager.Spr.BackPeerGroup(mockTxMsgs.NetID); err != nil { //返回子网ID对应的 peergroup
		t.Logf("fail in Back PeerGroup: %s", err)
	} else {
		testPg.MsgBroadcast(mockTxMsgs) //广播此交易(separator协议)
		time.Sleep(20 * time.Second)    //等待共识完成
	}

}
