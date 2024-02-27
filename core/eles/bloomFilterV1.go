package eles

import (
	"math"
	"math/big"
	"p3Chain/common"
	"strconv"
	"sync"
	"time"
)

// const BloomSize = 256
var seeds = []int64{1, 5, 7, 11, 17, 29, 37, 47, 59, 67}

type BloomFilter struct {
	bloomID     int
	bloomSize   uint32  // 布隆过滤器长度
	bloom       []byte  // 布隆过滤器的比特向量
	hashFuncNum int64   //哈希函数个数
	TxNum       uint32  //插入TxID个数
	p           float64 //误判概率
	poolMu      sync.RWMutex
	storedTime  time.Time
	lifeTime    time.Duration
}

// 自定义布隆过滤器
func CreateBloomFilter(bloomID int, bloomSize uint32, hashFuncNum int64, txNum uint32) *BloomFilter {
	return &BloomFilter{
		bloomID:     bloomID,
		bloomSize:   bloomSize,
		bloom:       make([]byte, bloomSize),
		hashFuncNum: hashFuncNum,
		TxNum:       txNum,
		p:           CalErrRate(txNum, hashFuncNum, bloomSize),
		storedTime:  time.Now(),
		lifeTime:    time.Duration(bloomID) * defaultLifeTime,
	}
}

// 从交易池添加TxID
func (bf *BloomFilter) Insert(tp *TransactionPool) {
	freshTxs := tp.RetrieveFreshTxs()
	for _, txs := range freshTxs {
		// v1, err := strconv.ParseInt(txs.Tx.TxID.Str(), 16, 64)
		// if err != nil {
		// 	fmt.Errorf("fail in Insert TxId: %v", err)
		// }
		v1 := new(big.Int).SetBytes(txs.Tx.TxID.Bytes())
		for i := int64(0); i < int64(bf.hashFuncNum); i++ {
			indexSet := bf.getIndexSet(v1, i).String()
			index, _ := strconv.Atoi(indexSet)
			bf.bloom[index] |= 1
		}
	}
}

// 添加一个TxID
func (bf *BloomFilter) Add(txID common.Hash) {
	// v1, err := strconv.ParseInt(txID.Str(), 16, 64) out of range(int64)
	// if err != nil {
	// 	  fmt.Println("fail in add TxId", err)
	// }
	// fmt.Println(txID.Bytes())
	v1 := new(big.Int).SetBytes(txID.Bytes())
	// fmt.Println(v1)

	for i := int64(0); i < int64(bf.hashFuncNum); i++ {
		indexSet := bf.getIndexSet(v1, i).String()
		index, _ := strconv.Atoi(indexSet)
		bf.bloom[index] |= 1
	}
}

// 检查交易信息是否存在
func (bf *BloomFilter) CheckTransaction(tx *Transaction) bool {
	// v1, err := strconv.ParseInt(txID.Str(), 16, 64)
	// if err != nil {
	// 	fmt.Errorf("fail in Check TxId: %v", err)
	// }
	v1 := new(big.Int).SetBytes(tx.TxID.Bytes())
	for i := int64(0); i < bf.hashFuncNum; i++ {
		indexSet := bf.getIndexSet(v1, i).String()
		index, _ := strconv.Atoi(indexSet)
		if bf.bloom[index]&1 == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) getIndexSet(x *big.Int, i int64) *big.Int {
	// return (x + seeds[i]) % (int64(len(bf.bloom)))
	seeds := new(big.Int).SetInt64(seeds[i])
	x_ := new(big.Int).Add(x, seeds)
	y_ := new(big.Int).SetInt64(int64(len(bf.bloom)))
	return new(big.Int).Mod(x_, y_)
}

// 计算BloomFilter误判率
func CalErrRate(TxNum uint32, hashFuncNum int64, bloomSize uint32) float64 {
	var N = float64(TxNum) * float64(hashFuncNum) / float64(bloomSize)
	return math.Pow(float64(1)-math.Pow(math.E, N*float64(-1)), float64(hashFuncNum))
}

// 布隆过滤器初始化
func InitBloom(BloomID int, TxNum int, p float64) *BloomFilter {
	// 哈希函数的个数
	k := -math.Log(p) * math.Log2E
	// 比特向量的位数
	bloomSize := float64(TxNum) * k * math.Log2E
	return &BloomFilter{
		bloomID:     BloomID,
		bloomSize:   uint32(bloomSize),
		bloom:       make([]byte, uint32(bloomSize)),
		hashFuncNum: int64(k),
		TxNum:       uint32(TxNum),
		p:           p,
		storedTime:  time.Now(),
		lifeTime:    time.Duration(BloomID) * defaultLifeTime,
	}
}

// 检查布隆过滤器是否该重置
func (bf *BloomFilter) CheckBloom() {
	bf.poolMu.Lock()
	defer bf.poolMu.Unlock()

	now := time.Now()
	threshold := now.Add(-bf.lifeTime)

	if bf.storedTime.Before(threshold) {
		bf.Clear()
		bf.storedTime = time.Now()
	}
}

// 清空布隆过滤器
func (bf *BloomFilter) Clear() {
	bf.bloom = make([]byte, bf.bloomSize)
}

// func (bf *BloomFilter) BloomValue(data []byte) {
// 	v1 := crc32.ChecksumIEEE(data)
// 	for i := 0; i < bf.hashFuncNum; i++ {
// 		indexSet := bf.getIndexSet(v1, i)
// 		bf.bloom[indexSet] |= 1
// 	}
// }
