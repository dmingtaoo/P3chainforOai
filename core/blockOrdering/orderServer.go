package blockOrdering

import (
	"fmt"
	"math"
	"p3Chain/coefficient"
	"p3Chain/common"
	"p3Chain/core/consensus"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/netconfig"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"
	"reflect"
	"sort"
	"sync"
	"time"
)

const (
	state_init      uint8 = 0x00
	state_spare     uint8 = 0x01
	state_wait_sign uint8 = 0x02
	state_wait_done uint8 = 0x03

	state_recovery uint8 = 0x10
)

var (
	promoteCycle         = coefficient.BlockOrder_ServerPromoteCycle
	messagePromoteCycle  = coefficient.BlockOrder_MessageProcessCycle
	blockExpireScanCycle = coefficient.BlockOrder_BlockExpireScanCycle
)

var (
	agent_block_max_hold_time = coefficient.BlockOrder_AgentBlockHoldTime
	signThresholdRate         = float64(coefficient.BlockOrder_SignThresholdRate) // to compute the threshold for signature message
	doneThresholdRate         = float64(coefficient.BlockOrder_DoneThresholdRate) // to compute the threshold for fone message
	initThresholdRate         = float64(coefficient.BlockOrder_InitThresholdRate) // to compute the threshold for initialization and restart of booter
)

var (
	signThreshold, doneThreshold, initThreshold int
)

var (
	init_number_count = 0
	init_pool         = map[uint64]common.Hash{}
	init_enough_flag  = make(chan struct{})
	init_pool_mux     sync.RWMutex
)

var (
	signWaitDuration = coefficient.BlockOrder_SignMsgMaxWait
	doneWaitDuration = coefficient.BlockOrder_DoneMsgMaxWait
)

type BlockOrderServer struct {
	selfNode *dpnet.Node

	knownAgents map[common.NodeID]*agentState
	agentsMux   sync.RWMutex
	selectTurns circleTurns

	serviceMaintainer *chainMaintainer

	msgChannel consensus.UpperChannel

	quit chan struct{}

	netManager *netconfig.NetManager
}

func NewBlockOrderServer(selfNode *dpnet.Node, initVersion common.Hash, initHeight uint64, channel consensus.UpperChannel, netManager *netconfig.NetManager) *BlockOrderServer {
	sm := NewChainMaintainer(initVersion, initHeight)

	bos := &BlockOrderServer{
		selfNode:          selfNode,
		knownAgents:       make(map[common.NodeID]*agentState),
		agentsMux:         sync.RWMutex{},
		selectTurns:       circleTurns{},
		serviceMaintainer: sm,
		msgChannel:        channel,
		quit:              make(chan struct{}),

		netManager: netManager,
	}

	liveMembers := bos.msgChannel.BackMembers()
	for _, mem := range liveMembers {
		us := NewAgentState(mem)
		bos.knownAgents[mem] = us
	}

	bos.selectTurns = NewCircleTurns()
	for member, _ := range bos.knownAgents {
		bos.selectTurns.add(member)
	}

	return bos
}

func (b *BlockOrderServer) checkSelf() bool {
	if b.selfNode == nil {
		return false
	}
	if b.knownAgents == nil {
		return false
	}
	if b.serviceMaintainer == nil {
		return false
	}
	if b.msgChannel == nil {
		return false
	}
	return true
}

func (b *BlockOrderServer) Start() error {
	if !b.checkSelf() {
		return fmt.Errorf("[Block Ordering] fail in start block order server: fail in check\n")
	}

	signThreshold = int(math.Ceil(float64(len(b.knownAgents)) * signThresholdRate))
	doneThreshold = int(math.Ceil(float64(len(b.knownAgents)) * doneThresholdRate))
	initThreshold = int(math.Ceil(float64(len(b.knownAgents)) * initThresholdRate))

	// TODO: do loops here!

	go b.promoteLoop()
	go b.messageProcessLoop()
	go b.blockExpireScanLoop()

	return nil
}

func (b *BlockOrderServer) Stop() {
	close(b.quit)
}

func (b *BlockOrderServer) promoteLoop() {
	tk := time.NewTicker(promoteCycle)
	for {
		select {
		case <-tk.C:
			b.servicePromote()
		case <-b.quit:
			tk.Stop()
			return
		}
	}
}

func (b *BlockOrderServer) messageProcessLoop() {
	tk := time.NewTicker(messagePromoteCycle)
	for {
		select {
		case <-tk.C:
			b.msgProcess()
		case <-b.quit:
			tk.Stop()
			return
		}
	}
}

func (b *BlockOrderServer) blockExpireScanLoop() {
	tk := time.NewTicker(blockExpireScanCycle)
	for {
		select {
		case <-tk.C:
			b.blockExpire()
		case <-b.quit:
			tk.Stop()
			return
		}
	}
}

func (b *BlockOrderServer) blockExpire() {
	b.agentsMux.RLock()
	defer b.agentsMux.RUnlock()

	for _, as := range b.knownAgents {
		as.expireBlocks()
	}
}

type agentState struct {
	nodeID    common.NodeID
	stability int  // 100 at the initialization
	credit    int  // 100 at the initialization
	ignore    bool // whether to serve this agent
	lastServe time.Time

	pendingBlocks map[common.Hash]*blockState
	pendingMux    sync.RWMutex
}

func NewAgentState(nodeID common.NodeID) *agentState {
	as := &agentState{
		nodeID:        nodeID,
		stability:     100,
		credit:        100,
		ignore:        false,
		lastServe:     time.Time{},
		pendingBlocks: make(map[common.Hash]*blockState),
		pendingMux:    sync.RWMutex{},
	}
	return as
}

type blockState struct {
	checkInTime time.Time
	expireTime  time.Time
	block       *eles.Block
}

// At this time, a simple FIFO algorithm is applied
// NOTE: Block retrieved will be deleted
func (as *agentState) retrieveBlock() (*eles.Block, error) {
	as.pendingMux.Lock()
	defer as.pendingMux.Unlock()
	if len(as.pendingBlocks) == 0 {
		return nil, fmt.Errorf("No block is in pending")
	}
	nowTime := time.Now()
	gotFlag := false
	var chosenBlock *eles.Block
	var chosenTime time.Time
	for _, bs := range as.pendingBlocks {
		if bs.expireTime.Before(nowTime) {
			continue
		} else {
			if gotFlag {
				if chosenTime.After(bs.checkInTime) {
					chosenTime = bs.checkInTime
					chosenBlock = bs.block
				}
			} else {
				chosenTime = bs.checkInTime
				chosenBlock = bs.block
				gotFlag = true
			}
		}
	}
	if !gotFlag {
		return nil, fmt.Errorf("No valid block can be retrieved")
	} else {
		delete(as.pendingBlocks, chosenBlock.BlockID)
		return chosenBlock, nil
	}
}

func (as *agentState) expireBlocks() {
	as.pendingMux.Lock()
	defer as.pendingMux.Unlock()

	th1 := time.Now()
	th2 := th1.Add(-agent_block_max_hold_time)

	for id, bs := range as.pendingBlocks {
		if bs.expireTime.Before(th1) {
			delete(as.pendingBlocks, id)
		} else {
			if bs.checkInTime.Before(th2) {
				delete(as.pendingBlocks, id)
			}
		}
	}
}

func (as *agentState) addBlock(block *eles.Block, expireTime time.Time) {
	as.pendingMux.Lock()
	defer as.pendingMux.Unlock()

	bs := &blockState{
		checkInTime: time.Now(),
		expireTime:  expireTime,
		block:       block,
	}

	as.pendingBlocks[block.BlockID] = bs
}

// At this time, we design a circle turn like block selection method, trying to make every subnet
// have equal probability to use the block order service.
// TODO: More efficient and smart block select strategy could be designed.
func (b *BlockOrderServer) selectNextBlock() (*eles.Block, error) {
	b.agentsMux.RLock()
	defer b.agentsMux.RUnlock()

	maxTry := b.selectTurns.length

	for i := 0; i < maxTry; i++ {
		nowTurn, err := b.selectTurns.getTurn()
		if err != nil {
			loglogrus.Log.Warnf("[Block Ordering] Upper Consensus failed: BlockOrderServer.selectRequire encountered an error while running getTurn, err:%v\n", err)
			break
		}
		us, ok := b.knownAgents[nowTurn]
		if !ok {
			loglogrus.Log.Warnf("[Block Ordering] Upper Consensus failed: BlockOrderServer.selectRequire is failed, node %x is not live\n", nowTurn)
			continue
		}
		slctBlk, err := us.retrieveBlock()
		if err == nil {
			return slctBlk, nil
		}
	}
	return nil, fmt.Errorf("no require is selected")
}

func (b *BlockOrderServer) addBlock(sender common.NodeID, block *eles.Block, expireTime time.Time) error {
	b.agentsMux.RLock()
	defer b.agentsMux.RUnlock()
	as, ok := b.knownAgents[sender]
	if !ok {
		return fmt.Errorf("fail in add block, the sender %x is not known", sender)
	}
	as.addBlock(block, expireTime)
	return nil
}

func (b *BlockOrderServer) msgProcess() {
	currentMsgs := b.msgChannel.CurrentMsgs()

	toMark := make([]common.Hash, 0)

	for _, msg := range currentMsgs {
		toMark = append(toMark, msg.CalculateHash())

		if !msg.IsUpperConsensus() {
			loglogrus.Log.Warnf("[Block Ordering] Receive Message isn't Upper Consensus Message!\n")
			continue
		}

		bom, err := DeserializeBlockOrderMsg(msg.PayLoad)

		if err != nil {
			loglogrus.Log.Warnf("[Block Ordering] Upper Consensus failed: Couldn't Deserialize separator.Message into Upper Consensus Msg, err:%v\n", err)
			continue
		}

		if msg.From != bom.Sender {
			loglogrus.Log.Warnf("[Block Ordering] Upper Consensus failed: Illegal message, from: %x could not match the sender: %x with type code: %v, the system maybe under attack\n", msg.From, bom.Sender, bom.TypeCode)
			continue
		}

		plainMsg, err := RetrievePayload(bom)

		if err != nil {
			loglogrus.Log.Warnf("[Block Ordering] Fail in msgProcess: %v\n", err)
		}

		switch bom.TypeCode {
		case CODE_UPLD:
			uplMsg, _ := plainMsg.(*UploadMessage)
			loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_UPLD Msg\n", msg.From)
			go b.handleUpload(bom.Sender, uplMsg)
			loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_UPLD Msg的处理\n", msg.From)

		case CODE_SIGN:
			sigMsg, _ := plainMsg.(*SignMessage)
			loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_SIGN Msg\n", msg.From)
			go b.handleSign(bom.Sender, sigMsg)
			loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_SIGN Msg的处理\n", msg.From)

		case CODE_DONE:
			donMsg, _ := plainMsg.(*DoneMessage)
			loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_DONE Msg\n", msg.From)
			go b.handleDone(bom.Sender, donMsg)
			loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_DONE Msg的处理\n", msg.From)

		case CODE_STAT:
			loglogrus.Log.Infof("[Block Ordering] Upper Channel收到来自节点(%x)的 CODE_STAT Msg\n", msg.From)
			staMsg, _ := plainMsg.(*StateMessage)
			go b.handleState(staMsg)
			loglogrus.Log.Infof("[Block Ordering] Upper Channel完成对来自节点(%x)的 CODE_STAT Msg的处理\n", msg.From)

		case CODE_UPDATEBLOCKVALIDATOR:
			ubvMsg, _ := plainMsg.(*UpdateBlockValidator)
			go b.handleUpdateValidator(ubvMsg, msg.From)

		default:
			loglogrus.Log.Warnf("[Block Ordering] Unknown block order message type : %d is received from %x\n", bom.TypeCode, bom.Sender)
			continue
		}

	}

	b.msgChannel.MarkRetrievedMsgs(toMark)
}

func (b *BlockOrderServer) handleUpload(sender common.NodeID, msg *UploadMessage) {
	block := &msg.Block
	expireTime := time.Unix(int64(msg.ExpireTime), 0)
	err := b.addBlock(sender, block, expireTime)
	if err != nil {
		loglogrus.Log.Warnf("[Block Ordering] fail in handleUpload: %v", err)
	}
}

func (b *BlockOrderServer) servicePromote() {
	switch b.serviceMaintainer.state {
	case state_init:
		b.simpleInit()
	case state_spare:
		b.tryLaunchNewRound()
	case state_wait_sign:
		b.tryCommit()
	case state_wait_done:
		b.tryFinish()
	case state_recovery:
		b.tryRelaunchNewRound()

	default:
		loglogrus.Log.Errorf("[Block Ordering] fatal error: unknown state: %d", b.serviceMaintainer.state)
	}
}

func (b *BlockOrderServer) tryLaunchNewRound() {
	block, err := b.selectNextBlock()
	if err != nil {
		// here we do not log the error, as most of the time, the error is caused by no block
		loglogrus.Log.Debugf("[Block Ordering] could not launch new round %v", err)
		return
	}

	b.serviceMaintainer.roundMux.RLock()
	defer b.serviceMaintainer.roundMux.RUnlock()

	nxtMsg := &NextMessage{
		Version: b.serviceMaintainer.currentVersion,
		Height:  b.serviceMaintainer.currentHeight,
		Block:   *block,
	}

	bom, err := CreateBlockOrderMsg(nxtMsg, b.selfNode.NodeID, CODE_NEXT)
	if err != nil {
		loglogrus.Log.Errorf("[Block Ordering] fail in tryLaunchNewRound: %v", err)
		return
	}

	b.serviceMaintainer.nxtMsg = nxtMsg
	b.broadcastMessage(bom)
	b.serviceMaintainer.signExpire = time.Now().Add(signWaitDuration)
	b.serviceMaintainer.state = state_wait_sign
	loglogrus.Log.Infof("[Block Ordering] A new next message is generated and sent, version: %x, height: %d, BlockID: %x\n", nxtMsg.Version, nxtMsg.Height, nxtMsg.Block.BlockID)
}

func (b *BlockOrderServer) tryRelaunchNewRound() {
	block, err := b.selectNextBlock()
	if err != nil {
		// here we do not log the error, as most of the time, the error is caused by no block
		loglogrus.Log.Debugf("[Block Ordering] could not launch new round %v", err)
		return
	}

	b.serviceMaintainer.roundMux.RLock()
	defer b.serviceMaintainer.roundMux.RUnlock()

	rcvMsg := &RecoveryMessage{
		RepeatCount: b.serviceMaintainer.repeatCount,
		Version:     b.serviceMaintainer.currentVersion,
		Height:      b.serviceMaintainer.currentHeight,
		Block:       *block,
	}

	nxtMsg := &NextMessage{
		Version: b.serviceMaintainer.currentVersion,
		Height:  b.serviceMaintainer.currentHeight,
		Block:   *block,
	}

	bom, err := CreateBlockOrderMsg(rcvMsg, b.selfNode.NodeID, CODE_RCVY)
	if err != nil {
		loglogrus.Log.Errorf("[Block Ordering] fail in tryRelaunchNewRound: %v", err)
		return
	}

	b.serviceMaintainer.nxtMsg = nxtMsg
	b.broadcastMessage(bom)
	b.serviceMaintainer.signExpire = time.Now().Add(signWaitDuration)
	b.serviceMaintainer.state = state_wait_sign

	loglogrus.Log.Warnf("[Block Ordering] Recovery message has been sent, the network is not stable, please check! Current version: %x, height: %d\n", b.serviceMaintainer.currentVersion, b.serviceMaintainer.currentHeight)
}

func (b *BlockOrderServer) tryCommit() {
	b.serviceMaintainer.roundMux.RLock()
	defer b.serviceMaintainer.roundMux.RUnlock()

	// restart a new round
	if b.serviceMaintainer.signExpire.Before(time.Now()) {
		b.serviceMaintainer.state = state_recovery
		b.serviceMaintainer.repeatCount += 1
		b.serviceMaintainer.flush()
		return
	}

	receiptID, signs, got := b.serviceMaintainer.receiptCache.getMaxSignedReceipt(signThreshold)
	if !got {
		// here we do not log the error, as most of the time, the error is caused by not collecting enough signatures
		loglogrus.Log.Warnf("[Block Ordering] Not enough signatures got in round version %x, height %d\n", b.serviceMaintainer.currentVersion, b.serviceMaintainer.currentHeight)
		return
	}

	cmtMsg := &CommitMessage{
		Height:     b.serviceMaintainer.currentHeight + 1,
		BlockID:    b.serviceMaintainer.nxtMsg.Block.BlockID,
		ReceiptID:  receiptID,
		Signatures: signs,
	}

	bom, err := CreateBlockOrderMsg(cmtMsg, b.selfNode.NodeID, CODE_CMIT)
	if err != nil {
		loglogrus.Log.Errorf("[Block Ordering] fail in tryCommit: %v", err)
		return
	}

	b.serviceMaintainer.cmtMsg = cmtMsg
	b.broadcastMessage(bom)
	loglogrus.Log.Infof("[Block Ordering] 成功发送Commit Msg, next Block Hash(%x) Height(%d)\n", cmtMsg.BlockID, cmtMsg.Height)
	b.serviceMaintainer.doneExpire = time.Now().Add(doneWaitDuration)
	b.serviceMaintainer.state = state_wait_done
}

func (b *BlockOrderServer) tryFinish() {
	// time run out, should resend the commit msg
	if b.serviceMaintainer.doneExpire.Before(time.Now()) {
		bom, err := CreateBlockOrderMsg(b.serviceMaintainer.cmtMsg, b.selfNode.NodeID, CODE_CMIT)
		if err != nil {
			loglogrus.Log.Errorf("[Block Ordering] fail in tryFinish: %v", err)
			return
		}
		b.broadcastMessage(bom)
		b.serviceMaintainer.doneExpire = time.Now().Add(doneWaitDuration)
		b.serviceMaintainer.state = state_wait_done
		return
	}

	votes, _ := b.serviceMaintainer.doneCache.readResult()

	// not enough done messages collected yet
	if votes < doneThreshold {
		// here we do not log the error, as most of the time, the error is caused by not collecting enough done messages
		loglogrus.Log.Debugf("[Block Ordering] Not enough done messages got in round version %x, height %d\n", b.serviceMaintainer.currentVersion, b.serviceMaintainer.currentHeight)
		return
	}

	b.serviceMaintainer.roundMux.Lock()
	defer b.serviceMaintainer.roundMux.Unlock()

	b.serviceMaintainer.currentHeight += 1
	b.serviceMaintainer.currentVersion = b.serviceMaintainer.cmtMsg.BlockID
	b.serviceMaintainer.flush()
	b.serviceMaintainer.repeatCount = 0
	b.serviceMaintainer.state = state_spare

}

func (b *BlockOrderServer) broadcastMessage(bom *BlockOrderMessage) {
	payload, err := rlp.EncodeToBytes(bom)
	if err != nil {
		loglogrus.Log.Warnf("[Block Ordering] Consensus: fail in broadcastMessage : %v", err)
		return
	}
	msg := b.msgChannel.NewUpperConsensusMessage(b.selfNode.NodeID, payload)
	b.msgChannel.MsgBroadcast(msg)
}

func (b *BlockOrderServer) simpleInit() {
	b.serviceMaintainer.roundMux.Lock()
	defer b.serviceMaintainer.roundMux.Unlock()
	iniMsg := &InitMessage{
		Version: b.serviceMaintainer.currentVersion,
		Height:  b.serviceMaintainer.currentHeight,
	}
	bom, err := CreateBlockOrderMsg(iniMsg, b.selfNode.NodeID, CODE_INIT)
	if err != nil {
		loglogrus.Log.Errorf("[Block Ordering] fail in simpleInit: %v\n", err)
		return
	}
	loglogrus.Log.Infof("[Block Ordering] simple init begins\n")
	b.broadcastMessage(bom)

	// wait for flag right
	time_out := time.NewTimer(5 * time.Second) // max wait is 5 seconds
	select {
	case <-time_out.C:
		loglogrus.Log.Errorf("[Block Ordering] fail in simpleInit, no enough state messages collected!\n")
		return
	case <-init_enough_flag:
	}

	init_pool_mux.RLock()
	defer init_pool_mux.RUnlock()

	if len(init_pool) == 0 {
		loglogrus.Log.Errorf("[Block Ordering] fail in simpleInit, no state messages is collected!\n")
		return
	}

	var maxHight uint64
	for h, _ := range init_pool {
		if h > maxHight {
			maxHight = h
		}
	}

	b.serviceMaintainer.currentVersion = init_pool[maxHight]
	b.serviceMaintainer.currentHeight = maxHight

	loglogrus.Log.Infof("[Block Ordering] service init succeed: current height %d, current version %x\n", b.serviceMaintainer.currentHeight, b.serviceMaintainer.currentVersion)

	synMsg := &SynchronizeMessage{
		Version: b.serviceMaintainer.currentVersion,
		Height:  b.serviceMaintainer.currentHeight,
	}

	bom2, err := CreateBlockOrderMsg(synMsg, b.selfNode.NodeID, CODE_SYNC)
	if err != nil {
		loglogrus.Log.Errorf("[Block Ordering] fail in simpleInit: %v\n", err)
		return
	}

	b.serviceMaintainer.state = state_spare
	b.broadcastMessage(bom2)
	return
}

// very simple state verification without any security concerns
func (b *BlockOrderServer) handleState(msg *StateMessage) {
	if b.serviceMaintainer.state != state_init {
		return
	}

	init_pool_mux.Lock()
	defer init_pool_mux.Unlock()
	init_pool[msg.Height] = msg.Version
	init_number_count += 1
	if init_number_count >= initThreshold {
		init_enough_flag <- struct{}{}
	}
	return
}

func (b *BlockOrderServer) handleSign(sender common.NodeID, msg *SignMessage) {
	if b.serviceMaintainer.state != state_wait_sign {
		return
	}

	if msg.Valid != BlockValid {
		loglogrus.Log.Warnf("[Block Ordering] Signatures with invalid flag is got from %x, malicious nodes may exist in the network! The invalid blockID %x\n", sender, msg.BlockID)
		return
	}

	b.serviceMaintainer.roundMux.RLock()
	defer b.serviceMaintainer.roundMux.RUnlock()

	if b.serviceMaintainer.nxtMsg == nil || b.serviceMaintainer.nxtMsg.Block.BlockID != msg.BlockID {
		loglogrus.Log.Warnf("[Block Ordering] Unexpected sign message blockID %x and receiptID %x is got\n", msg.BlockID, msg.ReceiptID)
		return
	}

	receipt := &receiptSign{
		author:    sender,
		receiptID: msg.ReceiptID,
		signature: msg.Signature,
	}

	loglogrus.Log.Infof("[Block Ordering] 接收到来自节点(%x)针对区块(%x)的交易回执(%x)\n", sender, msg.BlockID, msg.ReceiptID)

	b.serviceMaintainer.receiptCache.addReceipt(receipt)
}

func (b *BlockOrderServer) handleDone(sender common.NodeID, msg *DoneMessage) {
	if b.serviceMaintainer.state != state_wait_done {
		return
	}

	b.serviceMaintainer.roundMux.RLock()
	defer b.serviceMaintainer.roundMux.RUnlock()

	if b.serviceMaintainer.cmtMsg == nil || b.serviceMaintainer.cmtMsg.BlockID != msg.BlockID {
		loglogrus.Log.Warnf("[Block Ordering] Unexpected done message blockID %x and height %d is got\n", msg.BlockID, msg.Height)
		return
	}

	valid := true
	if msg.Valid != BlockValid {
		valid = false
	}

	b.serviceMaintainer.doneCache.add(sender, valid)

}

func (b *BlockOrderServer) handleUpdateValidator(ubvMsg *UpdateBlockValidator, sender common.NodeID) {

	faultGroup := b.netManager.GetDPNetInfo()[ubvMsg.GroupID]
	pbftFactor := (len(faultGroup.Nodes) - 1) / 3

	loglogrus.Log.Infof("[View Change] 故障分区(%s)节点总数(%d),应搜集的数字签名个数(%d),已搜集的数字签名个数(%d)\n", ubvMsg.GroupID, len(faultGroup.Nodes), 2*pbftFactor+1,
		len(ubvMsg.Sigs))

	if len(ubvMsg.Sigs) < 2*pbftFactor+1 {
		loglogrus.Log.Warnf("[View Change] 故障分区(%s)节点总数(%d),应搜集的数字签名个数(%d),已搜集的数字签名个数(%d)  搜集的数字签名数量不够,无法进行Leader节点的切换\n",
			ubvMsg.GroupID, len(faultGroup.Nodes), 2*pbftFactor+1, len(ubvMsg.Sigs))
		return
	}

	// 验证hash是否正确
	serialized, _ := rlp.EncodeToBytes(sender)
	newLeaderHash := crypto.Sha3Hash(serialized)

	loglogrus.Log.Infof("[View Change] 故障分区选举的新Leader Hash为(%x), 当前目标节点Hash为(%x)\n", ubvMsg.SenderHash, newLeaderHash)

	if !reflect.DeepEqual(newLeaderHash, ubvMsg.SenderHash) {
		loglogrus.Log.Warnf("[View Change] 要更换的Leader节点哈希不符,无法进行Leader节点的切换\n")
		return
	}

	// 1.计算出故障分区的所有节点的公钥
	pubKeySet := make([][]byte, 0)
	for nodeID, _ := range faultGroup.Nodes {
		pub, err := crypto.NodeIDtoKey(nodeID)
		if err != nil {
			loglogrus.Log.Warnf("[View Change] 无法从故障分区NodeID:%x 解析出其公钥\n", nodeID)
			continue
		}
		pubKey := crypto.FromECDSAPub(pub)
		pubKeySet = append(pubKeySet, pubKey)
	}

	// 2.验证消息中的所有数字签名是否正确(公钥是否全部存在于pubKeySet中)
	for _, sig := range ubvMsg.Sigs {
		sigPublicKeyByte, err := crypto.Ecrecover(ubvMsg.SenderHash[:], sig) // 获取签名公钥
		if err != nil {
			loglogrus.Log.Warnf("[View Change] fail in Ecrecover ViewChangeMsg")
			return
		}

		index := sort.Search(len(pubKeySet), func(i int) bool { // 检查签名公钥是否在pubKeySet中存在
			pubKeySet[i] = sigPublicKeyByte
			return true
		})

		if index == len(pubKeySet) {
			loglogrus.Log.Warnf("[View Change] Leader Change Msg中存在无效的数字签名,无法进行Leader节点的切换\n")
			return
		}
	}

	// 重新更新booter的knownAgents以及selectTurns
	b.agentsMux.Lock()
	defer b.agentsMux.Unlock()

	liveMembers := b.msgChannel.BackMembers()
	b.knownAgents = make(map[common.NodeID]*agentState)
	for _, mem := range liveMembers {
		us := NewAgentState(mem)
		b.knownAgents[mem] = us
	}
	b.selectTurns = NewCircleTurns()
	for member, _ := range b.knownAgents {
		b.selectTurns.add(member)
	}

	loglogrus.Log.Infof("[View Change] 当前Booter节点(%x)完成对故障分区Leader的切换\n", b.selfNode.NodeID)
}
