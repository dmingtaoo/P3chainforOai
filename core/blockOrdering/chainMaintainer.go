package blockOrdering

import (
	"p3Chain/common"
	loglogrus "p3Chain/log_logrus"
	"sync"
	"time"
)

type receiptSign struct {
	author    common.NodeID
	receiptID common.Hash
	signature []byte
}

type collectReceipt struct {
	receiptMux sync.RWMutex

	empty        bool
	maxCount     int
	maxReceiptID common.Hash

	countMap     map[common.Hash]int
	authorShot   map[common.NodeID]bool
	signatureMap map[common.Hash][][]byte
}

func (cr *collectReceipt) reset() {
	cr.receiptMux.Lock()
	defer cr.receiptMux.Unlock()

	cr.empty = true
	cr.maxCount = 0
	cr.maxReceiptID = common.Hash{}

	cr.countMap = make(map[common.Hash]int)
	cr.authorShot = make(map[common.NodeID]bool)
	cr.signatureMap = make(map[common.Hash][][]byte)
}

func (cr *collectReceipt) addReceipt(receipt *receiptSign) {
	cr.receiptMux.Lock()
	defer cr.receiptMux.Unlock()

	if cr.empty {
		cr.empty = false
		cr.maxCount = 1
		cr.maxReceiptID = receipt.receiptID

		cr.countMap[receipt.receiptID] += 1
		cr.authorShot[receipt.author] = true
		tempSigns := make([][]byte, 0)
		tempSigns = append(tempSigns, receipt.signature)
		cr.signatureMap[receipt.receiptID] = tempSigns
	} else {
		_, ok := cr.authorShot[receipt.author]
		if ok {
			loglogrus.Log.Warnf("[Block Ordering] repeat signature with the same author is got!")
			return
		}
		cr.authorShot[receipt.author] = true
		signs, ok := cr.signatureMap[receipt.receiptID]
		if ok {
			signs = append(signs, receipt.signature)
		} else {
			signs = make([][]byte, 0)
			signs = append(signs, receipt.signature)
		}
		cr.signatureMap[receipt.receiptID] = signs
		cr.countMap[receipt.receiptID] += 1

		if cr.maxCount < cr.countMap[receipt.receiptID] {
			cr.maxCount = cr.countMap[receipt.receiptID]
			cr.maxReceiptID = receipt.receiptID
		}
	}
}

// Only the receiptID with the max number of signatures and is over the threshold
// can be returned
func (cr *collectReceipt) getMaxSignedReceipt(threshold int) (common.Hash, [][]byte, bool) {
	cr.receiptMux.RLock()
	defer cr.receiptMux.RUnlock()

	if cr.empty {
		return common.Hash{}, nil, false
	}

	if cr.maxCount < threshold {
		return common.Hash{}, nil, false
	}

	return cr.maxReceiptID, cr.signatureMap[cr.maxReceiptID], true
}

type doneCollect struct {
	doneMux sync.RWMutex

	doneShot map[common.NodeID]bool
	valid    int
	invalid  int
}

func (dc *doneCollect) reset() {
	dc.doneMux.Lock()
	defer dc.doneMux.Unlock()

	dc.doneShot = make(map[common.NodeID]bool)
	dc.valid = 0
	dc.invalid = 0
}

func (dc *doneCollect) add(sender common.NodeID, valid bool) {
	dc.doneMux.Lock()
	defer dc.doneMux.Unlock()

	val, ok := dc.doneShot[sender]
	if ok {
		if val == valid {
			return
		} else {
			dc.doneShot[sender] = valid
			if valid {
				dc.valid += 1
				dc.invalid -= 1
			} else {
				dc.valid -= 1
				dc.invalid += 1
			}
		}
	} else {
		dc.doneShot[sender] = valid
		if valid {
			dc.valid += 1
		} else {
			dc.invalid += 1
		}
	}
}

func (dc *doneCollect) readResult() (int, int) {
	dc.doneMux.RLock()
	defer dc.doneMux.RUnlock()
	return dc.valid, dc.invalid
}

type chainMaintainer struct {
	state uint8
	// the block order server is running with a sequential logic, no mux is in need
	// stateMux sync.RWMutex

	roundMux sync.RWMutex

	currentVersion common.Hash
	currentHeight  uint64

	repeatCount uint8

	receiptCache *collectReceipt
	doneCache    *doneCollect

	signExpire time.Time
	doneExpire time.Time

	nxtMsg *NextMessage
	cmtMsg *CommitMessage
}

func NewChainMaintainer(initVersion common.Hash, initHeight uint64) *chainMaintainer {
	rc := &collectReceipt{}
	rc.reset()

	dc := &doneCollect{}
	dc.reset()

	cm := &chainMaintainer{
		state:          state_init,
		roundMux:       sync.RWMutex{},
		currentVersion: initVersion,
		currentHeight:  initHeight,
		receiptCache:   rc,
		doneCache:      dc,
		signExpire:     time.Time{},
		doneExpire:     time.Time{},
		nxtMsg:         nil,
		cmtMsg:         nil,
	}
	return cm
}

func (cm *chainMaintainer) flush() {
	cm.nxtMsg = nil
	cm.cmtMsg = nil
	cm.receiptCache.reset()
	cm.doneCache.reset()
}
