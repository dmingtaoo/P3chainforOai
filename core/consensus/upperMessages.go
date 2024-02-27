package consensus

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/crypto"
	"p3Chain/rlp"
)

const (
	// for subnet leaders
	CODE_INIT      uint8 = 0x09
	CODE_BROADCAST uint8 = 0x10
	CODE_REQUIRE   uint8 = 0x11
	CODE_READY     uint8 = 0x12
	CODE_VOTE      uint8 = 0x13
	CODE_FINAL     uint8 = 0x14
	CODE_FIX       uint8 = 0x15
	CODE_EMPTY     uint8 = 0x16

	// for booters (order service providers)
	CODE_PREPARE uint8 = 0x20
	CODE_NEXT    uint8 = 0x21
	CODE_COMMIT  uint8 = 0x22

	CODE_UPDATEBLOCKVALIDATOR uint8 = 0x23
)

const (
	BlockInvalid byte = 0x00 // for VoteMessage
	BlockValid   byte = 0x01 // for VoteMessage
)

// upper层共识消息
type UpperConsensusMessage struct {
	Sender   common.NodeID //发送节点NodeID
	TypeCode uint8         //消息码(7种类型)
	BlockID  common.Hash   //区块ID
	Payload  []byte        //消息载荷
}

// 区块验证投票
type VoteMessage struct {
	Valid          byte        // 0x01 is valid, and 0x00 is invalid 区块是否有效
	BlockReceiptID common.Hash //区块收据ID
	Signature      []byte      //节点签名
}

// Booter节点发送给所有Leader节点,通知希望获取的下一个区块
type NextMessage struct {
	CurrentVersion common.Hash
	CurrentHeight  uint64
	NextBlockID    common.Hash
}

// 区块提交消息(Leader --> Booter),包含若干Leader节点的签名
type BlockCommitMessage struct {
	Repeat         uint8
	BlockReceiptID common.Hash
	Signatures     [][]byte
}

type BooterInitMessage struct {
	CurrentVesion common.Hash
	BlockHeight   uint64
}

type FinalMessage struct {
	CurrentVersion common.Hash
}

// 解码上层共识消息(根据消息码类型,解码返回对应的消息结构体)
func DecodeUpperConsensusMessage(ucm *UpperConsensusMessage) (interface{}, error) {
	switch ucm.TypeCode {
	case CODE_INIT: // 类型为Booter的初始化消息
		bim := new(BooterInitMessage)
		err := rlp.DecodeBytes(ucm.Payload, bim) //解码出当前头哈希
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper consensus message: type code: %v, %v", ucm.TypeCode, err)
		}
		return bim, nil

	case CODE_BROADCAST: //类型为广播消息
		block := new(eles.Block)
		err := rlp.DecodeBytes(ucm.Payload, block) //解码出广播获取的区块
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper consensus message: type code: %v, %v", ucm.TypeCode, err)
		}
		return block, nil //返回获取的区块
	case CODE_REQUIRE:
		blockID := ucm.BlockID
		return blockID, nil
	case CODE_READY:
		blockID := ucm.BlockID
		return blockID, nil

	case CODE_VOTE:
		vote := new(VoteMessage)
		err := rlp.DecodeBytes(ucm.Payload, vote) //解包出投票消息
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper consensus message: type code: %v, %v", ucm.TypeCode, err)
		}
		return vote, nil

	case CODE_PREPARE:
		blockID := ucm.BlockID
		return blockID, nil

	case CODE_NEXT:
		nextMsg := new(NextMessage)
		err := rlp.DecodeBytes(ucm.Payload, nextMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper consensus message: type code: %v, %v", ucm.TypeCode, err)
		}
		return nextMsg, nil

	case CODE_COMMIT:
		commitMsg := new(BlockCommitMessage)
		err := rlp.DecodeBytes(ucm.Payload, commitMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper consensus message: type code: %v, %v", ucm.TypeCode, err)
		}
		return commitMsg, nil

	case CODE_FINAL:
		finalMsg := new(FinalMessage)
		err := rlp.DecodeBytes(ucm.Payload, finalMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper consensus message: type code: %v, %v", ucm.TypeCode, err)
		}
		return finalMsg, nil

	case CODE_UPDATEBLOCKVALIDATOR:
		ubv := new(UpdateBlockValidator)
		err := rlp.DecodeBytes(ucm.Payload, ubv)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper consensus message: type code: %v, %v", ucm.TypeCode, err)
		}
		return ubv, nil

	default:
		return nil, fmt.Errorf("fail in decode upper consensus message: unknown code: %v", ucm.TypeCode)
	}
}

func CreateUpperMsg_Final(sender common.NodeID, currentVersion common.Hash) (*UpperConsensusMessage, error) {
	final := new(FinalMessage)
	final.CurrentVersion = currentVersion
	payload, err := rlp.EncodeToBytes(final)
	if err != nil {
		return nil, err
	}

	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_FINAL,
		BlockID:  currentVersion,
		Payload:  payload,
	}
	return ucm, nil
}

// Booter节点用于向其他Leader节点申请当前头区块Hash的消息
func CreateUpperMsg_Init(sender common.NodeID, version common.Hash, height uint64) (*UpperConsensusMessage, error) {
	bim := new(BooterInitMessage)
	bim.CurrentVesion = version
	bim.BlockHeight = height
	payload, err := rlp.EncodeToBytes(bim)
	if err != nil {
		return nil, err
	}

	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_INIT,
		Payload:  payload,
	}

	return ucm, nil
}

// crete a UpperConsensusMessage with type code BROADCAST
// 创建一条区块广播消息(UpperConsensusMessage消息码为CODE_BROADCAST)
func CreateUpperMsg_Broadcast(sender common.NodeID, block *eles.Block) (*UpperConsensusMessage, error) {
	payload, err := rlp.EncodeToBytes(block)
	if err != nil {
		return nil, err
	}
	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_BROADCAST,
		BlockID:  block.BlockID,
		Payload:  payload,
	}
	return ucm, nil
}

// crete a UpperConsensusMessage with type code REQUIRE
// 创建一条区块申请消息(消息码为CODE_REQUIRE,内容为申请进行排序的区块BlockID)
func CreateUpperMsg_Require(sender common.NodeID, blockID common.Hash) *UpperConsensusMessage {
	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_REQUIRE,
		BlockID:  blockID,
		Payload:  make([]byte, 0),
	}
	return ucm
}

// crete a UpperConsensusMessage with type code READY
// 创建一条区块Ready消息(消息码为CODE_READY,内容为已准备好的区块BlockID)
func CreateUpperMsg_Ready(sender common.NodeID, blockID common.Hash) *UpperConsensusMessage {
	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_READY,
		BlockID:  blockID,
		Payload:  make([]byte, 0),
	}
	return ucm
}

// create a UpperConsensusMessage with type code VOTE
// 为某一区块进行投票(包含自己的数字签名)
func CreateUpperMsg_Vote(sender common.NodeID, blockID common.Hash, vote *VoteMessage) (*UpperConsensusMessage, error) {
	payload, err := rlp.EncodeToBytes(vote)
	if err != nil {
		return nil, err
	}
	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_VOTE,
		BlockID:  blockID,
		Payload:  payload,
	}
	return ucm, nil
}

// crete a UpperConsensusMessage with type code PREPARE
func CreateUpperMsg_Prepare(sender common.NodeID, blockID common.Hash) *UpperConsensusMessage {
	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_PREPARE,
		BlockID:  blockID,
		Payload:  make([]byte, 0),
	}
	return ucm
}

// crete a UpperConsensusMessage with type code NEXT
// BlockID为下一区块ID,载荷为当前区块ID
func CreateUpperMsg_Next(sender common.NodeID, nextBlockID, currentVersion common.Hash, currentHeight uint64) (*UpperConsensusMessage, error) {
	nextMsg := NextMessage{
		CurrentVersion: currentVersion,
		CurrentHeight:  currentHeight,
		NextBlockID:    nextBlockID,
	}
	payload, err := rlp.EncodeToBytes(&nextMsg)
	if err != nil {
		return nil, err
	}
	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_NEXT,
		BlockID:  nextBlockID,
		Payload:  payload,
	}
	return ucm, nil
}

// crete a UpperConsensusMessage with type code COMMIT
// 载荷为区块最终的上层共识结果(包含获取的所有签名)
func CreateUpperMsg_COMMIT(sender common.NodeID, blockID common.Hash, blockCommit *BlockCommitMessage) (*UpperConsensusMessage, error) {
	payload, err := rlp.EncodeToBytes(blockCommit)
	if err != nil {
		return nil, err
	}

	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_COMMIT,
		BlockID:  blockID,
		Payload:  payload,
	}
	return ucm, nil
}

// 新加入到upper层的Leader节点通过此消息通知upper层其他Leader更新自己的区块验证器
type UpdateBlockValidator struct {
	GroupID    string      // 分区的组ID
	SenderHash common.Hash // 发送者NodeID求出的Hash,用于数字签名
	Sigs       [][]byte    // 发送者搜集到的数字签名
}

// 进行view-change时会用到,故障分区的新Leader通过此消息通知其他分区的Leader根据新的netManger(因此上层完成更新必须等待separator层完成更新)更新自己的upperValidator
func CreateUpperMsg_UpdateBlockValidator(sender common.NodeID, group string, blockID common.Hash, sigs [][]byte) (*UpperConsensusMessage, error) {

	ubv := new(UpdateBlockValidator)
	ubv.GroupID = group
	serialized, _ := rlp.EncodeToBytes(sender)
	senderHash := crypto.Sha3Hash(serialized)
	ubv.SenderHash = senderHash
	ubv.Sigs = sigs

	payload, err := rlp.EncodeToBytes(ubv)
	if err != nil {
		return nil, err
	}

	ucm := &UpperConsensusMessage{
		Sender:   sender,
		TypeCode: CODE_UPDATEBLOCKVALIDATOR,
		BlockID:  blockID,
		Payload:  payload,
	}
	return ucm, nil
}
