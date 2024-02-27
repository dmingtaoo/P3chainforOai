package consensus

import (
	"p3Chain/common"
	"p3Chain/core/eles"
	loglogrus "p3Chain/log_logrus"
	"sync"
	"time"
)

// a round handler handles a round of lower-level consensus, it is
// usually running in an alone goroutine.
//
// all round handlers start from the state statePrepare
// 一个 roundHandler 对象负责处理一轮底层共识(子网组内共识)，需要由一个单独的协程负责运行
// 所有的 roundHandler 对象都是从statePrepare状态开始运行的
type roundHandler struct {
	// round info
	consensus  []byte              //所用的共识协议
	roundID    common.Hash         //本次共识round的ID值
	chosenTxs  []*eles.Transaction // the chosen transactions and the order.  符合条件的有序交易集合
	leaderNode common.NodeID       //组内Leader节点的ID
	selfNode   common.NodeID       //当前节点的ID
	version    common.Hash
	nonce      uint8

	state       uint8 //记录共识阶段状态
	expiredTime time.Duration

	liveMemberSnapshot map[common.NodeID]bool //组内节点活跃状态记录(活跃为true,下线为false)
	pbftFValue         int

	prepareMsgPool map[common.NodeID]*PrepareMsg //prepare阶段的消息池
	commitMsgPool  map[common.NodeID]*CommitMsg  //commit阶段消息池
	pmMux          sync.RWMutex
	cmMux          sync.RWMutex

	validOrder      []byte      //prepare阶段交易投票序列
	tidyResult      []byte      //tidy阶段获取的区块ID
	resultSignature []byte      //对上述区块ID的数字签名
	candidateBlock  *eles.Block //候选区块(commit阶段需要提交的)

	// 1.当前节点判断交易列表中交易是否合法(合法的交易validOrder中对应下表为1，否则为0)--获取交易有效验证序列
	transactionValidFunc func(txs []*eles.Transaction, version common.Hash) (validOrder []byte) // vote whether the chosen transactions are valid
	// 2.让当前节点在roundhandler启动之前发送自身的Prepare Msg
	prepareFunc func(*PrepareMsg) // send a prepare message before round handler starts
	// 3.从一组prepare共识消息中，找出合法有效的交易验证序列validOrder(也就是大多数prepareMsg都认为有效的validOrder)
	prepareCheck func(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*PrepareMsg) (ok bool, validOrder []byte) // check whether collects enough prepare messages from others could go to the tidy stage
	// 4.tidy处理函数(根据交易构建区块)
	tidyFunc func(leader common.NodeID, version common.Hash, nonce uint8, txs []*eles.Transaction, validOrder []byte) (result []byte, signature []byte, block *eles.Block) // the tidy function
	// 5.发送commit消息
	commitFunc func(cm *CommitMsg, leader common.NodeID) // send commit message
	// 6.检查区块是否可以提交，若可以则组装区块并提交
	submitFunc func(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*CommitMsg, block *eles.Block) bool // check whether could submit a block, if ok , construct a block and submit

	// 7.roundHandler自毁函数(调用token释放函数)，受consensus promoter监督
	selfDestruct func(roundID common.Hash) // to call token release func and be no more supervised by consensus promoter
	quit         chan struct{}             // quit channel
	quitOnce     sync.Once                 // to avoid repeated exits

	updateRequire chan struct{} // to infer whether should do update

}

// 在 prepare 阶段消息池中添加指定PrepareMsg，pm.head.Sender(发送者NodeID)为key值
func (rh *roundHandler) addPrepareMsg(pm *PrepareMsg) {
	rh.pmMux.Lock()
	defer rh.pmMux.Unlock()
	rh.prepareMsgPool[pm.head.Sender] = pm
}

// 在 commit 阶段消息池中添加指定CommitMsg，cm.head.Sender为ket值
func (rh *roundHandler) addCommitMsg(cm *CommitMsg) {
	rh.cmMux.Lock()
	defer rh.cmMux.Unlock()

	rh.commitMsgPool[cm.head.Sender] = cm
}

// processMsg process a WrappedLowerConsensusMessage, return whether the state of
// roundHandler is changed (or say whether a update is needed.)
// 对接收的WrappedLowerConsensusMessage统一格式消息进行解包,解包为相应共识阶段消息包,然后加入到对应阶段的消息池中
func (rh *roundHandler) processMsg(wlcm *WrappedLowerConsensusMessage) (flag bool) {
	unwrap, err := wlcm.UnWrap() //解包为对应阶段共识消息
	if err != nil {
		loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to unwarp WrappedLowerConsensusMessage into LowerConsensusMessage: %v\n", err)
		return
	}
	switch wlcm.Head.TypeCode {
	case StatePrepare: //prepare阶段
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: 共识消息类型为Prepare Msg...........\n")
		if rh.state != StatePrepare { //roundHandler只能在Prepare阶段处理prepare消息
			return
		}
		pm, ok := unwrap.(*PrepareMsg)
		if !ok {
			loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to assert LowerConsensusMessage into PrepareMsg\n")
			return
		}

		//检查当前roundHandler保存的交易数与收到的prepare消息中的交易数是否一致
		if len(rh.chosenTxs) != len(pm.ValidOrder) {
			loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: The Number of validated transactions include in PrepareMsg isn't match with PrepareMsg\n")
			return
		}
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: 准备将Prepare Msg放入消息池...........\n")
		//符合数量条件的prepare消息加入到 prepare 阶段消息池中
		rh.addPrepareMsg(pm)

		flag = true //标志位为true

		// TODO:此处的日志不是Leader，而是当前节点
		loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Node (%x) receive validated PrepareMsg from Node (%x) and has added it into roundHandler.prepareMsgPool\n",
			rh.selfNode, pm.head.Sender)

	case StateCommit: //commit阶段
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: 共识消息类型为Commit Msg...........\n")
		switch rh.state { //查看当前共识阶段(prepare阶段和commit阶段都可以接收commit消息)
		case StatePrepare:
		case StateCommit:
		default:
			//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Commit Msg, 共识阶段不匹配...........\n")
			return
		}
		cm, ok := unwrap.(*CommitMsg)
		if !ok {
			loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: Fail to assert LowerConsensusMessage into CommitMsg\n")
			return
		}
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: 准备将Commit Msg放入消息池...........\n")
		rh.addCommitMsg(cm) //添加到commit阶段消息池
		flag = true

		loglogrus.Log.Infof("[TPBFT Consensus] Lower Consensus: Leader (%x) receive CommitMsg from Node (%x) and has added it into roundHandler.commitMsgPool\n",
			rh.leaderNode, cm.head.Sender)

	default:
		loglogrus.Log.Warnf("[TPBFT Consensus] Lower Consensus failed: UnKnown kind of Lower Consensus Message, Type Code: %v\n", wlcm.Head.TypeCode)
	}
	return
}

// markRequire marks the updateRequire to notify the roundHandler
// do update.
// 调用此函数，向updateRequire管道发送信号，使得update所在协程执行Do方法
func (rh *roundHandler) markRequire() {
	//loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: 进入markRequire函数,等待向管道rh.updateRequire发送信号..............\n")
	select {
	case rh.updateRequire <- struct{}{}:
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: 进入markRequire函数,rh.updateRequire信号发送成功..............\n")
		return
	default:
		return
	}
}

// 启动 roundHandler 对象
func (rh *roundHandler) Start() {
	rh.updateRequire = make(chan struct{}, 1)
	rh.quit = make(chan struct{})
	loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: roudHandler with round ID: %x starts to running!\n", rh.roundID)
	go rh.update()
}

func (rh *roundHandler) Stop() {
	rh.quitOnce.Do(func() { //quitOnce保证匿名函数仅被执行一次
		close(rh.quit) //关闭管道，通知update结束
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: roundHandler 正在等待自毁.............\n")
		rh.selfDestruct(rh.roundID) //销毁指定ID的roundHandler
		loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: roudHandler with round ID: %x stop!\n", rh.roundID)
	})
}

func (rh *roundHandler) update() {
	expired := time.NewTimer(rh.expiredTime)
	for {
		select {
		case <-rh.updateRequire: //情况一：markRequire()方法被调用
			rh.Do()
		case <-rh.quit: //情况二：quit管道关闭，退出update
			return
		case <-expired.C: //情况三: 到达expired时限后，主动调用stop结束本次roundHandler
			rh.Stop()
		}
	}
}

func (rh *roundHandler) Do() {
	switch rh.state { //判断当前共识阶段
	case StatePrepare: //1. prepare阶段
		rh.pmMux.RLock()
		defer rh.pmMux.RUnlock()

		ok, validOrder := rh.prepareCheck(rh.liveMemberSnapshot, rh.pbftFValue, rh.prepareMsgPool) //从一组prepare消息中获取唯一的ValidOrder序列
		if ok {
			rh.validOrder = validOrder //从prepare消息中获取validOrder序列
			rh.state = StateTidy       //更新共识阶段状态为tidy阶段
			rh.markRequire()           //通知roundHandler执行下一次Do方法

			loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: Node (%x) has completed the prepare phase consensus, validOrder:%v\n", rh.selfNode, validOrder)
		}

	case StateTidy: //2.tidy阶段
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: Node (%x) 进入tidy阶段,计算参量 ----- rh.leaderNode (%x),rh.version (%x), rh.nonce (%d), rh.validOrder(%v)\n",
		//rh.selfNode, rh.leaderNode, rh.version, rh.nonce, rh.validOrder)
		// for i, v := range rh.chosenTxs {
		// 	loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: Node (%x) 进入tidy阶段,计算参量 ----- 第%d笔交易: %x\n",
		// 		rh.selfNode, i, v.TxID)
		// }

		rh.tidyResult, rh.resultSignature, rh.candidateBlock = rh.tidyFunc(rh.leaderNode, rh.version, rh.nonce, rh.chosenTxs, rh.validOrder) // TODO:Leader和其他Follower计算出来的结果不同
		//loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: Node (%x) tidy阶段, rh.tidyResult: %x ,rh.candidateBlock.BlockID:%x\n", rh.selfNode, rh.tidyResult, rh.candidateBlock.BlockID)

		if rh.candidateBlock == nil {
			loglogrus.Log.Warnf("[TPBFT Consensus] Lower Channel failed: Couldn't tidy roundHandler.chosenTxs into roundHandler.candidateBlock!\n")
			rh.state = StateCommit //状态变更为commit
			rh.markRequire()       //通知roundHandler执行下一次Do方法
			return
		}

		//根据tidy阶段获取的信息，组装commit消息
		commitMsg := &CommitMsg{
			head: CommonHead{
				Consensus: rh.consensus,
				RoundID:   rh.roundID,
				TypeCode:  StateCommit,
				Sender:    rh.selfNode,
			},
			Result:    rh.tidyResult,      //要提交的区块的区块ID
			Signature: rh.resultSignature, //发送commit共识消息的节点的数字签名
		}

		// rh.addCommitMsg(commitMsg)
		rh.commitFunc(commitMsg, rh.leaderNode) //发送commit消息
		rh.state = StateCommit                  //状态变更为commit
		rh.markRequire()                        //通知roundHandler执行下一次Do方法

		loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: Node (%x) has completed the tidy phase consensus and has sent CommitMsg to the Leader (%x)\n", rh.selfNode, rh.leaderNode)

	case StateCommit: //3.commit阶段
		if rh.leaderNode != rh.selfNode { //只有Leader节点才能执行commit阶段任务
			rh.Stop()
			rh.state = StateOver
			return
		}
		// only leader should do following
		rh.cmMux.RLock()
		defer rh.cmMux.RUnlock()
		//检查区块是否可以提交到整个dpnet网络

		if len(rh.commitMsgPool) < 2*rh.pbftFValue {
			return
		}

		if rh.submitFunc(rh.liveMemberSnapshot, rh.pbftFValue, rh.commitMsgPool, rh.candidateBlock) {

			rh.Stop()
			rh.state = StateOver

			loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: Leader (%x) has completed the commit phase consensus, Complete LowerChannel consensus\n", rh.selfNode)
		}
	}
}

// Do transaction valid before send the prepare message.
// NOTE: As only follower would call this func to send prepare message,
// a prepare message of itself should be add to roundHandler
// 在发送prepare阶段共识消息之前需要进行交易的有效性验证,获取validOrder序列(根据pre-prepare阶段获取的交易顺序集合)
// 将组装好的 prepare 消息 加入消息池并发送出去
func (rh *roundHandler) SendPrepareMsg(currentVersion common.Hash) {
	//1. roundHandler 判断当前节点的交易集合中交易的合法情况，合法交易在validOrder序列中对于bit为1，否则为0
	validOrder := rh.transactionValidFunc(rh.chosenTxs, rh.version)
	//2. 组装prepare消息
	pm := &PrepareMsg{
		head: CommonHead{
			Consensus: rh.consensus,
			RoundID:   rh.roundID,
			TypeCode:  StatePrepare,
			Sender:    rh.selfNode,
		},
		ValidOrder: validOrder, //有效验证序列
		Version:    currentVersion,
	}
	rh.addPrepareMsg(pm) //将prepare消息加入prepare阶段消息池
	rh.prepareFunc(pm)   //在roundHandler启动之前发送prepare消息

	loglogrus.Log.Infof("[TPBFT Consensus] Lower Channel: Current Consensus Node (%x) has broadcast prepareMsg within roundID (%x)\n", rh.selfNode, rh.roundID)
}
