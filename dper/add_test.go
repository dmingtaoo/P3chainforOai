package dper

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/contract"
	"p3Chain/core/dpnet"
	transactioncheck "p3Chain/dper/transactionCheck"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/utils"
	"testing"
	"time"
)

var mtw_add = []string{"ADDTPBFT"}

func InitConfigs_ADD() error {
	bootStrapNodes, err := utils.ReadNodesUrl(booterUrlFile)
	if err != nil {
		return fmt.Errorf("fail in read boot strap node, %v", err)
	}

	testBooterCfg = &OrderServiceServerConfig{
		NewAddressMode:    false,
		SelfNodeAddress:   testBooterKeyAddr,
		KeyStoreDir:       testKeyStore_OrderServiceServer,
		ServerName:        "test_booter",
		ListenAddr:        booterListenAddr,
		NAT:               nat.Any(),
		BootstrapNodes:    make([]*discover.Node, 0),
		MaxPeers:          20,
		CentralConfigMode: false,
	}

	testDpCfg1 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper1",
		ListenAddr:          dperListenAddr[0],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            20,
		MemoryDataBaseMode:  true,
		BlockDataBasePath:   "",
		StorageDataBasePath: "",
		Role:                dpnet.Leader,
		NetID:               "net1",
		CentralConfigMode:   false,
		ContractEngine:      contract.DEMO_CONTRACT_ADD_NAME,
		MTWSchemes:          mtw_add,
	}
	testDpCfg2 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper2",
		ListenAddr:          dperListenAddr[1],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            20,
		MemoryDataBaseMode:  true,
		BlockDataBasePath:   "",
		StorageDataBasePath: "",
		Role:                dpnet.Leader,
		NetID:               "net2",
		CentralConfigMode:   false,
		ContractEngine:      contract.DEMO_CONTRACT_ADD_NAME,
		MTWSchemes:          mtw_add,
	}
	testDpCfg3 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper3",
		ListenAddr:          dperListenAddr[2],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            20,
		MemoryDataBaseMode:  true,
		BlockDataBasePath:   "",
		StorageDataBasePath: "",
		Role:                dpnet.Follower,
		NetID:               "net1",
		CentralConfigMode:   false,
		ContractEngine:      contract.DEMO_CONTRACT_ADD_NAME,
		MTWSchemes:          mtw_add,
	}
	testDpCfg4 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper4",
		ListenAddr:          dperListenAddr[3],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            20,
		MemoryDataBaseMode:  true,
		BlockDataBasePath:   "",
		StorageDataBasePath: "",
		Role:                dpnet.Follower,
		NetID:               "net2",
		CentralConfigMode:   false,
		ContractEngine:      contract.DEMO_CONTRACT_ADD_NAME,
		MTWSchemes:          mtw_add,
	}
	return nil
}

func TestADDDemo(t *testing.T) {
	glog.SetToStderr(true)

	err := InitConfigs_ADD()
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

	account1, err := dper3.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	err = dper3.accountManager.Unlock(account1.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}

	dper3.StartCheckTransaction()

	printTxCheckResult := func(res transactioncheck.CheckResult) {
		fmt.Printf("Transaction ID: %x\n", res.TransactionID)
		fmt.Printf("Valid: %v\n", res.Valid)
		fmt.Printf("Transaction Results: %s\n", res.Result)
		fmt.Printf("Consensus Delay: %d ms\n", res.Interval.Milliseconds())
	}

	cresult, err := dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_ADD, contract.DEMO_CONTRACT_ADD_FuncInitAddr, nil)
	if err != nil {
		t.Fatalf("fail in publish transaction, %v", err)
	}
	printTxCheckResult(cresult)

	dper3.CloseCheckTransaction()

	// _, err = dper3.SimpleInvokeTransaction(account1, contract.DEMO_CONTRACT_ADD, contract.DEMO_CONTRACT_ADD_FuncInitAddr, nil)
	// if err != nil {
	// 	t.Fatalf("fail in invoke transaction, %v", err)
	// }

	time.Sleep(3 * time.Second)

	account2, err := dper4.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}

	result := dper4.SimpleInvokeTransactionLocally(account2, contract.DEMO_CONTRACT_ADD, contract.DEMO_CONTRACT_ADD_FuncGetAddr, [][]byte{[]byte("3")})
	if len(result) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("%s", result[0])

	_, err = dper3.SimpleInvokeTransaction(account1, contract.DEMO_CONTRACT_ADD, contract.DEMO_CONTRACT_ADD_FuncAddAddr, [][]byte{[]byte("1"), []byte("2"), []byte("250")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}

	time.Sleep(3 * time.Second)

	result = dper4.SimpleInvokeTransactionLocally(account2, contract.DEMO_CONTRACT_ADD, contract.DEMO_CONTRACT_ADD_FuncGetAddr, [][]byte{[]byte("250")})
	if len(result) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("%s", result[0])

	dper3.StartCheckTransaction()
	cresult, err = dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_ADD, contract.DEMO_CONTRACT_ADD_FuncSumAddr, [][]byte{[]byte("1"), []byte("250"), []byte("251")})
	if err != nil {
		t.Fatalf("fail in publish transaction, %v", err)
	}
	printTxCheckResult(cresult)

	result = dper4.SimpleInvokeTransactionLocally(account2, contract.DEMO_CONTRACT_ADD, contract.DEMO_CONTRACT_ADD_FuncGetAddr, [][]byte{[]byte("251")})
	if len(result) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("%s", result[0])

}
