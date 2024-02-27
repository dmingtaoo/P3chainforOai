package consensus

import (
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/core/separator"
	"p3Chain/rlp"
	"testing"
)

var (
	testTx = &eles.Transaction{
		TxID:      common.Hash{},
		Sender:    common.Address{},
		Nonce:     0,
		Version:   common.Hash{},
		LifeTime:  0,
		Signature: nil,
		Contract:  common.Address{},
		Function:  common.Address{},
		Args:      nil,
		CheckList: nil,
	}
)

func TestDeserializeTransactionsSeries(t *testing.T) {
	txg := NewTransactionGroup([]*eles.Transaction{testTx})
	payload, err := rlp.EncodeToBytes(txg)
	if err != nil {
		t.Fatalf("fail in rlp")
	}
	msgcode := separator.TransactionsCode
	msg := &separator.Message{
		MsgCode: uint64(msgcode), //消息类型为transactionCode
		NetID:   "test",          //目标子网
		From:    common.NodeID{}, //来源,也就是当前节点的NodeID
		PayLoad: payload,         //交易
	}

	txs, err := DeserializeTransactionsSeries(msg.PayLoad)
	if err != nil {
		t.Fatalf("fail in decode")
	}
	if txs[0].TxID != testTx.TxID {
		t.Logf("wrong tx")
	}
}
