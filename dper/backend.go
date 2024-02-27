package dper

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"p3Chain/accounts"
	"p3Chain/blockSync"
	"p3Chain/common"
	"p3Chain/core/blockOrdering"
	"p3Chain/core/centre"
	"p3Chain/core/consensus"
	"p3Chain/core/contract"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/netconfig"
	"p3Chain/core/separator"
	"p3Chain/core/validator"
	"p3Chain/core/worldstate"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	"p3Chain/crypto/randentropy"
	"p3Chain/database"
	"p3Chain/dper/transactionCheck"
	loglogrus "p3Chain/log_logrus"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
)

const (
	BlockDatabaseName      = "BlockDB"
	WorldStateDatabaseName = "WSDB"

	BenchMarkFetch = 1000

	// archiveScanInterval = 10 * time.Second // 归档协程扫描archiveThreshold间隔
	// archiveThreshold    = 60               // 本地存储区块数量达到该阈值后，触发归档(必须要大于50 --- 满足交易过滤器的区块数量需求) 适当配置，不能太低，否则区块同步将失效
)

var (
	REPEAT_CHECK_ENABLE = false
)

// 区块归档服务
type DataAchive struct {
	ArchiveMode         bool
	ArchiveScanInterval time.Duration
	ArchiveThreshold    int
}

var DataAchiveSetting = &DataAchive{}

// Dper is the actual implementation of P3-Chain. It contains all the databases, managers, networks and so on.
type Dper struct {
	selfKey        keys.Key
	selfNode       *dpnet.Node
	booterAddr     string
	blockDB        database.Database //本地维护一个区块数据库
	storageDB      database.Database //本地维护一个存储数据库
	accountManager *accounts.Manager

	netManager        *netconfig.NetManager //本地维护一个P3-Chain网络
	p2pServer         *server.Server
	stateManager      worldstate.StateManager
	intraStateManager worldstate.StateManager
	contractEngine    contract.ContractEngine
	contractName      string
	mtwSchemes        []string

	validateManager   *validator.ValidateManager
	consensusPromoter *consensus.ConsensusPromoter

	syncProtocal *blockSync.SyncProtocal

	txCheckPool  *transactionCheck.TxCheckPool
	txCheckChan  chan common.Hash
	txCheckClose chan bool

	transactionFilter *eles.TransactionFilter

	blockChain *eles.BlockChain

	CentralConfigMode bool
}

// Config configures the Dper.
type DperConfig struct {
	// basic information
	NewAddressMode bool //whether to create a new account when start
	DperKeyAddress common.Address
	DperPrivateKey *ecdsa.PrivateKey
	KeyStoreDir    string

	// connect configuration
	ServerName        string
	ListenAddr        string
	InitBooterAddress string
	NAT               nat.Interface
	BootstrapNodes    []*discover.Node //整个P3-Chain中已知的引导节点
	MaxPeers          int

	// about database
	MemoryDataBaseMode    bool // whether to use memory mode
	BlockDataBasePath     string
	StorageDataBasePath   string
	StorageV2DataBasePath string

	// about dpnet configuration
	Role              dpnet.RoleType // the role in subnet
	NetID             string         //所属的网络ID
	CentralConfigMode bool           // whether to use central protocol to config dpnet
	TotalDperCount    int            // Dper节点的总数量
	GroupCount        int            // 分区数量
	GroupLeader       int            // 分区的Leader数量
	GroupFollower     int            // 分区的Follower数量

	// about contract engine
	ContractEngine        string
	RemoteSupportPipeName string
	RemoteEnginePipeName  string
	MTWSchemes            []string
}

type DPNetwork struct {
	SubnetCount int `json:"subnetCount"`
	LeaderCount int `json:"leaderCount"`
	BooterCount int `json:"booterCount"`

	NetGroups []NetGroup `json:"netGroups"`
}

type NetGroup struct {
	NetID       string `json:"netID"`
	MemberCount int    `json:"memberCount"`
}

// 获取dper所属的网络的ID
func (dp *Dper) GetNetID() string {
	return dp.netManager.GetSelfNodeNetID()
}

// 获取depr在网络中的身份
func (dp *Dper) GetNodeRole() dpnet.RoleType {
	return dp.netManager.GetSelfNodeRole()
}

func (dp *Dper) BackStateManager() worldstate.StateManager {
	return dp.stateManager
}

func (dp *Dper) BackNetManager() *netconfig.NetManager {
	return dp.netManager
}

func (dp *Dper) CloseDiskDB() {
	dp.blockDB.Close()
	dp.storageDB.Close()
}

func saveAddress(path string, address common.Address) {
	byteValue, err := os.ReadFile(path) //读取json文件
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to open Dper configuration file at the specified path:%s ,err:%v\n", path, err)
		panic("NewDper failed: Unable to open Dper configuration file at the specified path!")
	}
	result := make(map[string]interface{}, 0)
	err = json.Unmarshal(byteValue, &result) //解析json k-v对
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to Unmarshal Dper configuration file, err:%v\n", err)
		panic("NewDper failed: Unable to Unmarshal Dper configuration file!")

	}
	result["DperKeyAddress"] = fmt.Sprintf("%x", address)

	temp, _ := json.MarshalIndent(result, "", "")
	err = os.WriteFile(path, temp, 0644)
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to modify Dper configuration file, err:%v\n", err)
		panic("NewDper failed: Unable to modify Dper configuration file!")
	}
}

// 初始化一个Dper对象
func NewDper(cfg *DperConfig) *Dper {
	err := os.Mkdir(cfg.KeyStoreDir, 0750)
	if err != nil && !os.IsExist(err) {
		loglogrus.Log.Errorf("NewDper failed: can't to create key store according to the given path:%s\n", cfg.KeyStoreDir)
		panic("NewDper failed: Failed to create key store according to the given path!")
	}
	// keyStore := keys.NewKeyStorePlain(cfg.KeyStoreDir) //创建一个key管理接口(负责key的增删改查)
	keyStore := keys.NewKeyStorePassphrase(cfg.KeyStoreDir) //创建一个key管理接口(负责key的增删改查)
	loglogrus.Log.Infof("NewDper: Initialize key manager...")
	accountManager := accounts.NewManager(keyStore) //创建一个账户管理对象
	var selfKey *keys.Key
	if cfg.NewAddressMode { //"NewAddressMode":"true"则生成新的Address   "NewAddressMode":"false"则读取"SelfNodeAddress"指定的Address
		var err error
		selfKey, err = keyStore.GenerateNewKey(randentropy.Reader, "")
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: can't to generate new key for current Dper ,err:%v\n", err)
			panic("NewDper failed: Generate New Key is failed!")
		}
		loglogrus.Log.Infof("NewDper: Complete generating key manager!")

	} else {
		var err error
		rootPath, _ := os.Getwd()

		dirPath := rootPath + string(os.PathSeparator) + strings.TrimLeft(cfg.KeyStoreDir, "./")
		files, err := ioutil.ReadDir(dirPath)
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: Unable to find the account directory of the current node in the specified path:%s\n", dirPath)
			panic("NewDper failed: Unable to reach the account directory of the current node!")
		}
		if len(files) == 0 {
			selfKey, err = keyStore.GenerateNewKey(randentropy.Reader, "")
			if err != nil {
				loglogrus.Log.Warnf("NewDper Warn: Can't generate new Key for current Dper,err:%v\n", err)
				selfKey, _ = keyStore.GenerateNewKey(randentropy.Reader, "") //TODO: 是否会出现反复创建不成功的情况？
			}
			jsonConfigPath := rootPath + string(os.PathSeparator) + "settings" + string(os.PathSeparator) + "dperConfig.json"
			saveAddress(jsonConfigPath, selfKey.Address)
			loglogrus.Log.Infof("NewDper: Welcome to Dper for the first time, the first account has been automatically generated for you!\n")

		} else {
			selfKey, err = keyStore.GetKey(cfg.DperKeyAddress, "")
			if err != nil {
				loglogrus.Log.Errorf("NewDper failed: The account you specified cannot be found in the account directory, err:%v\n", err)
				panic("NewDper failed: The specified account does not exist, login failed!")
			}
		}
		loglogrus.Log.Infof("NewDper: Complete loading key manager!\n")
	}

	publicKey := &selfKey.PrivateKey.PublicKey //从selfKey获取公钥

	selfNode := dpnet.NewNode(crypto.KeytoNodeID(publicKey), int(cfg.Role), cfg.NetID) //根据已知配置信息,生成dpnet.Node对象

	loglogrus.Log.Infof("NewDper failed: Complete creating dpnet.Node for self: NodeID: %x ,NetID: %s , Role: %v\n", selfNode.NodeID, selfNode.NetID, selfNode.Role)

	loglogrus.Log.Infof("NewDper: Initialize database...")
	var blockDB, storageDB, storageV2DB database.Database
	if cfg.MemoryDataBaseMode {
		memDB1, err := leveldb.Open(storage.NewMemStorage(), nil)
		if err != nil {
			loglogrus.Log.Warnf("NewDper Warn: Can't Create New memDB for BlockChain,err:%v\n", err)
			memDB1, _ = leveldb.Open(storage.NewMemStorage(), nil)
		}
		memDB2, err := leveldb.Open(storage.NewMemStorage(), nil)
		if err != nil {
			loglogrus.Log.Warnf("NewDper Warn: Can't Create New memDB for Storage,err:%v", err)
			memDB2, _ = leveldb.Open(storage.NewMemStorage(), nil)
		}
		memDB3, err := leveldb.Open(storage.NewMemStorage(), nil)
		if err != nil {
			loglogrus.Log.Warnf("NewDper Warn: Can't Create New memDB for Storage,err:%v", err)
			memDB3, _ = leveldb.Open(storage.NewMemStorage(), nil)
		}
		blockDB = database.NewSimpleLDB(BlockDatabaseName, memDB1)
		storageDB = database.NewSimpleLDB(WorldStateDatabaseName, memDB2)
		storageV2DB = database.NewSimpleLDB(WorldStateDatabaseName, memDB3)
	} else {
		DB1, err := database.NewLDBDatabase(cfg.BlockDataBasePath)
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: Can't Load Persistent data from Specified path:%s , err:%v\n", cfg.BlockDataBasePath, err)
			panic("NewDper failed: Can't Load Persistent data from Specified path")

		}
		DB2, err := database.NewLDBDatabase(cfg.StorageDataBasePath)
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: Can't Load Persistent data from Specified path:%s , err:%v\n", cfg.StorageDataBasePath, err)
			panic("NewDper failed: Can't Load Persistent data from Specified path")
		}
		DB3, err := database.NewLDBDatabase(cfg.StorageV2DataBasePath)
		if err != nil {
			loglogrus.Log.Errorf("NewDper failed: Can't Load Persistent data from Specified path:%s , err:%v\n", cfg.StorageV2DataBasePath, err)
			panic("NewDper failed: Can't Load Persistent data from Specified path")
		}
		blockDB = DB1
		storageDB = DB2
		storageV2DB = DB3
	}
	loglogrus.Log.Infof("NewDper: Complete loading database!\n")

	loglogrus.Log.Infof("NewDper: Initialize blockchain...\n")
	var transactionFilter *eles.TransactionFilter
	if REPEAT_CHECK_ENABLE {
		transactionFilter = eles.NewTransactionFilter()
	}

	TxCheckPool := transactionCheck.NewTaskPool() //创建一个交易检查事件池

	blockchain := eles.InitBlockChain(blockDB, TxCheckPool, transactionFilter, DataAchiveSetting.ArchiveMode)

	loglogrus.Log.Infof("NewDper: Complete initializing blockchain!\n")

	loglogrus.Log.Infof("NewDper: Initialize state manager...\n")
	stateManager := worldstate.NewDpStateManager(selfNode, storageDB, blockchain)
	loglogrus.Log.Infof("NewDper: Complete initializing state manager\n")

	loglogrus.Log.Infof("NewDper: Initialize the lagging state manager...\n")
	intraStateManager := worldstate.NewIntraStateManager(selfNode, storageV2DB, blockchain)
	err = intraStateManager.InitISM()
	if err != nil {
		panic("NewDper failed: Can't initialize the lagging state manager!")
	}
	loglogrus.Log.Infof("NewDper: Complete initializing the lagging state manager\n")

	loglogrus.Log.Infof("NewDper: Launch contract engine...\n")
	var contractEngine contract.ContractEngine

	switch cfg.ContractEngine {
	case contract.DEMO_PIPE_CONTRACT_NAME:
		tmp := contract.CreatePipeContract(stateManager, cfg.RemoteEnginePipeName)
		contractEngine = tmp
		go contract.PipeChainCodeSupport(tmp, cfg.RemoteSupportPipeName)
	case contract.DEMO_CONTRACT_1_NAME:
		contractEngine, err = contract.CreateDemo_Contract_1(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO1!")
		}
	case contract.DEMO_CONTRACT_2_NAME:
		contractEngine, err = contract.CreateDemo_Contract_2(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO2!")
		}
	case contract.DEMO_CONTRACT_3_NAME:
		contractEngine, err = contract.CreateDemo_Contract_3(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO3!")
		}
	case contract.DEMO_CONTRACT_MIX123_NAME:
		contractEngine, err = contract.CreateDemo_Contract_MIX123(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO_MAX123!")
		}
	case contract.DEMO_CONTRACT_FL_NAME:
		contractEngine, err = contract.CreateDemo_Contract_FL(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO_FL!")
		}
	case contract.DEMO_CONTRACT_ADD_NAME:
		contractEngine, err = contract.CreateDemo_Contract_ADD(stateManager)
		if err != nil {
			panic("NewDper failed: Can't install Contract DEMO_ADD!")
		}
	default:
		loglogrus.Log.Errorf("NewDper failed: Unknown contract engine: %s\n", cfg.ContractEngine)
		panic("NewDper failed: Unknown contract engine")
	}
	blockchain.InstallExecuteBlockFunc(contractEngine.CommitWriteSet)
	loglogrus.Log.Infof("NewDper: Contract engine launched\n")

	loglogrus.Log.Infof("NewDper: Now start net manager...\n")
	protocols := make([]server.Protocol, 0)
	shh := whisper.New() //本节点的whisper客户端对象
	protocols = append(protocols, shh.Protocol())
	spr := separator.New() //本节点的separator客户端对象
	protocols = append(protocols, spr.Protocol())

	viewNet := dpnet.NewDpNet()
	syn := blockSync.New(blockchain, *selfNode, viewNet)

	protocols = append(protocols, syn.Protocol())
	var cen *centre.Centre
	if cfg.CentralConfigMode { //若cfg.CentralConfigMode == true ,则采用中心化配置方式完成dpnet组网
		// cen = centre.CentralNew(*selfNode, cfg.GroupCount, cfg.GroupLeader, cfg.GroupFollower)
		// protocols = append(protocols, cen.Protocol())

		// booter 节点和 dper 节点都要额外运行一个服务器协程,专门用于网络初始化
		DperNetWorkInit(*selfNode, cfg.InitBooterAddress, viewNet)

	}
	srv := &server.Server{ //本节点的服务器对象
		PrivateKey:     selfKey.PrivateKey, //自身私钥
		Discovery:      true,               //允许被其他节点discover
		BootstrapNodes: cfg.BootstrapNodes, //引导节点
		MaxPeers:       cfg.MaxPeers,       // TODO: be customized  最大连接的节点数
		Name:           cfg.ServerName,
		Protocols:      protocols,      //使用的协议
		ListenAddr:     cfg.ListenAddr, //监听的地址
		NAT:            cfg.NAT,        // TODO: should be customized  NAT映射器
	}

	srv.Start() //启动节点的服务器模块
	// shh.Start() starts a whisper message expiration thread. As the net config
	// messages are not massive, it could be ignored.
	shh.Start()

	netManager := netconfig.NewNetManager(selfKey.PrivateKey, viewNet, srv, shh, spr, cen, syn, 0, selfNode) //让本地节点维护一个本地的P3-Chain网络
	err = netManager.ViewNetAddSelf()
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Unable to join the current node to the corresponding partition network, err:%v\n", err)
		panic("NewDper failed: Unable to join the current node to the corresponding partition network!")
	}
	err = netManager.StartConfigChannel()
	if err != nil {
		loglogrus.Log.Errorf("NewDper failed: Failed to config lowerchannel consensus message processor for the whisper client module of the current node, err:%v\n", err)
		panic("NewDper failed: Failed to config lowerchannel consensus message processor for the whisper client module of the current node!")
	}

	mtwSchemes := []string{}
	if len(cfg.MTWSchemes) == 0 {
		mtwSchemes = append(mtwSchemes, "CTPBFT")
		loglogrus.Log.Warnf("No MTW scheme is defined, now use the default CTPBFT.")
	} else {
		mtwSchemes = append(mtwSchemes, cfg.MTWSchemes...)
	}

	loglogrus.Log.Infof("NewDper: Net manager hsa started\n")

	loglogrus.Log.Infof("NewDper: Now construct dper...\n")
	dper := &Dper{
		booterAddr:        cfg.InitBooterAddress,
		selfKey:           *selfKey,
		selfNode:          selfNode,
		blockDB:           blockDB,
		storageDB:         storageDB,
		accountManager:    accountManager,
		netManager:        netManager,
		p2pServer:         srv,
		stateManager:      stateManager,
		intraStateManager: intraStateManager,
		contractEngine:    contractEngine,
		contractName:      cfg.ContractEngine,
		syncProtocal:      syn,
		txCheckPool:       TxCheckPool,
		txCheckChan:       make(chan common.Hash),
		txCheckClose:      make(chan bool),
		transactionFilter: transactionFilter,
		mtwSchemes:        mtwSchemes,

		blockChain:        blockchain,
		CentralConfigMode: cfg.CentralConfigMode,
	}
	loglogrus.Log.Infof("NewDper: Succeed in constructing dper\n")

	go dper.ArchiveHistoryBlock(cfg.BlockDataBasePath, blockchain, contractEngine.CommitWriteSet)

	return dper
}

func DperNetWorkInit(self dpnet.Node, booterAddrStr string, viewNet *dpnet.DpNet) {

	var waitTime time.Duration // 从配置文件获取的初始化等待最大时长
	networkInit(&waitTime)

	// Dper 节点负责与已知的 Booter 节点发起 TCP 连接，并发送 stateSelf 消息，等待 Booter 的 viewNet 响应
	conn, err := net.Dial("tcp4", booterAddrStr)
	if err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Dper can't connect with Booter(%s), err:%v\n", booterAddrStr, err)
		return
	} else {
		loglogrus.Log.Infof("[Dper Init] Network Init, Dper(%s) has connect with Booter(%s)\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
	}
	stateSelfJSON, err := json.Marshal(&self)
	if err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Dper can't encode stateSelf Msg into json, err:%v\n", err)
		return
	}
	if _, err := conn.Write(stateSelfJSON); err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Dper can't send stateSelfMsg to Booter(%s), err:%v\n", booterAddrStr, err)
		return
	} else {
		loglogrus.Log.Infof("[Dper Init] Network Init,Dper(%s) has send stateSelfMsg(%v) bytes(%d) to Booter(%s)\n", conn.LocalAddr().String(), self, len(stateSelfJSON),
			conn.RemoteAddr().String())
	}
	conn.SetReadDeadline(time.Now().Add(waitTime))

	var rBytes bytes.Buffer
	io.Copy(&rBytes, conn)

	booterInitMsgs := make(map[string]*dpnet.Node)
	if err := json.Unmarshal(rBytes.Bytes(), &booterInitMsgs); err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Dper can't decode JsonMsg From Booter(%s)\n", conn.RemoteAddr().String())
		return
	} else {
		loglogrus.Log.Infof("[Dper Init] Network Init, Dper(%s) has receive response viewNet from Booter(%s)\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
	}

	for _, node := range booterInitMsgs {
		viewNet.AddNode(*node)
	}
	conn.Close() // 完成初始化,结束Tcp连接

	loglogrus.Log.Infof("[Dper Init] Network Init, Dper has finish Network Init\n")
}

func networkInit(waitViewNet *time.Duration) bool {
	Cfg, err := ini.Load("./settings/extended.ini")
	if err != nil {
		loglogrus.Log.Errorf("Fail to parse './settings/http.ini': %v\n", err)
		return false
	} else {
		key, err := Cfg.Section("NetworkInit").GetKey("WaittingViewNet")
		if err != nil {
			loglogrus.Log.Errorf("Cfg.GetKey NetworkInit.WaittingViewNet err: %v\n", err)
			return false
		}
		WaittingViewNet, _ := key.Int()
		*waitViewNet = time.Duration(WaittingViewNet) * time.Second

		return true
	}
}

func (dp *Dper) StateSelf(reconnection bool) error {
	if dp.CentralConfigMode {
		return nil
	} else {
		if !reconnection {
			if err := dp.netManager.SendInitSelfNodeState(); err != nil {
				return err
			}
		} else { // 只有当正常连接方式错误时，才会发送断连请求消息
			if err := dp.netManager.SendReconnectState(); err != nil {
				return err
			}
		}
		return nil
	}

}

func (dp *Dper) ConstructDpnet() error {

	err := dp.netManager.ConfigLowerChannel()
	if err != nil {
		return err
	}
	//if dp.selfNode.Role == dpnet.Leader {
	err = dp.netManager.ConfigUpperChannel()
	if err != nil {
		return err
	}
	//}
	return nil
}

func (dp *Dper) Start() error {
	err := dp.netManager.StartSeparateChannels()
	if err != nil {
		return err
	}
	var upperChannel consensus.UpperChannel
	var lowerChannel consensus.LowerChannel
	lowerChannel, err = dp.netManager.BackLowerChannel()
	if err != nil {
		return err
	}
	if dp.selfNode.Role == dpnet.Leader {
		upperChannel, err = dp.netManager.BackUpperChannel()
		if err != nil {
			return err
		}
	}

	validateManager := new(validator.ValidateManager)
	exportedValidator, err := dp.netManager.ExportValidator()
	if err != nil {
		return err
	}
	validateManager.Update(exportedValidator)
	validateManager.InjectTransactionFilter(dp.transactionFilter)
	validateManager.SetLowerValidFactor(3)
	validateManager.SetUpperValidRate(0.8)

	dp.netManager.Syn.SetValidateManager(validateManager)
	go dp.netManager.Syn.Start()
	dp.netManager.Syn.OpenBroadcastSyncReq()

	commonBlockCache := new(eles.CommonBlockCache)
	commonBlockCache.Start()

	consensusPromoter := consensus.NewConsensusPromoter(dp.selfNode, dp.selfKey.PrivateKey, dp.stateManager, dp.intraStateManager, dp.netManager)
	consensusPromoter.SetLowerChannel(lowerChannel)
	consensusPromoter.SetUpperChannel(upperChannel)
	consensusPromoter.SetValidateManager(validateManager)
	consensusPromoter.SetBlockCache(commonBlockCache)
	consensusPromoter.SetContractEngine(dp.contractEngine)

	for i := 0; i < len(dp.mtwSchemes); i++ {
		switch dp.mtwSchemes[i] {
		case "CTPBFT":
			ctpbft := consensus.NewCtpbft()
			ctpbft.Install(consensusPromoter)
			ctpbft.Start()
		case "FLTPBFT":
			fltpbft := consensus.NewFLtpbft()
			fltpbft.Install(consensusPromoter)
			fltpbft.Start()
		case "ADDTPBFT":
			addtpbft := consensus.NewADDtpbft()
			addtpbft.Install(consensusPromoter)
			addtpbft.Start()
		default:
			continue
		}
	}

	if dp.selfNode.Role == dpnet.Leader {
		knownBooters := dp.netManager.BackBooters()
		if len(knownBooters) == 0 {
			loglogrus.Log.Warnf("NewDper failed: Current Node could not find any booter!\n")
			return fmt.Errorf("no booter is known")
		}
		servicePrivoderNodeID := knownBooters[0]
		orderServiceAgent := blockOrdering.NewBlockOrderAgent(servicePrivoderNodeID, dp.netManager.Syn)
		orderServiceAgent.Install(consensusPromoter)
		err = orderServiceAgent.Start()
		if err != nil {
			return err
		}
	}

	consensusPromoter.Start()

	dp.validateManager = validateManager
	dp.consensusPromoter = consensusPromoter

	return nil
}

func (dp *Dper) Stop() {
	dp.consensusPromoter.Stop()
	dp.blockDB.Close()
	dp.storageDB.Close()
	dp.netManager.Close()

}

func (dp *Dper) PublishTransactions(txs []*eles.Transaction) {
	dp.consensusPromoter.PublishTransactions(txs)
}

func (dp *Dper) SimpleInvokeTransaction(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) (common.Hash, error) {
	tx := eles.Transaction{
		Sender:    user.Address,
		Nonce:     0,
		Version:   dp.stateManager.CurrentVersion(),
		LifeTime:  0,
		Contract:  contratAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: make([]eles.CheckElement, 0),
	}
	tx.SetTxID()
	signature, err := dp.accountManager.SignHash(user, tx.TxID)
	if err != nil {
		return tx.TxID, err
	}
	tx.Signature = signature
	dp.PublishTransactions([]*eles.Transaction{&tx})
	return tx.TxID, nil
}

func (dp *Dper) SimpleInvokeTransactions(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) error {

	txs := make([]*eles.Transaction, 0)
	for i := 0; i < BenchMarkFetch; i++ {
		tx := eles.Transaction{
			Sender:    user.Address,
			Nonce:     uint64(i),
			Version:   dp.stateManager.CurrentVersion(),
			LifeTime:  0,
			Contract:  contratAddr,
			Function:  functionAddr,
			Args:      args,
			CheckList: make([]eles.CheckElement, 0),
		}
		tx.SetTxID()
		signature, err := dp.accountManager.SignHash(user, tx.TxID)
		if err != nil {
			return err
		}
		tx.Signature = signature

		txs = append(txs, &tx)
	}

	dp.PublishTransactions(txs)
	return nil
}

func (dp *Dper) SimpleInvokeTransactionLocally(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) [][]byte {
	tx := eles.Transaction{
		Sender:    user.Address,
		Nonce:     0,
		Version:   dp.stateManager.CurrentVersion(),
		LifeTime:  0,
		Contract:  contratAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: make([]eles.CheckElement, 0),
	}

	receipt, _ := dp.contractEngine.ExecuteTransactions([]eles.Transaction{tx})
	return receipt[0].Result
}

func (dp *Dper) SimplePublishTransaction(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) (transactionCheck.CheckResult, error) {
	tx := eles.Transaction{
		Sender:    user.Address,
		Nonce:     0,
		Version:   dp.stateManager.CurrentVersion(),
		LifeTime:  0,
		Contract:  contratAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: make([]eles.CheckElement, 0),
	}
	tx.SetTxID()
	signature, err := dp.accountManager.SignHash(user, tx.TxID)
	if err != nil {
		return transactionCheck.CheckResult{}, err
	}
	tx.Signature = signature

	dp.PublishTransactions([]*eles.Transaction{&tx})

	//1.直接将产生的TxID输入到TxIDChan中,由SimpleCheckTransaction打包成交易上链验证事件
	dp.txCheckChan <- tx.TxID
	//2.等待一段时间,如果收到事件验证结果则打印,若超时则显示超时信息
	checkTicker := time.NewTicker(1 * time.Second)
	finishTicker := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-finishTicker.C:
			return transactionCheck.CheckResult{}, errors.New("time out")
		case <-checkTicker.C:
			cresult := dp.txCheckPool.RetrievalResult(tx.TxID)
			if cresult != nil {
				return *cresult, nil
			}
		}
	}

}

func (dp *Dper) SimplePublishTransactions(user accounts.Account, contratAddr common.Address, functionAddr common.Address, args [][]byte) ([]*transactionCheck.CheckResult, error) {

	txs := make([]*eles.Transaction, 0)
	for i := 0; i < BenchMarkFetch; i++ {
		tx := eles.Transaction{
			Sender:    user.Address,
			Nonce:     uint64(i),
			Version:   dp.stateManager.CurrentVersion(),
			LifeTime:  0,
			Contract:  contratAddr,
			Function:  functionAddr,
			Args:      args,
			CheckList: make([]eles.CheckElement, 0),
		}
		tx.SetTxID()
		signature, err := dp.accountManager.SignHash(user, tx.TxID)
		if err != nil {
			return []*transactionCheck.CheckResult{}, err
		}
		tx.Signature = signature

		txs = append(txs, &tx)
	}
	dp.PublishTransactions(txs)

	//1.直接将产生的TxID输入到TxIDChan中,由SimpleCheckTransaction打包成交易上链验证事件
	for _, tx := range txs {
		dp.txCheckChan <- tx.TxID
	}

	//2.等待一段时间,如果收到事件验证结果则打印,若超时则显示超时信息
	checkTicker := time.NewTicker(1 * time.Second)
	finishTicker := time.NewTimer(6 * time.Second)
	txReceiptSet := make([]*transactionCheck.CheckResult, 0, len(txs))
	for {
		select {
		case <-finishTicker.C:
			if len(txReceiptSet) == 0 {
				return []*transactionCheck.CheckResult{}, errors.New("time out")
			} else {
				return txReceiptSet, nil
			}
		case <-checkTicker.C:
			for _, tx := range txs {
				cresult := dp.txCheckPool.RetrievalResult(tx.TxID)
				txReceiptSet = append(txReceiptSet, cresult)
			}
		}
	}

}

// 启动交易检测(检测交易是否上链成功--由协程捕获用户输入的交易ID)
func (dp *Dper) SimpleCheckTransaction(TxIDChan chan common.Hash, finish chan bool) error {

	var ErrorChan chan error = make(chan error, 1)

	go func() {
		for {
			err := <-ErrorChan //等待错误消息
			fmt.Println(err)
		}
	}()

	transactionCheck.TxCheckRun(dp.txCheckPool, ErrorChan, TxIDChan, finish) //启动交易查询功能
	return nil
}

func (dp *Dper) StartCheckTransaction() {
	go dp.SimpleCheckTransaction(dp.txCheckChan, dp.txCheckClose)
}

func (dp *Dper) BackAccountManager() *accounts.Manager {
	return dp.accountManager
}

func (dp *Dper) CloseCheckTransaction() {
	dp.txCheckClose <- true
}

func (dp *Dper) BackViewNetInfo() string {
	return dp.netManager.BackViewNetInfo()
}

func (dp *Dper) BackViewNet() DPNetwork {
	viewNet := dp.netManager.BackViewNet()

	dnw := DPNetwork{
		SubnetCount: len(viewNet.SubNets),
		LeaderCount: len(viewNet.Leaders),
		BooterCount: len(viewNet.Booters),
		NetGroups:   make([]NetGroup, 0),
	}

	for netID, subnet := range viewNet.SubNets {
		ng := NetGroup{
			NetID:       netID,
			MemberCount: len(subnet.Nodes),
		}
		dnw.NetGroups = append(dnw.NetGroups, ng)
	}
	return dnw
}

func (dp *Dper) BackBlockChain() *eles.BlockChain {
	return dp.blockChain
}

func (dp *Dper) BackSyn() *blockSync.SyncProtocal {
	return dp.syncProtocal
}

func (dp *Dper) BackContract() string {
	return dp.contractName
}

func (dp *Dper) BackMTW() []string {
	return dp.mtwSchemes
}

func (dp *Dper) BackConsensusPromoter() *consensus.ConsensusPromoter {
	return dp.consensusPromoter
}

func (dp *Dper) BackSubNetLeader(netID string) common.NodeID {
	return dp.netManager.GetSubNetLeader(netID)
}

func (dp *Dper) BackSelfNode() *dpnet.Node {
	return dp.selfNode
}

func dataArchiveSetup(das *DataAchive) bool {
	Cfg, err := ini.Load("./settings/extended.ini")
	if err != nil {
		loglogrus.Log.Errorf("Fail to parse './settings/http.ini': %v\n", err)
		return false
	} else {
		err = Cfg.Section("DataArchive").MapTo(das)
		if err != nil {
			loglogrus.Log.Errorf("Cfg.MapTo ServerSetting err: %v\n", err)
			return false
		}
		das.ArchiveScanInterval = das.ArchiveScanInterval * time.Second // 特殊项再赋值
		return true
	}
}

// 区块归档服务
func (dp *Dper) ArchiveHistoryBlock(blockDBPath string, oldBlockChain *eles.BlockChain, CommitWriteSet func(writeSet []eles.WriteEle) error) {

	DataAchiveSetting := &DataAchive{}
	if flag := dataArchiveSetup(DataAchiveSetting); !flag {
		return
	}
	if !DataAchiveSetting.ArchiveMode {
		return
	}
	archiveTicker := time.NewTicker(DataAchiveSetting.ArchiveScanInterval)

	// var reloadBlockCount uint64 = 50  // 为了保证切换数据库之后,交易过滤器的正常使用,必须加载50个最新区块

	for {
		select {
		case <-archiveTicker.C:
			if dp.stateManager.BackLocalBlockNum() >= uint64(DataAchiveSetting.ArchiveThreshold) {

				dp.stateManager.ResetLocalBlockNum()

				pointHash := dp.stateManager.CurrentVersion()        // 记录归档点区块Hash
				pointHeight := dp.stateManager.GetBlockChainHeight() // 记录归档点区块高度

				//pointBlock := dp.stateManager.GetBlockFromHash(pointHash)
				// archiveBlocks := dp.stateManager.GetBlocksFromHash(pointBlock.RawBlock.PrevBlock, reloadBlockCount) // 将归档点之前的全部区块进行归档

				cs := new(eles.ChainState) // 区块链状态量
				cs.CurrentHash = pointHash
				cs.GenesisHash = pointHash
				cs.Height = pointHeight
				cs.ArchiveHeight = pointHeight - 1
				cs.TxSum = dp.stateManager.BackTxSum()

				// 必须禁止提交任何区块到数据库
				dp.stateManager.BackBlockStorageMux().Lock()

				newDBPath := blockDBPath + fmt.Sprintf("%d", cs.Height)
				newBlockDB, err := database.NewLDBDatabase(newDBPath)
				if err != nil {
					loglogrus.Log.Errorf("NewDper failed: Can't Load Persistent data from Specified path:%s , err:%v\n", newDBPath, err)
					panic("NewDper failed: Can't Load Persistent data from Specified path")
				}

				// 将区块链状态量写入新的存储数据库
				key := []byte("chainState")
				value := eles.ChainStateSerialize(cs)
				newBlockDB.Put(key, value)

				// TODO: 交易过滤器和检查池或许可以使用旧的
				// var transactionFilter *eles.TransactionFilter
				// if REPEAT_CHECK_ENABLE {
				// 	transactionFilter = eles.NewTransactionFilter()
				// }
				// TxCheckPool := transactionCheck.NewTaskPool() //创建一个交易检查事件池
				// newBlockchain := eles.InitBlockChain(newBlockDB, TxCheckPool, transactionFilter)
				// newBlockchain.InstallExecuteBlockFunc(CommitWriteSet)

				dp.stateManager.BackBlockStorageMux().Unlock()
				oldBlockChain.Database.Close()
				// 更新BlockChain
				oldBlockChain.Database = newBlockDB

				ConfigBlockDBPath(newDBPath)
			}
		}
	}
}

func ConfigBlockDBPath(dbPath string) {
	result := make(map[string]interface{})
	dstPath := "." + string(os.PathSeparator) + "settings" + string(os.PathSeparator)
	jsonFile := dstPath + "dperConfig.json"
	if err := ReadJson(jsonFile, result); err != nil { // 读取json文件配置
		loglogrus.Log.Errorf("Read Dper Config file is failed,err:%v", err)
		panic(fmt.Sprintf("Read Dper Config file is failed,err:%v", err))
	}

	result["BlockDBPath"] = dbPath
	WriteJson(jsonFile, result) //重写json配置文件

}

func ReadJson(jsonFile string, result map[string]interface{}) error {
	byteValue, err := os.ReadFile(jsonFile) //读取json文件
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &result) //解析json k-v对
	if err != nil {
		return err
	}
	fmt.Printf("NewAddressMode:%v\n", result["NewAddressMode"])
	fmt.Printf("BooterKeyAddress:%v\n", result["BooterKeyAddress"])
	fmt.Printf("DperKeyAddress:%v\n", result["DperKeyAddress"])
	fmt.Printf("ListenAddress:%v\n", result["ListenAddress"])
	return nil
}

func WriteJson(jsonFile string, result interface{}) {
	temp, _ := json.MarshalIndent(result, "", "")
	if err := os.WriteFile(jsonFile, temp, 0644); err != nil {
		panic(fmt.Sprintf("Reset Dper Config file is failed,err:%v", err))
	}
}
