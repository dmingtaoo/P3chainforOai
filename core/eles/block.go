package eles

import (
	"bytes"
	"fmt"
	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/rlp"
)

type Block struct {
	// Subnet commit
	BlockID      common.Hash
	Subnet       []byte
	Leader       common.Address
	Version      common.Hash // help to validate whether transaction lifetime over
	Nonce        uint8       // the index of this block in the same version
	Transactions []Transaction
	SubnetVotes  []SubNetSignature

	// Block ordering service
	PrevBlock common.Hash

	// Help to check whether the storage is consistency. Could be state root or storage root.
	CheckRoot common.Hash // not used now

	// Leader validation
	Receipt     BlockReceipt
	LeaderVotes []LeaderSignature // Sign the receipt ID of block
}

type WrapBlock struct {
	RawBlock   *Block        //区块实体
	OriginPeer common.NodeID //区块提供者的标识符
	Number     uint64        //该区块在目标节点上的编号
	ReceivedAt string        //接收到该区块的时间戳
}
type WrapBlocks []*WrapBlock

type SubNetSignature []byte
type LeaderSignature []byte

// the block receipt ID is the hash of a concatenated text of BlockID, PrevBlock, TxReceipt and WriteSet
type BlockReceipt struct {
	ReceiptID common.Hash // the summary of block receipt and blockID
	TxReceipt []TransactionReceipt
	WriteSet  []WriteEle
}

var NullReceipt = BlockReceipt{
	ReceiptID: common.Hash{},
	TxReceipt: make([]TransactionReceipt, 0),
	WriteSet:  make([]WriteEle, 0),
}

type TransactionReceipt struct {
	Valid  byte     // 0 is false, 1 is right
	Result [][]byte // back some information of the result
}

type WriteEle struct {
	ValueAddress common.WsKey
	Value        []byte
}

type ReadEle struct {
	ValueAddress common.WsKey
	Value        []byte
}

func (b *Block) Hash() (common.Hash, error) {
	tempBlock := *b
	tempBlock.BlockID = common.Hash{}
	tempBlock.SubnetVotes = make([]SubNetSignature, 0)
	tempBlock.PrevBlock = common.Hash{}
	tempBlock.CheckRoot = common.Hash{}
	tempBlock.Receipt = NullReceipt
	tempBlock.LeaderVotes = make([]LeaderSignature, 0)

	summary, err := rlp.EncodeToBytes(&tempBlock)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Sha3Hash(summary), nil
}

// Compute and set BlockID
func (b *Block) ComputeBlockID() (common.Hash, error) {
	if b.BlockID == (common.Hash{}) {
		hash, err := b.Hash()
		if err != nil {
			return common.Hash{}, err
		}
		b.BlockID = hash
	}

	return b.BlockID, nil

}

// compute the receiptID
// NOTE: should first compute blockID and set the prevblock
func (b *Block) ComputeReceiptID() (common.Hash, error) {
	var receiptID common.Hash
	textTxReceipt, err := rlp.EncodeToBytes(&b.Receipt.TxReceipt)
	if err != nil {
		return receiptID, err
	}
	textWriteSet, err := rlp.EncodeToBytes(&b.Receipt.WriteSet)
	if err != nil {
		return receiptID, err
	}
	text := bytes.Join([][]byte{b.BlockID[:], b.PrevBlock[:], textTxReceipt, textWriteSet}, []byte{})
	receiptID = crypto.Sha3Hash(text)
	return receiptID, nil
}

// back the block receipt ID. If it is not set, set it and then back
// NOTE: should first compute blockID and set the prevblock
func (b *Block) BackReceiptID() (common.Hash, error) {
	if b.Receipt.ReceiptID != (common.Hash{}) {
		return b.Receipt.ReceiptID, nil
	}
	receiptID, err := b.ComputeReceiptID()
	if err != nil {
		return receiptID, err
	}
	b.Receipt.ReceiptID = receiptID
	return b.Receipt.ReceiptID, nil
}

// set the block receipt with the given transaction receipts and write set
func (b *Block) SetReceipt(prvBlock common.Hash, txsRcp []TransactionReceipt, wrtSet []WriteEle) (common.Hash, error) {
	if len(txsRcp) != len(b.Transactions) {
		return common.Hash{}, fmt.Errorf("fail in SetReceipt: not matched transaction receipts")
	}
	b.PrevBlock = prvBlock
	b.Receipt.TxReceipt = txsRcp
	b.Receipt.WriteSet = wrtSet
	rcpID, err := b.BackReceiptID()
	if err != nil {
		return common.Hash{}, fmt.Errorf("fail in SetReceipt: %v", err)
	}
	return rcpID, nil
}

func (b *Block) SetLeaderVotes(plainSignatures [][]byte) {
	signatures := make([]LeaderSignature, len(plainSignatures))
	for i := 0; i < len(signatures); i++ {
		signatures[i] = plainSignatures[i]
	}
	b.LeaderVotes = signatures
}

func (b *Block) BackTransactionIDs() []common.Hash {
	txIDs := make([]common.Hash, len(b.Transactions))
	for i := 0; i < len(txIDs); i++ {
		txIDs[i] = b.Transactions[i].TxID
	}
	return txIDs
}
