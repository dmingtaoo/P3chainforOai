package consensus

import (
	"p3Chain/coefficient"
	"p3Chain/common"
	"p3Chain/core/eles"
	"time"
)

const (
	// scan to check whether enough transactions could be wrapped to do consensus
	scanCycle = 300 * time.Millisecond
)

const (
	pbftThresholdFactor = 3 // the factor to do pbft
)

var DEFAULT_MAX_TRANSACTIONS = coefficient.Block_MaxTransactionNum

const (
	// the state of three stages
	StatePrePrepare = 0xA1
	StatePrepare    = 0xA2
	StateTidy       = 0xA3
	StateCommit     = 0xA4

	StateOver = 0x00

	StateHeartBeat         = 0xB1
	StateViewChange        = 0xB2
	StateConfirmLeaderReq  = 0xB3
	StateConfirmLeaderResp = 0xB4
	// StateNewView    = 0xB3
)

// Implement trimmed PBFT   tpbf_common.go的Ctpbft类实现了此接口
type TPBFT interface {
	//1.为consensus promoter安装一个交易池
	Install(*ConsensusPromoter) // install the txpool into consensus promoter

	//2. 检查当前交易是否与管理的交易池相匹配
	TxMatch(tx *eles.Transaction) bool // check out whether a transaction matches the associated pool.

	//3.从交易集合中筛选出符合条件的交易
	TxChoose(freshTxs []*eles.TransactionState) []*eles.Transaction // a strategy on how to choose transactions to do consensus in each round

	//4.当前使用的TPBFT协议的名称
	Name() string // return the unique name of this tpbft

	//5.进行视图切换
	ViewChange() // switch view when the net is changed

	//6.为当前共识协议启动新的round
	NewRound(leader common.NodeID, netID string) // launch a new round of consensus

	// Loads a consensus round and handle it. HandleRound will parse a PrePrepareMsg
	// and check the transaction pool have collected all the chosen transactions.
	// If succeed, returns a roundHandler and nil error.
	// 7.装载一个共识roundHandler并处理，具体的：
	// HandleRound函数负责解析 PrePrepareMsg 消息，然后检查交易池是否收集了全部筛选出的交易
	// 如果成功的话，返回一个roundHandler处理器
	HandleRound(ppm *PrePrepareMsg) (*roundHandler, error)

	//8.发送Pre-Prepare阶段共识消息(广播)
	SendPrePrepareMsg(*PrePrepareMsg)

	//9.发送Prepare阶段共识消息(广播)
	SendPrepareMsg(*PrepareMsg)

	//10.向目标节点发送Commit阶段共识消息(定点发送，给Leader节点)
	SendCommitMsg(*CommitMsg, common.NodeID)

	//11.验证传入交易集合中每条交易的有效性,获取交易有效认证序列
	ValidTransaction(txs []*eles.Transaction, version common.Hash) (validOrder []byte)

	//12.检查在prepare阶段是否从其他节点处获取了足够的prepare消息，以便进行交易整理(tidy)阶段
	CheckPrepare(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*PrepareMsg) (ok bool, validOrder []byte)

	//13.tidy阶段处理函数
	TidyTransactions(leader common.NodeID, version common.Hash, nonce uint8, txs []*eles.Transaction, validOrder []byte) (result []byte, signature []byte, block *eles.Block)

	//14.检查区块是否可以提交，若可以则组装区块并提交
	SubmitBlock(liveMemberSnapShot map[common.NodeID]bool, pbftFactor int, msgPool map[common.NodeID]*CommitMsg, block *eles.Block) bool
}
