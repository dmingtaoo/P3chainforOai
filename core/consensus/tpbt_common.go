package consensus

import (
	"fmt"
	"p3Chain/coefficient"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/core/utils"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"
	"sort"
	"sync"
	"time"
)

const (
	COMMON_TPBFT_NAME = "cpbft"

	cpbftRoundLifeTime = 5000 * time.Millisecond // the max life time to hold a round

)

// Ctpbft is a common kind of TPBFT which could be instanced in most scenarios.
type Ctpbft struct {
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
}

// NewCtpft use the default options to create a common tpbft
func NewCtpbft() *Ctpbft {
	ctpft := Ctpbft{
		name:              COMMON_TPBFT_NAME,
		running:           false,
		maxTransactionNum: DEFAULT_MAX_TRANSACTIONS,
		liveMember:        make(map[common.NodeID]int),
		lastVersion:       common.StringToHash("PRIMARY VERSION"),
		nonce:             0,
	}
	return &ctpft

}

// Install creates the associated transaction pool and set it to TPBFT, then injects both
// the transaction pool and TPBFT to consensus promoter.
func (ct *Ctpbft) Install(consensusPromoter *ConsensusPromoter) {
	ct.consensusPromoter = consensusPromoter //为当前Ctpbft对象绑定一个consensusPromoter对象

	ct.liveMemberMux.Lock()
	defer ct.liveMemberMux.Unlock()

	initialMembers := consensusPromoter.lowerChannel.BackMembers() //获取当前节点组内所有成员的NodeID
	for i := 0; i < len(initialMembers); i++ {                     //每个节点初始的可信度为100
		ct.liveMember[initialMembers[i]] = 100
	}
	ct.liveMember[ct.consensusPromoter.SelfNode.NodeID] = 100 // as lowerChannel.BackMembers() does not include self nodeID

	ct.consensusPromoter.poolMu.Lock()
	defer ct.consensusPromoter.poolMu.Unlock()

	ct.consensusPromoter.tpbftMux.Lock()
	defer ct.consensusPromoter.tpbftMux.Unlock()
	txPool := eles.NewTransactionPool(ct.name, ct.TxMatch, ct.TxChoose) //创建一个新的交易池(ct.TxMatch和ct.TxChoose是创建交易池必须自行定义的两个函数)
	txPool.Start()
	ct.txPool = txPool

	ct.consensusPromoter.installedTpbft[ct.name] = ct                                             //以及绑定的consensusPromoter对象内部也要注册使用的TPFBT
	ct.consensusPromoter.transactionPools = append(ct.consensusPromoter.transactionPools, txPool) //将当前节点新创建的交易池加入到交易池集合中

}

func (ct *Ctpbft) Name() string {
	return ct.name
}

func (ct *Ctpbft) ViewChange() {
	panic("To implement")
}

// 发送Pre-Prepare阶段消息(基于separator协议在组内完成广播) PeerGroup.go内可查NewLowerConsensusMessage和MsgBroadcast的实现
func (ct *Ctpbft) SendPrePrepareMsg(ppm *PrePrepareMsg) {
	wlcm := ppm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: RoundID (%x), Unable to encode PrePrepareMsg into LowerConsensusMessage.Payload, err:%v\n", ppm.head.RoundID, err)
	}
	msg := ct.consensusPromoter.lowerChannel.NewLowerConsensusMessage(ct.consensusPromoter.SelfNode.NodeID, payload)

	ct.consensusPromoter.lowerChannel.MsgBroadcast(msg)
}

// 发送Prepare阶段消息（同上，完成广播）
func (ct *Ctpbft) SendPrepareMsg(pm *PrepareMsg) {
	wlcm := pm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: RoundID (%x), Unable to encode PrepareMsg into LowerConsensusMessage.Payload, err:%v\n", pm.head.RoundID, err)
		return
	}
	msg := ct.consensusPromoter.lowerChannel.NewLowerConsensusMessage(ct.consensusPromoter.SelfNode.NodeID, payload)
	ct.consensusPromoter.lowerChannel.MsgBroadcast(msg)
}

// 向指定目标节点发送Commit阶段消息(也必须是组内的某一节点)
func (ct *Ctpbft) SendCommitMsg(cm *CommitMsg, nodeID common.NodeID) {
	if ct.consensusPromoter.SelfNode.NodeID == nodeID { // the node self is leader
		return
	}
	wlcm := cm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: RoundID (%x), Unable to encode CommitMsg into LowerConsensusMessage.Payload, err:%v\n", cm.head.RoundID, err)
		return
	}
	msg := ct.consensusPromoter.lowerChannel.NewLowerConsensusMessage(ct.consensusPromoter.SelfNode.NodeID, payload)
	err = ct.consensusPromoter.lowerChannel.MsgSend(msg, nodeID) //只能局限于组内定点发送
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: RoundID (%x), Unable to send CommitMsg to Leader (%x), err:%v", cm.head.RoundID, nodeID, err)
	}
}

// As transaction has been done match function to valid the signature,
// in this common tpbft we just check the stale.
// 由于交易已经完成了相应的match函数确保了签名的有效性，因此这里我们只根据交易的过时情况验证其有效性
func (ct *Ctpbft) ValidTransaction(txs []*eles.Transaction, version common.Hash) (validOrder []byte) {
	validOrder = make([]byte, len(txs))
	for i := 0; i < len(validOrder); i++ { //仅检查对应的交易消息是否已经过期
		if !ct.consensusPromoter.StateManager.StaleCheck(txs[i].Version, version, txs[i].LifeTime) {
			validOrder[i] = byte(1) //没过期的交易认为是有效的(valid)，对应bit位设置为1
		} else {
			validOrder[i] = byte(0) //过期的--> 无效 --> 对应bit位设为0
		}
	}
	return
}

// In common tpbft, we just trust every live member with reliability
// not less tha zero. The valid order with a number of votes over 2f
// of total wins.
// 在common tpbft协议中,我们认为那些 信任度 > 0 的live节点是可信任的
// 本方法的目的是从一组prepare共识消息中，找出唯一的合法有效的交易验证序列validOrder(也就是大多数prepareMsg都认为有效的validOrder)
func (ct *Ctpbft) CheckPrepare(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*PrepareMsg) (ok bool, validOrder []byte) {
	ct.liveMemberMux.RLock()
	defer ct.liveMemberMux.RUnlock()

	threshold := 2 * pbftFactor

	orderMap1 := make(map[uint64]int)
	orderMap2 := make(map[string]int)

	for node, pm := range msgPool { //遍历所有prepare消息(node为消息的发送者NodeID) -- 这些prepare消息每一个都存储着ValidOrder,也就是对一组交易的验证序列
		if _, ok := liveMemberSnapShot[node]; !ok { //检查prepare消息是否确实来自于形参指定的节点集合
			continue
		}
		if trust, ok := ct.liveMember[node]; ok { //查找prepare消息的发送节点在本节点上的信任值
			if trust >= 0 { // Credit evaluation mechanism could be applied here  信任值需要大于0
				if len(pm.ValidOrder) > 64 {
					ketStr := utils.BoolByteSliceToString(pm.ValidOrder)
					orderMap2[ketStr] += 1
					if orderMap2[ketStr] >= threshold {
						return true, pm.ValidOrder
					}
				} else {
					keyInt, err := utils.BoolByteSliceToUInt64(pm.ValidOrder) // 将bool切片(存储着交易验证序列)转换为对应的int64类型数值
					if err != nil {
						loglogrus.Log.Errorf("Lower Channel failed: Can't transform ValidOrder within prepare Message where from Node (%x) into Uin64!\n", node)
						continue
					}
					orderMap1[keyInt] += 1              //每当有一个可信节点对于交易序列投票数+1
					if orderMap1[keyInt] >= threshold { //某一交易验证序列的投票数达到了TPBFT标准,则认为当前交易序列是合法有效的
						return true, pm.ValidOrder
					}
				}

			} else { //一旦出现信任值小于0的节点，需要在日志中记录此节点以及其出现的交易roundID
				loglogrus.Log.Warnf("[TPBFT Consensus] Lower Channel warnning: round ID: %v has found prepare message from unknown node: %v, system may be in danger!\n", pm.head.RoundID, node)
				continue
			}
		}
	}

	return false, []byte{}
}

// 返回Ctpbft对象liveMember存储的所有live节点的NodeID(key为NodeID,value统一设置为true)
func (ct *Ctpbft) backSnapShotLiveMembers() map[common.NodeID]bool {
	ct.liveMemberMux.RLock()
	defer ct.liveMemberMux.RUnlock()

	snapShot := make(map[common.NodeID]bool)
	for nodeID, _ := range ct.liveMember {
		snapShot[nodeID] = true
	}
	return snapShot
}

// TidyTransactions tidies the ordered transactions and construct a block.
// This is the essential function to construct the connection of contract
// and the consensus.
//
// As common tpbft is a default lower-level consensus of p3Chain, we do nothing
// here but simply construct a block using the transaction order in agreement.
//
// TODO: Make this function more flexible.

// 本方法基于prepare阶段结束后的唯一交易有效认证序列(节点组内达成共识的那一个序列),构建相应区块
// 返回值:
// 1.组装好后的区块的ID
// 2.区块组装者的数字签名(用区块ID计算得到的)
// 3.组装好的整个区块(仅包含有效交易)
func (ct *Ctpbft) TidyTransactions(leader common.NodeID, version common.Hash, nonce uint8, txs []*eles.Transaction, validOrder []byte) (result []byte, signature []byte, block *eles.Block) {
	plainTxs := make([]eles.Transaction, 0) //存储所有有效的交易
	for i := 0; i < len(txs); i++ {         //遍历整个交易集合，只提取出有效的交易(validOrder中对应bit位==1的)
		if validOrder[i] == byte(1) {
			plainTxs = append(plainTxs, *txs[i])
		}
	}

	leaderAddress, err := crypto.NodeIDtoAddress(leader) //根据NodeID计算Address
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Couldn't tidy Transactions into candidateBlock, it's not to assert NodeID into Address!\n")
		return nil, nil, nil
	}

	//构建区块
	candidateBlock := &eles.Block{
		BlockID:      common.Hash{},                               //零值
		Subnet:       []byte(ct.consensusPromoter.SelfNode.NetID), //当前节点所在的子网ID
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
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Couldn't tidy Transactions into candidateBlock, Compute BlockID is failed, err:%v\n", err)
		return nil, nil, nil
	}
	result = blockID[:]
	signature, err = crypto.Sign(result, ct.consensusPromoter.prvKey) //生成数字签名(用区块ID产生的)
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Couldn't tidy Transactions into candidateBlock, Generate digtal signature is failed, err:%v\n", err)
		return nil, nil, nil
	}

	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Leader (%x) Tidy candidateBlock successfully, blockID:%x, txSum:%d\n", leaderAddress, candidateBlock.BlockID, len(candidateBlock.Transactions))

	return result, signature, candidateBlock
}

// 提交一个区块(Leader节点才能提交)，要满足的条件如下:
// 1.Leader必须事先获取足够多的来自于节点组内部成员节点的Commit共识消息
// 2.Leader必须获取足够多的可信任节点的数字签名
func (ct *Ctpbft) SubmitBlock(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*CommitMsg, block *eles.Block) bool {
	threshold := pbftFactor * 2

	if len(msgPool) < threshold { // 一个区块要提交，必须获取足够多的commit消息(相当于投票)
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: The number of commitMsg (%d) lower than threshold (%d), so it couldn't submit candidateBlock (%x) upto UpperChannel!\n",
			len(msgPool), threshold, block.BlockID)
		return false
	}

	// tempBlock := new(eles.Block)
	// tempBlock.BlockID = block.BlockID
	// copy(tempBlock.Subnet, block.Subnet)
	// tempBlock.Leader = block.Leader
	// tempBlock.Version = block.Version
	// tempBlock.Nonce = block.Nonce
	// copy(tempBlock.Transactions, block.Transactions)
	// copy(tempBlock.SubnetVotes, block.SubnetVotes)
	// tempBlock.PrevBlock = block.PrevBlock
	// tempBlock.CheckRoot = block.CheckRoot
	// tempBlock.Receipt = block.Receipt
	// copy(tempBlock.LeaderVotes, block.LeaderVotes)

	// for i, v := range msgPool {
	// 	loglogrus.Log.Infof("[TPBFT Consensus] 总共有%d条Commit Msg,当前来自于Node(%x)的Commit Msg, BlockID:%x, Signature:%v\n", len(msgPool), i, common.BytesToHash(v.Result), v.Signature)
	// }

	ct.liveMemberMux.RLock()
	defer ct.liveMemberMux.RUnlock()

	subNetVotes := make([]eles.SubNetSignature, 0)

	for node, cm := range msgPool { //遍历所有Commit共识消息(同时获取消息发送者ID)
		msg := cm
		//loglogrus.Log.Infof("[TPBFT Consensus] 总共有%d条Commit Msg,当前来自于Node(%x)的Commit Msg, BlockID:%x, Signature:%x\n", len(msgPool), node, common.BytesToHash(msg.Result), cm.Signature)

		if _, ok := liveMemberSnapShot[node]; !ok { //在形参的节点集合中查询 当前消息发送者ID
			continue
		}
		if trust, ok := ct.liveMember[node]; ok { //获取此Commit共识消息发送节点的 信任值
			if trust >= 0 { // Credit evaluation mechanism could be applied here
				blockIdHash := common.BytesToHash(msg.Result) //从Commit共识消息中获取待发布区块的ID(common.BytesToHash只是用于截断，并非重新计算哈希)
				if blockIdHash != block.BlockID {             //获得的 blockIdHash 需要等于形参传入区块的ID值才能向下执行

					continue
				}
				addr, err := crypto.NodeIDtoAddress(node) //根据节点的NodeID计算获取Address
				if err != nil {
					loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Couldn't submit candidateBlock (%x) upto UpperChannel, Unable to transform NodeID into Address, err:%v\n",
						block.BlockID, err)
					continue
				}

				valid, err := crypto.SignatureValid(addr, msg.Signature, blockIdHash) //验证Commit共识消息的数字签名是否有效
				if err != nil {
					loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Couldn't submit candidateBlock (%x) upto UpperChannel, Digtal signature is unvalid, err:%v\n",
						block.BlockID, err)
					continue
				}
				if valid { //将有效的数字签名记录到subNetVotes集合
					subNetVotes = append(subNetVotes, msg.Signature)
				}

			} else {
				loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: round ID: %x has found commit message from unknown node: %x, system may be in danger!\n", msg.head.RoundID, node)
				continue
			}
		}
	}

	if len(subNetVotes) < threshold { //必须要获取足够多节点的数字签名
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: The number of Signatrue Vote (%d) that is supported for candidateBlock (%x) lower than threshold (%d),"+
			"so it couldn't submit candidateBlock upto UpperChannel!\n", len(subNetVotes), block.BlockID, threshold)
		return false
	}

	// upload the block to do upper-level consensus
	block.SubnetVotes = subNetVotes

	ct.consensusPromoter.blockCache.AddLocalBlock(block) //提交该区块到upper层
	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Block: %x is submitted upto UpperChannel, Version: %x, Nonce: %d, txSum: %d\n", block.BlockID, block.Version, block.Nonce, len(block.Transactions))

	return true

}

// Start starts Ctpbft
// 运行C-TPBFT共识协议
func (ct *Ctpbft) Start() {
	ct.running = true
	loglogrus.Log.Infof("Ctpbft: Common Kind PBFT starts to running, transaction pool: %x\n", ct.txPool.PoolID)
	go ct.update()
}

// NOTE: Here we make a common first in and first out choose strategy,
// but also provides privilege to transactions with short life time.
// This kind of strategy might be unstable. Because we could not make each
// node of a subnet have the same current version during the lower-level
// consensus. The transactions with specific life time could be inferred as
// valid in some nodes but invalid in others, which might make the whole
// three stages of consensus dropped.
//
// TODO: Customize some more stable and efficient choose strategies.
// 本方法负责筛选出符合条件的交易集合
// 1.指定一个先进先出(FIFO)选择策略,但也为短life time的交易提供特权
// 2.这种策略是不稳定的,因为我们无法确保在底层共识阶段的每一个子网内节点具有相同的 current version
// 3.具有特定life time的交易在某些节点中可能会被认为是有效的，但在另一些节点中可能又是无效的,这会导致共识协议的三个阶段的整体性下降

// 实现上：从形参给定的交易集合中筛选出前maxTransactionNum条交易(按照交易的发布时间)
func (ct *Ctpbft) TxChoose(freshTxs []*eles.TransactionState) []*eles.Transaction {
	chosenTxs := make([]*eles.Transaction, 0, ct.maxTransactionNum)

	if len(freshTxs) <= ct.maxTransactionNum { //如果形参提供的交易集合中交易数目小于共识协议处理上限
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

	for i := 0; i < ct.maxTransactionNum; i++ { //提取出前maxTransactionNum条交易进行后续处理
		chosenTxs = append(chosenTxs, freshTxs[i].Tx)
	}
	return chosenTxs
}

// Check out whether a transaction matches the associated pool.
// At this time, we only check the signature.
// NOTE: One transaction could be matched to only one transaction pool,
// so this function needs to be carefully designed to avoid conflicts
// 通过检查交易的签名确定交易与交易池的匹配性
func (ct *Ctpbft) TxMatch(tx *eles.Transaction) bool {
	// First we should filter the contract. However, common tpbft
	// is used as a default tpbft in p3Chain, we expect it to
	// collect all kinds of transactions that are not specified.
	// Thus, we just do nothing here.
	// Example to do contract filter is like followed:
	//
	// if tx.Contract != common.StringToAddress("ANY CONTRACT"){return false}
	//
	// Also, the contract filter could be highly customized.

	valid, err := crypto.SignatureValid(tx.Sender, tx.Signature, tx.TxID) //验证交易的数字签名是否正确(签名过程是用私钥对tx.TxID进行签名)
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: TxMatch is failed, Unable to verify the validity of the digital signature, err:%v\n", err)
		return false
	}

	// signature not right
	if !valid {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Tx (%x) is unmathc with txPool (%x), tx is unvalid!\n", tx.TxID, ct.txPool.PoolID)
		return false
	}

	return true
}

// TODO: view changes could be done here
func (ct *Ctpbft) update() {

}

// nonceCount counts the nonce based on the given current version.
// 更新nonce字段,实际上就是记录版本号,每次更新nonce只需要在当前版本号的基础上+1(初次版本号从0开始)
func (ct *Ctpbft) nonceCount(currentVersion common.Hash) uint8 {
	if currentVersion == ct.lastVersion {
		ct.nonce += 1
	} else {
		ct.nonce = 0
	}
	return ct.nonce
}

// parseTXsOrder returns the transactions IDs as the order.
// 获取并返回形参传入的交易集合中所有交易的交易ID
func parseTXsOrder(txs []*eles.Transaction) []common.Hash {
	order := make([]common.Hash, len(txs))
	for i := 0; i < len(order); i++ {
		order[i] = txs[i].TxID
	}
	return order
}

// Loads a consensus round and handle it. HandleRound will parse a PrePrepareMsg
// and check the transaction pool have collected all the chosen transactions.
// If succeed, returns a roundHandler and nil error.
// 根据Pre-Prepare阶段获取的共识消息PrePrepareMsg(主要是基于消息中包含的交易集合)构建一个roundHandler处理器，负责完成prepare阶段、tidy阶段和commit阶段共识
func (ct *Ctpbft) HandleRound(ppm *PrePrepareMsg) (*roundHandler, error) {
	if len(ppm.TxOrder) > ct.maxTransactionNum { //TPBFT协议具有交易的处理上限,超过上限就会退出
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Fail in HandleRound -- over the max number of transactions in one round, current TxSum (%d), MaxTxSum (%d)\n",
			len(ppm.TxOrder), ct.maxTransactionNum)
		return nil, fmt.Errorf("consensus: %v, fail in HandleRound: over the max number of transactions in one round\n", ct.name)
	}
	// for i, v := range ppm.TxOrder {
	// 	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Node (%x) 等待创建roundHandler,pre-prepare有序交易集合第%d条：%x\n", ct.consensusPromoter.SelfNode.NodeID, i, v)
	// }

	chosenTxs, ok := ct.txPool.QuickFindTxs(ppm.TxOrder) //在交易池中快速查询筛选出pre-prepare消息中包含的交易
	// for i, v := range chosenTxs {
	// 	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Node (%x) 等待创建roundHandler,QuickFind的有序交易集合第%d条：%x\n", ct.consensusPromoter.SelfNode.NodeID, i, v)
	// }

	if !ok {
		loglogrus.Log.Warnf("[TPBFT Consensus] Ctpbft failed: Some transactions in PrePrepareMsg cann't be obtained in the transaction pool\n")
		return nil, fmt.Errorf("not all transactions found in pool yet")
	}

	// chosenTxs 存储的是从交易池中获取的按照pre-prepare消息中规定的顺序(ppm.TxOrder)组装好的交易集合
	rh := ct.parseRoundHandler(chosenTxs, ppm.head.RoundID, ppm.head.Sender, ppm.Version, ppm.Nonce)

	// for i, v := range rh.chosenTxs {
	// 	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Node (%x) 生成roundHandler,roundHandler的chosen交易有序集合第%d条：%x:%v\n", ct.consensusPromoter.SelfNode.NodeID, i, v)
	// }

	ct.txPool.RemoveTransactions(chosenTxs) //从交易池中删除对应的交易(已经交由roundHandler处理这些消息)
	return rh, nil
}

// Parses a round handler. This func would do no validation.
func (ct *Ctpbft) parseRoundHandler(chosenTxs []*eles.Transaction, roundID common.Hash, leader common.NodeID, version common.Hash, nonce uint8) *roundHandler {
	liveMemberSnapShot := ct.backSnapShotLiveMembers() //获取组内所有live节点
	//初始化一个 roundHandler,处理一轮(round)底层共识 --> 完成pre-prepare阶段后就需要使用roundHandler进行后续共识阶段
	rh := &roundHandler{
		consensus:      []byte(ct.name), //共识协议Name
		roundID:        roundID,
		chosenTxs:      chosenTxs, //pre-prepare阶段共识消息完成排序
		leaderNode:     leader,
		selfNode:       ct.consensusPromoter.SelfNode.NodeID,
		version:        version,
		nonce:          nonce,
		state:          StatePrepare,       //roundHandler从StatePrepare阶段开始
		expiredTime:    cpbftRoundLifeTime, //一轮交易round的上限时间
		prepareMsgPool: make(map[common.NodeID]*PrepareMsg),
		commitMsgPool:  make(map[common.NodeID]*CommitMsg),

		liveMemberSnapshot: liveMemberSnapShot,
		pbftFValue:         (len(liveMemberSnapShot) - 1) / pbftThresholdFactor,

		transactionValidFunc: ct.ValidTransaction, //用于验证交易合法性的方法
		prepareFunc:          ct.SendPrepareMsg,   //发送PrepareMsg的方法
		prepareCheck:         ct.CheckPrepare,     //验证prepare消息中交易合法性的方法(获取validOrder验证序列)
		tidyFunc:             ct.TidyTransactions, //根据有序交易集合构建区块的方法
		commitFunc:           ct.SendCommitMsg,    //发送CommitMsg的方法
		submitFunc:           ct.SubmitBlock,      //提交区块的方法

		selfDestruct: ct.consensusPromoter.RemoveRoundHandler,
	}
	return rh
}

var (
	newRoundScanCount     = 0
	newRoundScanThreshold = int(coefficient.Promoter_MaxPBFTRoundLaunchInterval / coefficient.Promoter_NewPBFTRoundScanCycle)
)

// launch a new round of consensus
// 启动新一轮共识(只有本组的Leader才能开启新一轮共识round)
func (ct *Ctpbft) NewRound(leader common.NodeID, netID string) {
	chosenTxs := ct.txPool.ChooseTransactions() //在交易池中检索未超时的交易消息

	// no new transaction is collected
	if len(chosenTxs) == 0 { //没有新收集的交易消息，直接退出
		//loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: No transaction without timeout was retrieved from txPool (%x)", ct.txPool.PoolID)
		return
	}

	//loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: 交易池中存在交易,数量为 :%d\n", len(chosenTxs))

	if newRoundScanCount < newRoundScanThreshold && len(chosenTxs) < coefficient.Block_MaxTransactionNum {
		newRoundScanCount += 1
		return
	} else {
		newRoundScanCount = 0
	}
	//loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: 下面要紧Token获取, 开启新一轮......\n")

	// over the max number of parallelled rounds
	if !ct.consensusPromoter.AcquireRoundToken() { //必须事先获取到Token才能进行新一轮共识(最多并行执行maxParallelRound轮共识)
		loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Failed to acquire free Round Token, Please wait!\n")
		return
	}

	//loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: 获取到了Token,下面要发送pre-prepare Msg.........\n")
	// a new round handler should be constructed
	ct.txPool.RemoveTransactions(chosenTxs)

	currentVersion := ct.consensusPromoter.StateManager.CurrentVersion() //获取当前区块版本
	nonce := ct.nonceCount(currentVersion)                               //根据区块版本计算nonce值
	ct.lastVersion = currentVersion

	//需要在节点组内广播发布新的pre-prepare共识消息
	ppm := &PrePrepareMsg{
		head: CommonHead{
			Consensus: []byte(COMMON_TPBFT_NAME),
			TypeCode:  StatePrePrepare,                      //Pre-Prepare共识阶段
			Sender:    ct.consensusPromoter.SelfNode.NodeID, //发送者的NodeID
		},
		Version: currentVersion,
		Nonce:   nonce,
		TxOrder: parseTXsOrder(chosenTxs), //获取chosenTxs中所有交易的交易ID.作为TxOrder
	}
	roudID := ppm.ComputeRoundID() //利用新建的pre-prepare共识消息计算哈希，得到本次共识round的ID
	ct.SendPrePrepareMsg(ppm)      //在组内广播此pre-prepare共识消息

	// for i, v := range ppm.TxOrder {
	// 	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Leader (%x) 等待创建roundHandler,pre-prepare有序交易集合第%d条：%x\n", ct.consensusPromoter.SelfNode.NodeID, i, v)
	// }

	//完成pre-prepare阶段后，注册一个RoundHandler处理器，负责处理prepare阶段、tidy阶段和commit阶段
	rh := ct.parseRoundHandler(chosenTxs, roudID, ct.consensusPromoter.SelfNode.NodeID, currentVersion, nonce)

	// for i, v := range rh.chosenTxs {
	// 	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: Leader (%x) 生成roundHandler,chosen交易有序集合第%d条：%x:%v\n", ct.consensusPromoter.SelfNode.NodeID, i, v)
	// }

	//运行RoundHandler处理器
	go func() { rh.Start(); ct.consensusPromoter.AddRoundHandler(rh) }()
	ct.consensusPromoter.AddRoundHandler(rh)

	loglogrus.Log.Infof("[TPBFT Consensus] Ctpbft: The Leader (%x) of group (%s) broadcast PrePrepareMsg successfully, New RoundHandler has been created (roundID:%x), Start a new round of consensus!\n",
		leader, netID, roudID)

}
