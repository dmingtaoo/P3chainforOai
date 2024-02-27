package validator

import (
	"crypto/ecdsa"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/crypto"
	"strconv"
	"testing"
)

const (
	UPNETID  string = "UPNET"
	LOWNETID string = "LOWNET"
)

type testNode struct {
	prvKey *ecdsa.PrivateKey
	nodeID common.NodeID
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
	temp := &testNode{
		prvKey: prvKey,
		nodeID: nodeID,
		addr:   addr,
	}
	//fmt.Printf("node addr: %x\n", addr)
	return temp
}

type subnet struct {
	netID string
	nodes []*testNode
}

var leaders *subnet
var subnets []*subnet

func generateUselessTX() eles.Transaction {
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

func testinit() {
	leaders = new(subnet)
	leaders.netID = UPNETID
	subnets = make([]*subnet, 0)
	for i := 0; i < 4; i++ {
		leader := generateTestNode()
		leaders.nodes = append(leaders.nodes, leader)
		newSN := new(subnet)
		newSN.netID = LOWNETID + strconv.Itoa(i)
		newSN.nodes = append(newSN.nodes, leader)
		for j := 0; j < 3; j++ {
			follower := generateTestNode()
			newSN.nodes = append(newSN.nodes, follower)
		}
		subnets = append(subnets, newSN)
	}

}

var testBlocks []*eles.Block

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
			txs = append(txs, generateUselessTX())
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
		upperSignatures := []eles.LeaderSignature{}

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

		receiptID, err := tblock.BackReceiptID()
		if err != nil {
			panic(err)
		}
		for j := 0; j < len(leaders.nodes); j++ {
			sig, err := crypto.SignHash(receiptID, leaders.nodes[j].prvKey)
			if err != nil {
				panic(err)
			}
			upperSignatures = append(upperSignatures, sig)
		}
		tblock.LeaderVotes = upperSignatures

		testBlocks = append(testBlocks, tblock)
	}
}

func TestCommonValidator(t *testing.T) {
	testinit()
	tCommonValidator := &CommonValidator{
		SubnetPool:   map[string]SubnetVoters{},
		LeaderVoters: map[common.Address]bool{},
	}

	for _, leader := range leaders.nodes {
		tCommonValidator.LeaderVoters[leader.addr] = true
	}
	for _, subnet := range subnets {
		voters := make(SubnetVoters)
		for _, follower := range subnet.nodes {
			voters[follower.addr] = true
		}
		tCommonValidator.SubnetPool[subnet.netID] = voters
	}

	tCommonValidator.SetDefaultThreshold()

	generateBlocks()
	if len(testBlocks) == 0 {
		t.Fatalf("fail in generate blocks")
	}

	for i := 0; i < len(testBlocks); i++ {
		if !tCommonValidator.LowerValidate(testBlocks[i]) {
			t.Fatalf("fail in lower validate")
		}

		if !tCommonValidator.UpperValidate(testBlocks[i]) {
			t.Fatalf("fail in upper validate")
		}
	}

}
