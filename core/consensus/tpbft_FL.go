package consensus

import (
	"crypto/ecdsa"
	"fmt"
	"p3Chain/common"
	"p3Chain/core/contract"
	"p3Chain/core/eles"
	"p3Chain/core/utils"
	"p3Chain/crypto"
	"p3Chain/flDemoUse/FLAPI"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/rlp"
	"sort"
	"sync"
	"time"
)

const (
	FL_TPBFT_NAME = "fltpbft"

	flRoundLifeTime = 10 * time.Second

	MaxCollectTime = 5 * time.Second
)

type privilegeKeyReader struct {
}

func (pkr *privilegeKeyReader) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		if i%2 == 0 {
			p[i] = 'F'
		} else {
			p[i] = 'L'
		}
	}
	return len(p), nil
}

type FLtpbft struct {
	name              string //当前使用的TPBFT协议的名称
	consensusPromoter *ConsensusPromoter

	running bool // whether this tpbft instance is running  共识协议是否正在运行

	// The max number of transactions could be ordered in one consensus round.
	// The max number is not over 64. If over 64, the CheckPrepare func should
	// be changed.
	maxTransactionNum int //一次共识round中容许的最大交易数目(不超过64)

	// Store the live members that we know, as well as their reliability degrees.
	// The reliability degree of a member node denotes the importance weight of the
	// vote from this node. That is to say node with low reliability plays an
	// unimportant role during this lower-level consensus.
	//
	// TODO: Create some award and punish strategies through the reliability degree
	// in the future.
	// 存储当前节点已知的其他live节点以及他们的可信度(可信度是衡量一个成员在进行共识投票时权重的指标，可信度越低作用越小)
	// key值为对应live节点的NodeID，value值为可信度
	liveMember    map[common.NodeID]int
	liveMemberMux sync.RWMutex

	txPool *eles.TransactionPool //对应的交易池

	// help to make nonce
	lastVersion common.Hash
	nonce       uint8

	flManager          *FLAPI.FLManager
	privilegeKey       *ecdsa.PrivateKey
	modelCache         []*eles.Transaction
	modelFirstHoldTime time.Time
}

func NewFLtpbft() *FLtpbft {
	tempReader := &privilegeKeyReader{}
	tempKey, _ := crypto.GenerateSpecificKey(tempReader)
	fltpbft := FLtpbft{
		name:              FL_TPBFT_NAME,
		running:           false,
		maxTransactionNum: DEFAULT_MAX_TRANSACTIONS,
		liveMember:        make(map[common.NodeID]int),
		lastVersion:       common.StringToHash("PRIMARY VERSION"),
		nonce:             0,
		flManager:         FLAPI.NewFLManager(0),
		privilegeKey:      tempKey,
		modelCache:        make([]*eles.Transaction, 0),
	}
	return &fltpbft
}

func (ft *FLtpbft) Install(promoter *ConsensusPromoter) {
	ft.consensusPromoter = promoter

	ft.liveMemberMux.Lock()
	defer ft.liveMemberMux.Unlock()

	initialMembers := promoter.lowerChannel.BackMembers() //获取当前节点组内所有成员的NodeID
	for i := 0; i < len(initialMembers); i++ {            //每个节点初始的可信度为100
		ft.liveMember[initialMembers[i]] = 100
	}
	ft.liveMember[ft.consensusPromoter.SelfNode.NodeID] = 100 // as lowerChannel.BackMembers() does not include self nodeID

	ft.consensusPromoter.poolMu.Lock()
	defer ft.consensusPromoter.poolMu.Unlock()

	ft.consensusPromoter.tpbftMux.Lock()
	defer ft.consensusPromoter.tpbftMux.Unlock()
	txPool := eles.NewTransactionPool(ft.name, ft.TxMatch, ft.TxChoose) //创建一个新的交易池(ct.TxMatch和ct.TxChoose是创建交易池必须自行定义的两个函数)
	txPool.Start()
	ft.txPool = txPool

	ft.consensusPromoter.installedTpbft[ft.name] = ft                                             //以及绑定的consensusPromoter对象内部也要注册使用的TPFBT
	ft.consensusPromoter.transactionPools = append(ft.consensusPromoter.transactionPools, txPool) //将当前节点新创建的交易池加入到交易池集合中
}

func (ft *FLtpbft) TxMatch(tx *eles.Transaction) bool {
	if tx.Contract != contract.DEMO_CONTRACT_FL {
		return false
	}

	valid, err := crypto.SignatureValid(tx.Sender, tx.Signature, tx.TxID) //验证交易的数字签名是否正确(签名过程是用私钥对tx.TxID进行签名)
	if err != nil {
		glog.V(logger.Error).Infof("fail in signature valid: %v", err)
	}

	// signature not right
	if !valid {
		return false
	}

	return true
}

func (ft *FLtpbft) TxChoose(freshTxs []*eles.TransactionState) []*eles.Transaction {
	chosenTxs := make([]*eles.Transaction, 0, ft.maxTransactionNum)

	if len(freshTxs) <= ft.maxTransactionNum { //如果形参提供的交易集合中交易数目小于共识协议处理上限
		for _, txs := range freshTxs {
			chosenTxs = append(chosenTxs, txs.Tx) //那么不必进行筛选，直接将形参提供的所有交易作为返回结果
		}
		return chosenTxs
	}

	sort.Slice(freshTxs, func(i, j int) bool { //对整个freshTxs集合中的交易进行排序(按照交易的发布时间)
		lft_i := freshTxs[i].Tx.LifeTime
		lft_j := freshTxs[j].Tx.LifeTime

		if lft_i == 0 && lft_j > 0 {
			return false
		}
		if lft_i > 0 && lft_j == 0 {
			return true
		}
		if lft_i == lft_j {
			return freshTxs[i].StoredTime.Before(freshTxs[j].StoredTime)
		}
		return lft_i < lft_j
	})

	for i := 0; i < ft.maxTransactionNum; i++ { //提取出前maxTransactionNum条交易进行后续处理
		chosenTxs = append(chosenTxs, freshTxs[i].Tx)
	}
	return chosenTxs
}

func (ft *FLtpbft) Name() string {
	return ft.name
}

func (ft *FLtpbft) ViewChange() {
	//TODO implement me
	panic("implement me")
}

func (ft *FLtpbft) NewRound(leader common.NodeID, netID string) {
	chosenTxs := ft.txPool.ChooseTransactions() //在交易池中检索未超时的交易消息
	selectedTxs := make([]*eles.Transaction, 0)

	if len(ft.modelCache) > 0 && time.Now().After(ft.modelFirstHoldTime.Add(MaxCollectTime)) {
		selectedTxs = append(selectedTxs, ft.modelCache...)
		ft.modelCache = make([]*eles.Transaction, 0)
	}

	for i := 0; i < len(chosenTxs); i++ {
		if chosenTxs[i].Contract == contract.DEMO_CONTRACT_FL && chosenTxs[i].Function == contract.DEMO_CONTRACT_FL_FuncStoreModelAddr {
			if len(ft.modelCache) == 0 {
				ft.modelFirstHoldTime = time.Now()
			}
			ft.modelCache = append(ft.modelCache, chosenTxs[i])
		} else {
			selectedTxs = append(selectedTxs, chosenTxs[i])
		}
	}

	// a new round handler should be constructed
	ft.txPool.RemoveTransactions(chosenTxs)

	// no new transaction is collected
	if len(selectedTxs) == 0 { //没有新收集的交易消息，直接退出
		return
	}

	// over the max number of parallelled rounds
	if !ft.consensusPromoter.AcquireRoundToken() { //必须事先获取到Token才能进行新一轮共识(最多并行执行maxParallelRound轮共识)
		return
	}

	currentVersion := ft.consensusPromoter.StateManager.CurrentVersion() //获取当前区块版本
	nonce := ft.nonceCount(currentVersion)                               //根据区块版本计算nonce值
	ft.lastVersion = currentVersion

	//需要在节点组内广播发布新的pre-prepare共识消息
	ppm := &PrePrepareMsg{
		head: CommonHead{
			Consensus: []byte(FL_TPBFT_NAME),
			TypeCode:  StatePrePrepare,                      //Pre-Prepare共识阶段
			Sender:    ft.consensusPromoter.SelfNode.NodeID, //发送者的NodeID
		},
		Version: currentVersion,
		Nonce:   nonce,
		TxOrder: parseTXsOrder(selectedTxs), //获取chosenTxs中所有交易的交易ID.作为TxOrder
	}
	roudID := ppm.ComputeRoundID() //利用新建的pre-prepare共识消息计算哈希，得到本次共识round的ID
	ft.SendPrePrepareMsg(ppm)      //在组内广播此pre-prepare共识消息

	//完成pre-prepare阶段后，注册一个RoundHandler处理器，负责处理prepare阶段、tidy阶段和commit阶段
	rh := ft.parseRoundHandler(selectedTxs, roudID, ft.consensusPromoter.SelfNode.NodeID, currentVersion, nonce)

	//运行RoundHandler处理器
	go func() { rh.Start(); ft.consensusPromoter.AddRoundHandler(rh) }()
	ft.consensusPromoter.AddRoundHandler(rh)
}

func (ft *FLtpbft) HandleRound(ppm *PrePrepareMsg) (*roundHandler, error) {
	if len(ppm.TxOrder) > ft.maxTransactionNum { //TPBFT协议具有交易的处理上限,超过上限就会退出
		return nil, fmt.Errorf("consensus: %v, fail in HandleRound: over the max number of transactions in one round", ft.name)
	}

	chosenTxs, ok := ft.txPool.QuickFindTxs(ppm.TxOrder) //在交易池中快速查询筛选出pre-prepare消息中包含的交易
	if !ok {
		return nil, fmt.Errorf("not all transactions found in pool yet")
	}

	// chosenTxs 存储的是从交易池中获取的按照pre-prepare消息中规定的顺序(ppm.TxOrder)组装好的交易集合
	rh := ft.parseRoundHandler(chosenTxs, ppm.head.RoundID, ppm.head.Sender, ppm.Version, ppm.Nonce)
	ft.txPool.RemoveTransactions(chosenTxs) //从交易池中删除对应的交易(已经交由roundHandler处理这些消息)
	return rh, nil
}

func (ft *FLtpbft) SendPrePrepareMsg(ppm *PrePrepareMsg) {
	wlcm := ppm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		glog.V(logger.Error).Infof("fail in SendPrePrepareMsg: round ID %v, %v", ppm.head.RoundID, err)
	}
	msg := ft.consensusPromoter.lowerChannel.NewLowerConsensusMessage(ft.consensusPromoter.SelfNode.NodeID, payload)
	ft.consensusPromoter.lowerChannel.MsgBroadcast(msg)
}

func (ft *FLtpbft) SendPrepareMsg(pm *PrepareMsg) {
	wlcm := pm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		glog.V(logger.Error).Infof("fail in SendPrepareMsg: round ID %v, %v", pm.head.RoundID, err)
	}
	msg := ft.consensusPromoter.lowerChannel.NewLowerConsensusMessage(ft.consensusPromoter.SelfNode.NodeID, payload)
	ft.consensusPromoter.lowerChannel.MsgBroadcast(msg)
}

func (ft *FLtpbft) SendCommitMsg(cm *CommitMsg, nodeID common.NodeID) {
	if ft.consensusPromoter.SelfNode.NodeID == nodeID { // the node self is leader
		return
	}
	wlcm := cm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		glog.V(logger.Error).Infof("fail in SendCommitMsg: round ID %v, %v", cm.head.RoundID, err)
	}
	msg := ft.consensusPromoter.lowerChannel.NewLowerConsensusMessage(ft.consensusPromoter.SelfNode.NodeID, payload)
	err = ft.consensusPromoter.lowerChannel.MsgSend(msg, nodeID) //只能局限于组内定点发送
	if err != nil {
		glog.V(logger.Error).Infof("fail in SendCommitMsg: round ID %v, %v", cm.head.RoundID, err)
	}
}

func (ft *FLtpbft) ValidTransaction(txs []*eles.Transaction, version common.Hash) (validOrder []byte) {
	validOrder = make([]byte, len(txs))
	for i := 0; i < len(validOrder); i++ { //仅检查对应的交易消息是否已经过期
		if !ft.consensusPromoter.StateManager.StaleCheck(txs[i].Version, version, txs[i].LifeTime) {
			validOrder[i] = byte(1) //没过期的交易认为是有效的(valid)，对应bit位设置为1
		} else {
			validOrder[i] = byte(0) //过期的--> 无效 --> 对应bit位设为0
		}
	}
	return
}

func (ft *FLtpbft) CheckPrepare(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*PrepareMsg) (ok bool, validOrder []byte) {
	ft.liveMemberMux.RLock()
	defer ft.liveMemberMux.RUnlock()

	threshold := 2 * pbftFactor

	orderMap := make(map[uint64]int)

	for node, pm := range msgPool { //遍历所有prepare消息(node为消息的发送者NodeID) -- 这些prepare消息每一个都存储着ValidOrder,也就是对一组交易的验证序列
		if _, ok := liveMemberSnapShot[node]; !ok { //检查prepare消息是否确实来自于形参指定的节点集合
			continue
		}
		if trust, ok := ft.liveMember[node]; ok { //查找prepare消息的发送节点在本节点上的信任值
			if trust >= 0 { // Credit evaluation mechanism could be applied here  信任值需要大于0
				keyInt, err := utils.BoolByteSliceToUInt64(pm.ValidOrder) // 将bool切片(存储着交易验证序列)转换为对应的int64类型数值
				if err != nil {
					glog.V(logger.Error).Infof("essential fatal: valid order: %v", err)
					continue
				}
				orderMap[keyInt] += 1              //每当有一个可信节点对于交易序列投票数+1
				if orderMap[keyInt] >= threshold { //某一交易验证序列的投票数达到了TPBFT标准,则认为当前交易序列是合法有效的
					return true, pm.ValidOrder
				}
			} else { //一旦出现信任值小于0的节点，需要在日志中记录此节点以及其出现的交易roundID
				glog.V(logger.Warn).Infof("round ID: %v has found prepare message from unknown node: %v, system may be in danger!", pm.head.RoundID, node)
				continue
			}
		}
	}

	return false, []byte{}
}

func (ft *FLtpbft) backSnapShotLiveMembers() map[common.NodeID]bool {
	ft.liveMemberMux.RLock()
	defer ft.liveMemberMux.RUnlock()

	snapShot := make(map[common.NodeID]bool)
	for nodeID, _ := range ft.liveMember {
		snapShot[nodeID] = true
	}
	return snapShot
}

func (ft *FLtpbft) TidyTransactions(leader common.NodeID, version common.Hash, nonce uint8, txs []*eles.Transaction, validOrder []byte) (result []byte, signature []byte, block *eles.Block) {
	plainTxs := make([]eles.Transaction, 0) //存储所有有效的交易
	modelParas := make([][]byte, 0)
	for i := 0; i < len(txs); i++ { //遍历整个交易集合，只提取出有效的交易(validOrder中对应bit位==1的)
		if validOrder[i] == byte(1) {
			tempTx := txs[i]
			if tempTx.Contract == contract.DEMO_CONTRACT_FL && tempTx.Function == contract.DEMO_CONTRACT_FL_FuncStoreModelAddr {
				if len(tempTx.Args) != 1 {
					continue
				}
				modelParas = append(modelParas, tempTx.Args[0])
			} else {
				plainTxs = append(plainTxs, *tempTx)
			}
		}
	}

	processModelTxs := func() {
		if len(modelParas) > 0 {
			agg_model, err := ft.flManager.Aggregate(modelParas...)
			if err != nil {
				glog.V(logger.Error).Infof("fail in TidyTransaction: %v", err)
				return
			}
			senderAddr := crypto.PubkeyToAddress(ft.privilegeKey.PublicKey)
			privilegeTx := eles.Transaction{
				Sender:    senderAddr,
				Nonce:     0,
				Version:   version,
				LifeTime:  0,
				Contract:  contract.DEMO_CONTRACT_FL,
				Function:  contract.DEMO_CONTRACT_FL_FuncStoreModelAddr,
				Args:      [][]byte{agg_model},
				CheckList: make([]eles.CheckElement, 0),
			}
			privilegeTx.SetTxID()
			txSignature, err := crypto.SignHash(privilegeTx.TxID, ft.privilegeKey)
			if err != nil {
				glog.V(logger.Error).Infof("fail in TidyTransaction: %v", err)
				return
			}
			privilegeTx.Signature = txSignature
			plainTxs = append(plainTxs, privilegeTx)
		}
	}

	processModelTxs()

	leaderAddress, err := crypto.NodeIDtoAddress(leader) //根据NodeID计算Address
	if err != nil {
		glog.V(logger.Error).Infof("fail in TidyTransaction: %v", err)
	}

	//构建区块
	candidateBlock := &eles.Block{
		BlockID:      common.Hash{},                               //零值
		Subnet:       []byte(ft.consensusPromoter.SelfNode.NetID), //当前节点所在的子网ID
		Leader:       leaderAddress,                               //Leader节点的Address
		Version:      version,
		Nonce:        nonce,
		Transactions: plainTxs,                        //有效交易组成的集合
		SubnetVotes:  make([]eles.SubNetSignature, 0), //需要用于存储共识过程中投票参与者的数字签名
		PrevBlock:    common.Hash{},                   //前一个区块的哈希
		CheckRoot:    common.Hash{},                   //根哈希
		Receipt:      eles.NullReceipt,
		LeaderVotes:  make([]eles.LeaderSignature, 0), //当区块发布到区块链上时，需要记录共识过程中所有Leader节点的签名(upper层共识)
	}

	blockID, err := candidateBlock.ComputeBlockID() //用初始化的区块计算的Hash作为区块的ID
	if err != nil {
		glog.V(logger.Error).Infof("fail in TidyTransaction: %v", err)
	}
	result = blockID[:]
	signature, err = crypto.Sign(result, ft.consensusPromoter.prvKey) //生成数字签名(用区块ID产生的)
	if err != nil {
		glog.V(logger.Error).Infof("fail in TidyTransaction: %v", err)
	}
	return result, signature, candidateBlock
}

func (ft *FLtpbft) SubmitBlock(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*CommitMsg, block *eles.Block) bool {
	threshold := pbftFactor * 2
	if len(msgPool) < threshold { // 一个区块要提交，必须获取足够多的commit消息(相当于投票)
		return false
	}

	ft.liveMemberMux.RLock()
	defer ft.liveMemberMux.RUnlock()

	subNetVotes := make([]eles.SubNetSignature, 0)

	for node, cm := range msgPool { //遍历所有Commit共识消息(同时获取消息发送者ID)
		if _, ok := liveMemberSnapShot[node]; !ok { //在形参的节点集合中查询 当前消息发送者ID
			continue
		}
		if trust, ok := ft.liveMember[node]; ok { //获取此Commit共识消息发送节点的 信任值
			if trust >= 0 { // Credit evaluation mechanism could be applied here
				blockIdHash := common.BytesToHash(cm.Result) //从Commit共识消息中获取待发布区块的ID(common.BytesToHash只是用于截断，并非重新计算哈希)
				if blockIdHash != block.BlockID {            //获得的 blockIdHash 需要等于形参传入区块的ID值才能向下执行
					continue
				}
				addr, err := crypto.NodeIDtoAddress(node) //根据节点的NodeID计算获取Address
				if err != nil {
					glog.V(logger.Error).Infof("%v", err)
					continue
				}

				valid, err := crypto.SignatureValid(addr, cm.Signature, blockIdHash) //验证Commit共识消息的数字签名是否有效
				if err != nil {
					glog.V(logger.Error).Infof("%v", err)
					continue
				}
				if valid { //将有效的数字签名记录到subNetVotes集合
					subNetVotes = append(subNetVotes, cm.Signature)
				}

			} else {
				glog.V(logger.Warn).Infof("round ID: %v has found commit message from unknown node: %v, system may be in danger!", cm.head.RoundID, node)
				continue
			}
		}
	}

	if len(subNetVotes) < threshold { //必须要获取足够多节点的数字签名
		return false
	}

	block.SubnetVotes = subNetVotes

	glog.V(logger.Info).Infof("Block: %x is submitted, Version: %x, Nonce: %d", block.BlockID, block.Version, block.Nonce)
	ft.consensusPromoter.blockCache.AddLocalBlock(block) //提交该区块到upper层

	return true
}

func (ft *FLtpbft) Start() {
	ft.running = true
	glog.V(logger.Info).Infof("FL Kind PBFT starts, transaction pool: %v", ft.txPool.PoolID)
	go ft.update()
}

// TODO: view changes could be done here
func (ft *FLtpbft) update() {
}

func (ft *FLtpbft) nonceCount(currentVersion common.Hash) uint8 {
	if currentVersion == ft.lastVersion {
		ft.nonce += 1
	} else {
		ft.nonce = 0
	}
	return ft.nonce
}

// Parses a round handler. This func would do no validation.
func (ft *FLtpbft) parseRoundHandler(chosenTxs []*eles.Transaction, roundID common.Hash, leader common.NodeID, version common.Hash, nonce uint8) *roundHandler {
	liveMemberSnapShot := ft.backSnapShotLiveMembers() //获取组内所有live节点
	//初始化一个 roundHandler,处理一轮(round)底层共识 --> 完成pre-prepare阶段后就需要使用roundHandler进行后续共识阶段
	rh := &roundHandler{
		consensus:      []byte(ft.name), //共识协议Name
		roundID:        roundID,
		chosenTxs:      chosenTxs, //pre-prepare阶段共识消息完成排序
		leaderNode:     leader,
		selfNode:       ft.consensusPromoter.SelfNode.NodeID,
		version:        version,
		nonce:          nonce,
		state:          StatePrepare,    //roundHandler从StatePrepare阶段开始
		expiredTime:    flRoundLifeTime, //一轮交易round的上限时间
		prepareMsgPool: make(map[common.NodeID]*PrepareMsg),
		commitMsgPool:  make(map[common.NodeID]*CommitMsg),

		liveMemberSnapshot: liveMemberSnapShot,
		pbftFValue:         (len(liveMemberSnapShot) - 1) / pbftThresholdFactor,

		transactionValidFunc: ft.ValidTransaction, //用于验证交易合法性的方法
		prepareFunc:          ft.SendPrepareMsg,   //发送PrepareMsg的方法
		prepareCheck:         ft.CheckPrepare,     //验证prepare消息中交易合法性的方法(获取validOrder验证序列)
		tidyFunc:             ft.TidyTransactions, //根据有序交易集合构建区块的方法
		commitFunc:           ft.SendCommitMsg,    //发送CommitMsg的方法
		submitFunc:           ft.SubmitBlock,      //提交区块的方法

		selfDestruct: ft.consensusPromoter.RemoveRoundHandler,
	}
	return rh
}
