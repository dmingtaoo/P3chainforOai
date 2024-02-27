package transactionCheck

import (
	"fmt"
	"p3Chain/common"
	"sync"
	"time"
)

const (
	TaskLimit  = 1500             //事件池允许的挂起事件上限
	ExpireTime = 20 * time.Second //超过此上限，则为过期事件
	CheckCycle = 25 * time.Second //事件过期检查周期
)

type CheckResult struct {
	TransactionID common.Hash
	Valid         bool
	Result        [][]byte
	Interval      time.Duration
}

type checkTask struct { //一个交易检查事件
	TxID      common.Hash
	TimeStamp time.Time
}

type TxCheckPool struct {
	taskNum      int
	checkingPool map[common.Hash]*checkTask //存放所有待检查事件
	poolMutex    sync.Mutex
	checkedPool  map[common.Hash]*CheckResult //存放所有已完成的事件及其结果
}

// 生成一个检查事件
func NewTask(TxID common.Hash, time time.Time) *checkTask {
	return &checkTask{
		TxID:      TxID,
		TimeStamp: time,
	}
}

// 创建一个事件池
func NewTaskPool() *TxCheckPool {
	return &TxCheckPool{
		checkingPool: make(map[common.Hash]*checkTask),
		checkedPool:  make(map[common.Hash]*CheckResult),
	}
}

// 增加一个待检查事件
func (tcp *TxCheckPool) AddTask(task *checkTask) bool {
	tcp.poolMutex.Lock()
	if tcp.taskNum >= TaskLimit {
		return false
	}
	defer tcp.poolMutex.Unlock()
	tcp.checkingPool[task.TxID] = task
	tcp.checkedPool[task.TxID] = nil
	tcp.taskNum++
	return true
}

// 删除一个检查事件
func (tcp *TxCheckPool) DelTask(txID common.Hash) {
	tcp.poolMutex.Lock()
	defer tcp.poolMutex.Unlock()
	delete(tcp.checkingPool, txID)

	tcp.taskNum--
}

// 返回当前交易池中事件的个数
func (tcp *TxCheckPool) LenTask() int {
	return tcp.taskNum
}

// 检索返回指定txID的事件
func (tcp *TxCheckPool) Retrieval(txID common.Hash) *checkTask {

	tcp.poolMutex.Lock()
	defer tcp.poolMutex.Unlock()

	if task, ok := tcp.checkingPool[txID]; !ok {
		return nil
	} else {
		return task
	}
}

func (tcp *TxCheckPool) RetrievalResult(txID common.Hash) *CheckResult {
	tcp.poolMutex.Lock()
	defer tcp.poolMutex.Unlock()

	if result, ok := tcp.checkedPool[txID]; !ok {
		return nil
	} else {
		delete(tcp.checkedPool, txID)
		return result
	}
}

// 删除过期事件(由协程单独运行)
func (tcp *TxCheckPool) Remove(currentTime time.Time) {
	for txID, task := range tcp.checkingPool {
		interval := currentTime.Sub(task.TimeStamp)
		if interval > ExpireTime {
			tcp.DelTask(txID)
			fmt.Printf("事件已过期,交易TxID:%x\n", txID)
		}
	}
}

// 循环、周期性删除过期事件
func (tcp *TxCheckPool) Expire() {
	Timer := time.Tick(CheckCycle)
	for {
		select {
		case <-Timer:
			tcp.Remove(time.Now())
		}
	}
}

// 用于通知用户交易上链验证的结果(设置checkedPool)
func (tcp *TxCheckPool) SetCheckedPool(txID common.Hash, valid byte, result [][]byte, duration time.Duration) {

	//1.检查验证的交易对应的事件否确实存在于当前taskPool中
	if _, ok := tcp.checkingPool[txID]; !ok {
		return
	}
	tcp.poolMutex.Lock()
	defer tcp.poolMutex.Unlock()
	//2.如果确实存在,需要通知用户交易验证已经完成,同时告知交易的执行结果

	var validFlag bool
	if valid == byte(1) {
		validFlag = true
	}
	cr := &CheckResult{
		TransactionID: txID,
		Valid:         validFlag,
		Result:        result,
		Interval:      duration,
	}
	tcp.checkedPool[txID] = cr
	//fmt.Printf("checkedPool[%x] = %s\n", txID, result)
}
