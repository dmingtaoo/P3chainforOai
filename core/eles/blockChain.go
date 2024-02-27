package eles

import (
	"p3Chain/common"

	"p3Chain/database"
	"p3Chain/dper/transactionCheck"
	"reflect"

	"fmt"
	"p3Chain/rlp"
	"sync"
	"time"

	loglogrus "p3Chain/log_logrus"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

const (
	maxQueryDepth = 100 //查找区块时的最大迭代查询深度
)

var upLinkedTxCacheLimit time.Duration = 3 * time.Second

// 每插入一个区块(或者插入若干个？)重新向levelDB中更新此代表本地区块链状态的变量
type ChainState struct {
	GenesisHash common.Hash
	CurrentHash common.Hash
	//CurrentBlock *WrapBlock
	Height        uint64
	LocalBlockNum uint64
	ArchiveHeight uint64

	TxSum uint64
}

type TransactionEvent struct {
	UpLinkTime time.Time
	Expired    bool
	Receipt    TransactionReceipt
}

type BlockChain struct {
	GenesisHash   common.Hash //创世区块hash
	CurrentHash   common.Hash //当前头区块hash
	CurrentBlock  *WrapBlock  //头区块
	Height        uint64      //当前区块链的高度
	ArchiveHeight uint64      //已完成归档的区块高度
	LocalBlockNum uint64      //本地已存储区块数
	TxSum         uint64      //交易总数
	Database      database.Database

	databaseMutex sync.RWMutex // 插入操作需要的锁,用于锁住数据库

	chainStatusMutex sync.RWMutex // 访问/修改区块链状态量时需要的锁

	TransactionFilter *TransactionFilter
	TxCheckPool       *transactionCheck.TxCheckPool
	ExecuteBlock      func(block *Block) error

	archiveMode bool // 是否启动了归档模式

	LatestUpLinkedTransaction map[common.Hash]*TransactionEvent // 缓存最近上链的若干笔交易哈希,作用是检查交易是否上链,一旦交易被检查完成对应交易从map中清除,因此理论上容量是有上限的
	latestMutex               sync.RWMutex
}

// 在本地创建一个BlockChain对象,存储所有区块
// create a blockchain by initializing a memory database
func InitBlockChain_Memory() (*BlockChain, error) {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}
	blockChain := new(BlockChain)
	blockChain.Height = 0
	blockChain.CurrentHash = common.Hash{}
	blockChain.Database = database.NewSimpleLDB("blockChain", db) //创建一个维护区块链的数据库

	return blockChain, nil
}

func InitBlockChain(db database.Database, TxCheckPool *transactionCheck.TxCheckPool, TxFilter *TransactionFilter, archiveMode bool) *BlockChain {
	if db == nil {
		loglogrus.Log.Errorf("BlockChain failed: LevelDB is nil, Unable to store data!\n")
		panic("BlockChain failed: LevelDB is nil, Unable to store data!")
	}

	chainState := []byte("chainState")
	byteStream, err := db.Get(chainState)
	cs := new(ChainState)
	if err != nil {
		loglogrus.Log.Infof("BlockChain: Hello,First use of p3Chain!\n")

	} else {
		loglogrus.Log.Infof("BlockChain: Restoring historical data(currentVersion,currentHeight and so on)...\n")
		if len(byteStream) != 0 {
			cs, err = ChainStateDeSerialize(byteStream)
			if err != nil {
				loglogrus.Log.Warnf("BlockChain Warn: Data loss, waiting to get relevant data from Leader...\n")
				// TODO: 发生了数据丢失，可能需要从其他节点申请获取chainState相关数据
			}
		} else {
			loglogrus.Log.Warnf("BlockChain Warn: Data loss, waiting to get relevant data from Leader...\n")
			// TODO: 发生了数据丢失，可能需要从其他节点申请获取chainState相关数据
		}
	}

	blockChain := new(BlockChain)
	blockChain.Height = cs.Height
	blockChain.LocalBlockNum = cs.LocalBlockNum
	blockChain.ArchiveHeight = cs.ArchiveHeight
	blockChain.TxSum = cs.TxSum
	blockChain.CurrentHash = cs.CurrentHash
	blockChain.GenesisHash = cs.GenesisHash

	blockChain.Database = db
	blockChain.TxCheckPool = TxCheckPool
	blockChain.TransactionFilter = TxFilter
	blockChain.archiveMode = archiveMode

	if blockChain.TransactionFilter != nil {
		blocks := blockChain.GetBlocksFromHash(blockChain.CurrentHash, 50)
		for i := len(blocks) - 1; i >= 0; i-- {
			blockChain.TransactionFilter.InputTxs(int(blocks[i].Number), blocks[i].RawBlock.BlockID, blocks[i].RawBlock.Transactions)
		}
	}

	blockChain.LatestUpLinkedTransaction = make(map[common.Hash]*TransactionEvent)

	go blockChain.ExpireTransactionEvent()

	return blockChain
}

// 将区块实体变为rlp字节流
func BlockSerialize(block *WrapBlock) []byte {
	enc, _ := rlp.EncodeToBytes(block)
	return enc
}

// 从rlp字节流解码出区块实体
func BlockDeSerialize(byteStream []byte) (*WrapBlock, error) {
	wb := new(WrapBlock)
	if err := rlp.DecodeBytes(byteStream, wb); err != nil {
		return nil, fmt.Errorf("failed in DeserializeBlock")
	} else {
		// wb.ReceivedAt = time.Now()
		return wb, nil
	}
}

func ChainStateSerialize(chainState *ChainState) []byte {
	enc, _ := rlp.EncodeToBytes(chainState)
	return enc
}

func ChainStateDeSerialize(byteStream []byte) (*ChainState, error) {
	cs := new(ChainState)
	if err := rlp.DecodeBytes(byteStream, cs); err != nil {
		return nil, fmt.Errorf("failed in DeserializeChainState")
	} else {
		return cs, nil
	}
}

func (chain *BlockChain) BackBlockStorageMux() *sync.RWMutex {
	return &chain.databaseMutex
}

func (chain *BlockChain) ResetLocalBlockNum() {
	chain.chainStatusMutex.Lock()
	defer chain.chainStatusMutex.Unlock()
	chain.LocalBlockNum = 0
}

func (chain *BlockChain) BackLocalBlockNum() uint64 {
	chain.chainStatusMutex.RLock()
	defer chain.chainStatusMutex.RUnlock()
	return chain.LocalBlockNum
}

func (chain *BlockChain) BackTxSum() uint64 {
	chain.chainStatusMutex.RLock()
	defer chain.chainStatusMutex.RUnlock()
	return chain.TxSum
}

// 获取当前链的头区块hash
func (chain *BlockChain) CurrentVersion() common.Hash {
	chain.chainStatusMutex.RLock()
	defer chain.chainStatusMutex.RUnlock()
	return chain.CurrentHash
}

// get current blockchain height
func (chain *BlockChain) CurrentHeight() uint64 {
	chain.chainStatusMutex.RLock()
	defer chain.chainStatusMutex.RUnlock()

	return chain.Height
}

// 新增,获取本地区块链高度,头区块hash和创世区块hash
func (chain *BlockChain) GetChainStatus() (Height uint64, head common.Hash, genesis common.Hash) {
	chain.chainStatusMutex.RLock()
	defer chain.chainStatusMutex.RUnlock()
	return chain.Height, chain.CurrentHash, chain.GenesisHash
}

// 插入一个区块
func (chain *BlockChain) InsertChain(block *WrapBlock) error {

	loglogrus.Log.Infof("[Block Chain] InsertChain 开始启动, block Hash:%x\n", block.RawBlock.BlockID)

	chain.databaseMutex.Lock()
	key := block.RawBlock.BlockID[:]
	value := BlockSerialize(block) //在数据库存储时的value值
	loglogrus.Log.Infof("[Block Chain] InsertChain 进行新区块的插入, block Hash:%x\n", block.RawBlock.BlockID)
	if err := chain.Database.Put(key, value); err != nil { //以键值对形式插入到数据库中
		chain.databaseMutex.Unlock()
		return err
	}
	chain.databaseMutex.Unlock()
	loglogrus.Log.Infof("[Block Chain] InsertChain 完成新区块的插入, block Hash:%x\n", block.RawBlock.BlockID)

	loglogrus.Log.Infof("[Block Chain] InsertChain 准备执行区块内的交易, block Hash:%x\n", block.RawBlock.BlockID)

	if chain.ExecuteBlock != nil {
		err := chain.ExecuteBlock(block.RawBlock)
		if err != nil {
			return err
		}
	}
	loglogrus.Log.Infof("[Block Chain] InsertChain 完成对区块内交易的执行, block Hash:%x\n", block.RawBlock.BlockID)

	chain.TxResultCheck(block.RawBlock)

	chain.chainStatusMutex.Lock()
	if (chain.CurrentHash == common.Hash{}) { //说明本次插入的区块是创世区块
		chain.GenesisHash = block.RawBlock.BlockID //更新创世区块hash
	}
	chain.CurrentBlock = block                              // 更新头区块为当前新插入的区块
	chain.CurrentHash = block.RawBlock.BlockID              // 更新最新区块hash
	chain.Height = chain.Height + 1                         // 区块链高度+1
	chain.LocalBlockNum = chain.LocalBlockNum + 1           // 本地存储区块数 + 1
	chain.TxSum += uint64(len(block.RawBlock.Transactions)) // 交易数递增
	chain.chainStatusMutex.Unlock()

	loglogrus.Log.Infof("[Block Chain] InsertChain 准备更新交易过滤器, block Hash:%x\n", block.RawBlock.BlockID)
	if chain.TransactionFilter != nil {
		chain.TransactionFilter.InputTxs(int(chain.Height), chain.CurrentHash, block.RawBlock.Transactions) // put all the transaction into the filter to protect repeating
	}
	loglogrus.Log.Infof("[Block Chain] InsertChain 完成对交易过滤器的更新, block Hash:%x\n", block.RawBlock.BlockID)

	cs := new(ChainState)
	//cs.CurrentBlock = chain.CurrentBlock
	cs.CurrentHash = chain.CurrentHash
	cs.GenesisHash = chain.GenesisHash
	cs.Height = chain.Height
	cs.LocalBlockNum = chain.LocalBlockNum
	cs.TxSum = chain.TxSum

	loglogrus.Log.Infof("[Block Chain] InsertChain 准备更新区块链状态量, block Hash:%x , block Height:%d \n", cs.CurrentHash, cs.Height)
	chain.databaseMutex.Lock()
	key = []byte("chainState")
	value = ChainStateSerialize(cs)
	if err := chain.Database.Put(key, value); err != nil { //额外存储chainState
		chain.databaseMutex.Unlock()
		return err
	}
	chain.databaseMutex.Unlock()
	loglogrus.Log.Infof("[Block Chain] InsertChain 完成对区块链状态量的更新, block Hash:%x , block Height:%d \n", cs.CurrentHash, cs.Height)

	go func() {
		chain.latestMutex.Lock()
		now := time.Now()
		for txIndex, tx := range block.RawBlock.Transactions {
			receipt := block.RawBlock.Receipt.TxReceipt[txIndex]
			chain.LatestUpLinkedTransaction[tx.TxID] = &TransactionEvent{UpLinkTime: now, Expired: false, Receipt: receipt}
		}
		chain.latestMutex.Unlock()
	}()

	return nil
}

// 插入一批区块(返回成功插入的区块的个数)
func (chain *BlockChain) InsertChains(blocks *WrapBlocks) (int, error) {
	index := 0
	block := &WrapBlock{}

	for index, block = range *blocks {
		if chain.FindVersion(block.RawBlock.BlockID) {
			continue
		}
		if err := chain.InsertChain(block); err != nil {
			return index, err
		}
	}
	return index + 1, nil
}

// commit a block use the given nodeID
func (chain *BlockChain) CommitBlock(selfNodeID common.NodeID, block *Block) error {

	loglogrus.Log.Infof("[Block Chain] CommitBlock开始启动, block Hash:%x\n", block.BlockID)

	if chain.CurrentHash != block.PrevBlock {
		loglogrus.Log.Warnf("fail in CommitBlock: unmatched current version\n")
		return fmt.Errorf("fail in CommitBlock: unmatched current version")
	}
	wb := &WrapBlock{
		RawBlock:   block,
		OriginPeer: selfNodeID,
		Number:     chain.Height,
		ReceivedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	chain.databaseMutex.Lock()
	loglogrus.Log.Infof("[Block Chain] 测试1: CommitBlock可以拿到databaseMutex的写锁\n")
	key := wb.RawBlock.BlockID[:]
	value := BlockSerialize(wb) //在数据库存储时的value值
	loglogrus.Log.Infof("[Block Chain] CommitBlock进行新区块的插入, block Hash:%x\n", block.BlockID)
	if err := chain.Database.Put(key, value); err != nil { //以键值对形式插入到数据库中
		fmt.Println(err)
		chain.databaseMutex.Unlock()
		return err
	}
	chain.databaseMutex.Unlock()
	loglogrus.Log.Infof("[Block Chain] CommitBlock完成新区块的插入, block Hash:%x\n", block.BlockID)

	loglogrus.Log.Infof("[Block Chain] CommitBlock准备执行区块内的交易, block Hash:%x\n", block.BlockID)
	if chain.ExecuteBlock != nil {
		err := chain.ExecuteBlock(wb.RawBlock)
		if err != nil {
			return err
		}
	}
	loglogrus.Log.Infof("[Block Chain] CommitBlock完成对区块内交易的执行, block Hash:%x\n", block.BlockID)
	chain.TxResultCheck(wb.RawBlock)

	chain.chainStatusMutex.Lock()
	if (chain.CurrentHash == common.Hash{}) { //说明本次插入的区块是创世区块
		chain.GenesisHash = wb.RawBlock.BlockID //更新创世区块hash
	}
	chain.CurrentBlock = wb                       // 更新头区块为当前新插入的区块
	chain.CurrentHash = wb.RawBlock.BlockID       // 更新最新区块hash
	chain.Height = chain.Height + 1               // 区块链高度+1
	chain.LocalBlockNum = chain.LocalBlockNum + 1 // 本地存储区块数 + 1
	chain.TxSum += uint64(len(wb.RawBlock.Transactions))
	chain.chainStatusMutex.Unlock()

	loglogrus.Log.Infof("[Block Chain] CommitBlock准备更新交易过滤器, block Hash:%x\n", block.BlockID)
	if chain.TransactionFilter != nil {
		chain.TransactionFilter.InputTxs(int(chain.Height), chain.CurrentHash, wb.RawBlock.Transactions) // put all the transaction into the filter to protect repeating
	}
	loglogrus.Log.Infof("[Block Chain] CommitBlock完成对交易过滤器的更新, block Hash:%x\n", block.BlockID)

	cs := new(ChainState)
	//cs.CurrentBlock = chain.CurrentBlock
	cs.CurrentHash = chain.CurrentHash
	cs.GenesisHash = chain.GenesisHash
	cs.Height = chain.Height
	cs.LocalBlockNum = chain.LocalBlockNum
	cs.TxSum = chain.TxSum

	loglogrus.Log.Infof("[Block Chain] CommitBlock准备更新区块链状态量, block Hash:%x , block Height:%d \n", cs.CurrentHash, cs.Height)
	chain.databaseMutex.Lock()
	key = []byte("chainState")
	value = ChainStateSerialize(cs)
	if err := chain.Database.Put(key, value); err != nil { //额外存储chainState
		chain.databaseMutex.Unlock()
		return err
	}
	chain.databaseMutex.Unlock()
	loglogrus.Log.Infof("[Block Chain] CommitBlock完成对区块链状态量的更新, block Hash:%x , block Height:%d \n", cs.CurrentHash, cs.Height)

	loglogrus.Log.Warnf("Current NodeID:%x commit Block into LevelDB\n", selfNodeID)

	go func() {
		chain.latestMutex.Lock()
		now := time.Now()
		for txIndex, tx := range block.Transactions {
			receipt := block.Receipt.TxReceipt[txIndex]
			chain.LatestUpLinkedTransaction[tx.TxID] = &TransactionEvent{UpLinkTime: now, Expired: false, Receipt: receipt}
		}
		chain.latestMutex.Unlock()
	}()

	return nil
}

// 根据区块hash值在本地链检索对应区块
func (chain *BlockChain) GetBlockFromHash(blockHash common.Hash) *WrapBlock {

	loglogrus.Log.Infof("[Block Chain] GetBlockFromHash 被调用\n")
	chain.databaseMutex.RLock()
	defer chain.databaseMutex.RUnlock()

	deadline := time.NewTimer(100 * time.Millisecond)
	blockStream := make(chan *WrapBlock)
	go func() {
		index := blockHash[:]
		if target, err := chain.Database.Get(index); err != nil {
			return
		} else {
			if block, err := BlockDeSerialize(target); err != nil { //将检索得到的value(bit流)解码为区块
				return
			} else {
				blockStream <- block
				return
			}
		}
	}()

	select {
	case <-deadline.C:
		close(blockStream)
		loglogrus.Log.Infof("[Block Chain] GetBlockFromHash 调用结束\n")
		return &WrapBlock{}
	case block := <-blockStream:
		close(blockStream)
		loglogrus.Log.Infof("[Block Chain] GetBlockFromHash 调用结束\n")
		return block
	}
}

// 检索当前本地链,判断某一区块是否存在
func (chain *BlockChain) FindVersion(expectVersion common.Hash) bool {
	loglogrus.Log.Infof("[Block Chain] FindVersion 被调用\n")

	chain.databaseMutex.RLock()
	defer chain.databaseMutex.RUnlock()
	index := expectVersion[:]
	if _, err := chain.Database.Get(index); err != nil { //数据库检索,是否存在区块
		return false
	} else {
		return true
	}
}

func (chain *BlockChain) GetBlocksFromHash(currentHash common.Hash, Max uint64) []*WrapBlock {
	loglogrus.Log.Infof("[Block Chain] GetBlocksFromHash 被调用\n")
	blocks := make([]*WrapBlock, 0)
	currentBlock := chain.GetBlockFromHash(currentHash)
	if currentBlock == nil || reflect.DeepEqual(*currentBlock, WrapBlock{}) {
		return nil
	}
	blocks = append(blocks, currentBlock)

	// if Max == 0 || Max > chain.Height-chain.ArchiveHeight {
	// 	Max = chain.Height - chain.ArchiveHeight
	// }
	if Max == 0 || Max > chain.Height {
		Max = chain.Height
	}

	for i := 0; i < int(Max)-1; i++ {
		parentBlock := chain.GetBlockFromHash(currentBlock.RawBlock.PrevBlock)
		if parentBlock == nil || reflect.DeepEqual(*parentBlock, WrapBlock{}) {
			break
		}
		blocks = append(blocks, parentBlock)
		// if parentBlock.RawBlock.BlockID == chain.GenesisHash {
		// 	break
		// }
		currentBlock = parentBlock
	}

	return blocks
}

// 检索当前本地链,获取从currentBlock开始的指定数量的区块hash。如果Max = 0,表示返回到创世区块的全部区块
func (chain *BlockChain) GetBlockHashesFromHash(currentHash common.Hash, Max uint64) []common.Hash {
	loglogrus.Log.Infof("[Block Chain] GetBlockHashesFromHash 被调用\n")
	Hashes := make([]common.Hash, 0)
	currentBlock := chain.GetBlockFromHash(currentHash)
	if currentBlock == nil || reflect.DeepEqual(*currentBlock, WrapBlock{}) {
		return nil
	}
	Hashes = append(Hashes, currentBlock.RawBlock.BlockID)

	// if Max == 0 || Max > chain.Height-chain.ArchiveHeight {
	// 	Max = chain.Height - chain.ArchiveHeight
	// }
	if Max == 0 || Max > chain.Height {
		Max = chain.Height
	}

	for i := 0; i < int(Max)-1; i++ {
		parentBlock := chain.GetBlockFromHash(currentBlock.RawBlock.PrevBlock)
		if parentBlock == nil || reflect.DeepEqual(*parentBlock, WrapBlock{}) {
			break
		}
		Hashes = append(Hashes, parentBlock.RawBlock.BlockID)
		// if parentBlock.RawBlock.BlockID == chain.GenesisHash {
		// 	break
		// }
		currentBlock = parentBlock
	}

	return Hashes
}

// 从本地区块链获取一个最新的区块
func (chain *BlockChain) GetCurrentBlock() *WrapBlock {
	loglogrus.Log.Infof("[Block Chain] GetCurrentBlock 被调用\n")
	return chain.GetBlockFromHash(chain.CurrentHash)
}

// 新增，从本地区块链获取指定编号的区块
func (chain *BlockChain) GetBlockByNumber(num uint64) *WrapBlock {

	loglogrus.Log.Infof("[Block Chain] GetBlockByNumber 被调用\n")
	if chain.Height <= num || (num <= chain.ArchiveHeight && chain.archiveMode) {
		return nil
	}

	if chain.Height-num > maxQueryDepth {
		return nil
	}
	currentBlock := chain.GetBlockFromHash(chain.CurrentHash)

	for {
		if currentBlock.Number == num {
			return currentBlock
		} else {
			currentBlock = chain.GetBlockFromHash(currentBlock.RawBlock.PrevBlock)
		}
		if (currentBlock == nil || reflect.DeepEqual(*currentBlock, WrapBlock{})) {
			return nil
		}
	}
}

// count the version number difference of the given stale version and the current one.
// if the stale version is not found, return -1
func (chain *BlockChain) CountVersionLag(staleVersion common.Hash) int {
	loglogrus.Log.Infof("[Block Chain] CountVersionLag 被调用\n")
	chain.databaseMutex.RLock()
	defer chain.databaseMutex.RUnlock()

	index := staleVersion[:]
	if target, err := chain.Database.Get(index); err != nil {
		return -1
	} else {
		if block, err := BlockDeSerialize(target); err != nil { //将检索得到的value(bit流)解码为区块
			return -1
		} else {
			lag := int(chain.Height - block.Number - 1)
			return lag
		}
	}
}

// Figure out wether a version is too old compared with another.
// if it is too old, back true, otherwise false is bcak.
// if staleLimit is 0, always back false.
// either of the two version is not found would back true.
func (chain *BlockChain) StaleCheck(oldVersion, newVersion common.Hash, staleLimit uint8) bool {
	loglogrus.Log.Infof("[Block Chain] StaleCheck 被调用\n")
	chain.databaseMutex.RLock()
	defer chain.databaseMutex.RUnlock()

	if staleLimit == 0 {
		return false
	}

	oldKey := oldVersion[:]
	newKey := newVersion[:]

	oldValue, err := chain.Database.Get(oldKey)
	if err != nil {
		loglogrus.Log.Warnf("[Block Chain] Old Block Version is not exist!\n")
		return true
	}
	oldBlock, err := BlockDeSerialize(oldValue)
	if err != nil {
		loglogrus.Log.Warnf("[Block Chain] Old Block is not exist!\n")
		return true
	}

	newValue, err := chain.Database.Get(newKey)
	if err != nil {
		loglogrus.Log.Warnf("[Block Chain] New Block Version is not exist!\n")
		return true
	}
	newBlock, err := BlockDeSerialize(newValue)
	if err != nil {
		loglogrus.Log.Warnf("[Block Chain] New Block is not exist!\n")
		return true
	}

	delta := newBlock.Number - oldBlock.Number
	if delta <= uint64(staleLimit) {
		loglogrus.Log.Infof("[Block Chain] Old Block is legal\n")
		return false
	} else {
		loglogrus.Log.Infof("[Block Chain] Old Block is too old for new Block, beyond staleLimit\n")
		return true
	}

}

func (chain *BlockChain) InstallExecuteBlockFunc(CommitWriteSet func(writeSet []WriteEle) error) {
	chain.ExecuteBlock = func(block *Block) error {
		return CommitWriteSet(block.Receipt.WriteSet)
	}
}

func (chain *BlockChain) TxCheck(txID common.Hash) (bool, *TransactionEvent) {
	chain.latestMutex.RLock()
	defer chain.latestMutex.RUnlock()
	if txEvent, ok := chain.LatestUpLinkedTransaction[txID]; ok {
		if !txEvent.Expired {
			txEvent.Expired = true
			return true, txEvent
		}
	}

	return false, nil
}

func (chain *BlockChain) ExpireTransactionEvent() {
	expireCircle := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-expireCircle.C:
			chain.latestMutex.RLock()
			for txID, event := range chain.LatestUpLinkedTransaction {
				if event.Expired || time.Since(event.UpLinkTime) > upLinkedTxCacheLimit { // 事件已读，或者存在时间超过3s
					delete(chain.LatestUpLinkedTransaction, txID)
				}
			}
			chain.latestMutex.RUnlock()
		}
	}
}

func (chain *BlockChain) TxResultCheck(block *Block) {
	if chain.TxCheckPool == nil {
		return
	}
	pool := chain.TxCheckPool
	var txIndex int //交易在区块中的下标
	var tx Transaction
	for pool.LenTask() > 0 { //需要处理完pool中所有等待处理的挂起事件
		for txIndex, tx = range block.Transactions {
			if task := pool.Retrieval(tx.TxID); task != nil { //在事件池中检索出指定TxID的交易检查事件
				targetTX := block.Receipt.TxReceipt[txIndex] //从区块中获取的目标交易

				//fmt.Printf("交易:%x上链成功,交易执行结果:%s\n", tx.TxID, targetTX.Result)
				//需要通知事件pool当前事件已经处理完成(也就是交易上链验证成功)
				now := time.Now()
				interval := now.Sub(task.TimeStamp)

				pool.SetCheckedPool(tx.TxID, targetTX.Valid, targetTX.Result, interval)

				pool.DelTask(tx.TxID) //检验完成的交易事件从事件池中删除
				break
			}
		}
	}
}

func (chain *BlockChain) GetTransactionByHash(txID common.Hash) (*Transaction, *TransactionReceipt) {
	currentBlock := chain.GetBlockFromHash(chain.CurrentHash)
	queryDepth := 0
	for {
		for index, tx := range currentBlock.RawBlock.Transactions {
			if reflect.DeepEqual(tx.TxID, txID) {
				return &tx, &currentBlock.RawBlock.Receipt.TxReceipt[index]
			}
		}
		parentBlock := chain.GetBlockFromHash(currentBlock.RawBlock.PrevBlock)
		if parentBlock == nil || reflect.DeepEqual(parentBlock, &WrapBlock{}) {
			break
		}
		currentBlock = parentBlock
		queryDepth++
		if (currentBlock.RawBlock.PrevBlock == common.Hash{}) || queryDepth == maxQueryDepth {
			break
		}
	}

	return nil, nil
}

func (chain *BlockChain) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index int) (*Transaction, *TransactionReceipt) {
	if wrapBlock := chain.GetBlockFromHash(blockHash); (wrapBlock != nil && !reflect.DeepEqual(*wrapBlock, WrapBlock{})) {
		if len(wrapBlock.RawBlock.Transactions)-1 < index {
			return nil, nil
		} else {
			return &wrapBlock.RawBlock.Transactions[index], &wrapBlock.RawBlock.Receipt.TxReceipt[index]
		}
	} else {
		return nil, nil
	}
}

func (chain *BlockChain) GetTransactionsByBlockHash(blockHash common.Hash) ([]Transaction, []TransactionReceipt) {

	if wrapBlock := chain.GetBlockFromHash(blockHash); (wrapBlock != nil && !reflect.DeepEqual(*wrapBlock, WrapBlock{})) {
		return wrapBlock.RawBlock.Transactions, wrapBlock.RawBlock.Receipt.TxReceipt
	} else {
		return nil, nil
	}
}

func (chain *BlockChain) GetTransactionsByBlockNumber(blockNum uint64) ([]Transaction, []TransactionReceipt) {
	chain.databaseMutex.RLock()
	defer chain.databaseMutex.RUnlock()
	if wrapBlock := chain.GetBlockByNumber(blockNum); (wrapBlock != nil && !reflect.DeepEqual(*wrapBlock, WrapBlock{})) {
		return wrapBlock.RawBlock.Transactions, wrapBlock.RawBlock.Receipt.TxReceipt
	} else {
		return nil, nil
	}
}

func (chain *BlockChain) GetTransactions(txNumber uint64) ([]Transaction, []TransactionReceipt) {

	if txNumber > chain.TxSum || txNumber == 0 {
		txNumber = chain.TxSum
	}

	currentBlock := chain.GetBlockFromHash(chain.CurrentHash)
	txList := make([]Transaction, 0)
	txReceiptList := make([]TransactionReceipt, 0)
	if (reflect.DeepEqual(*currentBlock, WrapBlock{})) {
		return txList, txReceiptList
	}
	for {

		for i := len(currentBlock.RawBlock.Transactions) - 1; i >= 0; i-- {
			txList = append(txList, currentBlock.RawBlock.Transactions[i])
			txReceiptList = append(txReceiptList, currentBlock.RawBlock.Receipt.TxReceipt[i])
			if uint64(len(txList)) >= txNumber {
				return txList, txReceiptList
			}
		}
		parentBlock := chain.GetBlockFromHash(currentBlock.RawBlock.PrevBlock)
		currentBlock = parentBlock

		if (reflect.DeepEqual(*currentBlock, WrapBlock{})) || currentBlock == nil {
			break
		}
	}

	return txList, txReceiptList
}
