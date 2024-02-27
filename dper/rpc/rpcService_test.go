package rpc

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/contract"
	"p3Chain/core/dpnet"
	"p3Chain/dper"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/utils"
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

var testDpCfg1, testDpCfg2, testDpCfg3, testDpCfg4 *dper.DperConfig
var testBooterCfg *dper.OrderServiceServerConfig

func InitConfigs() error {
	bootStrapNodes, err := utils.ReadNodesUrl(booterUrlFile)
	if err != nil {
		return fmt.Errorf("fail in read boot strap node, %v", err)
	}

	testBooterCfg = &dper.OrderServiceServerConfig{
		NewAddressMode:    false,
		SelfNodeAddress:   testBooterKeyAddr,
		KeyStoreDir:       testKeyStore_OrderServiceServer,
		ServerName:        "test_booter",
		ListenAddr:        booterListenAddr,
		NAT:               nat.Any(),
		BootstrapNodes:    make([]*discover.Node, 0),
		MaxPeers:          10,
		CentralConfigMode: false,
	}

	testDpCfg1 = &dper.DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper1",
		ListenAddr:          dperListenAddr[0],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            10,
		MemoryDataBaseMode:  true,
		BlockDataBasePath:   "",
		StorageDataBasePath: "",
		Role:                dpnet.Leader,
		NetID:               "net1",
		CentralConfigMode:   false,
		ContractEngine:      contract.DEMO_CONTRACT_MIX123_NAME,
	}
	testDpCfg2 = &dper.DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper2",
		ListenAddr:          dperListenAddr[1],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            10,
		MemoryDataBaseMode:  true,
		BlockDataBasePath:   "",
		StorageDataBasePath: "",
		Role:                dpnet.Leader,
		NetID:               "net2",
		CentralConfigMode:   false,
		ContractEngine:      contract.DEMO_CONTRACT_MIX123_NAME,
	}
	testDpCfg3 = &dper.DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper3",
		ListenAddr:          dperListenAddr[2],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            10,
		MemoryDataBaseMode:  true,
		BlockDataBasePath:   "",
		StorageDataBasePath: "",
		Role:                dpnet.Follower,
		NetID:               "net1",
		CentralConfigMode:   false,
		ContractEngine:      contract.DEMO_CONTRACT_MIX123_NAME,
	}
	testDpCfg4 = &dper.DperConfig{
		NewAddressMode:      true,
		DperKeyAddress:      common.Address{},
		KeyStoreDir:         testKeyStore_dpers,
		ServerName:          "test_dper4",
		ListenAddr:          dperListenAddr[3],
		NAT:                 nat.Any(),
		BootstrapNodes:      bootStrapNodes,
		MaxPeers:            10,
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

// this test is to simulate constructing a P3-Chain with 4 nodes and launch the RPC interface
func TestRpcRun(t *testing.T) {

	glog.SetToStderr(true)
	err := InitConfigs()

	if err != nil {
		t.Fatalf("fail in init configs, %v", err)
	}

	booter, err := dper.NewOrderServiceServer(testBooterCfg)
	if err != nil {
		t.Fatalf("fail in new booter, %v", err)
	}
	dper1 := dper.NewDper(testDpCfg1)

	dper2 := dper.NewDper(testDpCfg2)

	dper3 := dper.NewDper(testDpCfg3)

	dper4 := dper.NewDper(testDpCfg4)

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
	t.Logf("P3-Chain is contrcuted")

	RpcContractSupplyWith54front(dper3, "tcp", "127.0.0.1:5100")

}
