package dper

import (
	//"bytes"

	"p3Chain/common"
	"p3Chain/core/contract"

	//chaincodesupport "p3Chain/core/contract/chainCodeSupport"
	"fmt"
	"p3Chain/core/dpnet"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	"p3Chain/crypto/randentropy"
	transactioncheck "p3Chain/dper/transactionCheck"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/rlp"
	"p3Chain/utils"

	//"net"
	//"net/rpc"
	"os"
	"testing"
	"time"
)

const (
	testKeyStore                    = "./keyStoreForTest"
	testKeyStore_OrderServiceServer = testKeyStore + "/orderServiceServerKey"
	testKeyStore_dpers              = testKeyStore + "/dperKey"

	testUrlDir    = "./testNodeUrl"
	booterUrlFile = testUrlDir + "/booterUrl.txt"
)

var booterListenAddr = "127.0.0.1:20130"
var (
	testBooterKeyAddr = common.HexToAddress("f001bd944b38abf0a41ad2c5140918d3d83140d2") //once run TestGenerateOrderServiceServer, it should be changed
)

var dperListenAddr = []string{
	"127.0.0.1:20131",
	"127.0.0.1:20132",
	"127.0.0.1:20133",
	"127.0.0.1:20134",
}

func TestGenerateOrderServiceServer(t *testing.T) {
	err := os.Mkdir(testKeyStore_OrderServiceServer, 0750)
	if err != nil {
		if os.IsExist(err) {
			t.Logf("Key store path: %s, is existed", testKeyStore_OrderServiceServer)
		} else {
			t.Fatalf("fail in make dir, %v", err)
		}
	}
	keyStore := keys.NewKeyStorePlain(testKeyStore_OrderServiceServer)
	newKey, err := keyStore.GenerateNewKey(randentropy.Reader, "")
	if err != nil {
		t.Fatalf("fail in generate key, %v", err)
	}
	publicKey := &newKey.PrivateKey.PublicKey
	nodeID := crypto.KeytoNodeID(publicKey)
	url, err := utils.GenerateNodeUrl(nodeID, booterListenAddr)
	if err != nil {
		t.Fatalf("fail in generate url, %v", err)
	}
	err = utils.SaveString(url, booterUrlFile)
	if err != nil {
		t.Fatalf("fail in save url, %v", err)
	}
}

func TestReadBooterKey(t *testing.T) {
	err := os.Mkdir(testKeyStore_OrderServiceServer, 0750)
	if err != nil {
		if os.IsExist(err) {
			t.Logf("Key store path: %s, is existed", testKeyStore_OrderServiceServer)
		} else {
			t.Fatalf("fail in make dir, %v", err)
		}
	}
	keyStore := keys.NewKeyStorePlain(testKeyStore_OrderServiceServer)
	loadKey, err := keyStore.GetKey(testBooterKeyAddr, "")
	if err != nil {
		t.Fatalf("fail in load key, %v", err)
	}
	publicKey := &loadKey.PrivateKey.PublicKey
	nodeID := crypto.KeytoNodeID(publicKey)
	t.Logf("Succeed load key with addr: %x, the node id is: %x", testBooterKeyAddr, nodeID)

}

func TestGenerateOrderServiceServer_PassPhrase(t *testing.T) {
	err := os.Mkdir(testKeyStore_OrderServiceServer, 0750)
	if err != nil {
		if os.IsExist(err) {
			t.Logf("Key store path: %s, is existed", testKeyStore_OrderServiceServer)
		} else {
			t.Fatalf("fail in make dir, %v", err)
		}
	}
	keyStore := keys.NewKeyStorePassphrase(testKeyStore_OrderServiceServer)
	newKey, err := keyStore.GenerateNewKey(randentropy.Reader, "")
	if err != nil {
		t.Fatalf("fail in generate key, %v", err)
	}
	publicKey := &newKey.PrivateKey.PublicKey
	nodeID := crypto.KeytoNodeID(publicKey)
	url, err := utils.GenerateNodeUrl(nodeID, booterListenAddr)
	if err != nil {
		t.Fatalf("fail in generate url, %v", err)
	}
	err = utils.SaveString(url, booterUrlFile)
	if err != nil {
		t.Fatalf("fail in save url, %v", err)
	}
}

func TestReadBooterKey_PassPhrase(t *testing.T) {
	err := os.Mkdir(testKeyStore_OrderServiceServer, 0750)
	if err != nil {
		if os.IsExist(err) {
			t.Logf("Key store path: %s, is existed", testKeyStore_OrderServiceServer)
		} else {
			t.Fatalf("fail in make dir, %v", err)
		}
	}
	keyStore := keys.NewKeyStorePassphrase(testKeyStore_OrderServiceServer)
	loadKey, err := keyStore.GetKey(testBooterKeyAddr, "")
	if err != nil {
		t.Fatalf("fail in load key, %v", err)
	}
	publicKey := &loadKey.PrivateKey.PublicKey
	nodeID := crypto.KeytoNodeID(publicKey)
	t.Logf("Succeed load key with addr: %x, the node id is: %x", testBooterKeyAddr, nodeID)

}

var testDpCfg1, testDpCfg2, testDpCfg3, testDpCfg4 *DperConfig
var testBooterCfg *OrderServiceServerConfig

func InitConfigs() error {
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
		ContractEngine:      contract.DEMO_CONTRACT_1_NAME,
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
		ContractEngine:      contract.DEMO_CONTRACT_1_NAME,
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
		ContractEngine:      contract.DEMO_CONTRACT_1_NAME,
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
		ContractEngine:      contract.DEMO_CONTRACT_1_NAME,
	}
	return nil
}

func InitConfigs2() error {
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
		ContractEngine:      contract.DEMO_CONTRACT_MIX123_NAME,
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
		ContractEngine:      contract.DEMO_CONTRACT_MIX123_NAME,
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
		ContractEngine:      contract.DEMO_CONTRACT_MIX123_NAME,
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
		ContractEngine:      contract.DEMO_CONTRACT_MIX123_NAME,
	}
	return nil
}

func TestP3Chain(t *testing.T) {
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

	// dsm, ok := dper3.stateManager.(*worldstate.DpStateManager)
	// if !ok {
	// 	loglogrus.Log.Errorf("类型转换失败")
	// }
	// // _ = dsm
	// blockService := api.NewBlockService(dsm)

	// ginHttp.NewGinRouter(blockService, nil)

	// loglogrus.Log.Debugf("当前区块高度:%v", dsm.GetBlockChainHeight())

	// blockCount := blockService.GetCurrentBlockNumber()
	// loglogrus.Log.Debugf("当前dper3已有区块数目为:%d", blockCount)

	// block := blockService.GetBlockByNumber(2)
	// loglogrus.Log.Debugf("区块编号:%d,包含交易数:%d\n", block.Number, len(block.RawBlock.Transactions))

	// block = blockService.GetBlockByNumber(1)
	// loglogrus.Log.Debugf("区块编号:%d,包含交易数:%d\n", block.Number, len(block.RawBlock.Transactions))

	// block = blockService.GetBlockByNumber(0)
	// loglogrus.Log.Debugf("区块编号:%d,包含交易数:%d\n", block.Number, len(block.RawBlock.Transactions))

	// for {
	// 	time.Sleep(time.Second)
	// }
}

var testPurchaseForm = contract.PurchaseForm{
	Process_id:          "hcuiahciuyfhineuorfbhiuoreafherqf",
	Contract_identifier: "test",
	Buyer:               "rand_buyer",
	Provider_name:       "provider_name",
	Goods_name:          "goods_name",
	Goods_type:          "goods_type",
	Goods_specification: "goods_specification",
	Goods_amount:        "goods_amount",
	Goods_unit:          "goods_unit",
	Goods_price:         "goods_price",
	Goods_total_price:   "goods_total_price",
	Due_and_amount:      "due_and_amount",
	Sign_date:           "sign_date",
}
var testReceiptForm = contract.ReceiptForm{
	Process_id:          "hcuiahciuyfhineuorfbhiuoreafherqf",
	Contract_identifier: "test",
	Buyer:               "test",
	Provider_name:       "test",
	Goods_name:          "test",
	Goods_type:          "test",
	Goods_specification: "test",
	Goods_fahuo_amount:  "test",
	Goods_unit:          "test",
	Goods_price:         "test",
	Goods_total_price:   "test",
	Date:                "test",
}
var puchaseFormCode, _ = rlp.EncodeToBytes(testPurchaseForm)
var receiptFormCode, _ = rlp.EncodeToBytes(testReceiptForm)

// NOTE: not keep maintain
// it seems not work now
func TestP3Chain2(t *testing.T) {
	glog.SetToStderr(true)
	err := InitConfigs2()

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

	account1, err := dper3.accountManager.NewAccount("account1")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	account2, err := dper3.accountManager.NewAccount("account2")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	account3, err := dper3.accountManager.NewAccount("account3")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	account4, err := dper3.accountManager.NewAccount("account4")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	account5, err := dper3.accountManager.NewAccount("account5")
	if err != nil {
		t.Fatalf("fail in create new account, %v", err)
	}
	err = dper3.accountManager.Unlock(account1.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}
	err = dper3.accountManager.Unlock(account2.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}
	err = dper3.accountManager.Unlock(account4.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}
	err = dper3.accountManager.Unlock(account5.Address, "")
	if err != nil {
		t.Fatalf("fail in unlock account, %v", err)
	}
	//group
	accountAddr2 := fmt.Sprintf("%x", account2.Address)
	//this traction is invalid
	_, err = dper3.SimpleInvokeTransaction(account1, contract.DEMO_CONTRACT_2, contract.DEMO_CONTRACT_2_FuncSetAddr, [][]byte{[]byte("group1"), []byte(accountAddr2), []byte("10000")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account1, contract.DEMO_CONTRACT_2, contract.DEMO_CONTRACT_2_FuncConfigAddr, [][]byte{[]byte("group1")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account1, contract.DEMO_CONTRACT_2, contract.DEMO_CONTRACT_2_FuncSetAddr, [][]byte{[]byte("group1"), []byte(accountAddr2), []byte("10000")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	accountAddr3 := fmt.Sprintf("%x", account3.Address)
	_, err = dper3.SimpleInvokeTransaction(account2, contract.DEMO_CONTRACT_2, contract.DEMO_CONTRACT_2_FuncSendAddr, [][]byte{[]byte("group1"), []byte(accountAddr3), []byte("100")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}

	time.Sleep(5 * time.Second)
	result := dper4.SimpleInvokeTransactionLocally(account2, contract.DEMO_CONTRACT_2, contract.DEMO_CONTRACT_2_FuncBalanceAddr, [][]byte{[]byte("group1")})
	if len(result) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("%s", result[0])
	result2 := dper4.SimpleInvokeTransactionLocally(account3, contract.DEMO_CONTRACT_2, contract.DEMO_CONTRACT_2_FuncBalanceAddr, [][]byte{[]byte("group1")})
	if len(result2) == 0 {
		t.Fatalf("transaction is not executed")
	}
	t.Logf("%s", result2[0])

	//supplly

	_, err = dper3.SimpleInvokeTransaction(account4, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncCreateAddr, [][]byte{
		[]byte("hcuiahciuyfhineuorfbhiuoreafherqf"),
		account5.Address[:],
		[]byte("debtor_name"),
		[]byte("credit_side_name")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account4, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncSetPurAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf"), puchaseFormCode})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account4, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncSetReceiptAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf"), receiptFormCode})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account5, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncSignCreditAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account4, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncSignPayAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account5, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncDeliveryAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}
	time.Sleep(5 * time.Second)
	_, err = dper3.SimpleInvokeTransaction(account4, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncReceiveAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")})
	if err != nil {
		t.Fatalf("fail in invoke transaction, %v", err)
	}

	time.Sleep(5 * time.Second)
	result3 := dper4.SimpleInvokeTransactionLocally(account4, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncGetPurAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")})
	if len(result3) == 0 {
		t.Fatalf("transaction is not executed")
	}
	tmp := new(contract.PurchaseForm)
	err = rlp.DecodeBytes(result3[0], tmp)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("%+v %d", tmp, len(result3[0]))

	time.Sleep(5 * time.Second)
	result4 := dper4.SimpleInvokeTransactionLocally(account4, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncGetProcessAddr, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")})
	if len(result4) == 0 {
		t.Fatalf("transaction is not executed")
	}
	tmp2 := new(contract.ProcessForm)
	err = rlp.DecodeBytes(result4[0], tmp2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("%+v %d", tmp2, len(result4[0]))
}
