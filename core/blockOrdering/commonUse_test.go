package blockOrdering

import (
	"crypto/ecdsa"
	"fmt"
	"p3Chain/coefficient"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/separator"
	"p3Chain/core/validator"
	"p3Chain/crypto"
	"p3Chain/rlp"
	"strconv"
	"sync"
	"time"
)

const (
	UPNETID  string = "UPPER"
	LOWNETID string = "LOWNET"
)

var (
	pbftThresholdFactor = 3
	voteThresholdRate   = float64(coefficient.ServiceProvider_VoteThresholdRate)
)

// for test
var (
	subnets_num       = 4
	member_per_subnet = 4
)

type testNode struct {
	self   *dpnet.Node
	prvKey *ecdsa.PrivateKey
	addr   common.Address
}

func generateTestNode() *testNode {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	nodeID := crypto.KeytoNodeID(&prvKey.PublicKey)
	addr, err := crypto.NodeIDtoAddress(nodeID)
	if err != nil {
		panic(err)
	}
	dpnode := &dpnet.Node{
		NetID:  "temp",
		Role:   dpnet.UnKnown,
		NodeID: nodeID,
	}
	temp := &testNode{
		self:   dpnode,
		prvKey: prvKey,
		addr:   addr,
	}
	return temp
}

type subnet struct {
	netID string
	nodes []*testNode
}

var leaders *subnet
var subnets []*subnet

var testBlocks []*eles.Block

var globalBooter *mockPeerNode

var (
	mockUC         *mockUpperChannel
	mockVM         *validator.ValidateManager
	mockUpperNodes []*mockPeerNode // the booter is at the tail
)

func generateRandomTX() eles.Transaction {
	senderPrvKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	//创建一条新的交易对象
	tx := eles.Transaction{
		Sender:   crypto.PubkeyToAddress(senderPrvKey.PublicKey),
		Nonce:    0,
		Version:  common.StringToHash("test version"),
		LifeTime: 0,

		Contract:  common.StringToAddress("test contract"),
		Function:  common.StringToAddress("test function"),
		Args:      make([][]byte, 0),
		CheckList: make([]eles.CheckElement, 0),
	}
	tx.SetTxID()
	signature, err := crypto.SignHash(tx.TxID, senderPrvKey)
	if err != nil {
		panic(err)
	}
	tx.Signature = signature
	return tx
}

func generateBlocks() {
	testBlocks = make([]*eles.Block, 0)
	for i := 0; i < len(subnets); i++ {
		number := 5
		txs := make([]eles.Transaction, 0)
		rep := eles.BlockReceipt{
			TxReceipt: make([]eles.TransactionReceipt, number),
			WriteSet:  make([]eles.WriteEle, 0),
		}
		for j := 0; j < number; j++ {
			txs = append(txs, generateRandomTX())
		}

		tblock := &eles.Block{

			Subnet:       []byte(subnets[i].netID),
			Leader:       subnets[i].nodes[0].addr,
			Version:      common.StringToHash("test version"),
			Nonce:        uint8(0),
			Transactions: txs,
			SubnetVotes:  make([]eles.SubNetSignature, 0),
			PrevBlock:    common.StringToHash("previous"),
			Receipt:      rep,
			LeaderVotes:  make([]eles.LeaderSignature, 0),
		}

		lowerSignatures := []eles.SubNetSignature{}

		blockID, err := tblock.ComputeBlockID()
		if err != nil {
			panic(err)
		}

		for j := 0; j < len(subnets[i].nodes); j++ {
			sig, err := crypto.SignHash(blockID, subnets[i].nodes[j].prvKey)
			if err != nil {
				panic(err)
			}
			lowerSignatures = append(lowerSignatures, sig)
		}
		tblock.SubnetVotes = lowerSignatures

		testBlocks = append(testBlocks, tblock)
	}
}

func mockEnvInit() {
	leaders = new(subnet)
	leaders.netID = UPNETID
	subnets = make([]*subnet, 0)
	for i := 0; i < subnets_num; i++ {
		leader := generateTestNode()
		netID := LOWNETID + strconv.Itoa(i)
		leader.self.NetID = netID
		leader.self.Role = dpnet.Leader

		leaders.nodes = append(leaders.nodes, leader)
		newSN := new(subnet)
		newSN.netID = netID
		newSN.nodes = append(newSN.nodes, leader)
		for j := 0; j < member_per_subnet-1; j++ {
			follower := generateTestNode()
			follower.self.NetID = netID
			follower.self.Role = dpnet.Follower
			newSN.nodes = append(newSN.nodes, follower)
		}
		subnets = append(subnets, newSN)
	}
	generateBlocks()

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
	validateManager := new(validator.ValidateManager)
	validateManager.Update(tCommonValidator)
	validateManager.SetLowerValidFactor(pbftThresholdFactor)
	validateManager.SetUpperValidRate(voteThresholdRate)

	mockVM = validateManager
	mockUC = NewMockUpperChannel()
	mockUpperNodes = make([]*mockPeerNode, 0)
	for i := 1; i < len(leaders.nodes); i++ {
		tempMockPeer := &mockPeerNode{
			prevKey: leaders.nodes[i].prvKey,
			nodeID:  leaders.nodes[i].self.NodeID,
		}
		mockUpperNodes = append(mockUpperNodes, tempMockPeer)
	}

	globalBooter = newMockPeerNode()
	mockUpperNodes = append(mockUpperNodes, globalBooter)

	mockUC.mockMembers = mockUpperNodes

}

// 模拟一个节点
type mockPeerNode struct {
	prevKey *ecdsa.PrivateKey //私钥
	nodeID  common.NodeID     //节点的NodeID
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

type mockUpperChannel struct {
	mockMembers []*mockPeerNode
	msgPoolMux  sync.Mutex
	messagePool map[common.Hash]*separator.Message
	lastMsg     *BlockOrderMessage
}

func NewMockUpperChannel() *mockUpperChannel {
	temp := &mockUpperChannel{
		mockMembers: make([]*mockPeerNode, 0),
		messagePool: make(map[common.Hash]*separator.Message),
	}
	return temp
}

func (m *mockUpperChannel) AddMsgs(waitAddMsgs []*separator.Message) {
	m.msgPoolMux.Lock()
	defer m.msgPoolMux.Unlock()

	for _, msg := range waitAddMsgs {
		m.messagePool[msg.Hash] = msg //msg的哈希为key , key 为消息对象本身
	}
}

func (m *mockUpperChannel) AddMsg(waitAddMsg *separator.Message) {
	m.msgPoolMux.Lock()
	defer m.msgPoolMux.Unlock()
	m.messagePool[waitAddMsg.Hash] = waitAddMsg //msg的哈希为key , key 为消息对象本身
}

func (m *mockUpperChannel) CurrentMsgs() []*separator.Message {
	m.msgPoolMux.Lock()
	defer m.msgPoolMux.Unlock()
	msgs := make([]*separator.Message, 0)
	for _, msg := range m.messagePool {
		msgs = append(msgs, msg)
	}
	return msgs
}

func (m *mockUpperChannel) MarkRetrievedMsgs(msgIDs []common.Hash) {
	m.msgPoolMux.Lock()
	defer m.msgPoolMux.Unlock()

	for _, msgID := range msgIDs {
		delete(m.messagePool, msgID)
	}
}

func WrapUpperConsensusMsg(nodeID common.NodeID, netID string, bom *BlockOrderMessage) (*separator.Message, error) {
	payload, err := rlp.EncodeToBytes(bom)
	if err != nil {
		return nil, err
	}
	msg := &separator.Message{
		MsgCode: 3, // upper consensus Code
		NetID:   netID,
		From:    nodeID,
		PayLoad: payload,
	}
	msg.CalculateHash()
	return msg, nil
}

func (m *mockUpperChannel) NewUpperConsensusMessage(selfNode common.NodeID, payload []byte) *separator.Message {
	msg := &separator.Message{
		MsgCode: 3,       //upperConsensus
		NetID:   UPNETID, //目标子网NetID
		From:    selfNode,
		PayLoad: payload, //共识消息主体(pre-prepare/prepare/commit类型消息)
	}
	return msg
}

func (m *mockUpperChannel) MsgBroadcast(msg *separator.Message) {
	fmt.Printf("MsgBroadcast is called, MsgCode: %v, NetID: %v, From: %x \n", msg.MsgCode, msg.NetID, msg.From)
	if msg.IsUpperConsensus() { //检查msg消息的类型是否为lowerConsensusCode
		bom, err := DeserializeBlockOrderMsg(msg.PayLoad) //从msg消息的payload中解包出WrappedLowerConsensusMessage统一格式共识消息
		if err != nil {
			panic(err)
		}
		fmt.Printf("Type Code: %v, sender: %x\n", bom.TypeCode, bom.Sender)
		m.lastMsg = bom //当前msg消息更新到lastMsg字段
	}
}

func (m *mockUpperChannel) MsgSend(msg *separator.Message, receiver common.NodeID) error {
	fmt.Printf("MsgSend is called, MsgCode: %v, NetID: %v, From: %x, To: %x \n", msg.MsgCode, msg.NetID, msg.From, receiver)
	if msg.IsUpperConsensus() { //检查msg消息的类型是否为lowerConsensusCode
		bom, err := DeserializeBlockOrderMsg(msg.PayLoad) //从msg消息的payload中解包出WrappedLowerConsensusMessage统一格式共识消息
		if err != nil {
			panic(err)
		}
		fmt.Printf("Type Code: %v, sender: %x, to: %x\n", bom.TypeCode, bom.Sender, receiver)
		m.lastMsg = bom
	}
	return nil
}

func (m *mockUpperChannel) BackMembers() []common.NodeID {
	mockPeers := make([]common.NodeID, len(m.mockMembers))
	for i := 0; i < len(m.mockMembers); i++ {
		mockPeers[i] = m.mockMembers[i].nodeID
	}
	return mockPeers
}

type mockContractEngine struct {
}

func (m *mockContractEngine) ExecuteTransactions(txs []eles.Transaction) ([]eles.TransactionReceipt, []eles.WriteEle) {
	receipts := make([]eles.TransactionReceipt, 0)
	writeEles := make([]eles.WriteEle, 0)
	for i := 0; i < len(txs); i++ {
		receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
	}
	return receipts, writeEles
}

func (m *mockContractEngine) CommitWriteSet(writeSet []eles.WriteEle) error {
	return nil
}

type mockSyncProto struct {
	isNormal bool
}

func NewMockSyncProto() *mockSyncProto {
	return &mockSyncProto{
		isNormal: false,
	}
}

func (m *mockSyncProto) UseNormalMode() {
	m.isNormal = true
	fmt.Printf("now use the normal mode\n")
}

func (m *mockSyncProto) UseLeaderMode() {
	m.isNormal = false
	fmt.Printf("now use the leader mode\n")
}

func (m *mockSyncProto) IsInNormalMode() bool {
	return m.isNormal
}

func sleepSeconds(x int) {
	total := time.Duration(x) * time.Second
	time.Sleep(total)
}

// randomly generate some node id
func generateNodeID() common.NodeID {
	randomPrvKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	pubkey := randomPrvKey.PublicKey
	nodeID := crypto.KeytoNodeID(&pubkey)
	return nodeID
}
