package consensus

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/netconfig"
	"p3Chain/core/separator"
	"p3Chain/core/worldstate"
	"p3Chain/crypto"
	"p3Chain/rlp"
	"testing"
	"time"
)

// TestConsensusPromoter_RoleFollower test the functions of consensus promoter
// (using the common tpbft as the lower consensus) when its role is a subnet follower.
func TestConsensusPromoter_RoleFollower(t *testing.T) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("%v", err)
	}
	mockNode := dpnet.NewNode(crypto.KeytoNodeID(&prvKey.PublicKey), 3, TEST_NETID) //新建一个Node节点(Follower)
	mockSM := worldstate.MockStateManager{}
	mockIntraSM := worldstate.MockStateManager{}
	mockNetManager := netconfig.NetManager{}
	mockCP := NewConsensusPromoter(mockNode, prvKey, &mockSM, &mockIntraSM, &mockNetManager) //新建一个Promoter对象

	// Create mock lower channel with 5 peers and the total number of TESTNET is 6.
	// Assume the first peer in mock lower channel is the leader of the TESTNET.
	mockLC := newMockLowerChannel(5) //节点组内包含五个成员节点

	mockCP.SetLowerChannel(mockLC)              //节点组信息更新到Promoter对象-->不再是PeerGroup对象实现该接口，而是mockLC(即mockLowerChannel对象),因此要注意MsgSend和MsgBrodcast都已经被改变
	mockCP.SetUpperChannel(&mockUpperChannel{}) //upper层信息更新到Promoter对象
	mockCP.SetBlockCache(&mockBlockCache{})
	mockCP.Start() //Promoter对象开始运作

	ctpbft := NewCtpbft()  //创建新的Common-TPBFT协议对象
	ctpbft.Install(mockCP) //协议安装到Promoter对象

	mockTxs := make([]*eles.Transaction, 0)
	mockTxMsgs := make([]*separator.Message, 0)
	for _, pn := range mockLC.mockGroupPeers { //遍历组内所有成员节点
		tempTx := generateUselessTX()                                  //由当前节点产生新交易(依靠节点的私钥--这里的私钥每次都是随机的)
		mockTxs = append(mockTxs, &tempTx)                             //mockTxs为产生的交易集合
		mockTxMsgs = append(mockTxMsgs, WrapTxMsg(pn.nodeID, &tempTx)) //mockTxMsgs为separator消息集合,每条消息的发送者规定为pn.nodeID
	}
	mockSelfTx := generateUselessTX() // a transaction that is published by the mock node itself.
	mockCP.AddTransations([]*eles.Transaction{&mockSelfTx})
	mockTxs = append(mockTxs, &mockSelfTx)

	endflag := make(chan error) //错误结束通道

	go func() {
		mockLC.AddMsgs(mockTxMsgs) //将mockTxMsgs集合内的消息添加到组内separator消息池(全都是交易消息)

		time.Sleep(time.Second) // wait all the transactions are collected.

		// now craft a pre-prepare message to launch a new lower-level consensus round
		// 创建一条新的pre-prepare共识消息,来开启新一轮的底层共识
		mockTxOrder := make([]common.Hash, len(mockTxs))

		for i := 0; i < len(mockTxs); i++ {
			mockTxOrder[i] = mockTxs[i].TxID //获取所有产生的交易的ID,作为交易顺序集合
		}
		//创建一条新的Pre-Prepare共识消息(发送者为组内0号节点,我们假定组内0号节点为本组Leader)
		mockPrePrepareMsg := &PrePrepareMsg{
			head: CommonHead{
				Consensus: []byte(COMMON_TPBFT_NAME),
				TypeCode:  StatePrePrepare,
				Sender:    mockLC.mockGroupPeers[0].nodeID, // Assume the first peer in mock lower channel is the leader of the TESTNET.
			},
			Version: TEST_VERSION,
			Nonce:   0,
			TxOrder: mockTxOrder, //交易顺序序列
		}
		mockPrePrepareMsg.ComputeRoundID() //PrePrepare消息的哈希值作为本次交易round的ID值
		t.Logf("roundID: %x", mockPrePrepareMsg.ComputeRoundID())
		mockWrappedLowerConsensusMsg := mockPrePrepareMsg.Wrap()         //PrePrepare消息打包成WrappedLowerConsensusMessage统一格式消息
		payload, err := rlp.EncodeToBytes(&mockWrappedLowerConsensusMsg) //rlp编码,作为separator协议消息的PayLoad
		if err != nil {
			endflag <- err //如果出错,将错误原因写入通道中，然后结束
			return
		}
		//将共识消息打包到separator协议消息中
		mockPpmMsg := &separator.Message{
			MsgCode: 2, //lowerConsensusCode
			NetID:   TEST_NETID,
			From:    mockLC.mockGroupPeers[0].nodeID, //视为Leader节点
			PayLoad: payload,
		}
		mockPpmMsg.CalculateHash()

		//将pre-prepare共识消息打包成的separator消息加入到组内消息池
		mockLC.AddMsgs([]*separator.Message{mockPpmMsg}) // receive the pre-prepare message

		time.Sleep(time.Second) // wait to complete building a new tpbft round

		// imitate peers to send prepare message
		// 准备发送 prepare 阶段共识消息
		validOrder := make([]byte, len(mockPrePrepareMsg.TxOrder)) //默认情况下,交易认证序列所有bit位为1(认为pre-prepare消息中的交易都是有效的)
		for i := 0; i < len(validOrder); i++ {
			validOrder[i] = byte(1)
		}
		mockPrepareMsgs := make([]*separator.Message, 0)    //存储所有follower节点产生的Prepare共识消息(打包到separator协议消息中)
		for i := 0; i < len(mockLC.mockGroupPeers)-1; i++ { //除了组内0号节点之外的节点(也就是全部follower节点)负责生产prepare消息
			peer := mockLC.mockGroupPeers[i+1]
			//新建Prepare阶段消息
			mockpm := &PrepareMsg{
				head: CommonHead{
					Consensus: []byte(COMMON_TPBFT_NAME),
					RoundID:   mockPrePrepareMsg.ComputeRoundID(), //根据获取的Pre-Prepare阶段消息产生本次round的ID
					TypeCode:  StatePrepare,
					Sender:    peer.nodeID, //每个follower节点的NodeID
				},
				ValidOrder: validOrder, //交易有效认证序列(默认情况下为全1)
				Version:    TEST_VERSION,
			}

			wlcm := mockpm.Wrap() //prepare消息打包成WrappedLowerConsensusMessage统一格式

			t.Logf("wlcm Sender: %x", wlcm.Head.Sender)

			pmPayload, err := rlp.EncodeToBytes(&wlcm) //共识消息打包到separator协议消息的PayLoad
			if err != nil {
				endflag <- err //有错误的话会退出
				return
			}
			//根据prepare共识消息生成separator协议消息
			mockPmMsg := &separator.Message{
				MsgCode: 2, //lowerConsensusCode
				NetID:   TEST_NETID,
				From:    peer.nodeID,
				PayLoad: pmPayload,
			}
			mockPmMsg.CalculateHash()

			mockPrepareMsgs = append(mockPrepareMsgs, mockPmMsg)
		}
		mockLC.AddMsgs(mockPrepareMsgs) //将所有Prepare消息添加到组内消息池
		time.Sleep(time.Second)         // wait to collect all the prepare messages

		time.Sleep(time.Second) // wait to send commit messages
		endflag <- nil          //通知主协程，当前协程结束
	}()

	err = <-endflag
	if err != nil {
		t.Fatalf("%v", err)
	}

}

// TestConsensusPromoter_Leader test the functions of consensus promoter
// (using the common tpbft as the lower consensus) when its role is a subnet Leader.
func TestConsensusPromoter_RoleLeader(t *testing.T) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("%v", err)
	}
	mockNode := dpnet.NewNode(crypto.KeytoNodeID(&prvKey.PublicKey), 2, TEST_NETID)
	mockSM := worldstate.MockStateManager{}
	mockIntraSM := worldstate.MockStateManager{}
	mockNetManager := netconfig.NetManager{}
	mockCP := NewConsensusPromoter(mockNode, prvKey, &mockSM, &mockIntraSM, &mockNetManager)

	// Create mock lower channel with 5 peers and the total number of TESTNET is 6.
	mockLC := newMockLowerChannel(5)

	mockCP.SetLowerChannel(mockLC) //mockLC实现LowerChannel接口
	mockCP.SetUpperChannel(&mockUpperChannel{})
	mockBC := &mockBlockCache{}
	mockCP.SetBlockCache(mockBC)
	mockCP.Start() //启动Promoter对象

	ctpbft := NewCtpbft()
	ctpbft.Install(mockCP) //为Promoter对象按照Common-TPBFT协议

	mockTxs := make([]*eles.Transaction, 0) //存储随机生成的所有交易
	mockTxMsgs := make([]*separator.Message, 0)
	for _, pn := range mockLC.mockGroupPeers {
		tempTx := generateUselessTX()
		mockTxs = append(mockTxs, &tempTx)
		mockTxMsgs = append(mockTxMsgs, WrapTxMsg(pn.nodeID, &tempTx))
	}
	mockSelfTx := generateUselessTX() // a transaction that is published by the mock node itself.
	mockCP.AddTransations([]*eles.Transaction{&mockSelfTx})
	mockTxs = append(mockTxs, &mockSelfTx)

	endflag := make(chan error)

	go func() {
		mockLC.AddMsgs(mockTxMsgs)
		time.Sleep(time.Second) // wait all the transactions are collected.
		time.Sleep(time.Second) // wait to create a new round

		// now get the pre-prepare message that mock lower consensus has broadcast.
		if !mockLC.lastMsg.IsLowerConsensus() { //判断上一条广播的separator消息是否为共识消息
			endflag <- fmt.Errorf("last message is not lower consensus kind") //如果不是的话直接退出
			return
		}

		wlcmsg, err := DeserializeLowerConsensusMsg(mockLC.lastMsg.PayLoad) //解包获取的共识消息
		if err != nil {
			endflag <- err
			return
		}

		ppm, err := wlcmsg.UnWrap() //解包为pre-prepare阶段共识消息
		if err != nil {
			endflag <- err
			return
		}
		mockPpm, ok := ppm.(*PrePrepareMsg)
		if !ok {
			endflag <- fmt.Errorf("last message is not pre-prepare message")
		}

		// now imitate peers to send prepare message
		// 进入prepare共识阶段
		validOrder := make([]byte, len(mockPpm.TxOrder)) //根据pre-prepare共识消息中交易的数目产生交易有效认证序列(默认全为1)
		for i := 0; i < len(validOrder); i++ {
			validOrder[i] = byte(1)
		}
		mockPrepareMsgs := make([]*separator.Message, 0)
		for i := 0; i < len(mockLC.mockGroupPeers); i++ { //当前节点组内所有节点都要发送一条包含上述交易认证序列的prepare消息
			peer := mockLC.mockGroupPeers[i]
			mockpm := &PrepareMsg{
				head: CommonHead{
					Consensus: []byte(COMMON_TPBFT_NAME),
					RoundID:   mockPpm.ComputeRoundID(), //根据收到的pre-prepare共识消息设置本次共识round的ID
					TypeCode:  StatePrepare,
					Sender:    peer.nodeID,
				},
				ValidOrder: validOrder,
				Version:    TEST_VERSION,
			}

			wlcm := mockpm.Wrap() //prepare共识消息打包成WrappedLowerConsensusMessage统一格式

			t.Logf("wlcm Sender: %x", wlcm.Head.Sender)

			pmPayload, err := rlp.EncodeToBytes(&wlcm) //rlp编码作为separator协议消息的PayLoad
			if err != nil {
				endflag <- err
				return
			}

			mockPmMsg := &separator.Message{
				MsgCode: 2, //lowerConsensusCode类型
				NetID:   TEST_NETID,
				From:    peer.nodeID,
				PayLoad: pmPayload,
			}
			mockPmMsg.CalculateHash()

			mockPrepareMsgs = append(mockPrepareMsgs, mockPmMsg)
		}
		mockLC.AddMsgs(mockPrepareMsgs) //将所有产生的prepare共识消息添加到节点组的消息池中
		time.Sleep(time.Second)         // wait to collect all the prepare messages  等待prepare阶段节点收集足够的prepare消息

		// now imitate to send commit messages
		plainTxs := make([]eles.Transaction, 0) //存储既在本地交易集合中,也是pre-prepare消息中指定有效的Transaction交易(也就是只存储达成共识的交易)
		for i := 0; i < len(mockPpm.TxOrder); i++ {
			if validOrder[i] == byte(1) { //遍历交易顺序集合中所有交易，如果交易有效(对于bit位为1)
				for _, tx := range mockTxs { //遍历整个mockTxs交易集合(本地存储的)
					if tx.TxID == mockPpm.TxOrder[i] { //如果某一个本地记录交易与pre-prepare消息中有效交易TxID一致
						plainTxs = append(plainTxs, *tx) //将这个本地交易加入到plainTxs集合中
					}
				}
			}
		}
		//注意:只有子网组的Leader节点可以发送pre-prepare阶段共识消息
		leaderAddress, err := crypto.NodeIDtoAddress(mockPpm.head.Sender) //根据pre-prepare消息的发送者NodeID获取Leader节点Address
		if err != nil {
			endflag <- err
			return
		}
		//组建区块
		candidateBlock := &eles.Block{
			BlockID:      common.Hash{},
			Subnet:       []byte(TEST_NETID),
			Leader:       leaderAddress, //Leader节点地址
			Version:      mockPpm.Version,
			Nonce:        mockPpm.Nonce,
			Transactions: plainTxs, //存储经过前两个阶段验证获取的所有合法交易
			SubnetVotes:  make([]eles.SubNetSignature, 0),
			PrevBlock:    common.Hash{},
			CheckRoot:    common.Hash{},
			Receipt:      eles.NullReceipt,
			LeaderVotes:  make([]eles.LeaderSignature, 0),
		}

		blockID, err := candidateBlock.ComputeBlockID() //计算区块ID
		if err != nil {
			endflag <- err
			return
		}

		result := blockID[:]

		cmMsgs := make([]*separator.Message, 0)

		for i := 0; i < len(mockLC.mockGroupPeers); i++ { //遍历组内所有节点
			peer := mockLC.mockGroupPeers[i]
			signature, err := crypto.Sign(result, peer.prevKey) //获取所有节点对同一数据(blockID)的数字签名
			if err != nil {
				endflag <- err
				return
			}
			//组装commit消息(每个组内节点都需要发送给Leader节点)
			commitMsg := &CommitMsg{
				head: CommonHead{
					Consensus: mockPpm.head.Consensus,
					RoundID:   mockPpm.ComputeRoundID(), //依旧是根据pre-prepare消息获取的roundID
					TypeCode:  StateCommit,
					Sender:    peer.nodeID, //每个成员节点的NodeID
				},
				Result:    result,    //区块ID
				Signature: signature, //自身的数字签名(针对于区块ID)
			}

			wlcmCm := commitMsg.Wrap() //将commit共识消息打包成WrappedLowerConsensusMessage统一格式消息

			cmPayload, err := rlp.EncodeToBytes(&wlcmCm) //rlp编码作为separator协议消息的PayLoad
			if err != nil {
				endflag <- err
			}
			//打包到separator协议消息中
			mockCmMsg := &separator.Message{
				MsgCode: 2,
				NetID:   TEST_NETID,
				From:    peer.nodeID,
				PayLoad: cmPayload,
			}
			mockCmMsg.CalculateHash()
			cmMsgs = append(cmMsgs, mockCmMsg)
		}

		mockLC.AddMsgs(cmMsgs)  //将生成的所有commit阶段共识消息添加到组内separator协议消息池中
		time.Sleep(time.Second) // wait to collect all the commit messages  等待commit消息的发送与收集
		time.Sleep(time.Second) // wait to do submit block function   等待Leader提交区块

		if mockBC.lastBlock == nil { //检查区块是否已经提交
			endflag <- fmt.Errorf("no block is submitted.")
		}

		//区块正常提交,结束当前协程
		t.Logf("Block is submitted: BlockID: %x, Version: %x, Nonce: %d \n", mockBC.lastBlock.BlockID, mockBC.lastBlock.Version, mockBC.lastBlock.Nonce)

		endflag <- nil
	}()

	err = <-endflag
	if err != nil {
		t.Fatalf("%v", err)
	}

}
