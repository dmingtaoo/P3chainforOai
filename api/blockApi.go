package api

import (
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/core/worldstate"
)

type BlockHeader struct{} // TODO:区块头

// 提供p3Chain区块链的相关信息，包括区块查询和交易查询
type BlockService struct {
	stateManager *worldstate.DpStateManager
	blockChain   *eles.BlockChain
}

func NewBlockService(stateManager *worldstate.DpStateManager) *BlockService {
	return &BlockService{
		stateManager: stateManager,
		blockChain:   stateManager.GetBlockChain(),
	}
}

// 获取区块高度
func (bs *BlockService) GetCurrentBlockNumber() uint64 {
	return bs.blockChain.CurrentHeight()
}

func (bs *BlockService) GetCurrentBlockHash() common.Hash {
	return bs.blockChain.CurrentVersion()
}

// 根据区块Hash获取区块(区块整体)
func (bs *BlockService) GetBlockByHash(blockHash common.Hash) *eles.WrapBlock {
	return bs.blockChain.GetBlockFromHash(blockHash)
}

// 根据区块Hash获取区块(区块头)
func (bs *BlockService) GetBlockHeaderByHash(blockHash common.Hash) *BlockHeader {
	block := bs.blockChain.GetBlockFromHash(blockHash)
	_ = block
	// 取出获取区块的区块头
	panic("to do")
}

// 根据区块编号获取区块(区块整体)
func (bs *BlockService) GetBlockByNumber(blockNum uint64) *eles.WrapBlock {
	return bs.blockChain.GetBlockByNumber(blockNum)
}

// 根据区块编号获取区块(区块头)
func (bs *BlockService) GetBlockHeaderByNumber(blockNum uint64) BlockHeader {
	block := bs.blockChain.GetBlockByNumber(blockNum)
	_ = block
	// 取出获取区块的区块头
	panic("to do implement")
}

// 根据区块编号获取区块Hash
func (bs *BlockService) GetBlockHashByNumber(blockNum uint64) common.Hash {
	block := bs.blockChain.GetBlockByNumber(blockNum)
	if block == nil {
		return common.Hash{}
	}
	return block.RawBlock.BlockID
}

// 返回所有已有区块(阻塞操作，使用单独的协程负责处理)
func (bs *BlockService) GetAllBlocks() []*eles.WrapBlock {

	return bs.blockChain.GetBlocksFromHash(bs.blockChain.CurrentHash, 0)
}

// 返回所有已有区块(区块头)
func (bs *BlockService) GetAllBlocksHeader() []BlockHeader {

	blocks := bs.blockChain.GetBlocksFromHash(bs.blockChain.CurrentHash, 0)
	_ = blocks
	// 取出所有区块的区块头
	panic("to do implement")
}

// 返回从头区块开始的指定数量个区块
func (bs *BlockService) GetRecentBlocks(blockCount uint64) []*eles.WrapBlock {

	return bs.blockChain.GetBlocksFromHash(bs.blockChain.CurrentHash, blockCount)
}

// 返回从头区块开始的指定数量个区块(区块头)
func (bs *BlockService) GetRecentBlocksHeader(blockCount uint64) []BlockHeader {

	blocks := bs.blockChain.GetBlocksFromHash(bs.blockChain.CurrentHash, blockCount)
	_ = blocks
	// 取出所有区块的区块头
	panic("to do implement")
}

// 获取交易总数
func (bs *BlockService) GetTransactionNumber() uint64 {
	return bs.blockChain.TxSum
}

// 根据交易Hash获取交易(阻塞操作，使用单独的协程负责处理)
func (bs *BlockService) GetTransactionByHash(txID common.Hash) (*eles.Transaction, *eles.TransactionReceipt) {
	return bs.blockChain.GetTransactionByHash(txID)
}

// 根据区块Hash和索引获取区块内的对应索引的交易
func (bs *BlockService) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index int) (*eles.Transaction, *eles.TransactionReceipt) {
	return bs.blockChain.GetTransactionByBlockHashAndIndex(blockHash, index)
}

// 获取某一个区块内的全部交易（根据区块Hash）
func (bs *BlockService) GetTransactionsByBlockHash(blockHash common.Hash) ([]eles.Transaction, []eles.TransactionReceipt) {
	return bs.blockChain.GetTransactionsByBlockHash(blockHash)
}

// 获取某一个区块内的全部交易（根据区块编号）
func (bs *BlockService) GetTransactionsByBlockNumber(blockNum uint64) ([]eles.Transaction, []eles.TransactionReceipt) {
	return bs.blockChain.GetTransactionsByBlockNumber(blockNum)
}

// 获取最近的指定数量笔交易
func (bs *BlockService) GetRecentTransactions(txNumber uint64) ([]eles.Transaction, []eles.TransactionReceipt) {

	return bs.blockChain.GetTransactions(txNumber)
}

// 获取全部已有交易
func (bs *BlockService) GetAllTransactions() ([]eles.Transaction, []eles.TransactionReceipt) {
	return bs.blockChain.GetTransactions(0)
}
