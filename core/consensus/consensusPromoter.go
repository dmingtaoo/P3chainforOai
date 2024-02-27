package consensus

import (
	"crypto/ecdsa"
	"fmt"
	"p3Chain/coefficient"
	"p3Chain/common"
	"p3Chain/core/contract"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/netconfig"
	"p3Chain/core/separator"
	"p3Chain/core/validator"
	"p3Chain/core/worldstate"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"
	"sync"
	"time"
)

var (
	upperTickCycle     = coefficient.Promoter_UpperChannelUpdateScanCycle
	lowerTickCycle     = coefficient.Promoter_LowerChannelUpdateScanCycle
	newRoundCheckCycle = coefficient.Promoter_NewPBFTRoundScanCycle
	MaxParallelRound   = coefficient.Promoter_MaxParallelRoundNum
	updateIntraWSCycle = coefficient.Promoter_IntraWSUpdateScanCycle
)

// ConsensusPromoter follow the workflow of p3Chain to promote the
// whole consensus including the lower one (tpbft) and upper one.
// The promotion is realized by iteratively push transactions into
// transaction pools and collect candidate blocks from other subnet
// leaders.
type ConsensusPromoter struct {
	SelfNode          *dpnet.Node             //Node对象(NodeID/Role/NetID)
	prvKey            *ecdsa.PrivateKey       //私钥
	StateManager      worldstate.StateManager // manage the world states and blocks
	IntraStateManager worldstate.StateManager // manage the lagging world states and blocks
	contractEngine    contract.ContractEngine // the contract engine to execute smart contracts
	lowerChannel      LowerChannel            //separator.PeerGroup 类实现了此接口
	upperChannel      UpperChannel

	// ---------------------------------------------------
	// for lower-level consensus
	installedTpbft   map[string]TPBFT        // the installed TPBFTs   记录所有已安装的trimmed PBFT(一次可以平行运行多个共识协议的交易round)
	transactionPools []*eles.TransactionPool //记录了网络中所有节点的交易池
	poolMu           sync.RWMutex            // sync the installation and pool changes
	tpbftMux         sync.RWMutex            // sync the installed tpbft

	roundCache           map[common.Hash]*roundHandler // the cache to store low-level consensus round
	roundCacheMux        sync.RWMutex                  // sync the roundCache
	roundParalleledToken chan struct{}                 // 容量等于maxParallelRound，Promoter初始化时被填满

	addRoundMux sync.Mutex // sync the add round operation

	// ---------------------------------------------------
	// for upper-level consensus
	validateManager       *validator.ValidateManager // used to validate block
	blockCache            eles.BlockCache
	upperConsensusManager UpperConsensusManager

	LastPreprepareTime time.Time // 记录最近最后一次收到Leader节点pre-prepare Msg的时间

	netManager *netconfig.NetManager

	quit chan struct{}
}

// 初始化创建一个共识Promoter
func NewConsensusPromoter(selfNode *dpnet.Node, prvKey *ecdsa.PrivateKey, stateManager worldstate.StateManager, intraStateManager worldstate.StateManager, netManager *netconfig.NetManager) *ConsensusPromoter {
	cp := &ConsensusPromoter{
		SelfNode:             selfNode, // the self node
		prvKey:               prvKey,
		StateManager:         stateManager,
		IntraStateManager:    intraStateManager,
		installedTpbft:       make(map[string]TPBFT),
		transactionPools:     make([]*eles.TransactionPool, 0),
		roundCache:           make(map[common.Hash]*roundHandler),
		roundParalleledToken: make(chan struct{}, MaxParallelRound),
		netManager:           netManager,
	}
	for i := 0; i < MaxParallelRound; i++ {
		cp.roundParalleledToken <- struct{}{}
	}
	return cp
}

func (cp *ConsensusPromoter) SetUpperConsensusManager(manager UpperConsensusManager) {
	cp.upperConsensusManager = manager
}

func (cp *ConsensusPromoter) SetLowerChannel(lowerChannel LowerChannel) {
	cp.lowerChannel = lowerChannel
}

func (cp *ConsensusPromoter) BackLowerChannel() LowerChannel {
	return cp.lowerChannel
}

func (cp *ConsensusPromoter) SetUpperChannel(upperChannel UpperChannel) {
	cp.upperChannel = upperChannel
}

func (cp *ConsensusPromoter) SetContractEngine(contractEngine contract.ContractEngine) {
	cp.contractEngine = contractEngine
}

func (cp *ConsensusPromoter) SetBlockCache(blockCache eles.BlockCache) {
	cp.blockCache = blockCache
}

func (cp *ConsensusPromoter) SetValidateManager(validateManager *validator.ValidateManager) {
	cp.validateManager = validateManager
}

// 运行 Promoter
func (cp *ConsensusPromoter) Start() {
	if cp.lowerChannel != nil {
		go cp.LowerUpdateLoop() //各节点所属子网内部进入循环更新
	}
	if cp.upperChannel != nil && cp.upperConsensusManager != nil {
		go cp.UpperUpdateLoop() //upper层Leader节点们进入循环更新
	}
}

// 关闭 Promoter
func (cp *ConsensusPromoter) Stop() {
	cp.blockCache.Stop()
	cp.upperConsensusManager.Stop()
	close(cp.quit)
	// TODO: close the transaction pool and the TPBFT also.
	cp.poolMu.Lock()
	defer cp.poolMu.Unlock()
	for _, txpool := range cp.transactionPools {
		txpool.Stop()
	}
}

// 节点组内部的更新循环
func (cp *ConsensusPromoter) LowerUpdateLoop() {

	lowerTicker := time.NewTicker(lowerTickCycle)        //共识更新定时器
	newRoundTicker := time.NewTicker(newRoundCheckCycle) //新round产生定时器
	updateIntraWSTicker := time.NewTicker(updateIntraWSCycle)
	for {
		select {
		case <-lowerTicker.C:
			//loglogrus.Log.Infof("[TPBFT Consensus] Consensus Promoter: 开启一轮 lowerUpdate...........\n")
			cp.lowerUpdate() //子网组内部的消息处理与更新
			//loglogrus.Log.Infof("[TPBFT Consensus] Consensus Promoter: 一轮 lowerUpdate 处理完毕...........\n")
		case <-newRoundTicker.C:
			cp.checkNewRound() //负责由Leader节点为所有运行的TPBFT开启新一轮round
		case <-updateIntraWSTicker.C:
			// use goroutine, otherwise it will block the lowerUpdate
			go cp.updateIntraWorldState() //负责更新片内的滞后版本世界状态
		case <-cp.quit:
			lowerTicker.Stop()
			newRoundTicker.Stop()
			return
		}
	}
}

// upper层Leader节点之间的更新循环
func (cp *ConsensusPromoter) UpperUpdateLoop() {
	upperTicker := time.NewTicker(upperTickCycle) //共识更新定时器
	for {
		select {
		case <-upperTicker.C:
			cp.upperUpdate() //upper层Leader节点内部的消息处理与更新
		case <-cp.quit:
			upperTicker.Stop()
			return
		}
	}
}

// The lowerUpdate scans the current messages. Retrieves transactions and puts them into
// matched transaction pools. Retrieves lower-level consensus messages and realizes them
// to help consensus. Retrieves net control messages to do view change.
// 扫描当前子网内的消息.
// 1.检索接收到的交易并将它们加入到匹配的交易池中
// 2.检索组内共识信息，帮助实现共识过程(主要是处理pre-prepare消息,其他阶段消息由roundHandler处理)
// 3.检索网络控制信息实现视图切换
func (cp *ConsensusPromoter) lowerUpdate() {
	if cp.lowerChannel == nil {
		loglogrus.Log.Errorf("[TPBFT Consensus] Consensus Promoter failed: Current Node (%x) hasn't setup ConsensusPromoter.lowerChannel, it's unabled to reach Lower Channel consensus!\n", cp.SelfNode)
		return
	}
	currentVersion := cp.StateManager.CurrentVersion() //获取当前区块链的版本(也就是区块链的哈希)
	currentMsgs := cp.lowerChannel.CurrentMsgs()       //获取当前节点本组内(低共识层)的消息(从组内消息池获取)
	toMark := make([]common.Hash, 0)                   //消息已读队列(记录已读消息Hash值)

	mark := func(msgHash common.Hash) { //函数变量，负责将指定msgHash加入到toMark切片中
		toMark = append(toMark, msgHash)
	}

	cp.poolMu.RLock()
	defer cp.poolMu.RUnlock()

	cp.tpbftMux.RLock()
	defer cp.tpbftMux.RUnlock()

	for _, msg := range currentMsgs { //对接收到的消息(组内消息)进行处理
		switch {
		case msg.IsIntraWorldStateUpdate():
			//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: processing IntraWSUpdate message...........\n")
			mark(msg.CalculateHash())
			version := common.BytesToHash(msg.PayLoad)
			if version == cp.IntraStateManager.CurrentVersion() {
				continue
			}
			err := cp.IntraStateManager.UpdateWorldStateToVersion(version)
			if err != nil {
				loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to update intra-shard world state, err:%v\n", err)
			}
			loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Current Consensus Node (%x) has receive IntraWSUpdate message from leader, and updates to version %s\n", cp.SelfNode.NodeID, version.Hex())
		case msg.IsTransaction(): //交易消息
			//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: 处理单交易消息...........\n")
			mark(msg.CalculateHash())                      //记录已读标记(加入toMark队列)
			tx, err := DeserializeTransaction(msg.PayLoad) //从msg中解包出交易的信息(Transaction对象)
			if err != nil {
				loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to Deserialize separator.Message into Transaction Msg, err:%v\n", err)
				continue
			}
			for _, txpool := range cp.transactionPools { //遍历所有的交易池
				if txpool.Match(tx) { //查询此交易匹配的交易池
					if err := txpool.Add(tx); err != nil { //将交易加入匹配的交易池中
						loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Transaction Msg is nil!")
					}
				}
			}
			loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Current Consensus Node (%x) has receive transaction (count:%d) from other consensus node (%x)\n", cp.SelfNode.NodeID, 1, tx.Sender)
		case msg.IsTransactions():
			//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: 处理多交易消息...........\n")
			mark(msg.CalculateHash())
			txs, err := DeserializeTransactionsSeries(msg.PayLoad)
			if err != nil {
				loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to Deserialize separator.Message into Transaction Msg, err:%v\n", err)
				continue
			}
			// for i, tx := range txs {
			// 	loglogrus.Log.Infof("解包获取的第%d笔交易%x\n", i, tx.TxID)
			// }
			for i := 0; i < len(txs); i++ {
				tx := txs[i]
				for _, txpool := range cp.transactionPools { //遍历所有的交易池
					if txpool.Match(&tx) { //查询此交易匹配的交易池
						if err := txpool.Add(&tx); err != nil { //将交易加入匹配的交易池中
							loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: transaction pool: %v fail in add transaction: %v, %v", txpool.PoolID, tx.TxID, err)
						}
						//break
					}

				}
			}
			loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Current Consensus Node (%x) has receive transaction (count:%d) from other consensus node (%x) \n", cp.SelfNode.NodeID, len(txs), txs[0].Sender)

		case msg.IsLowerConsensus(): //共识消息
			//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: 处理共识消息...........\n")
			wlcm, err := DeserializeLowerConsensusMsg(msg.PayLoad) //解包为WrappedLowerConsensusMessage 统一格式消息
			if err != nil {                                        //解包失败
				mark(msg.CalculateHash()) //标记已读
				loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to Deserialize separator.Message into Lower Consensus Msg, err:%v\n", err)
				continue
			}
			if msg.From != wlcm.Head.Sender { //判断来源是否可靠
				mark(msg.CalculateHash()) //不可靠也要标记已读
				loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Deserialized Lower Consensus Msg -- msg.From (%x) could not match the sender (%x)\n", msg.From, wlcm.Head.Sender)
				continue
			}
			cp.roundCacheMux.RLock()
			rh, ok := cp.roundCache[wlcm.Head.RoundID] //根据收到的共识消息在本地查询其对应的roundHandler处理器
			cp.roundCacheMux.RUnlock()
			if !ok { //没能查询到对应的roundHandler处理器
				if wlcm.Head.TypeCode != StatePrePrepare { //判断消息是否为Pre-Prepare阶段共识消息
					loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus failed: Leader doesn't have corresponding roundHandler, Cann't process messages of non-StatePrePrepare Msg!\n")
				} else { //如果是的话

					loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Current Consensus Node (%x) has receive pre-prepare Message from Leader (%x) within group (%s)\n",
						cp.SelfNode.NodeID, wlcm.Head.Sender, cp.SelfNode.NetID)

					loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Leader doesn't have corresponding roundHandler, Waiting for creation....\n")

					// try to load a new round of consensus
					tpbft, ok := cp.installedTpbft[string(wlcm.Head.Consensus)] //根据共识消息查询其对应的共识协议
					if !ok {                                                    //未查询到
						loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Uninstalled pbft protocol (%s)\n", wlcm.Head.Consensus)
						continue //退出当前switch，处理下一条msg
					} else { //在本地查询到了对应的共识协议
						unwrapMsg, err := wlcm.UnWrap() //WrappedLowerConsensusMessage解包为对于阶段共识消息
						if err != nil {
							loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to unwarp WrappedLowerConsensusMessage into LowerConsensusMessage: %v\n", err)
							continue
						}
						ppm, ok := unwrapMsg.(*PrePrepareMsg) //类型断言为PrePrepareMsg
						if !ok {
							loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to assert LowerConsensusMessage into PrePrepareMsg\n")
							continue
						}
						// As roundID should be always right, we place this check at first
						// to improve the system efficiency.
						if !ppm.ValidateIntegrity() { //验证PrePrepareMsg共识消息的有效性(交易round的ID必须正确)
							loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: The roundID (%x) of pre-prepare message is mismatching with it MsgHash (%x)\n",
								ppm.head.RoundID, ppm.Hash())
							continue
						}

						if !cp.AcquireRoundToken() { //必须获取一个token才能继续执行(否则本次round结束)
							loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Couldn't get round token, so roundHandler cann't be created!\n")
							continue
						}

						newRoundHandler, err := tpbft.HandleRound(ppm) //基于pre-prepare阶段共识消息ppm构建新的roundHandler处理器
						if err != nil {
							loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Unable to create roundHandler based on pre-prepare message, err:%v\n", err)
							cp.ReleaseRoundToken()
							continue
						}
						mark(msg.CalculateHash()) //msg信号打上已读标签

						cp.LastPreprepareTime = time.Now()

						go func() {
							newRoundHandler.SendPrepareMsg(currentVersion) //在RoundHandler处理器运行之前，向其他节点广播PrepareMsg
							newRoundHandler.Start()                        //RoundHandler处理器开始运行,完成后续共识阶段
							cp.AddRoundHandler(newRoundHandler)            //promoter与RoundHandler相绑定

							newRoundHandler.markRequire() // 接收到了leader的pre-prepare Msg
						}()
						loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: The new roundHandler (roundID:%x) has been created and being running\n", newRoundHandler.roundID)
					}
				}
				continue
			}
			//如果查询到对应的roundHandler已经存在
			mark(msg.CalculateHash())
			if rh.processMsg(wlcm) { //解包并加入到对应消息池中

				rh.markRequire() //激励update协程调用Do函数进行处理
			}

		case msg.IsControl(): // TODO: complete this to realize view change.
		default:
			loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: UnKnown kind of separator.Message, Msg Code: %v\n", msg.MsgCode)
			continue
		}

	}
	cp.lowerChannel.MarkRetrievedMsgs(toMark) //separator协议的消息池中为以上已读的消息打上已读标签
}

// 启动新一轮共识round
func (cp *ConsensusPromoter) checkNewRound() {
	// only leader could launch new round of lower-level consensus.
	if cp.SelfNode.Role != dpnet.Leader { //只有一组的Leader阶段可以开启新一轮round
		return
	}
	//loglogrus.Log.Infof("[TPBFT Consensus] Consensus Promoter: 这里是Leader节点,检查是否能开始新一轮共识round.....\n")
	cp.tpbftMux.RLock()
	defer cp.tpbftMux.RUnlock()

	if len(cp.installedTpbft) == 0 {
		loglogrus.Log.Warnf("[TPBFT Consensus] Consensus Promoter failed: No PBFT consensus protocal is currently running!\n")
		return
	}
	//loglogrus.Log.Infof("[TPBFT Consensus] Consensus Promoter: 这里是Leader节点,为所有注册的共识协议开始新一轮共识round.....\n")
	for _, tpbft := range cp.installedTpbft { //为每一个注册的tpbft协议开启新一轮共识round(最多并行开启maxParallelRound个共识round)
		tpbft.NewRound(cp.SelfNode.NodeID, cp.SelfNode.NetID)
	}
}

// need to use with addRoundMux
func (cp *ConsensusPromoter) waitAllRoundFinish() {

}

// update the intra-shard world state to the newest version
func (cp *ConsensusPromoter) updateIntraWorldState() {
	// only leader could launch an intra-shard world state update.
	if cp.SelfNode.Role != dpnet.Leader {
		return
	}

	// stop adding new round and wait for all the rounds to finish, then update the intra-shard world state
	cp.addRoundMux.Lock()
	defer cp.addRoundMux.Unlock()
	checkTicker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-checkTicker.C:
			break
		}
		if len(cp.roundCache) == 0 {
			checkTicker.Stop()
			break
		}
	}

	newestVersion := cp.StateManager.CurrentVersion()
	// if the newest version is empty, do nothing
	if common.EmptyHash(newestVersion) {
		return
	}

	iwsVerson := cp.IntraStateManager.CurrentVersion()
	// if the intra-shard world state version isn't equal to the whole network world state version, update it
	if newestVersion != iwsVerson {
		err := cp.IntraStateManager.UpdateWorldStateToVersion(newestVersion)
		if err != nil {
			loglogrus.Log.Warnf("[TPBFT Consensus] Consensus Promoter failed: Fail to update intra world state, err:%v\n", err)
			return
		}
		// loglogrus.Log.Infof("[TPBFT Consensus] Consensus Promoter: leader update intra-shard world state version to %s\n", newestVersion.Hex())
	}
	// construct intra-shard world state update message and broadcast it in the lower channel (intra-shard)
	iwsUpdateMsg := cp.lowerChannel.NewIntraWorldStateUpdateMessage(cp.SelfNode.NodeID, newestVersion[:])
	iwsUpdateMsg.CalculateHash()
	cp.lowerChannel.MsgBroadcast(iwsUpdateMsg)

	loglogrus.Log.Infof("[TPBFT Consensus] Consensus Promoter: leader launch an intra-shard world state update to version %s, old version: %s\n", newestVersion.Hex(), iwsVerson.Hex())
	// TODO: wait for the response from followers to make sure their intra-shard world state update is done.
	// normally, if the network is stable, no need to do this TODO because the followers have enough time to update their intra-shard world state before next round reads it.
}

// acquire a token to handle a lower-level consensus round
// 为本次共识round获取token(从roundParalleledToken管道取出一个信号)
// 初始化时会填满roundParalleledToken个token信号
func (cp *ConsensusPromoter) AcquireRoundToken() bool {
	select {
	case <-cp.roundParalleledToken:
		return true
	default:
		return false
	}
}

// release a token
// 完成了一次共识round,重新归还一个token
func (cp *ConsensusPromoter) ReleaseRoundToken() {
	cp.roundParalleledToken <- struct{}{}
}

// Add a roundHandler into the round cache.
// NOTE: should first acquire round token to limit the paralleled rounds
// 绑定一个已有的roundHandler处理器到roundCache字段(roundID作为key值-->每轮round只能有一个roundHandler处理器)
func (cp *ConsensusPromoter) AddRoundHandler(rh *roundHandler) {
	// when updating the intra-shard world state, no new round should be added.
	cp.addRoundMux.Lock()
	defer cp.addRoundMux.Unlock()
	cp.roundCacheMux.Lock()
	defer cp.roundCacheMux.Unlock()
	cp.roundCache[rh.roundID] = rh
}

// Delete the round handler from the round cache.
// NOTE: ReleaseRoundToken is called when succeed in deleting.
// 卸载一个指定roundID的roundHandler处理器,同时调用ReleaseRoundToken() 恢复一个token
func (cp *ConsensusPromoter) RemoveRoundHandler(roundID common.Hash) {
	cp.roundCacheMux.Lock()
	defer cp.roundCacheMux.Unlock()
	if _, ok := cp.roundCache[roundID]; ok {
		delete(cp.roundCache, roundID)
		cp.ReleaseRoundToken()
	} else {
		// once roundID could not been found in roundCache, a token is permanently lost.
		// make sure this condition would never happen.
		loglogrus.Log.Errorf("Consensus Promoter failed: Couldn't find roundID: %x when deleting roundHandler, A token is permanently lost!\n", roundID)
	}
}

// upperUpdate retrieves messages in upper level consensus channel, read and process them.
// At this time, upperUpdate is designed to help use the order service provided by probably
// a booter.
// TODO: A upper level consensus without third party should be realized as an optional choice.
func (cp *ConsensusPromoter) upperUpdate() {
	if cp.upperChannel == nil {
		loglogrus.Log.Errorf("Consensus Promoter failed: Current Node (%x) hasn't setup ConsensusPromoter.upperChannel, it's unabled to reach Upper Channel consensus!\n", cp.SelfNode)
		return
	}
	currentMsgs := cp.upperChannel.CurrentMsgs()
	toMark := make([]common.Hash, 0)
	for _, msg := range currentMsgs {
		toMark = append(toMark, msg.CalculateHash())
		cp.upperConsensusManager.ProcessMessage(msg)
	}
	cp.upperChannel.MarkRetrievedMsgs(toMark)
}

// AddTransactions add the self published transactions into the
// transaction pool.
// NOTE: No more validation should be done in this function, as it
// call the Match function.
// 遍历形参指定的交易集合中的每一条交易，将其加入到与其相匹配的交易池中
func (cp *ConsensusPromoter) AddTransations(txs []*eles.Transaction) {
	cp.poolMu.RLock()
	defer cp.poolMu.RUnlock()
	for _, tx := range txs { //遍历形参指定的交易集合
		for _, txpool := range cp.transactionPools { //遍历本地所有的交易池
			if txpool.Match(tx) { //查询与当前交易tx匹配的交易池
				if err := txpool.Add(tx); err != nil { //将交易tx加入到与其匹配的交易池中
					loglogrus.Log.Warnf("Consensus Promoter failed: Transaction pool: %v fail in add transaction: %x ,err:%v\n", txpool.PoolID, tx.TxID, err)
				}
			}
		}
	}
}

// add transactions locally and broadcast them in the lower consensus channel
func (cp *ConsensusPromoter) PublishTransactions(txs []*eles.Transaction) {
	if len(txs) == 0 {
		return
	}
	cp.AddTransations(txs)
	left := len(txs)
	startIdx := 0
	fracNum := coefficient.Separator_MaxTransactionSeriesNum
	for left > 0 {
		if left <= fracNum {
			temp := txs[startIdx : startIdx+left]
			sprMsg, err := cp.WrapTxsMsg(temp)
			if err != nil {
				loglogrus.Log.Warnf("[TPBFT Consensus] fail in PublishTransactions, %v", err)
				continue
			}
			cp.lowerChannel.MsgBroadcast(sprMsg)
			left = 0
			break
		} else {
			temp := txs[startIdx : startIdx+fracNum]
			sprMsg, err := cp.WrapTxsMsg(temp)
			if err != nil {
				loglogrus.Log.Warnf("[TPBFT Consensus] fail in PublishTransactions, %v", err)
				continue
			}
			cp.lowerChannel.MsgBroadcast(sprMsg)
			startIdx += fracNum
			left -= fracNum
		}
	}
	loglogrus.Log.Infof("[TPBFT Consensus] Current Consensus Node (%x) has published new transactions (count:%d) and broadcast within lower group (%v)!\n",
		cp.SelfNode.NodeID, len(txs), cp.SelfNode.NetID)
}

func (cp *ConsensusPromoter) WrapTxsMsg(txs []*eles.Transaction) (*separator.Message, error) {
	txg := NewTransactionGroup(txs)
	payload, err := rlp.EncodeToBytes(txg)
	if err != nil {
		return nil, fmt.Errorf("fail in WrapTxMsg, %v", err)
	}
	msgcode := separator.TransactionsCode
	msg := &separator.Message{
		MsgCode: uint64(msgcode),    //消息类型为transactionCode
		NetID:   cp.SelfNode.NetID,  //目标子网
		From:    cp.SelfNode.NodeID, //来源,也就是当前节点的NodeID
		PayLoad: payload,            //交易
	}
	msg.CalculateHash()
	return msg, err
}

func (cp *ConsensusPromoter) WrapTxMsg(tx *eles.Transaction) (*separator.Message, error) {
	payload, err := rlp.EncodeToBytes(tx) //编码交易为msg消息的payload
	if err != nil {
		loglogrus.Log.Warnf("Consensus Promoter failed: WrapTxMsg is failed, err:%v\n", err)
		return nil, fmt.Errorf("fail in WrapTxMsg, %v", err)
	}
	msgcode := separator.TransactionCode
	msg := &separator.Message{
		MsgCode: uint64(msgcode),    //消息类型为transactionCode
		NetID:   cp.SelfNode.NetID,  //目标子网
		From:    cp.SelfNode.NodeID, //来源,也就是当前节点的NodeID
		PayLoad: payload,            //交易
	}
	msg.CalculateHash()
	return msg, err
}

func (cp *ConsensusPromoter) BackUpperChannel() UpperChannel {
	return cp.upperChannel
}

func (cp *ConsensusPromoter) BackPrvKey() *ecdsa.PrivateKey {
	return cp.prvKey
}

func (cp *ConsensusPromoter) BackBlockCache() eles.BlockCache {
	return cp.blockCache
}

func (cp *ConsensusPromoter) BackValidateManager() *validator.ValidateManager {
	return cp.validateManager
}

func (cp *ConsensusPromoter) BackContractEngine() contract.ContractEngine {
	return cp.contractEngine
}

func (cp *ConsensusPromoter) BackNetManager() *netconfig.NetManager {
	return cp.netManager
}

func (cp *ConsensusPromoter) BackRoundCacheMutex() *sync.RWMutex {
	return &cp.roundCacheMux
}

func (cp *ConsensusPromoter) BackRoundCache() map[common.Hash]*roundHandler {
	return cp.roundCache
}

func (cp *ConsensusPromoter) BackRoundParalleledToken() chan struct{} {
	return cp.roundParalleledToken
}

func (cp *ConsensusPromoter) BackTargetTPBFT(name string) TPBFT {
	return cp.installedTpbft[name]
}
