package blockOrdering

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/consensus"
	"p3Chain/core/eles"
	"p3Chain/core/netconfig"
	"p3Chain/core/worldstate"
	"p3Chain/crypto"
	"testing"
)

func TestBlockOrderAgent_NormalOperation(t *testing.T) {
	mockEnvInit()
	mockSM := worldstate.MockStateManager{}
	mockIntraSM := worldstate.MockStateManager{}
	if err := mockSM.InitTestDataBase(); err != nil {
		t.Fatalf("fail in init database: %v", err)
	}
	mockCE := new(mockContractEngine)
	mockNetManager := netconfig.NetManager{}
	mockCP := consensus.NewConsensusPromoter(leaders.nodes[0].self, leaders.nodes[0].prvKey, &mockSM, &mockIntraSM, &mockNetManager)
	mockSyn := NewMockSyncProto()

	commonBlockCache := new(eles.CommonBlockCache)
	commonBlockCache.Start()

	//mockCP.SetLowerChannel()  //do not need in this test
	mockCP.SetBlockCache(commonBlockCache)
	mockCP.SetUpperChannel(mockUC)
	mockCP.SetContractEngine(mockCE)
	mockCP.SetValidateManager(mockVM)

	testBoa := NewBlockOrderAgent(globalBooter.nodeID, mockSyn)
	testBoa.Install(mockCP)

	if err := testBoa.Start(); err != nil {
		t.Fatalf("fail in start BlockOrderAgent: %v", err)
	}

	mockCP.Start() // now promoter is running

	commonBlockCache.AddLocalBlock(testBlocks[0]) // test the block upload function

	sleepSeconds(1)

	if mockUC.lastMsg.TypeCode != CODE_UPLD {
		t.Fatalf("fail in upload block!")
	}

	errflag := make(chan error)

	go func() {
		nowVersion := common.Hash{}
		nowHeight := uint64(0)
		// first test initialize
		iniMsg := &InitMessage{
			Version: nowVersion,
			Height:  nowHeight,
		}
		plainMsg1, _ := CreateBlockOrderMsg(iniMsg, globalBooter.nodeID, CODE_INIT)
		mockMsg1, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg1)
		mockUC.AddMsg(mockMsg1)

		sleepSeconds(1)
		if mockUC.lastMsg.TypeCode != CODE_STAT {
			errflag <- fmt.Errorf("fail in initialization!")
			return
		}

		rtvMsg1, _ := RetrievePayload(mockUC.lastMsg)
		staMsg := rtvMsg1.(*StateMessage)
		if staMsg.Height != nowHeight || staMsg.Version != nowVersion {
			errflag <- fmt.Errorf("unmatch state message is sent!")
			return
		}

		// test receive next round block
		nextBlock := testBlocks[1]
		nexMsg := &NextMessage{
			Version: nowVersion,
			Height:  nowHeight,
			Block:   *nextBlock,
		}
		plainMsg2, _ := CreateBlockOrderMsg(nexMsg, globalBooter.nodeID, CODE_NEXT)
		mockMsg2, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg2)
		mockUC.AddMsg(mockMsg2)

		sleepSeconds(1)
		if mockUC.lastMsg.TypeCode != CODE_SIGN {
			errflag <- fmt.Errorf("fail in signature!")
			return
		}

		rtvMsg2, _ := RetrievePayload(mockUC.lastMsg)
		sigMsg := rtvMsg2.(*SignMessage)
		fmt.Printf("got sign message, blockID: %x, receiptID: %x, signature: %x \n", sigMsg.BlockID, sigMsg.ReceiptID, sigMsg.Signature)

		// test commit
		blockReceiptID := sigMsg.ReceiptID
		upperSignatures := make([][]byte, 0)
		for i := 0; i < len(leaders.nodes); i++ {
			sig, errr := crypto.SignHash(blockReceiptID, leaders.nodes[i].prvKey)
			if errr != nil {
				errflag <- fmt.Errorf("fail in create mockCommitMsg: block is not valid")
				return
			}
			upperSignatures = append(upperSignatures, sig)
		}
		cmiMsg := &CommitMessage{
			Height:     nowHeight + 1,
			BlockID:    sigMsg.BlockID,
			ReceiptID:  blockReceiptID,
			Signatures: upperSignatures,
		}
		plainMsg3, _ := CreateBlockOrderMsg(cmiMsg, globalBooter.nodeID, CODE_CMIT)
		mockMsg3, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg3)
		mockUC.AddMsg(mockMsg3)

		sleepSeconds(1)
		if mockUC.lastMsg.TypeCode != CODE_DONE {
			errflag <- fmt.Errorf("fail in commit!")
			return
		}

		nowVersion, nowHeight = mockSM.GetCurrentState()
		fmt.Printf("now version is: %x, height is: %d \n", nowVersion, nowHeight)

		rtvMsg3, _ := RetrievePayload(mockUC.lastMsg)
		donMsg := rtvMsg3.(*DoneMessage)
		if nowVersion != donMsg.BlockID || nowHeight != donMsg.Height {
			errflag <- fmt.Errorf("unmatch version or height!")
			return
		}

		// test whether boa can enter the next round
		nextBlock = testBlocks[2]
		nexMsg = &NextMessage{
			Version: nowVersion,
			Height:  nowHeight,
			Block:   *nextBlock,
		}
		plainMsg4, _ := CreateBlockOrderMsg(nexMsg, globalBooter.nodeID, CODE_NEXT)
		mockMsg4, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg4)
		mockUC.AddMsg(mockMsg4)

		sleepSeconds(1)
		if mockUC.lastMsg.TypeCode != CODE_SIGN {
			errflag <- fmt.Errorf("fail in signature!")
			return
		}

		rtvMsg4, _ := RetrievePayload(mockUC.lastMsg)
		sigMsg = rtvMsg4.(*SignMessage)
		fmt.Printf("got sign message, blockID: %x, receiptID: %x, signature: %x \n", sigMsg.BlockID, sigMsg.ReceiptID, sigMsg.Signature)

		// test receive recovery message
		recoveryBlock := testBlocks[3]
		rcvMsg := &RecoveryMessage{
			RepeatCount: 1,
			Version:     nowVersion,
			Height:      nowHeight,
			Block:       *recoveryBlock,
		}
		plainMsg5, _ := CreateBlockOrderMsg(rcvMsg, globalBooter.nodeID, CODE_RCVY)
		mockMsg5, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg5)
		mockUC.AddMsg(mockMsg5)

		sleepSeconds(1)
		if mockUC.lastMsg.TypeCode != CODE_SIGN {
			errflag <- fmt.Errorf("fail in signature!")
			return
		}

		rtvMsg5, _ := RetrievePayload(mockUC.lastMsg)
		sigMsg = rtvMsg5.(*SignMessage)
		fmt.Printf("got sign message, blockID: %x, receiptID: %x, signature: %x \n", sigMsg.BlockID, sigMsg.ReceiptID, sigMsg.Signature)

		// test whether boa can finish the round after recovery
		blockReceiptID = sigMsg.ReceiptID
		upperSignatures = make([][]byte, 0)
		for i := 0; i < len(leaders.nodes); i++ {
			sig, errr := crypto.SignHash(blockReceiptID, leaders.nodes[i].prvKey)
			if errr != nil {
				errflag <- fmt.Errorf("fail in create mockCommitMsg: block is not valid")
				return
			}
			upperSignatures = append(upperSignatures, sig)
		}
		cmiMsg = &CommitMessage{
			Height:     nowHeight + 1,
			BlockID:    sigMsg.BlockID,
			ReceiptID:  blockReceiptID,
			Signatures: upperSignatures,
		}
		plainMsg6, _ := CreateBlockOrderMsg(cmiMsg, globalBooter.nodeID, CODE_CMIT)
		mockMsg6, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg6)
		mockUC.AddMsg(mockMsg6)

		sleepSeconds(1)
		if mockUC.lastMsg.TypeCode != CODE_DONE {
			errflag <- fmt.Errorf("fail in commit!")
			return
		}

		nowVersion, nowHeight = mockSM.GetCurrentState()
		fmt.Printf("now version is: %x, height is: %d \n", nowVersion, nowHeight)

		rtvMsg6, _ := RetrievePayload(mockUC.lastMsg)
		donMsg = rtvMsg6.(*DoneMessage)
		if nowVersion != donMsg.BlockID || nowHeight != donMsg.Height {
			errflag <- fmt.Errorf("unmatch version or height!")
			return
		}

		errflag <- nil
	}()
	select {
	case res := <-errflag:
		if res != nil {
			t.Fatalf("fail: %v", res)
		}
	}
}

func TestBlockOrderAgent_ProtectMode(t *testing.T) {
	mockEnvInit()
	mockSM := worldstate.MockStateManager{}
	mockIntraSM := worldstate.MockStateManager{}
	mockNetManager := netconfig.NetManager{}
	if err := mockSM.InitTestDataBase(); err != nil {
		t.Fatalf("fail in init database: %v", err)
	}
	mockCE := new(mockContractEngine)
	mockCP := consensus.NewConsensusPromoter(leaders.nodes[0].self, leaders.nodes[0].prvKey, &mockSM, &mockIntraSM, &mockNetManager)
	mockSyn := NewMockSyncProto()

	commonBlockCache := new(eles.CommonBlockCache)
	commonBlockCache.Start()

	//mockCP.SetLowerChannel()  //do not need in this test
	mockCP.SetBlockCache(commonBlockCache)
	mockCP.SetUpperChannel(mockUC)
	mockCP.SetContractEngine(mockCE)
	mockCP.SetValidateManager(mockVM)

	testBoa := NewBlockOrderAgent(globalBooter.nodeID, mockSyn)
	testBoa.Install(mockCP)

	if err := testBoa.Start(); err != nil {
		t.Fatalf("fail in start BlockOrderAgent: %v", err)
	}

	mockCP.Start() // now promoter is running

	errflag := make(chan error)

	go func() {
		nowVersion := testBlocks[0].BlockID
		nowHeight := uint64(6)
		// first test the sync message
		if mockSyn.IsInNormalMode() {
			errflag <- fmt.Errorf("should not be normal\n")
			return
		}
		fmt.Printf("now is in leader mode\n")
		synMsg := &SynchronizeMessage{
			Version: nowVersion,
			Height:  nowHeight,
		}

		plainMsg1, _ := CreateBlockOrderMsg(synMsg, globalBooter.nodeID, CODE_SYNC)
		mockMsg1, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg1)
		mockUC.AddMsg(mockMsg1)

		sleepSeconds(1)
		if !mockSyn.IsInNormalMode() {
			errflag <- fmt.Errorf("should be normal\n")
			return
		}
		fmt.Printf("after receive the recv message, protect mode is on\n")

		mockSM.ChangeState(nowVersion, nowHeight) // update the version and height

		// test whether boa can ge t out of the protect mode
		nextBlock := testBlocks[1]
		nexMsg := &NextMessage{
			Version: nowVersion,
			Height:  nowHeight,
			Block:   *nextBlock,
		}
		plainMsg2, _ := CreateBlockOrderMsg(nexMsg, globalBooter.nodeID, CODE_NEXT)
		mockMsg2, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg2)
		mockUC.AddMsg(mockMsg2)

		sleepSeconds(1)
		if mockSyn.IsInNormalMode() {
			errflag <- fmt.Errorf("should not be normal\n")
			return
		}
		fmt.Printf("After synchronizing, the protect mode ends\n")

		if mockUC.lastMsg.TypeCode != CODE_SIGN {
			errflag <- fmt.Errorf("fail in signature!")
			return
		}

		rtvMsg2, _ := RetrievePayload(mockUC.lastMsg)
		sigMsg := rtvMsg2.(*SignMessage)
		fmt.Printf("got sign message, blockID: %x, receiptID: %x, signature: %x \n", sigMsg.BlockID, sigMsg.ReceiptID, sigMsg.Signature)

		// test whether boa can get into protect mode if next message with higher height is got
		nowHeight += 1
		nowVersion = testBlocks[2].BlockID

		nextBlock = testBlocks[2]
		nexMsg = &NextMessage{
			Version: nowVersion,
			Height:  nowHeight,
			Block:   *nextBlock,
		}
		plainMsg3, _ := CreateBlockOrderMsg(nexMsg, globalBooter.nodeID, CODE_NEXT)
		mockMsg3, _ := WrapUpperConsensusMsg(globalBooter.nodeID, UPNETID, plainMsg3)
		mockUC.AddMsg(mockMsg3)

		sleepSeconds(1)
		if !mockSyn.IsInNormalMode() {
			errflag <- fmt.Errorf("should be normal\n")
			return
		}

		fmt.Printf("the boa now is in protect mode!\n")

		rtvMsg3, _ := RetrievePayload(mockUC.lastMsg)
		sigMsg = rtvMsg3.(*SignMessage)
		fmt.Printf("got next message, blockID: %x\n", nowVersion)
		fmt.Printf("last sign message, blockID: %x, receiptID: %x, signature: %x \n", sigMsg.BlockID, sigMsg.ReceiptID, sigMsg.Signature)

		errflag <- nil
	}()
	select {
	case res := <-errflag:
		if res != nil {
			t.Fatalf("fail: %v", res)
		}
	}
}
