package eles

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"sync"
	"time"
)

const (
	NormalTxPool = "BASEPOOL"

	// The max staleness of transaction chosen to do consensus.
	// NOTICE: this should be shorter than transaction max life time (that is to say
	// expiration delay) in transaction pool to make sure we won't choose transactions
	// that would be expired in transaction pool to do consensus.
	//
	// TODO: Make this customized.
	defaultStaleThreshold = 10000 * time.Millisecond

	defaultExpirationDelay = 20000 * time.Millisecond

	defaultLifeTime = 30000 * time.Millisecond
)

type TransactionState struct {
	Tx         *Transaction
	StoredTime time.Time // when to store
}

func NewTransactionState(tx *Transaction) *TransactionState {
	ts := &TransactionState{
		Tx:         tx,
		StoredTime: time.Now(),
	}
	return ts
}

type TransactionPool struct {
	PoolID string // the ID of this transaction pool

	staleLimit time.Duration // help to determine whether a transaction is fresh or not
	holdTime   time.Duration // the max time for a transaction could be seen in pool, should be larger than stateLimit

	poolMu sync.RWMutex                      // sync operations on pool
	Txs    map[common.Hash]*TransactionState // the implement of pool

	matchFunc  func(tx *Transaction) bool // check out whether a transaction matches this pool
	chooseFunc func(freshTxs []*TransactionState) []*Transaction

	quit chan struct{}
}

func NewTransactionPool(poolID string, matchFunc func(tx *Transaction) bool, chooseFunc func(freshTxs []*TransactionState) []*Transaction) *TransactionPool {
	if poolID == "" {
		poolID = NormalTxPool
	}
	tp := &TransactionPool{
		PoolID:     poolID,
		staleLimit: defaultStaleThreshold,
		holdTime:   defaultExpirationDelay,
		Txs:        make(map[common.Hash]*TransactionState),
		matchFunc:  matchFunc,
		chooseFunc: chooseFunc,
		quit:       make(chan struct{}),
	}
	return tp
}

func (tp *TransactionPool) Start() {
	go tp.update()
	glog.V(logger.Info).Infof("transaction pool: %v starts", tp.PoolID)
}

func (tp *TransactionPool) Stop() {
	close(tp.quit)
	glog.V(logger.Info).Infof("transaction pool: %v stops", tp.PoolID)
}

func (tp *TransactionPool) update() {
	expireTicker := time.NewTicker(defaultExpirationDelay)
	for {
		select {
		case <-expireTicker.C:
			tp.Expire()
		case <-tp.quit:
			return
		}
	}
}

// check out whether a transaction matches this pool
func (tp *TransactionPool) Match(tx *Transaction) bool {
	return tp.matchFunc(tx)
}

// add a transaction into pool
func (tp *TransactionPool) Add(tx *Transaction) error {
	if tx == nil {
		return fmt.Errorf("could not add nil transaction into pool.")
	}
	tp.poolMu.Lock()
	defer tp.poolMu.Unlock()
	tp.Txs[tx.TxID] = NewTransactionState(tx)
	return nil
}

// kick out those stale transactions from the pool
func (tp *TransactionPool) Expire() {
	tp.poolMu.Lock()
	defer tp.poolMu.Unlock()

	now := time.Now()
	threshold := now.Add(-tp.holdTime)

	for id, ts := range tp.Txs {
		if ts.StoredTime.Before(threshold) {
			delete(tp.Txs, id)
		}
	}
}

// RetrieveFreshTxs retrieves fresh transactions in the pool.
//
// NOTE: This function is not thread safe, so could be used in ChooseTransactions.
func (tp *TransactionPool) RetrieveFreshTxs() []*TransactionState {
	freshTxs := make([]*TransactionState, 0)
	now := time.Now()
	threshold := now.Add(-tp.staleLimit)
	for _, txs := range tp.Txs {
		if txs.StoredTime.After(threshold) {
			freshTxs = append(freshTxs, txs)
		}
	}
	return freshTxs
}

// ChooseTransactions first find out all fresh transactions in pool and
// then follows a strategy to choose some of them.
func (tp *TransactionPool) ChooseTransactions() []*Transaction {
	tp.poolMu.Lock()
	defer tp.poolMu.Unlock()

	freshTxs := tp.RetrieveFreshTxs()
	chosenTxs := tp.chooseFunc(freshTxs)
	return chosenTxs
}

// RemoveTransactions removes the referred transactions from the pool
func (tp *TransactionPool) RemoveTransactions(txs []*Transaction) {
	tp.poolMu.Lock()
	defer tp.poolMu.Unlock()

	for _, tx := range txs {
		delete(tp.Txs, tx.TxID)
	}
}

// FindTxs finds the transactions in order through the given txID.
// It returns transactions (which are found successfully) and a flag
// to tell whether all transactions are found.
func (tp *TransactionPool) FindTxs(expectTxs []common.Hash) ([]*Transaction, bool) {
	tp.poolMu.RLock()
	defer tp.poolMu.RUnlock()

	allFindFlag := true
	txs := make([]*Transaction, 0, len(expectTxs))

	for _, txID := range expectTxs {
		if ts, ok := tp.Txs[txID]; !ok {
			allFindFlag = false
			continue
		} else {
			txs = append(txs, ts.Tx)
		}
	}

	return txs, allFindFlag
}

// QuickFindTxs finds transactions in order through the given txID.
// If a transaction is not found, this function would end and immediately return.
func (tp *TransactionPool) QuickFindTxs(expectTxs []common.Hash) ([]*Transaction, bool) {
	tp.poolMu.RLock()
	defer tp.poolMu.RUnlock()

	txs := make([]*Transaction, 0, len(expectTxs))

	for _, txID := range expectTxs {
		ts, ok := tp.Txs[txID]
		if ok {
			txs = append(txs, ts.Tx)
		} else {
			return []*Transaction{}, false
		}
	}
	return txs, true
}
