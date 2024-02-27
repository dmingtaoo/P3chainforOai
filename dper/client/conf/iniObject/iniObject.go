package iniObject

import "time"

// 分区拓扑配置信息
type Network struct {
	BooterCount   int
	DperCount     int
	ObserverCount int
	GroupCount    int
	LeaderCount   int
	FollowerCount int
}

// booter节点的基础配置信息
type Booter struct {
	NewAddressMode         bool
	BooterKeyAddress       string
	KeyStoreDir            string
	ServerName             string
	ListenAddress          string
	InitBooterAddress      string
	NATKind                string
	BootstrapNodesFilePath string
	MaxPeerNum             int

	IsDisturbBooterURL bool
	CentralConfigMode  bool
	TotalDperCount     int
	GroupCount         int
	GroupLeader        int
	GroupFollower      int
}

// dper节点的基础配置信息
type Dper struct {
	NewAddressMode         bool
	DperKeyAddress         string
	DperPrivateKey         string
	AccountsDir            string
	ServerName             string
	ListenAddress          string
	InitBooterAddress      string
	NATKind                string
	BootstrapNodesFilePath string
	MaxPeerNum             int
	MemoryDBMode           bool
	BlockDBPath            string
	StorageDBPath          string
	StorageV2DBPath        string
	DperRole               string
	SubNetName             string
	CentralConfigMode      bool
	GroupCount             int
	GroupLeader            int
	GroupFollower          int

	ContractEngine string

	MTWSchemes []string

	ViewChangeFunc bool
}

// 节点的启动模式
type DperRunMode struct {
	ExecFileName      string // dper可执行文件名称
	RunMode           string // dper客户端启动模式
	DaemonMode        bool   // dper客户端是否以后台模式运行
	GnomeTerminalMode bool   // 当前运行服务器是否有可视化界面

	LogLevel string
}

// 网络初始化相关
type NetworkInit struct {
	WaittingViewNet int
	SleepTime       int
	SleepLongTime   int
}

// 合约相关配置信息
type Contract struct {
	ContractMode         bool
	ContractExecFileName string

	ContractEngine           string
	RemoteSupportPipeNameWin string
	RemoteEnginePipeNameWin  string

	RemoteSupportPipeNameUnix string
	RemoteEnginePipeNameUnix  string
}

type Log struct {
	LogLevel string
}

// dper节点restful接口配置信息
type Http struct {
	RunMode      string
	IP           string
	HttpPort     int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// 数据归档相关功能
type DataAchive struct {
	ArchiveMode         bool
	ArchiveScanInterval time.Duration
	ArchiveThreshold    int
}

// 使用的加密算法
type Crypto struct {
}
