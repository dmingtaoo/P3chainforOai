package consensus

import (
	"fmt"
	"p3Chain/core/eles"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"
)

type TransactionGroup struct {
	Txs []eles.Transaction
}

func (tg *TransactionGroup) BackGroup() []eles.Transaction {
	return tg.Txs
}

func NewTransactionGroup(txs []*eles.Transaction) *TransactionGroup {
	plainTxs := make([]eles.Transaction, len(txs))
	for i := 0; i < len(txs); i++ {
		plainTxs[i] = *txs[i]
	}
	return &TransactionGroup{Txs: plainTxs}

}

// Try to decode a message payload into transaction. If failed, return nil.
// 从字节流(一条交易信息)中解包出 eles.Transaction 对象
func DeserializeTransaction(payload []byte) (*eles.Transaction, error) {
	tx := new(eles.Transaction)
	if err := rlp.DecodeBytes(payload, tx); err != nil {
		return nil, fmt.Errorf("failed in DeserializeTransaction")
	}
	return tx, nil
}

func DeserializeTransactionsSeries(payload []byte) ([]eles.Transaction, error) {
	txg := new(TransactionGroup)
	if err := rlp.DecodeBytes(payload, txg); err != nil {
		loglogrus.Log.Warnf("无法从byte流中解析出交易集合,err:%v\n", err)
		return nil, fmt.Errorf("failed in DeserializeTransaction")
	}
	return txg.BackGroup(), nil
}

// Try to decode a message payload into lower consensus message. If failed, return nil.
// 从字节流中解包出 WrappedLowerConsensusMessage 组内共识消息
func DeserializeLowerConsensusMsg(payload []byte) (*WrappedLowerConsensusMessage, error) {
	wlcm := new(WrappedLowerConsensusMessage)
	if err := rlp.DecodeBytes(payload, wlcm); err != nil {
		return nil, fmt.Errorf("failed in DeserializeLowerConsensusMsg")
	}
	return wlcm, nil
}

// Try to decode a message payload into upper consensus message. If failed, return nil.
// NOTE: Not used any more
// 从字节流中解包出 WrappedUpperConsensusMessage 上层共识消息
func DeserializeUpperConsensusMsg(payload []byte) (*UpperConsensusMessage, error) {
	ucm := new(UpperConsensusMessage)
	if err := rlp.DecodeBytes(payload, ucm); err != nil {
		return nil, fmt.Errorf("failed in DeserializeUpperConsensusMsg")
	}
	return ucm, nil
}
