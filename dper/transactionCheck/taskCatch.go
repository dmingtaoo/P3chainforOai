package transactionCheck

import (
	"errors"
	"p3Chain/common"
	"time"
)

// 将用户输入的txID组装成检查时间添加到事件池
func TxCheckRun(TaskPool *TxCheckPool, errorChan chan error, TxIDChan chan common.Hash, finish chan bool) {
	go TaskPool.Expire() //循环运行，删除过期事件
	// 根据用户输入的TxID不断向事件池中添加事件
	for {
		select {
		case t := <-TxIDChan: //不断从管道中接收用户输入的TxID(阻塞等待)
			Task := NewTask(t, time.Now()) //创建一个待检查事件
			if !TaskPool.AddTask(Task) {   //向事件池中添加该事件
				errorChan <- errors.New("TaskPool is full") //可以提示用户事件池已满,可以等待一会再重新输入
				continue
			} else {
				//fmt.Printf("事件添加成功,对应交易TxID:%x\n", t)
			}
		case flag := <-finish:
			if flag == true {
				return
			}
		}

	}

}
