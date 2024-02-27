package dper

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"p3Chain/common"
	"p3Chain/core/blockOrdering"
	"p3Chain/core/centre"
	"p3Chain/core/dpnet"
	"p3Chain/core/netconfig"
	"p3Chain/core/separator"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	"p3Chain/crypto/randentropy"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
	"sync"
	"time"
)

type OrderServiceServer struct {
	selfKey              keys.Key
	selfNode             *dpnet.Node
	netManager           *netconfig.NetManager
	p2pServer            *server.Server
	orderServiceProvider *blockOrdering.BlockOrderServer

	TotalDperCount int // Dper节点的总数量
	BooterAddr     string

	CentralConfigMode bool
}

type OrderServiceServerConfig struct {
	// basic information
	NewAddressMode  bool //whether to create a new account when start
	SelfNodeAddress common.Address
	KeyStoreDir     string

	// connect configuration
	ServerName        string
	ListenAddr        string
	InitBooterAddress string
	NAT               nat.Interface
	BootstrapNodes    []*discover.Node //整个P3-Chain中已知的引导节点
	MaxPeers          int

	IsDisturbBooterURL bool

	// about dpnet configuration
	CentralConfigMode bool // whether to use central protocol to config dpnet
	TotalDperCount    int  // Dper节点的总数量
	GroupCount        int  // 分区数量
	GroupLeader       int  // 分区的Leader数量
	GroupFollower     int  // 分区的Follower数量
}

func NewOrderServiceServer(cfg *OrderServiceServerConfig) (*OrderServiceServer, error) {
	glog.V(logger.Info).Infof("Initialize the dper...")
	glog.V(logger.Info).Infof("Loading key store...")
	err := os.Mkdir(cfg.KeyStoreDir, 0750)
	if err != nil {
		if os.IsExist(err) {
			glog.V(logger.Info).Infof("Key store path: %s, is existed", cfg.KeyStoreDir)
		} else {
			return nil, fmt.Errorf("Fail in loading key store, path: %s", cfg.KeyStoreDir)
		}
	}
	// keyStore := keys.NewKeyStorePlain(cfg.KeyStoreDir)
	keyStore := keys.NewKeyStorePassphrase(cfg.KeyStoreDir)
	var selfKey *keys.Key
	if cfg.NewAddressMode {
		var err error
		selfKey, err = keyStore.GenerateNewKey(randentropy.Reader, "")
		if err != nil {
			return nil, fmt.Errorf("fail in create new dper: %v", err)
		}
	} else {
		var err error
		selfKey, err = keyStore.GetKey(cfg.SelfNodeAddress, "")
		if err != nil {
			return nil, fmt.Errorf("fail in create new dper: %v", err)
		}
	}
	publicKey := &selfKey.PrivateKey.PublicKey
	glog.V(logger.Info).Infof("Complete loading key store")

	glog.V(logger.Info).Infof("Create self node...")
	selfNode := dpnet.NewNode(crypto.KeytoNodeID(publicKey), int(dpnet.Booter), dpnet.UpperNetID)
	glog.V(logger.Info).Infof("Complete creating self node")

	glog.V(logger.Info).Infof("Now start net manager...")
	protocols := make([]server.Protocol, 0)
	shh := whisper.New() //本节点的whisper客户端对象
	protocols = append(protocols, shh.Protocol())
	spr := separator.New() //本节点的separator客户端对象
	protocols = append(protocols, spr.Protocol())

	viewNet := dpnet.NewDpNet()

	var cen *centre.Centre
	if cfg.CentralConfigMode {
		// cen = centre.CentralNew(*selfNode, cfg.GroupCount, cfg.GroupLeader, cfg.GroupFollower)
		// protocols = append(protocols, cen.Protocol())
		if !cfg.IsDisturbBooterURL {
			BooterNetWorkInit(cfg.InitBooterAddress, selfNode, cfg.TotalDperCount, viewNet)
		}
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

	netManager := netconfig.NewNetManager(selfKey.PrivateKey, viewNet, srv, shh, spr, cen, nil, 0, selfNode) //让本地节点维护一个本地的P3-Chain网络

	err = netManager.ViewNetAddSelf()
	if err != nil {
		return nil, err
	}
	err = netManager.StartConfigChannel()
	if err != nil {
		return nil, err
	}
	glog.V(logger.Info).Infof("Net manager hsa started")

	glog.V(logger.Info).Infof("Now construct order service server...")
	oss := &OrderServiceServer{
		selfKey:    *selfKey,
		selfNode:   selfNode,
		netManager: netManager,
		p2pServer:  srv,

		TotalDperCount: cfg.TotalDperCount,
		BooterAddr:     cfg.ListenAddr,

		CentralConfigMode: cfg.CentralConfigMode,
	}
	glog.V(logger.Info).Infof("Complete constructing order service server")

	return oss, nil
}

func BooterNetWorkInit(selfAddr string, selfNode *dpnet.Node, totalDperCount int, viewNet *dpnet.DpNet) {
	var waitTime time.Duration // 从配置文件获取的初始化等待最大时长
	networkInit(&waitTime)

	ln, err := net.Listen("tcp", selfAddr)
	if err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Booter can't listen on %s\n", selfAddr)
		return
	} else {
		loglogrus.Log.Infof("[Dper Init] Network Init, Booter listen on %s\n", selfAddr)
	}

	booterInitMsgs := make(map[string]*dpnet.Node) // Booter节点用于记录收集到的来自各个 Dper 节点的 stateSelf 消息
	var booterInitMsgsMutex sync.RWMutex

	connSet := make([]net.Conn, 0)
	notices := make([]chan bool, totalDperCount)

	for i := 0; i < totalDperCount; i++ {
		conn, err := ln.Accept()
		if err != nil {
			loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Booter(%s) can't accept tcp SYN From (%s)\n", selfAddr, conn.RemoteAddr().String())
			continue
		} else {
			connSet = append(connSet, conn)
			loglogrus.Log.Infof("[Dper Init] Network Init, Booter(%s) has connect with Dper(%s)\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
		}
		notices[i] = make(chan bool)
		go handleConnection(conn, booterInitMsgs, &booterInitMsgsMutex, totalDperCount, viewNet, notices[i])
	}

	for i := 0; i < totalDperCount; i++ {
		<-notices[i]
	}

	loglogrus.Log.Infof("[Dper Init] Network Init, Booter(%s) has receive (%d) stateSelf Msg\n", selfAddr, len(connSet)) // Booter 已经完成对所有 Dper 节点 stateSelf 消息的接收

	booterInitMsgs[selfAddr] = selfNode
	bytes, err := json.Marshal(booterInitMsgs)
	if err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Booter can't encode JsonMsg \n")
		return
	}
	for _, conn := range connSet {
		if _, err = conn.Write(bytes); err != nil {
			loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Booter(%s) can't Send JsonMsg to (%s)\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
			continue
		} else {
			loglogrus.Log.Infof("[Dper Init] Network Init, Booter(%s) has Send ViewNet to (%s)\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
		}
	}
	loglogrus.Log.Infof("[Dper Init] Network Init, Booter(%s) has finish Network Init\n", selfAddr)

}

func handleConnection(conn net.Conn, booterInitMsgs map[string]*dpnet.Node, booterInitMsgsMutex *sync.RWMutex, totalDperCount int, viewNet *dpnet.DpNet, notice chan bool) {

	// 等待接收来自 Dper 节点的 stateSelf 消息

	stateSelfJSON := make([]byte, 300)
	if n, err := conn.Read(stateSelfJSON); err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Booter(%s) can't read Msg From Dper(%s)\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
		return
	} else {
		stateSelfJSON = stateSelfJSON[:n]
		// loglogrus.Log.Infof("[Dper Init] Network Init, Booter(%s) read bytes(%v) from Dper(%s)\n", conn.LocalAddr().String(), stateSelfJSON, conn.RemoteAddr().String())
	}
	stateSelf := &dpnet.Node{}
	if err := json.Unmarshal(stateSelfJSON, stateSelf); err != nil {
		loglogrus.Log.Warnf("[Dper Init] Network Init is failed, Booter(%s) can't decode JsonMsg From Dper(%s)\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
		return
	} else {
		loglogrus.Log.Infof("[Dper Init] Network Init, Booter(%s) has receive StateSelf Msg(%v) from Dper(%s)\n", conn.LocalAddr().String(), *stateSelf, conn.RemoteAddr().String())
	}
	booterInitMsgsMutex.Lock()
	booterInitMsgs[conn.RemoteAddr().String()] = stateSelf
	booterInitMsgsMutex.Unlock()

	viewNet.AddNode(*stateSelf) // 将 Dper 节点信息添加到 Booter 自己的 viewNet 中

	notice <- true
	loglogrus.Log.Infof("[Dper Init] Network Init, Booter(%s) has add Node(%v) into viewNet\n", conn.LocalAddr().String(), *stateSelf)

}

func (oss *OrderServiceServer) StateSelf() error {
	if oss.CentralConfigMode {
		return nil
	} else {
		err := oss.netManager.SendInitSelfNodeState()
		if err != nil {
			return err
		}
		return nil
	}
}

func (oss *OrderServiceServer) ConstructDpnet() error {
	err := oss.netManager.ConfigUpperChannel()
	if err != nil {
		return err
	}
	return nil
}

func (oss *OrderServiceServer) Start() error {
	err := oss.netManager.StartSeparateChannels()
	if err != nil {
		return err
	}
	upperChannel, err := oss.netManager.BackUpperChannel()
	if err != nil {
		return err
	}
	//orderServiceProvider := consensus.NewOrderServiceProvider(oss.selfNode, common.Hash{}, upperChannel)
	orderServiceProvider := blockOrdering.NewBlockOrderServer(oss.selfNode, common.Hash{}, 0, upperChannel, oss.netManager)
	err = orderServiceProvider.Start()
	if err != nil {
		return err
	}
	oss.orderServiceProvider = orderServiceProvider
	return nil
}

func (oss *OrderServiceServer) Stop() {
	oss.netManager.Close()
	oss.p2pServer.Stop()
	oss.orderServiceProvider.Stop()
}

func (oss *OrderServiceServer) BackSelfUrl() (string, error) {
	return oss.netManager.BackSelfUrl()
}

func (oss *OrderServiceServer) BackViewNetInfo() string {
	return oss.netManager.BackViewNetInfo()
}

func (oss *OrderServiceServer) BackNetManager() *netconfig.NetManager {
	return oss.netManager
}
