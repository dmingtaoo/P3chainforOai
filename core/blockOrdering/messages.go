package blockOrdering

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/rlp"
)

const (
	// for order agent
	CODE_UPLD uint8 = 0x10 // upload block
	CODE_SIGN uint8 = 0x11
	CODE_DONE uint8 = 0x12

	// for order server
	CODE_NEXT uint8 = 0x20
	CODE_CMIT uint8 = 0x21
	CODE_RCVY uint8 = 0x22 // for recovery

	// for initialization
	CODE_INIT uint8 = 0x30
	CODE_STAT uint8 = 0x31 // now state
	CODE_SYNC uint8 = 0x32 // notify the agent to synchronize to catch the latest height

	CODE_UPDATEBLOCKVALIDATOR uint8 = 0x40
)

const (
	BlockInvalid byte = 0x00 // for VoteMessage
	BlockValid   byte = 0x01 // for VoteMessage
)

type BlockOrderMessage struct {
	Sender   common.NodeID
	TypeCode uint8
	Payload  []byte
}

// for order service agent

type UploadMessage struct {
	ExpireTime uint64
	Block      eles.Block
}

type SignMessage struct {
	Valid     byte
	BlockID   common.Hash
	ReceiptID common.Hash
	Signature []byte
}

type DoneMessage struct {
	Valid   byte
	BlockID common.Hash
	Height  uint64
}

// for order service server

type NextMessage struct {
	Version common.Hash // previous block hash
	Height  uint64      // previous block height
	Block   eles.Block
}

type CommitMessage struct {
	Height     uint64      // now height
	BlockID    common.Hash // now block hash
	ReceiptID  common.Hash
	Signatures [][]byte
}

type RecoveryMessage struct {
	RepeatCount uint8
	Version     common.Hash // previous block hash
	Height      uint64      // previous block height
	Block       eles.Block
}

// for initialization

type InitMessage struct {
	Version common.Hash // known the latest version
	Height  uint64      // known the latest height
}

type StateMessage struct {
	Version common.Hash // now latest version
	Height  uint64      // now latest height
}

type SynchronizeMessage struct {
	Version common.Hash // the latest version
	Height  uint64      // the latest height
}

// 新加入到upper层的Leader节点通过此消息通知upper层其他Leader更新自己的区块验证器
type UpdateBlockValidator struct {
	GroupID    string      // 分区的组ID
	SenderHash common.Hash // 发送者NodeID求出的Hash,用于数字签名
	Sigs       [][]byte    // 发送者搜集到的数字签名
}

func DeserializeBlockOrderMsg(payload []byte) (*BlockOrderMessage, error) {
	bom := new(BlockOrderMessage)
	if err := rlp.DecodeBytes(payload, bom); err != nil {
		return nil, fmt.Errorf("failed in DeserializeBlockOrderMsg: %v", err)
	}
	return bom, nil
}

func CreateBlockOrderMsg(element any, sender common.NodeID, typeCode uint8) (*BlockOrderMessage, error) {
	val, err := rlp.EncodeToBytes(element)
	if err != nil {
		return nil, fmt.Errorf("failed in encode payload of BlockOrderMessage of CreateBlockOrderMsg: %v", err)
	}
	bom := &BlockOrderMessage{
		Sender:   sender,
		TypeCode: typeCode,
		Payload:  val,
	}
	return bom, nil
}

func RetrievePayload(bom *BlockOrderMessage) (any, error) {
	switch bom.TypeCode {
	case CODE_UPLD:
		uplMsg := new(UploadMessage)
		err := rlp.DecodeBytes(bom.Payload, uplMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return uplMsg, nil

	case CODE_SIGN:
		sigMsg := new(SignMessage)
		err := rlp.DecodeBytes(bom.Payload, sigMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return sigMsg, nil

	case CODE_DONE:
		donMsg := new(DoneMessage)
		err := rlp.DecodeBytes(bom.Payload, donMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return donMsg, nil

	case CODE_NEXT:
		nexMsg := new(NextMessage)
		err := rlp.DecodeBytes(bom.Payload, nexMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return nexMsg, nil

	case CODE_CMIT:
		cmiMsg := new(CommitMessage)
		err := rlp.DecodeBytes(bom.Payload, cmiMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return cmiMsg, nil

	case CODE_RCVY:
		rcvMsg := new(RecoveryMessage)
		err := rlp.DecodeBytes(bom.Payload, rcvMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return rcvMsg, nil

	case CODE_INIT:
		iniMsg := new(InitMessage)
		err := rlp.DecodeBytes(bom.Payload, iniMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return iniMsg, nil
	case CODE_STAT:
		staMsg := new(StateMessage)
		err := rlp.DecodeBytes(bom.Payload, staMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return staMsg, nil

	case CODE_SYNC:
		synMsg := new(SynchronizeMessage)
		err := rlp.DecodeBytes(bom.Payload, synMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return synMsg, nil

	case CODE_UPDATEBLOCKVALIDATOR:
		ubvMsg := new(UpdateBlockValidator)
		err := rlp.DecodeBytes(bom.Payload, ubvMsg)
		if err != nil {
			return nil, fmt.Errorf("fail in decode upper block order message: type code: %v, %v", bom.TypeCode, err)
		}
		return ubvMsg, nil

	default:
		return nil, fmt.Errorf("unknown typecode: %d is got, can be parsed into nothing", bom.TypeCode)
	}
}
