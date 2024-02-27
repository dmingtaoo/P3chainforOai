package worldstate

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/database"
	"sync"
)

type StateManager interface {
	CurrentVersion() common.Hash
	FindVersion(expectVersion common.Hash) bool
	CountVersionLag(staleVersion common.Hash) int
	StaleCheck(oldVersion, newVersion common.Hash, staleLimit uint8) bool
	CommitBlock(block *eles.Block) error // commit a block and add it to the blockchain
	ReadWorldStateValues(keys ...common.WsKey) ([][]byte, error)
	WriteWorldStateValues(keys []common.WsKey, values [][]byte) error

	GetBlockChainHeight() uint64
	GetBlockByNumber(num uint64) (*eles.Block, error)
	GetBlockFromHash(common.Hash) *eles.WrapBlock
	GetBlocksFromHash(common.Hash, uint64) []*eles.WrapBlock

	GetCurrentState() (common.Hash, uint64)

	BackTxSum() uint64
	BackBlockStorageMux() *sync.RWMutex
	BackLocalBlockNum() uint64
	ResetLocalBlockNum()

	UpdateWorldStateToVersion(version common.Hash) error
}

type DpStateManager struct {
	selfNode   *dpnet.Node
	stateDB    database.Database
	blockChain *eles.BlockChain

	stateMux sync.RWMutex
}

func NewDpStateManager(selfNode *dpnet.Node, stateDB database.Database, blockchain *eles.BlockChain) *DpStateManager {
	newSM := &DpStateManager{
		selfNode:   selfNode,
		stateDB:    stateDB,
		blockChain: blockchain,
	}
	return newSM
}

func (dsm *DpStateManager) GetBlockChain() *eles.BlockChain {
	return dsm.blockChain
}

func (dsm *DpStateManager) CommitBlock(block *eles.Block) error {
	return dsm.blockChain.CommitBlock(dsm.selfNode.NodeID, block)
}

// return the current block chain version.
func (dsm *DpStateManager) CurrentVersion() common.Hash {
	return dsm.blockChain.CurrentVersion()
}

// CountVersionLag returns the num of versions that a stale version lags the current one.
// if not found, back -1
func (dsm *DpStateManager) CountVersionLag(staleVersion common.Hash) int {
	return dsm.blockChain.CountVersionLag(staleVersion)
}

// Figure out wether a version is too old compared with another.
// if it is too old, back true, otherwise false is bcak.
// if staleLimit is 0, always back false.
// either of the two version is not found would back true.
func (dsm *DpStateManager) StaleCheck(oldVersion, newVersion common.Hash, staleLimit uint8) bool {
	return dsm.blockChain.StaleCheck(oldVersion, newVersion, staleLimit)
}

func (dsm *DpStateManager) FindVersion(expectVersion common.Hash) bool {
	return dsm.blockChain.FindVersion(expectVersion)
}

func (dsm *DpStateManager) ReadWorldStateValues(keys ...common.WsKey) ([][]byte, error) {
	dsm.stateMux.RLock()
	defer dsm.stateMux.RUnlock()
	res := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		val, err := dsm.stateDB.Get(keys[i][:])
		if err != nil {
			return res, err
		}
		res[i] = val
	}
	return res, nil
}

func (dsm *DpStateManager) WriteWorldStateValues(keys []common.WsKey, values [][]byte) error {
	if len(keys) != len(values) {
		return fmt.Errorf("numbers of keys and values are not matched")
	}

	dsm.stateMux.Lock()
	defer dsm.stateMux.Unlock()

	for i := 0; i < len(keys); i++ {
		err := dsm.stateDB.Put(keys[i][:], values[i])
		if err != nil {
			return err // NOTE: in real system, we need roll back!
		}
	}

	return nil
}

func (dsm *DpStateManager) GetBlockChainHeight() uint64 {
	return dsm.blockChain.CurrentHeight()
}

func (dsm *DpStateManager) GetBlockByNumber(num uint64) (*eles.Block, error) {
	wrapBlock := dsm.blockChain.GetBlockByNumber(num)
	if wrapBlock == nil {
		return nil, fmt.Errorf("cannot find block with heright %d", num)
	}
	return wrapBlock.RawBlock, nil
}

func (dsm *DpStateManager) GetBlockFromHash(hash common.Hash) *eles.WrapBlock {
	return dsm.blockChain.GetBlockFromHash(hash)
}

func (dsm *DpStateManager) GetBlocksFromHash(hash common.Hash, max uint64) []*eles.WrapBlock {
	return dsm.blockChain.GetBlocksFromHash(hash, max)
}

func (dsm *DpStateManager) BackTxSum() uint64 {
	return dsm.blockChain.BackTxSum()
}

func (dsm *DpStateManager) BackBlockStorageMux() *sync.RWMutex {
	return dsm.blockChain.BackBlockStorageMux()
}

func (dsm *DpStateManager) BackLocalBlockNum() uint64 {
	return dsm.blockChain.LocalBlockNum
}

func (dsm *DpStateManager) ResetLocalBlockNum() {
	dsm.blockChain.ResetLocalBlockNum()
}

func (dsm *DpStateManager) GetCurrentState() (common.Hash, uint64) {

	height, head, _ := dsm.blockChain.GetChainStatus()
	return head, height
}

func (dsm *DpStateManager) UpdateWorldStateToVersion(version common.Hash) error {
	panic("to do")
}
