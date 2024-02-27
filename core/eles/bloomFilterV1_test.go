package eles

import (
	"fmt"
	"p3Chain/common"
	"testing"
	"time"
)

func TestBloomFilter(t *testing.T) {
	var id0 = common.HexToHash("21ec00578b7fdd85ccf654ee162af4a5eef16f88ab94c1fa40957f7e7cdf09a6")
	var id1 = common.HexToHash("29a43622ca9fbf1facf8ac513509ce44976fe858f2cef1e2a52fd2a3a9834fb9")
	var id2 = common.HexToHash("471c0e248ae1fa39068c5d87c7ae388ffb6f7d713fd1fba40748229a10fd4a7f")
	// var id3 = common.HexToHash("693ae584ce14b5ed667415d3445c6aa11b8ee7484fe7e6ddeb2832584f5a9246")
	transaction := &[3]Transaction{
		{
			TxID: id0,
		},
		{
			TxID: id1,
		},
		{
			TxID: id2,
		},
	}
	// transaction_test := Transaction{
	// 	TxID: id3,
	// }

	tp := &TransactionPool{
		PoolID:     NormalTxPool,
		staleLimit: defaultStaleThreshold,
		holdTime:   defaultExpirationDelay,
		Txs:        make(map[common.Hash]*TransactionState),
	}

	// matchTransaction := func(tx *Transaction) bool {
	// 	if _, ok := tp.Txs[tx.TxID]; !ok {
	// 		return false
	// 	}
	// 	return true
	// }
	// tp.matchFunc = matchTransaction

	for _, v := range transaction {
		tp.Txs[v.TxID] = NewTransactionState(&v)
		// if err != nil {
		// 	fmt.Errorf("fail in add transaction[%v]", key)
		// 	fmt.Println("fail in add transaction")
		// }
	}
	fmt.Println(tp.Txs)
	// 	fmt.Println("Txs[id0]:", *tp.Txs[id0])
	// 	fmt.Println("Txs[id1]:", *tp.Txs[id1])
	// 	fmt.Println("Txs[id2]:", *tp.Txs[id2])

	fmt.Println("---------------------------------------InitBloom----------------------------------------")
	//初始化3个布隆过滤器
	bf := [3]*BloomFilter{
		InitBloom(1, 3, 0.01),
		InitBloom(2, 3, 0.1),
		InitBloom(3, 3, 0.2),
	}

	fmt.Println("---------------------------------------Add TxID in Bloom----------------------------------------")
	for key, value := range bf {
		// value.Insert(tp) //无法从交易池中添加Tx
		for k, v := range transaction {
			value.Add(v.TxID) //添加TxID
			result := bf[key].CheckTransaction(&v)
			fmt.Println("is transaction[", k, "]in bloom[", key, "]:", result)
		}
		fmt.Println("Bloom:", value)
	}

	fmt.Println("---------------------------------------Clear/Reset Bloom-----------------------------------------")
	time.Sleep(30 * time.Second) //重置bloom[0]
	bf[0].CheckBloom()
	fmt.Println("bloom[ 0 ]:", bf[0])

	for _, v := range transaction { //如果TxID存在于另外两个bloom 则不再往bloom1里添加TxID
		if bf[1].CheckTransaction(&v) {
			if bf[2].CheckTransaction(&v) {
				continue
			}
		}
		bf[0].Add(v.TxID) //往bloom[0]添加TxID
	}
	time.Sleep(30 * time.Second) //重置bloom[1]
	bf[1].CheckBloom()
	bf[2].CheckBloom()
	fmt.Println("---------------------------------------result-----------------------------------------")
	for key, value := range bf {
		fmt.Println("bloom[", key, "]:", value)
	}
}
