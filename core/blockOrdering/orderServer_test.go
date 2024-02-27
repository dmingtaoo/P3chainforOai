package blockOrdering

import (
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/netconfig"
	"p3Chain/core/separator"
	"p3Chain/crypto"
	"testing"
	"time"
)

func TestBlockOrderServer_CommonOperation(t *testing.T) {
	mockEnvInit()
	mockCE := new(mockContractEngine)
	mockNetManager := netconfig.NetManager{}
	booterNode := &dpnet.Node{
		NetID:  UPNETID,
		Role:   dpnet.Booter,
		NodeID: globalBooter.nodeID,
	}
	bos := NewBlockOrderServer(booterNode, common.Hash{}, 0, mockUC, &mockNetManager)
	bos.Start()

	// first test initialization
	sleepSeconds(1)
	if mockUC.lastMsg.TypeCode != CODE_INIT {
		t.Fatalf("fail in init!")
	}
	rvtMsg0, _ := RetrievePayload(mockUC.lastMsg)
	iniMsg, _ := rvtMsg0.(*InitMessage)
	t.Logf("got init message: version: %x, height: %d\n", iniMsg.Version, iniMsg.Height)

	nowVersion := testBlocks[0].BlockID
	nowHeight := uint64(5)

	mockMsgs1 := make([]*separator.Message, 0)
	for _, leader := range leaders.nodes {
		staMsg := &StateMessage{
			Version: nowVersion,
			Height:  nowHeight,
		}
		bom, _ := CreateBlockOrderMsg(staMsg, leader.self.NodeID, CODE_STAT)
		tempMsg, _ := WrapUpperConsensusMsg(leader.self.NodeID, UPNETID, bom)
		mockMsgs1 = append(mockMsgs1, tempMsg)
	}

	mockUC.AddMsgs(mockMsgs1)
	sleepSeconds(1)

	if bos.serviceMaintainer.currentHeight != nowHeight || bos.serviceMaintainer.currentVersion != nowVersion {
		t.Fatalf("fail in initilization with wrong version and height\n")
	}

	t.Logf("now current version: %x, current height: %d\n", bos.serviceMaintainer.currentVersion, bos.serviceMaintainer.currentHeight)
	if mockUC.lastMsg.TypeCode != CODE_SYNC {
		t.Fatalf("fail in init!")
	}
	rvtMsg1, _ := RetrievePayload(mockUC.lastMsg)
	synMsg, _ := rvtMsg1.(*SynchronizeMessage)
	t.Logf("got sync message: version: %x, height: %d\n", synMsg.Version, synMsg.Height)

	// test upload blocks
	mockMsgs2 := make([]*separator.Message, 0)
	for i := 0; i < subnets_num; i++ {
		leader := leaders.nodes[i]
		block := testBlocks[i]
		expireTime := time.Now().Add(block_expiration)

		uplMsg := &UploadMessage{
			ExpireTime: uint64(expireTime.Unix()),
			Block:      *block,
		}
		bom, _ := CreateBlockOrderMsg(uplMsg, leader.self.NodeID, CODE_UPLD)
		tempMsg, _ := WrapUpperConsensusMsg(leader.self.NodeID, UPNETID, bom)
		mockMsgs2 = append(mockMsgs2, tempMsg)
	}
	mockUC.AddMsgs(mockMsgs2)

	sleepSeconds(1)
	if mockUC.lastMsg.TypeCode != CODE_NEXT {
		t.Fatalf("fail in send next message!\n")
	}
	rvtMsg2, _ := RetrievePayload(mockUC.lastMsg)
	nexMsg, _ := rvtMsg2.(*NextMessage)
	t.Logf("got next message: version: %x, height: %d, next blockID: %x\n", nexMsg.Version, nexMsg.Height, nexMsg.Block.BlockID)

	// test	whether bos can commit
	mockMsgs3 := make([]*separator.Message, 0)

	blockChosen1 := nexMsg.Block
	txRcp, wrtSet := mockCE.ExecuteTransactions(blockChosen1.Transactions)
	receiptID, _ := blockChosen1.SetReceipt(nowVersion, txRcp, wrtSet)
	for _, leader := range leaders.nodes {
		signature, _ := crypto.SignHash(receiptID, leader.prvKey)
		sigMsg := &SignMessage{
			Valid:     BlockValid,
			BlockID:   blockChosen1.BlockID,
			ReceiptID: receiptID,
			Signature: signature,
		}
		bom, _ := CreateBlockOrderMsg(sigMsg, leader.self.NodeID, CODE_SIGN)
		tempMsg, _ := WrapUpperConsensusMsg(leader.self.NodeID, UPNETID, bom)
		mockMsgs3 = append(mockMsgs3, tempMsg)
	}

	mockUC.AddMsgs(mockMsgs3)

	sleepSeconds(1)
	if mockUC.lastMsg.TypeCode != CODE_CMIT {
		t.Fatalf("fail in send commit message!\n")
	}
	rvtMsg3, _ := RetrievePayload(mockUC.lastMsg)
	cmiMsg, _ := rvtMsg3.(*CommitMessage)
	t.Logf("got commit message: blockID: %x, height: %d, receiptID: %x\n", cmiMsg.BlockID, cmiMsg.Height, cmiMsg.ReceiptID)

	// test whether bos can step in next round
	mockMsgs4 := make([]*separator.Message, 0)

	for _, leader := range leaders.nodes {
		donMsg := &DoneMessage{
			Valid:   BlockValid,
			BlockID: blockChosen1.BlockID,
			Height:  nowHeight + 1,
		}
		bom, _ := CreateBlockOrderMsg(donMsg, leader.self.NodeID, CODE_DONE)
		tempMsg, _ := WrapUpperConsensusMsg(leader.self.NodeID, UPNETID, bom)
		mockMsgs4 = append(mockMsgs4, tempMsg)
	}

	mockUC.AddMsgs(mockMsgs4)

	sleepSeconds(1)
	if mockUC.lastMsg.TypeCode != CODE_NEXT { // this is because the bos has stepped into next round
		t.Fatalf("fail in send next message!\n")
	}
	rvtMsg4, _ := RetrievePayload(mockUC.lastMsg)
	nexMsg, _ = rvtMsg4.(*NextMessage)
	t.Logf("got next message: version: %x, height: %d, next blockID: %x\n", nexMsg.Version, nexMsg.Height, nexMsg.Block.BlockID)

	// test whether bos can launch recovery message if long time not collect enough signatures
	sleepSeconds(5)
	if mockUC.lastMsg.TypeCode != CODE_RCVY {
		t.Fatalf("fail in send recovery message!\n")
	}
	rvtMsg5, _ := RetrievePayload(mockUC.lastMsg)
	recMsg, _ := rvtMsg5.(*RecoveryMessage)
	t.Logf("got recovery message: repeat time: %d, version: %x, height: %d, next blockID: %x\n", recMsg.RepeatCount, recMsg.Version, recMsg.Height, recMsg.Block.BlockID)

}
