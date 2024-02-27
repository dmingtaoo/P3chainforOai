package dper

import (
	"fmt"
	"p3Chain/core/contract"
	transactioncheck "p3Chain/dper/transactionCheck"
	"p3Chain/logger/glog"
	"testing"
	"time"
)

func Testp3ChainTuning(t *testing.T) {
	REPEAT_CHECK_ENABLE = false

	glog.SetToStderr(true)
	err := InitConfigs()

	if err != nil {
		t.Fatalf("fail in init configs, %v", err)
	}

	booter, err := NewOrderServiceServer(testBooterCfg)
	if err != nil {
		t.Fatalf("fail in new booter, %v", err)
	}

	dper1 := NewDper(testDpCfg1)

	dper2 := NewDper(testDpCfg2)

	dper3 := NewDper(testDpCfg3)

	dper4 := NewDper(testDpCfg4)

	time.Sleep(1 * time.Second)

	err = booter.StateSelf()
	if err != nil {
		t.Fatalf("fail in state booter, %v", err)
	}
	err = dper1.StateSelf(false)
	if err != nil {
		t.Fatalf("fail in state dper, %v", err)
	}
	err = dper2.StateSelf(false)
	if err != nil {
		t.Fatalf("fail in state dper, %v", err)
	}
	err = dper3.StateSelf(false)
	if err != nil {
		t.Fatalf("fail in state dper, %v", err)
	}
	err = dper4.StateSelf(false)
	if err != nil {
		t.Fatalf("fail in state dper, %v", err)
	}

	time.Sleep(2 * time.Second)
	t.Logf("booter view net: %s", booter.netManager.BackViewNetInfo())
	t.Logf("dper1 view net: %s", dper1.netManager.BackViewNetInfo())
	t.Logf("dper2 view net: %s", dper2.netManager.BackViewNetInfo())
	t.Logf("dper3 view net: %s", dper3.netManager.BackViewNetInfo())
	t.Logf("dper4 view net: %s", dper4.netManager.BackViewNetInfo())

	err = booter.ConstructDpnet()
	if err != nil {
		t.Fatalf("fail in construct booter dpnet, %v", err)
	}
	err = dper1.ConstructDpnet()
	if err != nil {
		t.Fatalf("fail in construct dper dpnet, %v", err)
	}
	err = dper2.ConstructDpnet()
	if err != nil {
		t.Fatalf("fail in construct dper dpnet, %v", err)
	}
	err = dper3.ConstructDpnet()
	if err != nil {
		t.Fatalf("fail in construct dper dpnet, %v", err)
	}
	err = dper4.ConstructDpnet()
	if err != nil {
		t.Fatalf("fail in construct dper dpnet, %v", err)
	}

	time.Sleep(1 * time.Second)
	err = booter.Start()
	if err != nil {
		t.Fatalf("fail in start booter, %v", err)
	}
	err = dper1.Start()
	if err != nil {
		t.Fatalf("fail in start dper, %v", err)
	}
	err = dper2.Start()
	if err != nil {
		t.Fatalf("fail in start dper, %v", err)
	}
	err = dper3.Start()
	if err != nil {
		t.Fatalf("fail in start dper, %v", err)
	}
	err = dper4.Start()
	if err != nil {
		t.Fatalf("fail in start dper, %v", err)
	}

	time.Sleep(1 * time.Second)
	//pg, err := dper1.netManager.Spr.BackPeerGroup(dpnet.UpperNetID)
	//if err != nil {
	//	t.Fatalf("fail in back peer group")
	//}
	//t.Logf("peers in upper is: %v", len(pg.BackLiveMembers()))

	account1, err := dper3.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	err = dper3.accountManager.Unlock(account1.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}
	_, err = dper3.SimpleInvokeTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("Leo"), []byte("Awesome")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	// 发布若干笔笔交易,返回交易的TxID,存入管道TxIDChan
	dper3.StartCheckTransaction()

	printTxCheckResult := func(res transactioncheck.CheckResult) {
		fmt.Printf("Transaction ID: %x\n", res.TransactionID)
		fmt.Printf("Valid: %v\n", res.Valid)
		fmt.Printf("Transaction Results: %s\n", res.Result)
		fmt.Printf("Consensus Delay: %d ms\n", res.Interval.Milliseconds())
	}

	cresult, err := dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("Leo1"), []byte("Awesome1")})
	if err != nil {
		t.Fatalf("fail in publish transaction, %v", err)
	}
	printTxCheckResult(cresult)

	cresult, err = dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("Leo2"), []byte("Awesome2")})
	if err != nil {
		t.Fatalf("fail in publish transaction, %v", err)
	}
	printTxCheckResult(cresult)

	cresult, err = dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("Leo3"), []byte("Awesome3")})
	if err != nil {
		t.Fatalf("fail in publish transaction, %v", err)
	}
	printTxCheckResult(cresult)

	dper3.CloseCheckTransaction() //卸载交易检测模块

	account2, err := dper4.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	result := dper4.SimpleInvokeTransactionLocally(account2, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncGetAddr, [][]byte{[]byte("Leo")})
	if len(result) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("%s", result[0])

}
