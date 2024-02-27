package consensus

import (
	"crypto/ecdsa"
	"fmt"
	"p3Chain/common"
	"p3Chain/core/separator"
	"p3Chain/crypto"
	"p3Chain/rlp"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestWrappedLowerConsensusMessage(t *testing.T) {
	ppm := &PrePrepareMsg{
		head: CommonHead{
			Consensus: []byte("test consensus"),
			TypeCode:  StatePrePrepare,
			Sender:    common.NodeID{},
		},
		Version: common.StringToHash("TestVersion"),
		Nonce:   0,
		TxOrder: []common.Hash{common.StringToHash("tx1"), common.StringToHash("tx2"), common.StringToHash("tx3")},
	}
	roundID := ppm.ComputeRoundID()
	if roundID != ppm.head.RoundID {
		t.Fatalf("fail in set roundID")
	}
	wlcm := ppm.Wrap()
	reconstructppm, err := wlcm.UnWrap()
	if err != nil {
		t.Fatalf("fail in unwrap: %v", err)
	}
	if !reflect.DeepEqual(reconstructppm, ppm) {
		t.Fatalf("PrePrepareMsg changed after deserialized")
	}

}

func TestHeartBeatMessage(t *testing.T) {
	heartBeat := &HeartBeat{
		Head: CommonHead{
			Consensus: []byte("test consensus"),
			TypeCode:  StateHeartBeat,
			Sender:    common.NodeID{}, //发送者的NodeID
		},
		TimeStampSecond: uint64(time.Now().Unix()),
	}
	wlcm := heartBeat.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		fmt.Printf("Heart Beate Msg Encode failed,err:%v\n", err)
		return
	}
	msg := &separator.Message{
		MsgCode: 0x04,
		NetID:   "testnet1",
		From:    common.NodeID{},
		PayLoad: payload,
	}

	if msg.IsControl() {
		newWlcm, err := DeserializeLowerConsensusMsg(msg.PayLoad) //解包为WrappedLowerConsensusMessage 统一格式消息
		if err != nil {
			fmt.Printf("反序列化失败,err:%v\n", err)
			return
		}

		unwrapMsg, err := newWlcm.UnWrap()
		if err != nil {
			fmt.Printf("解包失败,err:%v\n", err)
			return
		}
		hb, ok := unwrapMsg.(*HeartBeat) //类型断言为HeartBeat
		if !ok {
			fmt.Printf("类型断言失败,err:%v\n", err)
			return
		}
		fmt.Println(hb.TimeStampSecond)
	}
}

func TestViewChangeMsg(t *testing.T) {
	// 生成一条ViewChange Msg,检查能否能解析(重点是数字签名能否完成验证)

	// 1.生成4个私钥
	prvKeySet := make([]*ecdsa.PrivateKey, 0)
	for i := 0; i < 4; i++ {
		prv, _ := crypto.GenerateKey()
		prvKeySet = append(prvKeySet, prv)
	}
	// 2.生成4个公钥
	pubKeySet := make([]*ecdsa.PublicKey, 0)
	for i := 0; i < 4; i++ {
		pub := prvKeySet[i].Public().(*ecdsa.PublicKey)
		pubKeySet = append(pubKeySet, pub)
	}
	// 3.生成4个nodeID
	nodeIDSet := make([]common.NodeID, 0)
	for i := 0; i < 4; i++ {
		nodeID := crypto.KeytoNodeID(pubKeySet[i])
		nodeIDSet = append(nodeIDSet, nodeID)
	}

	vcm := ViewChangeMsg{
		Head: CommonHead{
			Consensus: []byte("test consensus"),
			TypeCode:  StateViewChange, //View-Change阶段
			Sender:    nodeIDSet[0],    //发送者的NodeID
		},
		Sender:          nodeIDSet[0],
		TimeStampSecond: uint64(time.Now().Unix()),
		NewLeader:       nodeIDSet[0],
	}
	vcm.Hash()
	vcm.SignatureVcm(prvKeySet[0])

	wlcm := vcm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		fmt.Printf("send view-change Msg failed,err:%v\n", err)
		return
	}
	msg := &separator.Message{
		MsgCode: 0x04,
		NetID:   "testnet1",
		From:    nodeIDSet[0],
		PayLoad: payload,
	}

	if msg.IsControl() {
		newWlcm, err := DeserializeLowerConsensusMsg(msg.PayLoad) //解包为WrappedLowerConsensusMessage 统一格式消息
		if err != nil {
			fmt.Printf("反序列化失败,err:%v\n", err)
			return
		}

		unwrapMsg, err := newWlcm.UnWrap()
		if err != nil {
			fmt.Printf("解包失败,err:%v\n", err)
			return
		}
		hb, ok := unwrapMsg.(*ViewChangeMsg) //类型断言为ViewChangeMsg
		if !ok {
			fmt.Printf("类型断言失败,err:%v\n", err)
			return
		}
		fmt.Println(hb.TimeStampSecond)
	}

}

func TestLeaderSelect(t *testing.T) {
	// 1.生成4个私钥
	prvKeySet := make([]*ecdsa.PrivateKey, 0)
	for i := 0; i < 4; i++ {
		prv, _ := crypto.GenerateKey()
		prvKeySet = append(prvKeySet, prv)
	}
	// 2.生成4个公钥
	pubKeySet := make([]*ecdsa.PublicKey, 0)
	for i := 0; i < 4; i++ {
		pub := prvKeySet[i].Public().(*ecdsa.PublicKey)
		pubKeySet = append(pubKeySet, pub)
	}
	// 3.生成4个nodeID
	nodeIDSet := make([]common.NodeID, 0)
	for i := 0; i < 4; i++ {
		nodeID := crypto.KeytoNodeID(pubKeySet[i])
		nodeIDSet = append(nodeIDSet, nodeID)
	}

	nodeIDMap := make(map[string]common.NodeID)
	nodeIDStrSet := make([]string, 0)
	for _, nodeID := range nodeIDSet {
		nodeIDMap[fmt.Sprintf("%x", nodeID)] = nodeID
		nodeIDStrSet = append(nodeIDStrSet, fmt.Sprintf("%x", nodeID))
	}

	sort.Strings(nodeIDStrSet)

	newLeader := nodeIDMap[nodeIDStrSet[0]]

	for _, nodeID := range nodeIDMap {
		fmt.Printf("Node:%x\n", nodeID)
	}
	fmt.Println()
	fmt.Printf("newLeader:%x\n", newLeader)
	fmt.Println()

	nodeIDSet = nodeIDSet[:2]
	nodeIDMap = make(map[string]common.NodeID)
	nodeIDStrSet = make([]string, 0)
	for _, nodeID := range nodeIDSet {
		nodeIDMap[fmt.Sprintf("%x", nodeID)] = nodeID
		nodeIDStrSet = append(nodeIDStrSet, fmt.Sprintf("%x", nodeID))
	}

	sort.Strings(nodeIDStrSet)

	newLeader = nodeIDMap[nodeIDStrSet[0]]

	for _, nodeID := range nodeIDMap {
		fmt.Printf("Node:%x\n", nodeID)
	}
	fmt.Println()
	fmt.Printf("newLeader:%x\n", newLeader)

}
