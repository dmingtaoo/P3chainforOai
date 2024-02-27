package dper

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/consensus"
	"p3Chain/core/contract"
	"p3Chain/core/dpnet"
	"p3Chain/core/viewChange"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/utils"
	"testing"
	"time"
)

var booterAddr = "127.0.0.1:20130"
var (
	viewChangeBooterKeyAddr = common.HexToAddress("f001bd944b38abf0a41ad2c5140918d3d83140d2") //once run TestGenerateOrderServiceServer, it should be changed
)

var dperAddr = []string{
	"127.0.0.1:20131",
	"127.0.0.1:20132",
	"127.0.0.1:20133",
	"127.0.0.1:20134",
	"127.0.0.1:20135",
	"127.0.0.1:20136",
}

var viewChangeDpCfg1, viewChangeDpCfg2, viewChangeDpCfg3, viewChangeDpCfg4, viewChangeDpCfg5, viewChangeDpCfg6 *DperConfig
var viewChangeBooterCfg *OrderServiceServerConfig

func InitConfigsViewChange() error {
	bootStrapNodes, err := utils.ReadNodesUrl(booterUrlFile)
	if err != nil {
		return fmt.Errorf("fail in read boot strap node, %v", err)
	}

	viewChangeBooterCfg = &OrderServiceServerConfig{
		NewAddressMode:    false,
		SelfNodeAddress:   viewChangeBooterKeyAddr,
		KeyStoreDir:       testKeyStore_OrderServiceServer,
		ServerName:        "test_booter",
		ListenAddr:        booterAddr,
		NAT:               nat.Any(),
		BootstrapNodes:    make([]*discover.Node, 0),
		MaxPeers:          20,
		CentralConfigMode: false,
	}
	// net1四个节点(1个Leader,三个Follower)作为故障分区   net2两个节点(1个Leader,1个Follower)
	viewChangeDpCfg1 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper1",
		ListenAddr:          dperAddr[0],
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
	viewChangeDpCfg2 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper2",
		ListenAddr:          dperAddr[1],
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
	viewChangeDpCfg3 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper3",
		ListenAddr:          dperAddr[2],
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
	viewChangeDpCfg4 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper4",
		ListenAddr:          dperAddr[3],
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

	// net2
	viewChangeDpCfg5 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper5",
		ListenAddr:          dperAddr[4],
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

	viewChangeDpCfg6 = &DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper6",
		ListenAddr:          dperAddr[5],
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

func TestViewChange(t *testing.T) {
	if err := InitConfigsViewChange(); err != nil {
		t.Fatalf("fail in init configs, %v", err)
		return
	}

	booter, err := NewOrderServiceServer(viewChangeBooterCfg)
	if err != nil {
		t.Fatalf("fail in new booter, %v", err)
	}

	dper1 := NewDper(viewChangeDpCfg1)

	dper2 := NewDper(viewChangeDpCfg2)

	dper3 := NewDper(viewChangeDpCfg3)

	dper4 := NewDper(viewChangeDpCfg4)

	dper5 := NewDper(viewChangeDpCfg5)

	dper6 := NewDper(viewChangeDpCfg6)

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

	err = dper5.StateSelf(false)
	if err != nil {
		t.Fatalf("fail in state dper, %v", err)
	}
	err = dper6.StateSelf(false)
	if err != nil {
		t.Fatalf("fail in state dper, %v", err)
	}

	time.Sleep(2 * time.Second)
	t.Logf("booter view net: %s", booter.netManager.BackViewNetInfo())
	t.Logf("dper1 view net: %s", dper1.netManager.BackViewNetInfo())
	t.Logf("dper2 view net: %s", dper2.netManager.BackViewNetInfo())
	t.Logf("dper3 view net: %s", dper3.netManager.BackViewNetInfo())
	t.Logf("dper4 view net: %s", dper4.netManager.BackViewNetInfo())
	t.Logf("dper5 view net: %s", dper3.netManager.BackViewNetInfo())
	t.Logf("dper6 view net: %s", dper4.netManager.BackViewNetInfo())

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

	err = dper5.ConstructDpnet()
	if err != nil {
		t.Fatalf("fail in construct dper dpnet, %v", err)
	}
	err = dper6.ConstructDpnet()
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
	err = dper5.Start()
	if err != nil {
		t.Fatalf("fail in start dper, %v", err)
	}
	err = dper6.Start()
	if err != nil {
		t.Fatalf("fail in start dper, %v", err)
	}
	time.Sleep(1 * time.Second)

	// 让测试分区net1的各节点注册view-change服务
	net1Leader := dper1.selfNode.NodeID

	node1viewChange := viewChange.NewViewChange(dper1.consensusPromoter, net1Leader, dper1.consensusPromoter.BackTargetTPBFT(consensus.COMMON_TPBFT_NAME), dper1.BackSyn())
	node2viewChange := viewChange.NewViewChange(dper2.consensusPromoter, net1Leader, dper2.consensusPromoter.BackTargetTPBFT(consensus.COMMON_TPBFT_NAME), dper2.BackSyn())
	node3viewChange := viewChange.NewViewChange(dper3.consensusPromoter, net1Leader, dper3.consensusPromoter.BackTargetTPBFT(consensus.COMMON_TPBFT_NAME), dper3.BackSyn())
	node4viewChange := viewChange.NewViewChange(dper4.consensusPromoter, net1Leader, dper4.consensusPromoter.BackTargetTPBFT(consensus.COMMON_TPBFT_NAME), dper4.BackSyn())

	go node1viewChange.ViewChangeStart()
	go node2viewChange.ViewChangeStart()
	go node3viewChange.ViewChangeStart()
	go node4viewChange.ViewChangeStart()

	time.Sleep(3 * time.Second)

	// 建链完毕，发布交易进行检测

	// account1, err := dper3.accountManager.NewAccount("")
	// if err != nil {
	// 	t.Fatalf("fail in create new account, %v", err)
	// }
	// err = dper3.accountManager.Unlock(account1.Address, "")
	// if err != nil {
	// 	t.Fatalf("fail in unlock account, %v", err)
	// }

	// dper3.StartCheckTransaction() // 开启交易检测模块

	// printTxCheckResult := func(res transactioncheck.CheckResult) {
	// 	fmt.Printf("Transaction ID: %x\n", res.TransactionID)
	// 	fmt.Printf("Valid: %v\n", res.Valid)
	// 	fmt.Printf("Transaction Results: %s\n", res.Result)
	// 	fmt.Printf("Consensus Delay: %d ms\n", res.Interval.Milliseconds())
	// }

	// cresult, err := dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("Leo1"), []byte("Awesome1")})
	// if err != nil {
	// 	t.Fatalf("fail in publish transaction, %v", err)
	// }
	// printTxCheckResult(cresult)

	// cresult, err = dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("Leo2"), []byte("Awesome2")})
	// if err != nil {
	// 	t.Fatalf("fail in publish transaction, %v", err)
	// }
	// printTxCheckResult(cresult)

	// 在view-change完成前打印upper层
	upperGroup, err := dper5.BackNetManager().BackUpperChannel() // testnet2 leader
	if err != nil {
		fmt.Printf("[testnet2 Leader] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Leader] 更新前: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Leader] 更新前: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}

	upperGroup, err = dper6.BackNetManager().BackUpperChannel() // testnet2 follower
	if err != nil {
		fmt.Printf("[testnet2 Follower] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Follower] 更新前: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Follower] 更新前: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}

	time.Sleep(150 * time.Second) // 等待一个view-change

	// 在完成view-change之后打印upper层，查看是否完成切换
	upperGroup, err = dper5.BackNetManager().BackUpperChannel() // testnet2 leader
	if err != nil {
		fmt.Printf("[testnet2 Leader] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Leader] 更新后: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Leader] 更新后: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}

	upperGroup, err = dper6.BackNetManager().BackUpperChannel() // testnet2 follower
	if err != nil {
		fmt.Printf("[testnet2 Follower] 获取Upper Channel层失败,err:%v", err)
	} else {
		upperNodes := upperGroup.BackMembers()
		if len(upperNodes) == 0 {
			fmt.Printf("[testnet2 Follower] 更新后: upper层没有发现其他Leader节点\n")
		}
		for _, nodeID := range upperNodes {
			fmt.Printf("[testnet2 Follower] 更新后: upper层节点 -- %x\n", nodeID)
		}
		fmt.Println()
	}

	// // 重新发交易
	// cresult, err = dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("big1"), []byte("handsome1")})
	// if err != nil {
	// 	t.Fatalf("fail in publish transaction, %v", err)
	// }
	// printTxCheckResult(cresult)

	// cresult, err = dper3.SimplePublishTransaction(account1, contract.DEMO_CONTRACT_1, contract.DEMO_CONTRACT_1_FuncSetAddr, [][]byte{[]byte("big2"), []byte("handsome2")})
	// if err != nil {
	// 	t.Fatalf("fail in publish transaction, %v", err)
	// }
	// printTxCheckResult(cresult)

	// dper3.CloseCheckTransaction() //卸载交易检测模块

}
