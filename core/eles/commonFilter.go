package eles

import (
	"p3Chain/common"
	"sync"
)

const (
	PoolCapacity       = 10                            // 池容量(容纳区块数)
	PoolNums           = 6                             // 池数量
	MaxValidVersionLag = (PoolNums - 1) * PoolCapacity // if a transaction lifetime is 0, it is promised to be regarded as valid less than this max version lag
)

type TransactionFilter struct {
	poolMutex sync.RWMutex

	versionPool  [MaxValidVersionLag]common.Hash // 存储本地最近上链的 MaxValidVersionLag 个区块的Hash,可以想象为一个环结构
	currentIndex int

	filterPools    [PoolNums]map[common.Hash]bool
	cachePoolIndex int
}

func NewTransactionFilter() *TransactionFilter {
	trasactionFilter := new(TransactionFilter)
	for i := 0; i < PoolNums; i++ {
		trasactionFilter.filterPools[i] = make(map[common.Hash]bool)
	}
	return trasactionFilter
}

func (tf *TransactionFilter) InputTxs(currentBlockHeight int, currentVersion common.Hash, txs []Transaction) {
	tf.poolMutex.Lock()
	defer tf.poolMutex.Unlock()

	tf.currentIndex = currentBlockHeight % MaxValidVersionLag
	tf.versionPool[tf.currentIndex] = currentVersion

	if (currentBlockHeight % PoolCapacity) == 0 {
		tf.cachePoolIndex = (tf.cachePoolIndex + 1) % PoolNums
		tf.filterPools[tf.cachePoolIndex] = make(map[common.Hash]bool)
	}

	for i := 0; i < len(txs); i++ {
		tf.filterPools[tf.cachePoolIndex][txs[i].TxID] = true
	}
}

// if no repeat transaction found, return true
// if repeat transaction found or the transaction is over MaxValidVersionLag, return false
func (tf *TransactionFilter) IsTxUnique(tx Transaction) bool {
	tf.poolMutex.RLock()
	defer tf.poolMutex.RUnlock()

	if !tf.IsVersionIn(tx.Version) {
		return false
	}

	for i := 0; i < PoolNums; i++ {
		if _, ok := tf.filterPools[i][tx.TxID]; ok {
			return false
		}
	}
	return true
}

// if find repeat transaction, return true
// if repeat transaction found or the transaction is over MaxValidVersionLag, return false
func (tf *TransactionFilter) HasRepeatTxs(txs []Transaction) bool {
	tf.poolMutex.RLock()
	defer tf.poolMutex.RUnlock()
	for i := 0; i < len(txs); i++ {
		if !tf.IsVersionIn(txs[i].Version) {
			return true
		}
		for j := 0; j < PoolNums; j++ {
			if _, ok := tf.filterPools[j][txs[i].TxID]; ok {
				return true
			}
		}
	}
	return false
}

func (tf *TransactionFilter) IsVersionIn(currentVersion common.Hash) bool {
	for i := 0; i < MaxValidVersionLag; i++ {
		if len(tf.versionPool) < (tf.currentIndex-i)%MaxValidVersionLag-1 {
			continue
		}
		if tf.versionPool[(tf.currentIndex-i)%MaxValidVersionLag] == currentVersion { //TODO: 索引会出现-1
			return true
		}
	}
	return false
}
