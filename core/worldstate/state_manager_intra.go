package worldstate

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/database"
	loglogrus "p3Chain/log_logrus"
	"sync"
	"time"
)

type IntraStateManager struct {
	selfNode   *dpnet.Node
	stateDB    database.Database
	version    common.Hash // the version of the lagging intra-shard world state
	blockChain *eles.BlockChain

	stateMux  sync.RWMutex
	updateMux sync.Mutex // prevent the updating world state from being read by the StdWrapper
}

var (
	iwsVersionKey = []byte("IntraWorldStateVersion")
)

func NewIntraStateManager(selfNode *dpnet.Node, stateDB database.Database, blockchain *eles.BlockChain) *IntraStateManager {
	newSM := &IntraStateManager{
		selfNode:   selfNode,
		stateDB:    stateDB,
		blockChain: blockchain,
	}
	return newSM
}

func (ism *IntraStateManager) GetBlockChain() *eles.BlockChain {
	return ism.blockChain
}

func (ism *IntraStateManager) CommitBlock(block *eles.Block) error {
	panic("to do")
}

// return the lagging world state version.
func (ism *IntraStateManager) CurrentVersion() common.Hash {
	return ism.version
}

// CountVersionLag returns the num of versions that a stale version lags the current one.
// if not found, back -1
func (ism *IntraStateManager) CountVersionLag(staleVersion common.Hash) int {
	return ism.blockChain.CountVersionLag(staleVersion)
}

// Figure out wether a version is too old compared with another.
// if it is too old, back true, otherwise false is bcak.
// if staleLimit is 0, always back false.
// either of the two version is not found would back true.
func (ism *IntraStateManager) StaleCheck(oldVersion, newVersion common.Hash, staleLimit uint8) bool {
	panic("to do")
}

func (ism *IntraStateManager) FindVersion(expectVersion common.Hash) bool {
	return ism.blockChain.FindVersion(expectVersion)
}

func (ism *IntraStateManager) GetBlockChainHeight() uint64 {
	return ism.blockChain.CurrentHeight()
}

func (ism *IntraStateManager) GetBlockByNumber(num uint64) (*eles.Block, error) {
	wrapBlock := ism.blockChain.GetBlockByNumber(num)
	if wrapBlock == nil {
		return nil, fmt.Errorf("cannot find block with heright %d", num)
	}
	return wrapBlock.RawBlock, nil
}

func (ism *IntraStateManager) GetBlockFromHash(hash common.Hash) *eles.WrapBlock {
	return ism.blockChain.GetBlockFromHash(hash)
}

func (ism *IntraStateManager) GetBlocksFromHash(hash common.Hash, max uint64) []*eles.WrapBlock {
	return ism.blockChain.GetBlocksFromHash(hash, max)
}

func (ism *IntraStateManager) BackTxSum() uint64 {
	return ism.blockChain.BackTxSum()
}

func (ism *IntraStateManager) BackBlockStorageMux() *sync.RWMutex {
	return ism.blockChain.BackBlockStorageMux()
}

func (ism *IntraStateManager) BackLocalBlockNum() uint64 {
	return ism.blockChain.LocalBlockNum
}

func (ism *IntraStateManager) ResetLocalBlockNum() {
	ism.blockChain.ResetLocalBlockNum()
}

func (ism *IntraStateManager) GetCurrentState() (common.Hash, uint64) {
	height, head, _ := ism.blockChain.GetChainStatus()
	return head, height
}

func (ism *IntraStateManager) WriteWorldStateValues(keys []common.WsKey, values [][]byte) error {
	if len(keys) != len(values) {
		return fmt.Errorf("numbers of keys and values are not matched")
	}

	ism.stateMux.Lock()
	defer ism.stateMux.Unlock()
	for i := 0; i < len(keys); i++ {
		err := ism.stateDB.Put(keys[i][:], values[i])
		if err != nil {
			return err // NOTE: in real system, we need roll back!
		}
	}

	return nil
}

func (ism *IntraStateManager) ReadWorldStateValues(keys ...common.WsKey) ([][]byte, error) {
	ism.stateMux.RLock()
	defer ism.stateMux.RUnlock()
	// when the world state is updating, the read operation is blocked
	ism.updateMux.Lock()
	defer ism.updateMux.Unlock()
	res := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		val, err := ism.stateDB.Get(keys[i][:])
		if err != nil {
			return res, err
		}
		res[i] = val
	}
	return res, nil
}

func (ism *IntraStateManager) SetWorldStateVersion(version common.Hash) error {
	ism.stateMux.Lock()
	defer ism.stateMux.Unlock()
	iwsVersionVal := version.Bytes()
	err := ism.stateDB.Put(iwsVersionKey, iwsVersionVal)
	if err != nil {
		return err
	}
	ism.version = version
	loglogrus.Log.Infof("set intra world state version to %s", version.Hex())
	return nil
}

func (ism *IntraStateManager) GetWorldStateVersion() common.Hash {
	iwsVersionVal, err := ism.stateDB.Get(iwsVersionKey)
	if err != nil {
		return common.Hash{}
	}
	ism.version = common.BytesToHash(iwsVersionVal)
	return ism.version
}

func (ism *IntraStateManager) InitISM() error {
	ret, err := ism.stateDB.Has(iwsVersionKey)
	if err != nil {
		return err
	}

	if ret {
		ism.GetWorldStateVersion()
	} else {
		err := ism.SetWorldStateVersion(common.Hash{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ism *IntraStateManager) CommitWriteSet(writeSet []eles.WriteEle) error {
	keys := make([]common.WsKey, len(writeSet))
	values := make([][]byte, len(writeSet))
	for i := 0; i < len(writeSet); i++ {
		keys[i] = writeSet[i].ValueAddress
		values[i] = writeSet[i].Value
	}
	err := ism.WriteWorldStateValues(keys, values)
	return err
}

func (ism *IntraStateManager) ExecuteBlocks(blocks []*eles.WrapBlock) error {
	// prevent the updating world state from reading by the StdWrapper
	ism.updateMux.Lock()
	defer ism.updateMux.Unlock()

	execBNum := len(blocks)
	if execBNum == 0 {
		return nil
	}

	for i := execBNum - 1; i >= 0; i-- {
		err := ism.CommitWriteSet(blocks[i].RawBlock.Receipt.WriteSet)
		if err != nil {
			// update the world state height to the successfully committed block height
			if i == execBNum-1 {
				return err
			}
			ism.SetWorldStateVersion(blocks[i+1].RawBlock.BlockID)
			return err
		}
	}

	// update the world state version
	err := ism.SetWorldStateVersion(blocks[0].RawBlock.BlockID)
	if err != nil {
		return err
	}

	return nil
}

// func (ism *IntraStateManager) UpdateWorldStateToVersion(version common.Hash) error {
// 	// get the blockchain height and verson where the world state updates to
// 	iwsVersion := ism.version
// 	if common.EmptyHash(iwsVersion) {
// 		blocks := ism.blockChain.GetBlocksFromHash(version, 0)
// 		err := ism.ExecuteBlocks(blocks)
// 		if err != nil {
// 			return err
// 		}
// 		return nil
// 	}

// 	wb1 := ism.blockChain.GetBlockFromHash(iwsVersion)
// 	if common.EmptyHash(wb1.RawBlock.BlockID) {
// 		return fmt.Errorf("cannot find block with version %s", iwsVersion.Hex())
// 	}
// 	iwsHeight := wb1.Number + 1

// 	wb2 := ism.blockChain.GetBlockFromHash(version)
// 	if common.EmptyHash(wb1.RawBlock.BlockID) {
// 		return fmt.Errorf("cannot find block with version %s", version.Hex())
// 	}
// 	bcHeight := wb2.Number + 1

// 	loglogrus.Log.Infof("iwsVersion: %s, iwsHeight: %d, bcVersion: %s, bcHeight: %d", iwsVersion.Hex(), iwsHeight, version.Hex(), bcHeight)

// 	if iwsHeight > bcHeight {
// 		return fmt.Errorf("world state height is larger")
// 	} else if iwsHeight == bcHeight {
// 		return nil
// 	} else {
// 		// get blocks from the lagging height to the newest height
// 		blocks := ism.blockChain.GetBlocksFromHash(version, bcHeight-iwsHeight)

// 		loglogrus.Log.Infof("blocks length: %d", len(blocks))

// 		// update blocks' writeset to the world state
// 		err := ism.ExecuteBlocks(blocks)
// 		if err != nil {
// 			return err
// 		}

// 		return nil
// 	}
// }

func (ism *IntraStateManager) UpdateWorldStateToVersion(version common.Hash) error {
	var bNum, iwsHeight, vHeight uint64

	if common.EmptyHash(version) {
		return nil
	}

	iwsVersion := ism.version
	if common.EmptyHash(iwsVersion) {
		bNum = 0
	} else {
		// to get the number of update blocks
		wb1 := ism.blockChain.GetBlockFromHash(iwsVersion)
		if common.EmptyHash(wb1.RawBlock.BlockID) {
			return fmt.Errorf("cannot find block with version %s", iwsVersion.Hex())
		}
		iwsHeight = wb1.Number + 1

		wb2 := ism.blockChain.GetBlockFromHash(version)
		if common.EmptyHash(wb1.RawBlock.BlockID) {
			return fmt.Errorf("cannot find block with version %s", version.Hex())
		}
		vHeight = wb2.Number + 1

		if iwsHeight < vHeight {
			bNum = vHeight - iwsHeight
		} else if iwsHeight == vHeight {
			return nil
		} else {
			return fmt.Errorf("world state height is larger")
		}
	}

	// loglogrus.Log.Infof("iwsVersion: %s, iwsHeight: %d, bcVersion: %s, bcHeight: %d; update num: %d", iwsVersion.Hex(), iwsHeight, version.Hex(), vHeight, bNum)

	var blocks []*eles.WrapBlock
	checkTicker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-checkTicker.C:
			blocks = ism.blockChain.GetBlocksFromHash(version, bNum)
		}
		if len(blocks) > 0 {
			checkTicker.Stop()
			break
		}
	}

	err := ism.ExecuteBlocks(blocks)
	if err != nil {
		return err
	}
	return nil
}
