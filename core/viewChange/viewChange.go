package viewChange

import (
	"context"
	"fmt"
	"math/rand"
	"p3Chain/blockSync"
	"p3Chain/common"
	"p3Chain/core/blockOrdering"
	"p3Chain/core/consensus"
	"p3Chain/core/dpnet"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"
	"reflect"
	"sort"
	"sync"
	"time"
)

var (
	viewChange_msgScanInterval    = 3 * time.Second
	viewChange_msgExpireInterval  = 5 * time.Second
	viewChange_liveCheckInterval  = 15 * time.Second // 发送心跳包的时间间隔
	viewChange_responseThreshould = 60 * time.Second
)

// 1.viewChange模块必须持续运行,Follower通过此模块检测Leader节点是否在正常工作;Leader通过此模块检查自己是否还是本分区的Leader(长时间下线后又重新上线)
// 2.在进行View Change期间，当前分区不会进行区块的提交，不会进行PBFT共识。可以接收交易消息，但是需要存储到缓存池中。
// 3.
type ViewChange struct {
	consensusPromoter *consensus.ConsensusPromoter

	syncProtocal *blockSync.SyncProtocal

	running bool // 是否进行Leader节点切换

	responseThreshould time.Duration // 超过此时间阈值没有获得Leader/Follower节点的响应，发起一次LeaderChange

	subNetLiver      map[common.NodeID]bool // 存储当前分区的所有节点的NodeID,以及其是否正在运行（可能需要通过keep Alive包不断更新此map）
	subNetLiverMutex sync.RWMutex

	liveCheckInterval time.Duration // 心跳包发送间隔

	msgScanInterval   time.Duration // 消息扫描协程的扫描周期
	msgExpireInterval time.Duration // 删除过期消息的扫描周期

	oldLeader common.NodeID // 上一轮的leader
	leader    common.NodeID // 记录当前分区的leader

	pbftFactor int // pbft系数

	tpbftProtocal consensus.TPBFT // 当前在哪一个TPBFT协议上运行viewChange

	candidateLeader      []common.NodeID // 候选Leader节点
	candidateLeaderMutex sync.RWMutex

	heartBeatPool      map[common.NodeID]*consensus.HeartBeat // 存放来自组内各节点的心跳包(key值为对方节点的NodeID)
	heartBeatPoolMutex sync.RWMutex

	viewChangeMsgPool      map[common.NodeID]*consensus.ViewChangeMsg // 存放来自各个节点的view-change消息
	viewChangeMsgPoolMutex sync.RWMutex

	sequence                   uint64
	confirmLeaderRespPool      map[common.NodeID]*consensus.ConfirmLeaderResp // Leader节点用来存放接收到的 leader确认消息 回应
	confirmLeaderRespPoolMutex sync.RWMutex

	// newViewMsgPool      map[common.NodeID]*NewViewMsg // 存放来自新Leader节点的New-View Msg
	// newViewMsgPoolMutex sync.RWMutex

	close chan bool
}

func NewViewChange(consensusPromoter *consensus.ConsensusPromoter, leader common.NodeID, tpbftProtocal consensus.TPBFT, syncProtocal *blockSync.SyncProtocal) *ViewChange {
	vc := new(ViewChange)
	vc.consensusPromoter = consensusPromoter
	vc.syncProtocal = syncProtocal
	vc.running = false

	initialSet := vc.consensusPromoter.BackNetManager().BackViewNet().BackSubnetNodesID(common.NodeID{}, vc.consensusPromoter.SelfNode.NetID)

	vc.subNetLiver = make(map[common.NodeID]bool)
	for _, nodeID := range initialSet {
		vc.AddSubNetLive(nodeID)
	}
	vc.leader = leader
	vc.tpbftProtocal = tpbftProtocal

	vc.pbftFactor = (len(vc.subNetLiver) - 1) / 3 // 3f + 1 = 分区节点总数

	vc.responseThreshould = viewChange_responseThreshould
	vc.liveCheckInterval = viewChange_liveCheckInterval
	vc.msgScanInterval = viewChange_msgScanInterval
	vc.msgExpireInterval = viewChange_msgExpireInterval

	vc.candidateLeader = make([]common.NodeID, 0)

	vc.close = make(chan bool)

	vc.heartBeatPool = make(map[common.NodeID]*consensus.HeartBeat)
	vc.viewChangeMsgPool = make(map[common.NodeID]*consensus.ViewChangeMsg)
	vc.confirmLeaderRespPool = make(map[common.NodeID]*consensus.ConfirmLeaderResp)
	// vc.newViewMsgPool = make(map[common.NodeID]*NewViewMsg)

	return vc
}

func (vc *ViewChange) ViewChangeStart() {

	ReloadConfigPath()

	ctx, cancle := context.WithCancel(context.Background())
	go vc.LiverCheck(ctx)
	go vc.CheckLeader(ctx)
	go vc.MsgRead(ctx)

	// Leader需要有一个协程，通过这个协程来确认自己当前是否还是Leader
	go vc.IsNotLeader(ctx)

	<-vc.close
	cancle()
}

func (vc *ViewChange) ViewChangeStop() {
	close(vc.close)
}

func (vc *ViewChange) SetPbftFactor(factor int) {
	vc.pbftFactor = factor
}

// 选择候选可以成为Leader的节点 (此处的策略是选择已连接的live节点)
func (vc *ViewChange) SelectCandidateLeader() {
	vc.candidateLeaderMutex.Lock()
	defer vc.candidateLeaderMutex.Unlock()
	vc.candidateLeader = make([]common.NodeID, 0)
	for node, status := range vc.subNetLiver {
		if status && !reflect.DeepEqual(node, vc.leader) { // 当前的Leader不能进入候选队列中
			vc.candidateLeader = append(vc.candidateLeader, node)
			loglogrus.Log.Infof("[View Change] 当前节点(%x) ,可供选择的候选Leader节点有:%x(总计%d个)\n", vc.consensusPromoter.SelfNode.NodeID, node, len(vc.candidateLeader))
		}
	}

}

// 进行Leader Change之前，先要选出一个新的Leader（轮询？Pow方式？）
func (vc *ViewChange) SelectNewLeader(time int, candidate []common.NodeID) { // time表示轮次
	if len(candidate) == 0 {
		loglogrus.Log.Warnf("View Change Failed: 当前节点(%x)没有候选Leader,无法进行Leader Change\n", vc.consensusPromoter.SelfNode.NodeID)
		return
	}
	vc.oldLeader = vc.leader

	nodeIDMap := make(map[string]common.NodeID)
	nodeIDStrSet := make([]string, 0)
	for _, nodeID := range candidate {
		nodeIDMap[fmt.Sprintf("%x", nodeID)] = nodeID
		nodeIDStrSet = append(nodeIDStrSet, fmt.Sprintf("%x", nodeID))
	}

	sort.Strings(nodeIDStrSet)

	vc.leader = nodeIDMap[nodeIDStrSet[time]]
	loglogrus.Log.Infof("[View Change] 当前节点(%x)挑选出下一轮共识的新Leader(%x), oldLeader为(%x)\n", vc.consensusPromoter.SelfNode.NodeID, vc.leader, vc.oldLeader)
}

func (vc *ViewChange) SetRespThreshould(threshould time.Duration) {
	vc.responseThreshould = threshould
}

func (vc *ViewChange) AddSubNetLive(node common.NodeID) {
	vc.subNetLiverMutex.Lock()
	defer vc.subNetLiverMutex.Unlock()
	if status, ok := vc.subNetLiver[node]; !ok {
		vc.subNetLiver[node] = true
	} else {
		if !status {
			vc.subNetLiver[node] = true
		}
	}
}

func (vc *ViewChange) AddSubNetDeadth(node common.NodeID) {
	vc.subNetLiverMutex.Lock()
	defer vc.subNetLiverMutex.Unlock()
	if status, ok := vc.subNetLiver[node]; !ok {
		vc.subNetLiver[node] = false
	} else {
		if status {
			vc.subNetLiver[node] = false
		}
	}
}

func (vc *ViewChange) RemoveNodeFromSubNet(node common.NodeID) {
	vc.subNetLiverMutex.Lock()
	defer vc.subNetLiverMutex.Unlock()
	delete(vc.subNetLiver, node)
}

func (vc *ViewChange) MsgRead(ctx context.Context) {
	msgScanCycle := time.NewTicker(vc.msgScanInterval)
	msgExpireCycle := time.NewTicker(vc.msgExpireInterval)
	for {
		select {
		case <-ctx.Done():
			msgScanCycle.Stop()
			msgExpireCycle.Stop()
			return
		case <-msgExpireCycle.C: // 从消息池中删除过期的view-Change相关的消息(针对长期断连或者已经退出网络的节点)
			now := time.Now()
			// 1.删除过期的心跳包
			vc.heartBeatPoolMutex.Lock()
			expireHeartBeatSet := make([]common.NodeID, 0)
			for nodeID, hb := range vc.heartBeatPool {
				if now.Unix()-int64(hb.TimeStampSecond) > int64(vc.responseThreshould.Seconds()) {
					expireHeartBeatSet = append(expireHeartBeatSet, nodeID)
					loglogrus.Log.Infof("[View Change] 删除来自节点(%x)的过期心跳包,心跳包时间戳(%d),当前时间(%d)\n", nodeID, hb.TimeStampSecond, now.Unix())
				}
			}
			for _, ehb := range expireHeartBeatSet {
				delete(vc.heartBeatPool, ehb)
			}
			vc.heartBeatPoolMutex.Unlock()

			// 2.删除过期的view-change Msg
			vc.viewChangeMsgPoolMutex.Lock()
			expireVcmSet := make([]common.NodeID, 0)
			for nodeID, vcm := range vc.viewChangeMsgPool {
				if (now.Unix() - int64(vcm.TimeStampSecond)) > int64(vc.responseThreshould.Seconds()) {
					expireVcmSet = append(expireVcmSet, nodeID)
					loglogrus.Log.Infof("[View Change] 删除来自节点(%x)的过期View-Change Msg,Msg时间戳(%d),当前时间(%d)\n", nodeID, vcm.TimeStampSecond, now.Unix())
				}
			}
			for _, evcm := range expireVcmSet {
				delete(vc.viewChangeMsgPool, evcm)
			}
			vc.viewChangeMsgPoolMutex.Unlock()

		case <-msgScanCycle.C: // 处理接收到的消息，将其放入相应的消息池中(总是最新的消息)
			currentMsgs := vc.consensusPromoter.BackLowerChannel().CurrentMsgs()
			toMark := make([]common.Hash, 0)    //消息已读队列(记录已读消息Hash值)
			mark := func(msgHash common.Hash) { //函数变量，负责将指定msgHash加入到toMark切片中
				toMark = append(toMark, msgHash)
			}
			for _, msg := range currentMsgs {
				if msg.IsControl() {
					// 如果是心跳包,读取后要将其从消息池中删除,按照心跳包更新vc.subNetLiver；如果是其他类型直接跳过不处理
					wlcm, err := consensus.DeserializeLowerConsensusMsg(msg.PayLoad) //解包为WrappedLowerConsensusMessage 统一格式消息
					if err != nil {                                                  //解包失败
						mark(msg.CalculateHash()) //标记已读
						loglogrus.Log.Warnf("[View Change] View Change failed: Fail to Deserialize separator.Message into Lower Consensus Msg, err:%v\n", err)
						continue
					}
					if msg.From != wlcm.Head.Sender { //判断来源是否可靠
						mark(msg.CalculateHash()) //不可靠也要标记已读
						loglogrus.Log.Warnf("[View Change] View Change failed: Deserialized Lower Consensus Msg -- msg.From (%x) could not match the sender (%x)\n", msg.From, wlcm.Head.Sender)
						continue
					}
					// 1.是心跳包
					if wlcm.Head.TypeCode == consensus.StateHeartBeat {
						unwrapMsg, err := wlcm.UnWrap() // WrappedLowerConsensusMessage解包为对于阶段共识消息
						if err != nil {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to unwarp WrappedLowerConsensusMessage into LowerConsensusMessage: %v\n", err)
							continue
						}
						hb, ok := unwrapMsg.(*consensus.HeartBeat) //类型断言为HeartBeat
						if !ok {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to assert LowerConsensusMessage into HeartBeat\n")
							continue
						}
						vc.heartBeatPoolMutex.Lock()
						if heatBeat, ok := vc.heartBeatPool[wlcm.Head.Sender]; !ok { // 从未收到过来自该节点的心跳包,那么新增
							vc.heartBeatPool[wlcm.Head.Sender] = hb
						} else { // 接收到过来自该节点的心跳包,则更新(时间戳在后才更新)
							if hb.TimeStampSecond > heatBeat.TimeStampSecond {
								vc.heartBeatPool[wlcm.Head.Sender] = hb
							}
						}
						vc.heartBeatPoolMutex.Unlock()
						mark(msg.CalculateHash()) //msg信号打上已读标签
						// 2.是view-Change 消息
					} else if wlcm.Head.TypeCode == consensus.StateViewChange {
						unwrapMsg, err := wlcm.UnWrap() // WrappedLowerConsensusMessage解包为对于阶段共识消息
						if err != nil {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to unwarp WrappedLowerConsensusMessage into LowerConsensusMessage: %v\n", err)
							continue
						}
						vcm, ok := unwrapMsg.(*consensus.ViewChangeMsg) //类型断言为 ViewChangeMsg
						if !ok {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to assert LowerConsensusMessage into ViewChangeMsg\n")
							continue
						}
						vc.viewChangeMsgPoolMutex.Lock()
						if viewChangeMsg, ok := vc.viewChangeMsgPool[wlcm.Head.Sender]; !ok { // 从未收到过来自该节点的view-change Msg,那么新增
							vc.viewChangeMsgPool[wlcm.Head.Sender] = vcm
						} else { // 接收到过来自该节点的view-change Msg,则更新(时间戳在后才更新)
							if vcm.TimeStampSecond > viewChangeMsg.TimeStampSecond {
								vc.viewChangeMsgPool[wlcm.Head.Sender] = vcm
								loglogrus.Log.Infof("[View Change] 接收到来自节点(%x)的最新View-Change Msg, 最新Msg时间戳(%d), 上一Msg时间戳(%d)\n", wlcm.Head.Sender, vcm.TimeStampSecond, viewChangeMsg.TimeStampSecond)
							} else {
								loglogrus.Log.Infof("[View Change] 接收到来自节点(%x)的已过期View-Change Msg, 此Msg时间戳(%d), 上一Msg时间戳(%d)\n", wlcm.Head.Sender, vcm.TimeStampSecond, viewChangeMsg.TimeStampSecond)
							}
						}
						vc.viewChangeMsgPoolMutex.Unlock()
						mark(msg.CalculateHash()) //msg信号打上已读标签
						// 3.是 ConfirmLeaderReq 消息
					} else if wlcm.Head.TypeCode == consensus.StateConfirmLeaderReq {
						if vc.running {
							loglogrus.Log.Infof("[View Change] 当前节点(%x)当前正在进行view-change,因此不能对 ConfirmLeaderReq Msg 进行回应\n", vc.consensusPromoter.SelfNode.NodeID)
							continue
						}

						if wlcm.Head.Sender == vc.consensusPromoter.SelfNode.NodeID { // 不能自发自收
							continue
						}

						unwrapMsg, err := wlcm.UnWrap() // WrappedLowerConsensusMessage解包为对于阶段共识消息
						if err != nil {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to unwarp WrappedLowerConsensusMessage into LowerConsensusMessage: %v\n", err)
							continue
						}
						clreq, ok := unwrapMsg.(*consensus.ConfirmLeaderReq) //类型断言为 ConfirmLeaderReq Msg
						if !ok {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to assert LowerConsensusMessage into ConfirmLeaderReq\n")
							continue
						}
						mark(msg.CalculateHash()) //msg信号打上已读标签

						loglogrus.Log.Infof("[View Change] 当前节点(%x)接收到来自节点(%x)的ConfirmLeaderReq Msg\n", vc.consensusPromoter.SelfNode.NodeID, wlcm.Head.Sender)

						if reflect.DeepEqual(wlcm.Head.Sender, vc.leader) { // 如果对方依旧还是Leader节点,则不需要进行回复
							loglogrus.Log.Infof("[View Change] 对方节点(%x)依旧是Leader,不进行回复\n", wlcm.Head.Sender)
							continue
						}

						clResp := &consensus.ConfirmLeaderResp{
							Head: consensus.CommonHead{
								Consensus: []byte(vc.tpbftProtocal.Name()),
								TypeCode:  consensus.StateConfirmLeaderResp,
								Sender:    vc.consensusPromoter.SelfNode.NodeID, //发送者的NodeID
							},
							Sender:          vc.consensusPromoter.SelfNode.NodeID,
							Leader:          vc.leader,
							Sequence:        clreq.Sequence,
							TimeStampSecond: uint64(time.Now().Unix()),
						}
						clResp.Hash()
						clResp.SignatureClResp(vc.consensusPromoter.BackPrvKey())

						wlcmResp := clResp.Wrap()
						payload, err := rlp.EncodeToBytes(&wlcmResp)
						if err != nil {
							loglogrus.Log.Warnf("[View Change] View Change failed: ConfirmLeaderResp Msg Send failed!, err:%v\n", err)
						}
						msg := vc.consensusPromoter.BackLowerChannel().NewControlMessage(vc.consensusPromoter.SelfNode.NodeID, payload)
						vc.consensusPromoter.BackLowerChannel().MsgSend(msg, wlcm.Head.Sender) // 向请求节点进行回复

						vc.consensusPromoter.LastPreprepareTime = time.Now() // 为了保证当前节点与旧Leader能够同步进行下一次view-change

						loglogrus.Log.Infof("[View Change] 当前节点(%x)向请求方(%x)回复ConfirmLeaderResp Msg\n", vc.consensusPromoter.SelfNode.NodeID, wlcm.Head.Sender)

					} else if wlcm.Head.TypeCode == consensus.StateConfirmLeaderResp {
						unwrapMsg, err := wlcm.UnWrap() // WrappedLowerConsensusMessage解包为对于阶段共识消息
						if err != nil {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to unwarp WrappedLowerConsensusMessage into LowerConsensusMessage: %v\n", err)
							continue
						}
						clresp, ok := unwrapMsg.(*consensus.ConfirmLeaderResp) //类型断言为 ConfirmLeaderResp Msg
						if !ok {
							loglogrus.Log.Warnf("[View Change] View Change failed: Fail to assert LowerConsensusMessage into ConfirmLeaderResp\n")
							continue
						}
						mark(msg.CalculateHash()) //msg信号打上已读标签

						loglogrus.Log.Infof("[View Change] 当前节点(%x)已经收到来自(%x)的confirmLeaderResp Msg\n", vc.consensusPromoter.SelfNode.NodeID, wlcm.Head.Sender)

						// 需要一个缓存池保留收到的 ConfirmLeaderResp Msg , 当大于 2f+1 条时,说明Leader已经被换过了
						vc.confirmLeaderRespPoolMutex.Lock()
						if clresp.Sequence == vc.sequence {
							pubKey, _ := crypto.NodeIDtoKey(wlcm.Head.Sender)
							if flag := clresp.ValidateSign(pubKey); flag {
								vc.confirmLeaderRespPool[wlcm.Head.Sender] = clresp
								loglogrus.Log.Infof("[View Change] 当前节点(%x) 完成对来自(%x)的confirmLeaderResp Msg的验证,回应的当前Leader应为:%x\n",
									vc.consensusPromoter.SelfNode.NodeID, wlcm.Head.Sender, clresp.Leader)
							} else {
								loglogrus.Log.Infof("[View Change] 当前节点(%x) 无法完成对来自(%x)的confirmLeaderResp Msg的验证, 原因:数字签名验证未通过\n",
									vc.consensusPromoter.SelfNode.NodeID, wlcm.Head.Sender)
							}
						} else {
							loglogrus.Log.Infof("[View Change] 当前节点(%x) 无法完成对来自(%x)的confirmLeaderResp Msg的验证, 原因:消息Sequence(%d)与本轮Sequence(%d)不匹配\n",
								vc.consensusPromoter.SelfNode.NodeID, wlcm.Head.Sender, clresp.Sequence, vc.sequence)
						}
						vc.confirmLeaderRespPoolMutex.Unlock()

						vc.confirmLeaderRespPoolMutex.RLock()
						if len(vc.confirmLeaderRespPool) < 2*vc.pbftFactor+1 {
							loglogrus.Log.Infof("[View Change] 节点(%x)依旧还是Leader,因为没有足够的节点表示Leader节点已经切换\n", vc.leader)
							vc.confirmLeaderRespPoolMutex.RUnlock()
							continue
						}

						voteMap := make(map[common.NodeID]int)
						for _, clresp := range vc.confirmLeaderRespPool {
							voteMap[clresp.Leader] += 1
						}

						var maxVoteNode common.NodeID
						var maxVoteCount int = 0
						for curLeader, vote := range voteMap {
							if vote > maxVoteCount {
								maxVoteNode = curLeader
								maxVoteCount = vote
							}
						}

						loglogrus.Log.Infof("[View Change] 当前节点(%x)已知的获得票数最多的新Leader是(%x)\n", vc.consensusPromoter.SelfNode.NodeID, maxVoteNode)
						vc.confirmLeaderRespPoolMutex.RUnlock()

						if maxVoteCount >= 2*vc.pbftFactor+1 { // 可以进行Leader切换
							vc.leader = maxVoteNode

							tokenCount := vc.SuspendConsensus()

							vc.consensusPromoter.SelfNode.Role = dpnet.Follower
							vc.consensusPromoter.LastPreprepareTime = time.Now().Add(vc.responseThreshould / 2) // 重新开始计时

							loglogrus.Log.Infof("[View Change] 当前节点(%x)变为Follower, 新的Leader节点是(%x)\n", vc.consensusPromoter.SelfNode.NodeID, vc.leader)

							vc.RestartConsensus(tokenCount)

							vc.confirmLeaderRespPoolMutex.Lock()
							vc.confirmLeaderRespPool = make(map[common.NodeID]*consensus.ConfirmLeaderResp)
							vc.confirmLeaderRespPoolMutex.Unlock()

							loglogrus.Log.Infof("[View Change] 当前节点(%x)重新恢复下层共识的工作,新leader(%x)......\n", vc.consensusPromoter.SelfNode.NodeID, vc.leader)

							// 旧Leader更新本地的配置文件,将自己改为Follower,保证下次启动时能正常运行
							dc := ReloadConfigFile()
							UpdateConfigFile(dc, "Follower")
							vc.syncProtocal.UseNormalMode() // 成为了follower,需要打开区块同步的主动同步功能
						} else {
							loglogrus.Log.Infof("[View Change] 当前节点(%x)依旧还是Leader,因为没有足够的节点表示Leader节点已经切换\n", vc.leader)
							continue
						}

					}
				}
			}
			vc.consensusPromoter.BackLowerChannel().MarkRetrievedMsgs(toMark) //separator协议的消息池中将这些已读的view-change相关的Msg标记已读
		}
	}
}

// 进行活跃检测,更新vc.subNetLiver
func (vc *ViewChange) LiverCheck(ctx context.Context) {
	heartBeatCycle := time.NewTicker(vc.liveCheckInterval)
	for {
		select {
		case <-heartBeatCycle.C: // 发送心跳包
			for nodeID, _ := range vc.subNetLiver {
				if nodeID == vc.consensusPromoter.SelfNode.NodeID { // 不需要给自己发送心跳包，而是将自己直接加入到vc.subNetLiver
					continue
				}
				go func(node common.NodeID) {
					heartBeat := &consensus.HeartBeat{
						Head: consensus.CommonHead{
							Consensus: []byte(vc.tpbftProtocal.Name()),
							TypeCode:  consensus.StateHeartBeat,
							Sender:    vc.consensusPromoter.SelfNode.NodeID, //发送者的NodeID
						},
						TimeStampSecond: uint64(time.Now().Unix()),
					}
					wlcm := heartBeat.Wrap()
					payload, err := rlp.EncodeToBytes(&wlcm)
					if err != nil {
						loglogrus.Log.Warnf("[View Change] View Change failed: Heart Beat Send failed!, err:%v\n", err)
					}
					msg := vc.consensusPromoter.BackLowerChannel().NewControlMessage(vc.consensusPromoter.SelfNode.NodeID, payload)

					if err := vc.consensusPromoter.BackLowerChannel().MsgSend(msg, node); err != nil {
						loglogrus.Log.Warnf("[View Change] 发送心跳包失败, 当前NodeID(%x),目标NodeID(%x), err:%v\n", vc.consensusPromoter.SelfNode.NodeID, node, err)
					} else {
						//fmt.Printf("[View Change] 成功发送心跳包, 当前NodeID(%x), 目标NodeID(%x)\n", vc.consensusPromoter.SelfNode.NodeID, node)
						loglogrus.Log.Infof("[View Change] 成功发送心跳包, 当前NodeID(%x), 目标NodeID(%x)\n", vc.consensusPromoter.SelfNode.NodeID, node)
					}
				}(nodeID)
			}

			time.Sleep(2 * vc.msgScanInterval) // 等待接收来自于其他节点的心跳回复

			for node, live := range vc.subNetLiver {
				if live {
					vc.AddSubNetDeadth(node)
				}
			}
			now := time.Now()
			vc.heartBeatPoolMutex.RLock()
			for nodeID, heartBeat := range vc.heartBeatPool {
				interval := now.Unix() - int64(heartBeat.TimeStampSecond)
				if interval < int64(vc.responseThreshould.Seconds()) { // 必须是本view内的心跳包
					vc.AddSubNetLive(nodeID)
					loglogrus.Log.Infof("[View Change] 成功接收到来自节点(%x)在时刻(%d)发送的心跳包(未超时), 时间间隔为(%d)\n", nodeID, heartBeat.TimeStampSecond, interval)
				} else {
					loglogrus.Log.Infof("[View Change] 成功接收到来自节点(%x)在时刻(%d)发送的心跳包(超时), 时间间隔为(%d)\n", nodeID, heartBeat.TimeStampSecond, interval)
				}
			}

			vc.AddSubNetLive(vc.consensusPromoter.SelfNode.NodeID) // 将自己直接加入到vc.subNetLiver

			vc.heartBeatPoolMutex.RUnlock()
			if vc.running { // 正在进行Leader Change
				loglogrus.Log.Infof("[View Change] 正在进行Leader Change, 无法更新vc.candidateLeader....\n")
				continue
			}
			vc.SelectCandidateLeader() // 根据vc.subNetLiver重新更新vc.candidateLeader
		case <-ctx.Done():
			heartBeatCycle.Stop()
			return
		default:
			continue
		}
	}
}

// Leader节点检查自己是否还是Leader(通过发送Msg确认)
func (vc *ViewChange) IsNotLeader(ctx context.Context) {
	checkCycle := time.NewTicker(vc.responseThreshould * 1 / 4) // 速率应该比follower的CheckLeader()函数更快
	for {
		select {
		case <-checkCycle.C:
			selfNode := vc.consensusPromoter.SelfNode.NodeID
			if vc.consensusPromoter.SelfNode.Role == dpnet.Follower {
				continue
			}
			// 在lower channel节点组内广播 ConfirmLeaderReq 确认请求
			rand.Seed(time.Now().UnixNano())
			vc.sequence = rand.Uint64()
			confirmLeader := &consensus.ConfirmLeaderReq{
				Head: consensus.CommonHead{
					Consensus: []byte(vc.tpbftProtocal.Name()),
					TypeCode:  consensus.StateConfirmLeaderReq,
					Sender:    selfNode, //发送者的NodeID
				},
				Sender:   selfNode,
				Sequence: vc.sequence,
			}
			wlcm := confirmLeader.Wrap()
			payload, err := rlp.EncodeToBytes(&wlcm)
			if err != nil {
				loglogrus.Log.Warnf("[View Change] View Change failed: Confirm Leader Msg Send failed!, err:%v\n", err)
			}
			msg := vc.consensusPromoter.BackLowerChannel().NewControlMessage(selfNode, payload)
			vc.consensusPromoter.BackLowerChannel().MsgBroadcast(msg)
			loglogrus.Log.Infof("[View Change] 当前节点(%x)已经发送 ConfirmLeaderReq Msg\n", vc.consensusPromoter.SelfNode.NodeID)

		case <-ctx.Done():
			checkCycle.Stop()
			return
		default:
			continue

		}

	}
}

// Follower检查Leader节点是否还在正常工作，如果发现不正常则启动view-change
func (vc *ViewChange) CheckLeader(ctx context.Context) {
	checkCycle := time.NewTicker(vc.responseThreshould)
	for {
		select {
		case <-checkCycle.C:
			if vc.consensusPromoter.SelfNode.Role == dpnet.Leader {
				loglogrus.Log.Infof("[View Change] 当前节点(%x)是Leader,不会启动view-change功能\n", vc.consensusPromoter.SelfNode.NodeID)
				continue
			}
			now := time.Now()
			interval := now.Sub(vc.consensusPromoter.LastPreprepareTime)        // 距离上次收到Leader节点pre-prepare Msg的时间间隔
			if !vc.subNetLiver[vc.leader] || interval > vc.responseThreshould { // 如果Leader节点长时间断连，或者长时间不执行PBFT共识(当前Follower持续接收不到Leader的pre-prepare Msg)执行Leader Change
				for i := 0; i < 3; i++ {
					vc.running = true
					if !vc.subNetLiver[vc.leader] {
						loglogrus.Log.Infof("[View Change] 当前节点(%x)无法与Leader节点(%x)正常通信,进行View Change\n", vc.consensusPromoter.SelfNode.NodeID, vc.leader)
					}
					if interval > vc.responseThreshould {
						loglogrus.Log.Infof("[View Change] 当前节点(%x)长时间未收到Leader节点(%x)的pre-prepare Msg,进行View Change\n", vc.consensusPromoter.SelfNode.NodeID, vc.leader)
					}

					candidate := make([]common.NodeID, 0)
					candidate = append(candidate, vc.candidateLeader...)

					//for i := 0; i < len(candidate); i++ {
					vc.SelectNewLeader(0, candidate) // 从候选节点中选择出一个新的Leader  TODO: 剩余Follower节点尽可能都选举同一个节点作为新Leader(可以引入视图ID的概念，用视图ID来计算新Leader的NodeID)

					// 1.关停当前节点组的共识功能(下层+上层共识)
					tokenCount := vc.SuspendConsensus()

					// 2.发送支持新Leader的view-change消息给组内其他存活节点
					vc.SendViewChangeMsg()

					// 3.等待其他节点的回复
					time.Sleep(5 * time.Second)
					isSuccess, leaderVotes := vc.CheckViewChangeVoteMsg()

					// 4.是否有候选节点获得了2*f条选票
					if !isSuccess { // 一轮选举失败了
						vc.running = false
						vc.RestartConsensus(tokenCount)
						time.Sleep(3 * time.Second)
						continue
					}
					// 5.新的Leader节点负责通知全网执行Leader节点的切换（同时也要在本地完成更新）
					if vc.leader == vc.consensusPromoter.SelfNode.NodeID {
						loglogrus.Log.Infof("[View Change] 当前节点(%x)将作为新的Leader通知全网进行本分区的Leader节点切换\n", vc.consensusPromoter.SelfNode.NodeID)
						go vc.NewLeaderNotify(leaderVotes)

						// 新Leader更新本地的配置文件,将自己改为Leader,保证下次启动时能正常运行
						go func() {
							dc := ReloadConfigFile()
							UpdateConfigFile(dc, "Leader")
						}()

						vc.syncProtocal.UseLeaderMode() // 成为了leader,需要关闭区块同步的主动同步功能
					}

					time.Sleep(5 * time.Second) // 等待所有节点完成Leader节点的切换(本分区的需要在separator完成更新,upper层的需要在upper channel完成更新)
					loglogrus.Log.Infof("[View Change] 警戒线: 所有节点必须在此时间线之前完成故障分区的Leader节点切换工作......\n")

					// 6.重新恢复下层共识
					vc.RestartConsensus(tokenCount)
					loglogrus.Log.Infof("[View Change] 当前节点(%x)重新恢复下层共识的工作,新leader(%x)......\n", vc.consensusPromoter.SelfNode.NodeID, vc.leader)

					vc.consensusPromoter.LastPreprepareTime = time.Now() // 重新开始计时
					vc.running = false

					break
				}
			}
		case <-ctx.Done():
			checkCycle.Stop()
			return
		default:
			continue
		}
	}
}

func (vc *ViewChange) SuspendConsensus() int {
	// 1.中止当前正在进行的roundHandler
	roundSet := make([]common.Hash, 0)
	vc.consensusPromoter.BackRoundCacheMutex().RLock()
	for roundID, _ := range vc.consensusPromoter.BackRoundCache() {
		roundSet = append(roundSet, roundID)
	}
	vc.consensusPromoter.BackRoundCacheMutex().RUnlock()
	for _, roundID := range roundSet {
		vc.consensusPromoter.RemoveRoundHandler(roundID)
		loglogrus.Log.Infof("[View Change] 准备停止共识流程,正在停止已运行共识roundHandler(%x)\n", roundID)
	}
	// 2.取出所有的token,避免新的共识round开始
	tokenCount := 0
	ddl := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-vc.consensusPromoter.BackRoundParalleledToken():
			tokenCount++
			if tokenCount == consensus.MaxParallelRound {
				loglogrus.Log.Infof("[View Change] 当前节点(%x)已回收全部Token(count:%d),已成功停止所有运行中pbft共识\n", vc.consensusPromoter.SelfNode.NodeID, tokenCount)
				return tokenCount
			}
		case <-ddl.C:
			loglogrus.Log.Infof("[View Change] 当前节点(%x)回收超时,部分回收Token(count:%d),未能成功停止所有运行中pbft共识\n", vc.consensusPromoter.SelfNode.NodeID, tokenCount)
			return tokenCount
		}
	}
}

func (vc *ViewChange) RestartConsensus(tokenCount int) {
	for i := 0; i < tokenCount; i++ {
		vc.consensusPromoter.BackRoundParalleledToken() <- struct{}{}
	}
}

func (vc *ViewChange) SendViewChangeMsg() {
	vcm := consensus.ViewChangeMsg{
		Head: consensus.CommonHead{
			Consensus: []byte(vc.tpbftProtocal.Name()),
			TypeCode:  consensus.StateViewChange,            //View-Change阶段
			Sender:    vc.consensusPromoter.SelfNode.NodeID, //发送者的NodeID
		},
		Sender:          vc.consensusPromoter.SelfNode.NodeID,
		TimeStampSecond: uint64(time.Now().Unix()),
		NewLeader:       vc.leader,
	}
	vcm.Hash()
	vcm.SignatureVcm(vc.consensusPromoter.BackPrvKey())

	wlcm := vcm.Wrap()
	payload, err := rlp.EncodeToBytes(&wlcm)
	if err != nil {
		loglogrus.Log.Warnf("send view-change Msg failed,err:%v\n", err)
		return
	}
	msg := vc.consensusPromoter.BackLowerChannel().NewControlMessage(vc.consensusPromoter.SelfNode.NodeID, payload)
	vc.consensusPromoter.BackLowerChannel().MsgBroadcast(msg)

	loglogrus.Log.Infof("[View Change] 当前节点(%x)已成功广播View-Change Msg\n", vc.consensusPromoter.SelfNode.NodeID)
}

// 检查是否至少有一个节点获得了至少2*f条ViewChange Msg的支持
func (vc *ViewChange) CheckViewChangeVoteMsg() (bool, map[common.NodeID]consensus.ViewChangeMsg) {

	vc.viewChangeMsgPoolMutex.RLock()
	defer vc.viewChangeMsgPoolMutex.RUnlock()

	votesSet := make(map[common.NodeID]map[common.NodeID]consensus.ViewChangeMsg) // 第一个key是支持的新Leader，第二个key是vcm的Sender
	for nodeID, vcm := range vc.viewChangeMsgPool {                               // 每个NodeID只能投一票
		if nodeID == vc.consensusPromoter.SelfNode.NodeID {
			continue
		}
		// 验证该View-Change 消息是否可信(签名、哈希都正确)
		if isTrust := ValidateViewChangeMsg(vc.consensusPromoter.SelfNode.NodeID, vcm); isTrust {
			if _, ok := votesSet[vcm.NewLeader]; !ok {
				votesSet[vcm.NewLeader] = make(map[common.NodeID]consensus.ViewChangeMsg)
			}
			votesSet[vcm.NewLeader][vcm.Sender] = *vcm
			loglogrus.Log.Infof("[View Change] 当前节点(%x)签名验证通过, 接收到来自节点(%x)的View-Change Msg, 支持的新Leader(%x)\n", vc.consensusPromoter.SelfNode.NodeID, nodeID, vcm.NewLeader)
		}
	}

	var maxVoteNode common.NodeID = vc.leader
	var maxVoteCount int = len(votesSet[vc.leader])
	for candidate, votes := range votesSet {
		if len(votes) > maxVoteCount {
			maxVoteNode = candidate
			maxVoteCount = len(votes)
		}
	}

	loglogrus.Log.Infof("[View Change] 得票数最多的Follower节点为(%x), 得票数(%d), 达标票数为(%d)\n", maxVoteNode, maxVoteCount, 2*vc.pbftFactor)

	if maxVoteCount >= 2*vc.pbftFactor {
		vc.leader = maxVoteNode
		return true, votesSet[maxVoteNode]
	} else {
		return false, nil
	}
}

// 新的Leader节点通知全网进行Leader切换
func (vc *ViewChange) NewLeaderNotify(leaderVotes map[common.NodeID]consensus.ViewChangeMsg) {
	// 1.通知separator层完成切换
	if isSuccess := vc.UpdateSeparator(leaderVotes); !isSuccess {
		loglogrus.Log.Warnf("[View Change] New Leader: 无法在separator层完成Leader节点的切换\n")
	} else {
		loglogrus.Log.Infof("[View Change] New Leader: 完成在separator层的Leader节点切换\n")
	}

	// 2.在lower channel层完成切换
	if isSuccess := vc.UpdateLowerChannel(); !isSuccess {
		loglogrus.Log.Warnf("[View Change] New Leader: 无法在lower channel完成Leader节点的切换\n")
	} else {
		loglogrus.Log.Infof("[View Change] New Leader: 完成在lower channnel层的Leader节点切换\n")
	}

	time.Sleep(3 * time.Second) // 等待其他所有上层节点完成separator层netManger的更新
	// 3.在upper channel层完成切换
	loglogrus.Log.Infof("[View Change] 警戒线: 其他分区的Leader节点必须在此时间线之前完成separator层的更新......\n")

	if isSuccess := vc.UpdateUpperChannel(leaderVotes); !isSuccess {
		loglogrus.Log.Warnf("[View Change] New Leader: 无法在upper channel完成Leader节点的切换\n")
	} else {
		loglogrus.Log.Infof("[View Change] New Leader: 完成在upper channnel层的Leader节点切换\n")
	}

}

func (vc *ViewChange) UpdateSeparator(leaderVotes map[common.NodeID]consensus.ViewChangeMsg) bool {
	sigs := make([][]byte, 0)

	for _, vcm := range leaderVotes {
		if vcm.NewLeader == vc.consensusPromoter.SelfNode.NodeID {
			//fmt.Printf("[View Change] 收集到第(%d)个签名:%s\n", index, vcm.Signature)
			sigs = append(sigs, vcm.Signature)
		}
	}

	serialized, err := rlp.EncodeToBytes(vc.consensusPromoter.SelfNode.NodeID)
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 发送Leader Change Msg出错,err:=%v\n", err)
		return false
	}
	msgHash := crypto.Sha3Hash(serialized)

	selfSignature, err := crypto.Sign(msgHash.Bytes(), vc.consensusPromoter.BackPrvKey())
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 进行数字签名失败\n")
		return false
	} else {
		sigs = append(sigs, selfSignature) //包含新节点自己的数字签名(应对一个节点组只有2个节点的情况)
	}

	loglogrus.Log.Infof("[View Change] 通知组内其他节点 (包括节点自己) 在separator层更新自己的Leader,当前已搜集数字签名个数(%d)\n", len(sigs))
	if err := vc.consensusPromoter.BackNetManager().SendUpdateLeaderChange(msgHash, sigs); err != nil {
		loglogrus.Log.Warnf("[View Change] 发送Leader Change Msg失败,err:%v\n", err)
		return false
	}
	return true
}

func (vc *ViewChange) UpdateLowerChannel() bool {
	vc.consensusPromoter.SelfNode.Role = dpnet.Leader                                                            // 可以以Leader节点的身份开启pbft round
	vc.consensusPromoter.SetUpperChannel(vc.consensusPromoter.BackNetManager().Spr.PeerGroups[dpnet.UpperNetID]) // 加入到upper层节点组，可以接收到来自于upper层其他节点的消息
	return true
}

func (vc *ViewChange) UpdateUpperChannel(leaderVotes map[common.NodeID]consensus.ViewChangeMsg) bool {
	// 1.Validatator需要更新(区块验证器主要就是跟各分区主节点NodeID进行数字签名验证)
	exportedValidator, err := vc.consensusPromoter.BackNetManager().ExportValidator()
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 新Leader节点生成区块验证器失败,err:%v\n", err)
		return false
	}
	validateManager := vc.consensusPromoter.BackValidateManager()
	validateManager.Update(exportedValidator)
	validateManager.SetLowerValidFactor(3)
	validateManager.SetUpperValidRate(0.8)
	vc.consensusPromoter.BackNetManager().Syn.ValidateManager.Update(exportedValidator)

	// 2.新的Leader节点需要拿到一个属于自己的upperConsensusManager(设置cp.upperConsensusManager),否则无法在upper层内发送上层共识消息
	knownBooters := vc.consensusPromoter.BackNetManager().BackBooters()
	if len(knownBooters) == 0 {
		loglogrus.Log.Warnf("[View Change] 新Leader节点无法获知任何Booter节点...................\n")
		return false
	}
	servicePrivoderNodeID := knownBooters[0]

	orderServiceAgent := blockOrdering.NewBlockOrderAgent(servicePrivoderNodeID, vc.consensusPromoter.BackNetManager().Syn)
	orderServiceAgent.Install(vc.consensusPromoter)
	err = orderServiceAgent.Start()
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 新Leader节点无法启动upper层分区代理功能,err:%v\n", err)
		return false
	}

	// 3.向upper层其他Leader节点发送upper层共识消息,通知他们更新自己的区块验证器
	vc.NoticeUpdateValidator(leaderVotes)

	// 4.启动upper层接收协程
	go vc.consensusPromoter.UpperUpdateLoop()

	return true
}

// 新Leader节点通知upper层其他Leader更新区块验证器
func (vc *ViewChange) NoticeUpdateValidator(leaderVotes map[common.NodeID]consensus.ViewChangeMsg) bool {

	sigs := make([][]byte, 0)

	for _, vcm := range leaderVotes { // leaderVotes不包含节点自己发送给自己的viewchange Msg
		if vcm.NewLeader == vc.consensusPromoter.SelfNode.NodeID {
			sigs = append(sigs, vcm.Signature)
		}
	}

	serialized, err := rlp.EncodeToBytes(vc.consensusPromoter.SelfNode.NodeID)
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 发送Leader Change Msg出错,err:=%v\n", err)
		return false
	}
	msgHash := crypto.Sha3Hash(serialized)

	selfSignature, err := crypto.Sign(msgHash.Bytes(), vc.consensusPromoter.BackPrvKey())
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 进行数字签名失败\n")
		return false
	} else {
		sigs = append(sigs, selfSignature) //包含新节点自己的数字签名(应对一个节点组只有2个节点的情况)
	}

	loglogrus.Log.Infof("[View Change] 最终阶段: 搜集的数字签名个数:%d\n", len(sigs))

	bom, err := CreateUpperMsg_UpdateBlockValidator(vc.consensusPromoter.SelfNode.NodeID, vc.consensusPromoter.SelfNode.NetID, vc.consensusPromoter.StateManager.CurrentVersion(), sigs)
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 节点(%x) 生成UpdateBlockValidator Msg消息失败,err:%v\n", vc.consensusPromoter.SelfNode.NodeID, err)
		return false
	}

	payload, err := rlp.EncodeToBytes(bom)
	msg := vc.consensusPromoter.BackUpperChannel().NewUpperConsensusMessage(vc.consensusPromoter.SelfNode.NodeID, payload)

	if err != nil {
		loglogrus.Log.Warnf("[View Change] 节点(%x) 打包UpdateBlockValidator Msg为separator Msg失败,err:%v\n", vc.consensusPromoter.SelfNode.NodeID, err)
		return false
	}
	vc.consensusPromoter.BackUpperChannel().MsgBroadcast(msg)

	loglogrus.Log.Infof("[View Change] 当前节点(%x)完成 UpdateBlockValidator 消息的发送\n", vc.consensusPromoter.SelfNode.NodeID)

	return true
}

func CreateUpperMsg_UpdateBlockValidator(sender common.NodeID, group string, blockID common.Hash, sigs [][]byte) (*blockOrdering.BlockOrderMessage, error) {
	ubvMsg := new(blockOrdering.UpdateBlockValidator)
	ubvMsg.GroupID = group
	serialized, _ := rlp.EncodeToBytes(sender)
	senderHash := crypto.Sha3Hash(serialized)
	ubvMsg.SenderHash = senderHash
	ubvMsg.Sigs = sigs

	bom, err := blockOrdering.CreateBlockOrderMsg(ubvMsg, sender, blockOrdering.CODE_UPDATEBLOCKVALIDATOR)
	if err != nil {
		loglogrus.Log.Errorf("[View Change] 当前节点编辑UpperChannel Msg UpdateBlockValidator失败: %v", err)
		return nil, err
	}
	return bom, nil
}

func ValidateViewChangeMsg(self common.NodeID, vcm *consensus.ViewChangeMsg) bool {
	pub, err := crypto.NodeIDtoKey(vcm.Sender)
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 无法从发送方NodeID:(%x) 解析出其公钥\n", vcm.Sender)
		return false
	}

	if !vcm.ValidateSign(pub) {
		loglogrus.Log.Warnf("[View Change] 当前节点:(%x) 接收到副本节点:(%x)的的ViewChange Msg, 数字签名验证失败\n", self, vcm.Sender)
		return false
	}
	return true
}
