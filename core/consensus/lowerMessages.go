package consensus

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"p3Chain/common"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"
)

type CommonHead struct {
	Consensus []byte        //标注当前共识协议
	RoundID   common.Hash   //当前共识round的ID
	TypeCode  uint8         //阶段类型(tpbft有三个阶段)
	Sender    common.NodeID //发送者的NodeID
}

// Outer wrapper of pre-prepare, prepare, commit messages
// tpbft的三类共识消息(三个阶段各自使用不同的共识消息)
type WrappedLowerConsensusMessage struct {
	Head    CommonHead //消息头(共识消息的common消息头)
	Content []byte     //消息实体(整条共识消息的rlp编码)
}

// 打包接口，负责将对应的共识消息(PrePrepareMsg/PrepareMsg/CommitMsg)统一打包成WrappedLowerConsensusMessage
type LowerConsensusMessage interface {
	Wrap() WrappedLowerConsensusMessage
}

// Pre-Prepare阶段使用的共识消息(注意:只有子网组的Leader节点可以发送pre-prepare阶段共识消息)
type PrePrepareMsg struct {
	// common
	head CommonHead //消息头

	// message
	Version common.Hash   //当前区块哈希
	Nonce   uint8         //暂不了解。。。
	TxOrder []common.Hash //当前节点所记录的区块内交易的顺序
}

func (ppm *PrePrepareMsg) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(ppm)
	if err != nil {
		loglogrus.Log.Warnf("fail in PrePrepareMsg wrap")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    ppm.head,
		Content: serialized,
	}
	return wlcm
}

// 计算PrePrepare消息的哈希值
func (ppm *PrePrepareMsg) Hash() common.Hash {
	temp := *ppm
	serialized, err := rlp.EncodeToBytes(&temp)
	if err != nil {
		loglogrus.Log.Warnf("fail in encode prepreparemsg")
	}
	return crypto.Sha3Hash(serialized)
}

// PrePrepare消息的哈希值作为本次交易round的ID值
// compute the round ID and set it
func (ppm *PrePrepareMsg) ComputeRoundID() common.Hash {
	if (ppm.head.RoundID == common.Hash{}) {
		ppm.head.RoundID = ppm.Hash()
	}
	return ppm.head.RoundID
}

type PrepareMsg struct {
	// common
	head CommonHead

	// message
	ValidOrder []byte // vote the valid orders, true is 1, false is 0  一个交易对应 1bit, 注意在prepare阶段时的交易列表就是有序的了
	Version    common.Hash
}

// 将 PrepareMsg 打包成 WrappedLowerConsensusMessage统一格式消息
func (pm *PrepareMsg) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(pm)
	if err != nil {
		loglogrus.Log.Warnf("fail in PrepareMsg wrap")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    pm.head,
		Content: serialized,
	}
	return wlcm
}

type CommitMsg struct {
	// common
	head CommonHead

	// message
	Result []byte // the result after tidy  完成整理的待发送区块的区块ID

	// vote
	Signature []byte // to support the result  发送此commit共识消息的节点的数字签名
}

// 将 CommitMsg 打包成 WrappedLowerConsensusMessage统一格式消息
func (cm *CommitMsg) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(cm)
	if err != nil {
		loglogrus.Log.Warnf("fail in CommitMsg wrap\n")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    cm.head,
		Content: serialized,
	}
	return wlcm
}

type HeartBeat struct {
	Head CommonHead
	//TimeStamp       time.Time
	TimeStampSecond uint64
}

func (hb *HeartBeat) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(hb)
	if err != nil {
		loglogrus.Log.Warnf("fail in HeartBeat wrap\n")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    hb.Head,
		Content: serialized,
	}
	return wlcm
}

type ViewChangeMsg struct {
	Head CommonHead //消息头

	Sender common.NodeID //发送者的NodeID
	//TimeStamp       time.Time     //时间戳(防止旧的viewChange覆盖新的viewChange)
	NewLeader       common.NodeID //新的主节点的NodeID,所有副本节点都必须指定同一个新主节点
	TimeStampSecond uint64
	MsgHash         common.Hash
	Signature       []byte //对MsgHash进行签名
}

func (vcm *ViewChangeMsg) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(vcm)
	if err != nil {
		loglogrus.Log.Warnf("fail in ViewChangeMsg wrap\n")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    vcm.Head,
		Content: serialized,
	}
	return wlcm
}

// 计算NewLeader字段的哈希值(后续对此哈希进行数字签名)
func (vcm *ViewChangeMsg) Hash() common.Hash {
	serialized, err := rlp.EncodeToBytes(vcm.NewLeader)
	if err != nil {
		loglogrus.Log.Warnf("fail in encode ViewChangeMsg\n")
	}

	vcm.MsgHash = crypto.Sha3Hash(serialized)

	return vcm.MsgHash
}

// 返回对ViewChange MsgHash的数字签名
func (vcm *ViewChangeMsg) SignatureVcm(privateKey *ecdsa.PrivateKey) []byte {

	signature, err := crypto.Sign(vcm.MsgHash.Bytes(), privateKey)
	if err != nil {
		loglogrus.Log.Warnf("fail in sign ViewChangeMsg\n")
	}
	vcm.Signature = signature
	return signature
}

// 接收方用发送方的公钥对其发送的ViewChangeMsg进行解密，看得到的签名公钥是否正确
func (vcm *ViewChangeMsg) ValidateSign(pub *ecdsa.PublicKey) bool {
	// 1.获取公钥的字节流
	publicKeyBytes := crypto.FromECDSAPub(pub)

	// 2.从Msg中获取出签名公钥
	sigPublicKey, err := crypto.Ecrecover(vcm.MsgHash.Bytes(), vcm.Signature)
	if err != nil {
		loglogrus.Log.Warnf("fail in Ecrecover ViewChangeMsg\n")
		return false
	}

	// 3.比较签名公钥是否正确
	matches := bytes.Equal(sigPublicKey, publicKeyBytes)

	return matches
}

// Leader节点用于确认自己是否还是Leader(因为Leader本身无法参与view-change)
type ConfirmLeaderReq struct {
	Head     CommonHead    // 消息头
	Sender   common.NodeID // 发送者的NodeID
	Sequence uint64
}

func (clreq *ConfirmLeaderReq) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(clreq)
	if err != nil {
		loglogrus.Log.Warnf("fail in ConfirmLeaderReq wrap\n")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    clreq.Head,
		Content: serialized,
	}
	return wlcm
}

// 其他节点对ConfirmLeaderReq Msg的回复
type ConfirmLeaderResp struct {
	Head            CommonHead    // 消息头
	Sender          common.NodeID // 发送者的NodeID
	Leader          common.NodeID // 接收方收到后，通过此字段来回复当前的Leader
	Sequence        uint64
	TimeStampSecond uint64 // 用于超时销毁
	MsgHash         common.Hash
	Signature       []byte //对MsgHash进行签名
}

func (clresp *ConfirmLeaderResp) Wrap() WrappedLowerConsensusMessage {
	serialized, err := rlp.EncodeToBytes(clresp)
	if err != nil {
		loglogrus.Log.Warnf("fail in ConfirmLeaderResp wrap\n")
	}
	wlcm := WrappedLowerConsensusMessage{
		Head:    clresp.Head,
		Content: serialized,
	}
	return wlcm
}

// 计算Leader字段的哈希值(后续对此哈希进行数字签名)
func (clresp *ConfirmLeaderResp) Hash() common.Hash {
	serialized, err := rlp.EncodeToBytes(clresp.Leader)
	if err != nil {
		loglogrus.Log.Warnf("fail in encode ConfirmLeaderResp\n")
	}

	clresp.MsgHash = crypto.Sha3Hash(serialized)

	return clresp.MsgHash
}

// 返回对ConfirmLeader MsgHash的数字签名
func (clresp *ConfirmLeaderResp) SignatureClResp(privateKey *ecdsa.PrivateKey) []byte {

	signature, err := crypto.Sign(clresp.MsgHash.Bytes(), privateKey)
	if err != nil {
		loglogrus.Log.Warnf("fail in sign ConfirmLeaderResp\n")
	}
	clresp.Signature = signature
	return signature
}

// 接收方用发送方的公钥对其发送的ConfirmLeaderResp进行解密，看得到的签名公钥是否正确
func (clresp *ConfirmLeaderResp) ValidateSign(pub *ecdsa.PublicKey) bool {
	// 1.获取公钥的字节流
	publicKeyBytes := crypto.FromECDSAPub(pub)

	// 2.从Msg中获取出签名公钥
	sigPublicKey, err := crypto.Ecrecover(clresp.MsgHash.Bytes(), clresp.Signature)
	if err != nil {
		loglogrus.Log.Warnf("fail in Ecrecover ConfirmLeaderResp\n")
		return false
	}

	// 3.比较签名公钥是否正确
	matches := bytes.Equal(sigPublicKey, publicKeyBytes)

	return matches
}

// 将统一格式的WrappedLowerConsensusMessage消息解包成对应共识阶段类型的消息
// (Pre-Prepare阶段/Prepare阶段/Commit阶段)分别解包成(PrePrepareMsg/PrepareMsg/CommitMsg)消息
func (wlcm *WrappedLowerConsensusMessage) UnWrap() (LowerConsensusMessage, error) {
	switch wlcm.Head.TypeCode {
	case StatePrePrepare:
		ppm := new(PrePrepareMsg)
		if err := rlp.DecodeBytes(wlcm.Content, ppm); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		ppm.head = wlcm.Head
		return ppm, nil
	case StatePrepare:
		pm := new(PrepareMsg)
		if err := rlp.DecodeBytes(wlcm.Content, pm); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		pm.head = wlcm.Head
		return pm, nil
	case StateCommit:
		cm := new(CommitMsg)
		if err := rlp.DecodeBytes(wlcm.Content, cm); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		cm.head = wlcm.Head
		return cm, nil
	case StateHeartBeat:
		hb := new(HeartBeat)
		if err := rlp.DecodeBytes(wlcm.Content, hb); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		hb.Head = wlcm.Head
		return hb, nil
	case StateViewChange:
		vcm := new(ViewChangeMsg)
		if err := rlp.DecodeBytes(wlcm.Content, vcm); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		vcm.Head = wlcm.Head
		return vcm, nil
	case StateConfirmLeaderReq:
		clreq := new(ConfirmLeaderReq)
		if err := rlp.DecodeBytes(wlcm.Content, clreq); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		clreq.Head = wlcm.Head
		return clreq, nil

	case StateConfirmLeaderResp:
		clresp := new(ConfirmLeaderResp)
		if err := rlp.DecodeBytes(wlcm.Content, clresp); err != nil {
			return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
		}
		clresp.Head = wlcm.Head
		return clresp, nil
	// case StateNewView:
	// 	nvm := new(NewViewMsg)
	// 	if err := rlp.DecodeBytes(wlcm.Content, nvm); err != nil {
	// 		return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: %v", err)
	// 	}
	// 	nvm.head = wlcm.Head
	// 	return nvm, nil

	default:
		return nil, fmt.Errorf("fail in WrappedLowerConsensusMessage unwrap: unknown code: %v", wlcm.Head.TypeCode)
	}
}

// Validate the integrity of a pre-prepare message by compare the
// round ID and the in-time computed hash.
// 验证PrePrepareMsg消息的正确性
func (ppm *PrePrepareMsg) ValidateIntegrity() bool {
	return ppm.Hash() == ppm.head.RoundID
}
