package configer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/dper"
	"p3Chain/dper/rpc"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/p2p/nat"
	"p3Chain/utils"
	"strconv"
	"strings"
)

const (
	defaultExternalIPSavePath    = "./settings/natKind/exIP.txt"
	defaultServerAddressSavePath = "./settings/natKind/serverAddress.txt"
)

var SupportContractEngineType = []string{
	"EMS",
	"WASM",
	"PCE",
	"DEMO_CONTRACT_1",
	"DEMO_CONTRACT_2",
	"DEMO_CONTRACT_3",
	"DEMO_CONTRACT_MIX123",
	"DEMO_CONTRACT_FL",
	"DEMO_PIPE_CONTRACT_NAME",
	"DEMO_CONTRACT_ADD",
}

var SupportMTWSchemes = []string{
	"CTPBFT",
	"FLTPBFT",
	"ADDTPBFT",
}

type PlainDperConfig struct {
	NewAddressMode         bool   `json:"NewAddressMode"`
	DperKeyAddress         string `json:"DperKeyAddress"`
	DperPrivateKey         string `json:"DperPrivateKey"`
	AccountsDir            string `json:"AccountsDir"`
	ServerName             string `json:"ServerName"`
	ListenAddress          string `json:"ListenAddress"`
	InitBooterAddress      string `json:"InitBooterAddress"`
	NATKind                string `json:"NATKind"`
	BootstrapNodesFilePath string `json:"BootstrapNodesFilePath"`
	MaxPeerNum             int    `json:"MaxPeerNum"`
	MemoryDBMode           bool   `json:"MemoryDBMode"`
	BlockDBPath            string `json:"BlockDBPath"`
	StorageDBPath          string `json:"StorageDBPath"`
	StorageV2DBPath        string `json:"StorageV2DBPath"`
	DperRole               string `json:"DperRole"`
	SubNetName             string `json:"SubNetName"`

	CentralConfigMode bool `json:"CentralConfigMode"`
	TotalDperCount    int  `json:"GroupCount"`    // Dper节点的总数量
	GroupCount        int  `json:"GroupCount"`    // 分区数量
	GroupLeader       int  `json:"GroupLeader"`   // 分区的Leader数量
	GroupFollower     int  `json:"GroupFollower"` // 分区的Follower数量

	ContractEngine        string   `json:"ContractEngine"`
	RemoteSupportPipeName string   `json:"RemoteSupportPipeName"`
	RemoteEnginePipeName  string   `json:"RemoteEnginePipeName"`
	MTWSchemes            []string `json:"MTWSchemes"`
}

type PlainBooterConfig struct {
	NewAddressMode         bool   `json:"NewAddressMode"`
	BooterKeyAddress       string `json:"BooterKeyAddress"`
	KeyStoreDir            string `json:"KeyStoreDir"`
	ServerName             string `json:"ServerName"`
	ListenAddress          string `json:"ListenAddress"`
	InitBooterAddress      string `json:"InitBooterAddress"`
	NATKind                string `json:"NATKind"`
	BootstrapNodesFilePath string `json:"BootstrapNodesFilePath"`
	MaxPeerNum             int    `json:"MaxPeerNum"`
	IsDisturbBooterURL     bool   `json:"IsDisturbBooterURL"`
	CentralConfigMode      bool   `json:"CentralConfigMode"`
	TotalDperCount         int    `json:"TotalDperCount"` // Dper节点的总数量
	GroupCount             int    `json:"GroupCount"`     // 分区数量
	GroupLeader            int    `json:"GroupLeader"`    // 分区的Leader数量
	GroupFollower          int    `json:"GroupFollower"`  // 分区的Follower数量
}
type PlainRpcServiceConfig struct {
	NetWorkMethod string `json:"NetWorkMethod"`
	RpcIPAddress  string `json:"RpcIPAddress"`
}

// 加载读取dper节点的json配置文件,返回读取内容结构体PlainDperConfig
func LoadDperCfg(cfgPath string) *PlainDperConfig {
	filePtr, err := os.Open(cfgPath)
	if err != nil {
		loglogrus.Log.Errorf("Can't open Config Json File,err:%s\n", err.Error())
		panic("The startup configuration file of Dper node is not in the specified path!")
	}
	defer filePtr.Close()
	var cfgs PlainDperConfig
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&cfgs)
	if err != nil {
		loglogrus.Log.Errorf("Can't open Decode Json File,err:%s\n", err.Error())
		panic("Unable to properly parse the startup configuration file of Dper node!")
	}
	return &cfgs
}

// func PrivateKeyRevert(str string) (ecdsa.PrivateKey, error) {
// 	//loglogrus.Log.Debugf("%s",str)
// 	var key ecdsa.PrivateKey
// 	byteStream := []byte(str)

// 	gob.Register(curve.S256())

// 	decoder := gob.NewDecoder(bytes.NewBuffer(byteStream))
// 	err := decoder.Decode(&key)
// 	if err != nil {
// 		return key, err
// 	}
// 	key.Curve = curve.S256()

// 	loglogrus.Log.Debugf("%v", key.PublicKey)

// 	return key, nil
// }

// 解析json读取内容结构体PlainDperConfig,返回dper.DperConfig配置对象
func ParseDperConfig(plainCfg *PlainDperConfig) *dper.DperConfig {
	dperCfg := new(dper.DperConfig)
	dperCfg.NewAddressMode = plainCfg.NewAddressMode
	dperCfg.DperKeyAddress = common.HexToAddress(plainCfg.DperKeyAddress)

	dperCfg.KeyStoreDir = plainCfg.AccountsDir
	dperCfg.ServerName = plainCfg.ServerName
	dperCfg.ListenAddr = plainCfg.ListenAddress
	dperCfg.InitBooterAddress = plainCfg.InitBooterAddress
	switch plainCfg.NATKind { //设置NAT穿越方式
	case "Any", "any", "ANY", "":
		dperCfg.NAT = nat.Any()
	case "Hole", "hole":
		var serverAddress []byte = make([]byte, 0)
		var err error
		if serverAddress, err = ioutil.ReadFile(defaultServerAddressSavePath); err != nil {
			loglogrus.Log.Errorf("Can't find default Server Address File in %s\n", defaultServerAddressSavePath)
			panic(fmt.Sprintf("%s is not found", defaultServerAddressSavePath))
		}
		server := strings.Split(string(serverAddress), ":")
		if len(server) != 2 {
			loglogrus.Log.Errorf("The network address format of the relay server is incorrect(should be: ip port), Please check the configuration file : %s\n", defaultServerAddressSavePath)
			panic(fmt.Sprintf("%s content format error", defaultServerAddressSavePath))
		}
		ip := []byte(server[0])
		port, _ := strconv.Atoi(server[1])
		dperCfg.NAT = nat.NatHole(ip, port)

	case "Static", "static", "reflect", "Reflect":
		dperCfg.NAT = nat.StaticReflect(defaultExternalIPSavePath)

	default:
		loglogrus.Log.Warnf("Please configure Nat Kind correctly('any'/'hole'/'static'), Select 'any' by default\n")
		dperCfg.NAT = nat.Any()
	}
	bootNodes, err := utils.ReadNodesUrl(plainCfg.BootstrapNodesFilePath) //从默认位置读取booter节点的url
	if err != nil {
		loglogrus.Log.Errorf("Unable to try to read the url of the bootNode , Please Check File:%s\n", plainCfg.BootstrapNodesFilePath)
		panic("Start failed , Unable to get the url of the bootNode")
	}
	dperCfg.BootstrapNodes = bootNodes //配置booter节点的url信息
	dperCfg.MaxPeers = plainCfg.MaxPeerNum
	dperCfg.MemoryDataBaseMode = plainCfg.MemoryDBMode
	dperCfg.BlockDataBasePath = plainCfg.BlockDBPath
	dperCfg.StorageDataBasePath = plainCfg.StorageDBPath
	dperCfg.StorageV2DataBasePath = plainCfg.StorageV2DBPath
	switch plainCfg.DperRole {
	case "Leader", "leader":
		dperCfg.Role = dpnet.Leader
	case "Follower", "follower":
		dperCfg.Role = dpnet.Follower
	case "Booter", "booter":
		loglogrus.Log.Warnf("Dper role can't be set as booter, Has been set to 'follower' by default for you!\n")
		dperCfg.Role = dpnet.Follower
	default:
		loglogrus.Log.Warnf("Unknown dper role(Dper role can only be 'leader' or 'follower'), Has been set to 'follower' by default for you!\n")
		dperCfg.Role = dpnet.Follower
	}
	dperCfg.NetID = plainCfg.SubNetName
	dperCfg.CentralConfigMode = plainCfg.CentralConfigMode
	dperCfg.TotalDperCount = plainCfg.TotalDperCount
	dperCfg.GroupCount = plainCfg.GroupCount
	dperCfg.GroupLeader = plainCfg.GroupLeader
	dperCfg.GroupFollower = plainCfg.GroupFollower
	validCEFlag := false
	for i := 0; i < len(SupportContractEngineType); i++ { //查看json配置文件中指定的合约引擎是否支持(支持则validCEFlag = true)
		if SupportContractEngineType[i] == plainCfg.ContractEngine {
			validCEFlag = true
			dperCfg.ContractEngine = plainCfg.ContractEngine
			if plainCfg.ContractEngine == "DEMO_REMOTE_CONTRACT_NAME" {
				dperCfg.RemoteEnginePipeName = plainCfg.RemoteEnginePipeName
				dperCfg.RemoteSupportPipeName = plainCfg.RemoteSupportPipeName
			}
			if plainCfg.ContractEngine == "DEMO_PIPE_CONTRACT_NAME" {
				dperCfg.RemoteEnginePipeName = plainCfg.RemoteEnginePipeName
				dperCfg.RemoteSupportPipeName = plainCfg.RemoteSupportPipeName
			}
		}
	}
	if !validCEFlag {
		loglogrus.Log.Warnf("Unsupport contract engine type:%s ----  The contract engines currently supported are: %v\n", plainCfg.ContractEngine, SupportContractEngineType)
		panic("unsupport contract engine type")
	}

	mtw := make([]string, 0)
	if len(plainCfg.MTWSchemes) == 0 {
		loglogrus.Log.Infof("No MTW scheme is filled, use CTPBFT. No support MTW schemes are: %v\n", SupportMTWSchemes)
		mtw = append(mtw, "CTPBFT")
	} else {
		for _, scheme := range plainCfg.MTWSchemes {
			inFlag := false
			for _, supmtw := range SupportMTWSchemes {
				if supmtw == scheme {
					inFlag = true
					mtw = append(mtw, supmtw)
				}
			}
			if !inFlag {
				loglogrus.Log.Warnf("Unsupport MTW scheme:%s ----  The MTW schemes currently supported are: %v\n", scheme, SupportMTWSchemes)
				panic("unsupport MTW")
			}
		}
	}
	dperCfg.MTWSchemes = mtw

	return dperCfg
}

// 读取cfgPath路径下的dper节点json配置文件,最终返回dper.DperConfig配置对象
func LoadToDperCfg(cfgPath string) *dper.DperConfig {
	plainCfg := LoadDperCfg(cfgPath)
	dperCfg := ParseDperConfig(plainCfg)
	return dperCfg
}

func LoadBooterCfg(cfgPath string) (*PlainBooterConfig, error) {
	filePtr, err := os.Open(cfgPath)
	if err != nil {
		fmt.Printf("could not open json [Err:%s]", err.Error())
		return nil, err
	}
	defer filePtr.Close()
	var cfgs PlainBooterConfig
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&cfgs)
	if err != nil {
		fmt.Printf("could not decode json [Err:%s]", err.Error())
		return nil, err
	}
	return &cfgs, nil
}

func ParseBooterConfig(plainCfg *PlainBooterConfig) (*dper.OrderServiceServerConfig, error) {
	booterCfg := new(dper.OrderServiceServerConfig)
	booterCfg.NewAddressMode = plainCfg.NewAddressMode
	booterCfg.SelfNodeAddress = common.HexToAddress(plainCfg.BooterKeyAddress)
	booterCfg.KeyStoreDir = plainCfg.KeyStoreDir
	booterCfg.ServerName = plainCfg.ServerName
	booterCfg.ListenAddr = plainCfg.ListenAddress
	booterCfg.InitBooterAddress = plainCfg.InitBooterAddress
	switch plainCfg.NATKind { //设置NAT穿越模式
	case "Any", "any", "ANY", "":
		booterCfg.NAT = nat.Any()
	case "Hole", "hole":
		var serverAddress []byte = make([]byte, 0)
		var err error
		if serverAddress, err = ioutil.ReadFile(defaultServerAddressSavePath); err != nil {
			panic(fmt.Sprintf("%s is not found", defaultServerAddressSavePath))
		}
		server := strings.Split(string(serverAddress), ":")
		ip := []byte(server[0])
		port, _ := strconv.Atoi(server[1])

		fmt.Printf("server address ip:%s , port:%d\n", ip, port)
		booterCfg.NAT = nat.NatHole(ip, port)

	case "Static", "static", "reflect", "Reflect":
		//booterCfg.NAT = nat.NatHole()
		// var externalIP []byte = make([]byte, 0)
		// var err error
		// if externalIP, err = ioutil.ReadFile(defaultExternalIPSavePath); err != nil {
		// 	panic(fmt.Sprintf("%s is not found", defaultExternalIPSavePath))
		// }
		// fmt.Printf("external ip:%s\n", externalIP)
		// booterCfg.NAT = nat.ExtIP(net.ParseIP(string(externalIP)))
		booterCfg.NAT = nat.StaticReflect(defaultExternalIPSavePath)
	default:
		return nil, fmt.Errorf("unknown nat kind")
	}
	bootNodes, err := utils.ReadNodesUrl(plainCfg.BootstrapNodesFilePath)
	if err != nil {
		return nil, err
	}
	booterCfg.IsDisturbBooterURL = plainCfg.IsDisturbBooterURL
	booterCfg.BootstrapNodes = bootNodes
	booterCfg.MaxPeers = plainCfg.MaxPeerNum
	booterCfg.CentralConfigMode = plainCfg.CentralConfigMode
	booterCfg.TotalDperCount = plainCfg.TotalDperCount
	booterCfg.GroupCount = plainCfg.GroupCount
	booterCfg.GroupLeader = plainCfg.GroupLeader
	booterCfg.GroupFollower = plainCfg.GroupFollower

	return booterCfg, nil
}

// 读取cfgPath路径下的booter节点json配置文件,最终返回dper.OrderServiceServerConfig配置对象
func LoadToBooterCfg(cfgPath string) (*dper.OrderServiceServerConfig, error) {
	plainCfg, err := LoadBooterCfg(cfgPath)
	if err != nil {
		return nil, err
	}
	booterCfg, err := ParseBooterConfig(plainCfg)
	if err != nil {
		return nil, err
	}
	return booterCfg, nil
}

func LoadRpcCfg(cfgPath string) (*rpc.RpcServiceConfig, error) {
	filePtr, err := os.Open(cfgPath)
	if err != nil {
		fmt.Printf("could not open json [Err:%s]", err.Error())
		return nil, err
	}
	defer filePtr.Close()
	var cfgs PlainRpcServiceConfig
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&cfgs)
	if err != nil {
		fmt.Printf("could not decode json [Err:%s]", err.Error())
		return nil, err
	}
	tmp := new(rpc.RpcServiceConfig)
	tmp.NetWorkmethod = cfgs.NetWorkMethod
	tmp.RpcIPAddress = cfgs.RpcIPAddress
	return tmp, nil
}
