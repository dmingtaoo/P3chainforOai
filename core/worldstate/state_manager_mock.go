package worldstate

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/database"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

type MockStateManager struct {
	currentVersion common.Hash
	currentHeight  uint64
	StateStorage   database.Database

	WorldStateMu sync.RWMutex // to sync the changes of world state
}

func (msm *MockStateManager) GetCurrentState() (common.Hash, uint64) {
	msm.WorldStateMu.RLock()
	defer msm.WorldStateMu.RUnlock()
	return msm.currentVersion, msm.currentHeight
}

func (msm *MockStateManager) GetBlockChainHeight() uint64 {
	msm.WorldStateMu.RLock()
	defer msm.WorldStateMu.RUnlock()
	return msm.currentHeight
}

func (msm *MockStateManager) GetBlockByNumber(num uint64) (*eles.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (msm *MockStateManager) CommitBlock(block *eles.Block) error {
	msm.WorldStateMu.Lock()
	defer msm.WorldStateMu.Unlock()
	msm.currentVersion = block.BlockID
	msm.currentHeight += 1
	return nil
}

func (msm *MockStateManager) CurrentVersion() common.Hash {
	msm.WorldStateMu.RLock()
	defer msm.WorldStateMu.RUnlock()
	return msm.currentVersion
}

func (msm *MockStateManager) FindVersion(expectVersion common.Hash) bool {
	panic("Do this")
}

func (msm *MockStateManager) CountVersionLag(staleVersion common.Hash) int {
	panic("Do this")
}

// in mock state manager, version will never be staled
func (msm *MockStateManager) StaleCheck(oldVersion, newVersion common.Hash, staleLimit uint8) bool {
	return false
}

func (msm *MockStateManager) ReadWorldStateValues(keys ...common.WsKey) ([][]byte, error) {
	msm.WorldStateMu.RLock()
	defer msm.WorldStateMu.RUnlock()

	res := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		val, err := msm.StateStorage.Get(keys[i][:])
		if err != nil {
			return res, err
		}
		res[i] = val
	}
	return res, nil
}

func (msm *MockStateManager) WriteWorldStateValues(keys []common.WsKey, values [][]byte) error {
	if len(keys) != len(values) {
		return fmt.Errorf("numbers of keys and values are not matched")
	}

	msm.WorldStateMu.Lock()
	defer msm.WorldStateMu.Unlock()

	for i := 0; i < len(keys); i++ {
		err := msm.StateStorage.Put(keys[i][:], values[i])
		if err != nil {
			return err // NOTE: in real system, we need roll back!
		}
	}

	return nil
}

// InitTestDataBase init a disposable database for test,
// which is a memory level-db
func (msm *MockStateManager) InitTestDataBase() error {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return err
	}

	testDB := database.NewSimpleLDB("TESTDB", db)
	msm.StateStorage = testDB
	return nil
}

func (msm *MockStateManager) GetBlockFromHash(common.Hash) *eles.WrapBlock {
	panic("to do")
}

func (msm *MockStateManager) GetBlocksFromHash(common.Hash, uint64) []*eles.WrapBlock {
	panic("to do")
}

func (msm *MockStateManager) BackTxSum() uint64 {
	panic("to do")
}

func (msm *MockStateManager) BackBlockStorageMux() *sync.RWMutex {
	panic("to do")
}

func (msm *MockStateManager) BackLocalBlockNum() uint64 {
	panic("to do")
}

func (msm *MockStateManager) ResetLocalBlockNum() {
	panic("to do")
}

// this function is only for testing use
func (msm *MockStateManager) ChangeState(version common.Hash, height uint64) {
	msm.WorldStateMu.Lock()
	defer msm.WorldStateMu.Unlock()
	msm.currentVersion = version
	msm.currentHeight = height
}

func (msm *MockStateManager) UpdateWorldStateToVersion(version common.Hash) error {
	fmt.Println("update version")
	panic("to do")
	fmt.Println("update finish")
	return nil
}
