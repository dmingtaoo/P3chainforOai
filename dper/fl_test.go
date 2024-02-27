package dper

import (
	"fmt"
	"io/ioutil"
	"p3Chain/common"
	"p3Chain/core/contract"
	"p3Chain/core/dpnet"
	transactioncheck "p3Chain/dper/transactionCheck"
	"p3Chain/flDemoUse/FLAPI"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/utils"
	"testing"
	"time"
)

var (
	testModelFilePath  = "./fltmp/testpara.pt"
	saveModelFilePath  = "./fltmp/gotpara.pt"
	plainModelFilePath = "./fltmp/plainpara.pt"
)

var mtw_fl = []string{"FLTPBFT"}

func InitConfigs_FL() error {
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
		ContractEngine:      contract.DEMO_CONTRACT_FL_NAME,
		MTWSchemes:          mtw_fl,
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
		ContractEngine:      contract.DEMO_CONTRACT_FL_NAME,
		MTWSchemes:          mtw_fl,
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
		ContractEngine:      contract.DEMO_CONTRACT_FL_NAME,
		MTWSchemes:          mtw_fl,
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
		ContractEngine:      contract.DEMO_CONTRACT_FL_NAME,
		MTWSchemes:          mtw_fl,
	}
	return nil
}

func TestFLDemo(t *testing.T) {
	flManager1 := FLAPI.NewFLManager(1)
	//flManager2 := FLAPI.NewFLManager(2)
	flManager3 := FLAPI.NewFLManager(3)
	//flManager4 := FLAPI.NewFLManager(4)
	err := InitConfigs_FL()
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

	dper1.StartCheckTransaction()

	account1, err := dper1.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	err = dper1.accountManager.Unlock(account1.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}

	account2, err := dper2.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	err = dper2.accountManager.Unlock(account2.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}

	account3, err := dper3.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	err = dper3.accountManager.Unlock(account3.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}

	account4, err := dper4.accountManager.NewAccount("")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	err = dper4.accountManager.Unlock(account4.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}

	res1, err := dper1.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_FL, contract.DEMO_CONTRACT_FL_FuncInitAddr, [][]byte{})
	if err != nil {
		t.Fatalf("fail in publish transaction, %v", err)
	}
	printTxCheckResult(res1)

	res2 := dper1.SimpleInvokeTransactionLocally(account1, contract.DEMO_CONTRACT_FL, contract.DEMO_CONTRACT_FL_FuncGetHeightAddr, [][]byte{})
	if len(res2) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("The global model height is: %s", res2[0])

	plaindata, err := ioutil.ReadFile(plainModelFilePath)
	if err != nil {
		t.Fatalf("fail in read file, %v", err)
	}
	t.Logf("start fl node 1 training...")
	newModel1, err := flManager1.ModelTrain(plaindata)
	if err != nil {
		t.Fatalf("fail in model train: %v", err)
	}
	evalRes1, err := flManager1.Evaluate(newModel1)
	if err != nil {
		t.Fatalf("fail in model evaluation: %v", err)
	}
	t.Logf("fl node after training: %s", evalRes1)

	t.Logf("start fl node 3 training...")
	newModel3, err := flManager3.ModelTrain(plaindata)
	if err != nil {
		t.Fatalf("fail in model train: %v", err)
	}
	evalRes3, err := flManager3.Evaluate(newModel3)
	if err != nil {
		t.Fatalf("fail in model evaluation: %v", err)
	}
	t.Logf("fl node after training: %s", evalRes3)

	t.Logf("start publish local models")

	_, err = dper1.SimpleInvokeTransaction(account1, contract.DEMO_CONTRACT_FL, contract.DEMO_CONTRACT_FL_FuncStoreModelAddr, [][]byte{newModel1})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}

	_, err = dper3.SimpleInvokeTransaction(account3, contract.DEMO_CONTRACT_FL, contract.DEMO_CONTRACT_FL_FuncStoreModelAddr, [][]byte{newModel3})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}

	time.Sleep(15 * time.Second)

	res3 := dper1.SimpleInvokeTransactionLocally(account1, contract.DEMO_CONTRACT_FL, contract.DEMO_CONTRACT_FL_FuncGetHeightAddr, [][]byte{})
	if len(res3) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("The global model height is: %s", res3[0])

	res4 := dper1.SimpleInvokeTransactionLocally(account1, contract.DEMO_CONTRACT_FL, contract.DEMO_CONTRACT_FL_FuncGetCurrentAddr, [][]byte{})
	agg_model := res4[0]
	evalLastRes, err := flManager1.Evaluate(agg_model)
	if err != nil {
		t.Fatalf("fail in model evaluate: %v", err)
	}
	t.Logf("Now global model %s: %s", res3[0], evalLastRes)

}

func printTxCheckResult(res transactioncheck.CheckResult) {
	fmt.Printf("Transaction ID: %x\n", res.TransactionID)
	fmt.Printf("Valid: %v\n", res.Valid)
	fmt.Printf("Transaction Results: %s\n", res.Result)
	fmt.Printf("Consensus Delay: %d ms\n", res.Interval.Milliseconds())
}
