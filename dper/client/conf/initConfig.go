package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	loglogrus "p3Chain/log_logrus"

	dperfile "p3Chain/dper/client/conf/dperfile"
	jsonconfig "p3Chain/dper/client/conf/jsonConfig"
	"p3Chain/dper/client/conf/utils"

	iniObject "p3Chain/dper/client/conf/iniObject"

	"github.com/go-ini/ini"
)

// 大写，作为全局配置对象
var NetworkSetting = &iniObject.Network{}

var BooterSetting = &iniObject.Booter{}

var DperSetting = &iniObject.Dper{}

var DperRunModeSetting = &iniObject.DperRunMode{}

var ContractSetting = &iniObject.Contract{}

var LogSetting = &iniObject.Log{}

var HttpSetting = &iniObject.Http{}

var DataArchiveSetting = &iniObject.DataAchive{}

var NetworkInitSetting = &iniObject.NetworkInit{}

// 读取网络分区拓扑配置信息
func ReloadNetwork(cfg *ini.File) {

	NetworkSetting.BooterCount, _ = cfg.Section("Network").Key("BooterCount").Int()
	NetworkSetting.DperCount, _ = cfg.Section("Network").Key("DperCount").Int()
	NetworkSetting.ObserverCount, _ = cfg.Section("Network").Key("ObserverCount").Int()

	NetworkSetting.GroupCount, _ = cfg.Section("Network").Key("GroupCount").Int()
	NetworkSetting.LeaderCount, _ = cfg.Section("Network").Key("LeaderCount").Int()
	NetworkSetting.FollowerCount, _ = cfg.Section("Network").Key("FollowerCount").Int()
}

// 读取Booter节点的基础配置信息
func ReloadBooterBasic(cfg *ini.File) {
	BooterSetting.NewAddressMode, _ = cfg.Section("BooterBasic").Key("NewAddressMode").Bool()
	BooterSetting.BooterKeyAddress = cfg.Section("BooterBasic").Key("BooterKeyAddress").String()
	BooterSetting.KeyStoreDir = cfg.Section("BooterBasic").Key("KeyStoreDir").String()
	BooterSetting.ServerName = cfg.Section("BooterBasic").Key("ServerName").String()
	BooterSetting.ListenAddress = cfg.Section("BooterBasic").Key("ListenAddress").String()
	BooterSetting.InitBooterAddress = cfg.Section("BooterBasic").Key("InitBooterAddress").String()

	BooterSetting.NATKind = cfg.Section("BooterBasic").Key("NATKind").String()
	BooterSetting.BootstrapNodesFilePath = cfg.Section("BooterBasic").Key("BootstrapNodesFilePath").String()
	BooterSetting.MaxPeerNum, _ = cfg.Section("BooterBasic").Key("MaxPeerNum").Int()
	BooterSetting.CentralConfigMode, _ = cfg.Section("BooterBasic").Key("CentralConfigMode").Bool()
	BooterSetting.IsDisturbBooterURL, _ = cfg.Section("BooterBasic").Key("IsDisturbBooterURL").Bool()

	BooterSetting.TotalDperCount, _ = cfg.Section("Network").Key("DperCount").Int()
	BooterSetting.GroupCount, _ = cfg.Section("Network").Key("GroupCount").Int()
	BooterSetting.GroupLeader, _ = cfg.Section("Network").Key("LeaderCount").Int()
	BooterSetting.GroupFollower, _ = cfg.Section("Network").Key("FollowerCount").Int()
}

func ReloadDperBasic(cfg *ini.File) {
	DperSetting.NewAddressMode, _ = cfg.Section("DperBasic").Key("NewAddressMode").Bool()
	DperSetting.DperKeyAddress = cfg.Section("DperBasic").Key("DperKeyAddress").String()
	DperSetting.DperPrivateKey = cfg.Section("DperBasic").Key("DperPrivateKey").String()
	DperSetting.AccountsDir = cfg.Section("DperBasic").Key("AccountsDir").String()
	DperSetting.ServerName = cfg.Section("DperBasic").Key("ServerName").String()
	DperSetting.ListenAddress = cfg.Section("DperBasic").Key("ListenAddress").String()
	DperSetting.InitBooterAddress = cfg.Section("DperBasic").Key("InitBooterAddress").String()

	DperSetting.NATKind = cfg.Section("DperBasic").Key("NATKind").String()
	DperSetting.BootstrapNodesFilePath = cfg.Section("BooterBasic").Key("BootstrapNodesFilePath").String()
	DperSetting.MaxPeerNum, _ = cfg.Section("DperBasic").Key("MaxPeerNum").Int()
	DperSetting.MemoryDBMode, _ = cfg.Section("DperBasic").Key("MemoryDBMode").Bool()
	DperSetting.BlockDBPath = cfg.Section("DperBasic").Key("BlockDBPath").String()
	DperSetting.StorageDBPath = cfg.Section("DperBasic").Key("StorageDBPath").String()
	DperSetting.StorageV2DBPath = cfg.Section("DperBasic").Key("StorageV2DBPath").String()

	DperSetting.DperRole = cfg.Section("DperBasic").Key("DperRole").String()
	DperSetting.SubNetName = cfg.Section("DperBasic").Key("SubNetName").String()

	DperSetting.CentralConfigMode, _ = cfg.Section("DperBasic").Key("CentralConfigMode").Bool()
	DperSetting.GroupCount, _ = cfg.Section("Network").Key("GroupCount").Int()
	DperSetting.GroupLeader, _ = cfg.Section("Network").Key("LeaderCount").Int()
	DperSetting.GroupFollower, _ = cfg.Section("Network").Key("FollowerCount").Int()

	DperSetting.ContractEngine = cfg.Section("DperBasic").Key("ContractEngine").String()

	DperSetting.ViewChangeFunc, _ = cfg.Section("DperBasic").Key("ViewChangeFunc").Bool()

	DperSetting.MTWSchemes = cfg.Section("MTW").Key("MTWSchemes").Strings(" ")
}

func ReloadDperRunMode(cfg *ini.File) {
	DperRunModeSetting.RunMode = cfg.Section("DperRunMode").Key("RunMode").String()
	DperRunModeSetting.ExecFileName = cfg.Section("DperRunMode").Key("ExecFileName").String()
	DperRunModeSetting.DaemonMode, _ = cfg.Section("DperRunMode").Key("DaemonMode").Bool()
	DperRunModeSetting.GnomeTerminalMode, _ = cfg.Section("DperRunMode").Key("GnomeTerminalMode").Bool()
}

func ReloadNetworkInit(cfg *ini.File) {
	NetworkInitSetting.WaittingViewNet, _ = cfg.Section("NetworkInit").Key("WaittingViewNet").Int()
	NetworkInitSetting.SleepTime, _ = cfg.Section("NetworkInit").Key("SleepTime").Int()
	NetworkInitSetting.SleepLongTime, _ = cfg.Section("NetworkInit").Key("SleepLongTime").Int()
}

func ReloadContract(cfg *ini.File) {
	ContractSetting.ContractMode, _ = cfg.Section("Contract").Key("ContractMode").Bool()
	ContractSetting.ContractExecFileName = cfg.Section("Contract").Key("ContractExecFileName").String()

	ContractSetting.ContractEngine = cfg.Section("Contract").Key("ContractEngine").String()
	ContractSetting.RemoteSupportPipeNameWin = cfg.Section("Contract").Key("RemoteSupportPipeNameWin").String()
	ContractSetting.RemoteEnginePipeNameWin = cfg.Section("Contract").Key("RemoteEnginePipeNameWin").String()

	ContractSetting.RemoteSupportPipeNameUnix = cfg.Section("Contract").Key("RemoteSupportPipeNameUnix").String()
	ContractSetting.RemoteEnginePipeNameUnix = cfg.Section("Contract").Key("RemoteEnginePipeNameUnix").String()

}

func ReloadLog(cfg *ini.File) {
	LogSetting.LogLevel = cfg.Section("Log").Key("LogLevel").String()
}

func ReloadHttp(cfg *ini.File) {

	HttpSetting.RunMode = cfg.Section("Http").Key("RunMode").String()
	HttpSetting.IP = cfg.Section("Http").Key("IP").String()
	HttpSetting.HttpPort, _ = cfg.Section("Http").Key("HttpPort").Int()

	readTimeout, _ := cfg.Section("Http").Key("ReadTimeout").Int()
	HttpSetting.ReadTimeout = time.Duration(readTimeout) * time.Second

	writeTimeout, _ := cfg.Section("Http").Key("WriteTimeout").Int()
	HttpSetting.WriteTimeout = time.Duration(writeTimeout) * time.Second
}

func ReloadDataArchive(cfg *ini.File) {
	DataArchiveSetting.ArchiveMode, _ = cfg.Section("DataArchive").Key("ArchiveMode").Bool()
	interval, _ := cfg.Section("DataArchive").Key("ArchiveScanInterval").Int()
	DataArchiveSetting.ArchiveScanInterval = time.Duration(interval) * time.Second
	DataArchiveSetting.ArchiveThreshold, _ = cfg.Section("DataArchive").Key("ArchiveThreshold").Int()
}

func main() {
	dperRootDir := ".." + string(os.PathSeparator) + "auto"
	cfg, err := ini.Load("./config.ini")
	if err != nil {
		loglogrus.Log.Errorf("Fail to parse './config.ini': %v\n", err)
	}
	// 1.读取网络分区部署配置文件（分区网络的部署拓扑），生成对应数量的节点文件夹，并修改各节点的DperRole、SubNetName、ListenAddress
	ReloadNetwork(cfg)
	ReloadBooterBasic(cfg)
	ReloadDperBasic(cfg)
	ReloadDperRunMode(cfg)
	ReloadNetworkInit(cfg)

	// 智能合约
	ReloadContract(cfg)

	// 扩展功能
	ReloadLog(cfg)
	ReloadHttp(cfg)
	ReloadDataArchive(cfg)

	/******************************根据节点总数，生成对应数量的节点文件夹***************************************/
	dperInitDir := dperfile.NewDperInit(NetworkSetting, NetworkInitSetting, BooterSetting, DperSetting, HttpSetting, DperRunModeSetting, DataArchiveSetting, ContractSetting, LogSetting)
	// 1.创建根目录
	dperInitDir.CreateRootDir(dperRootDir)
	// 2.根据booter节点数量、共识节点数量、观察节点数量分别在dperRootDir目录下生成对应数量节点文件
	dperInitDir.CreateNodeDir(NetworkSetting.BooterCount, NetworkSetting.DperCount, NetworkSetting.ObserverCount)
	/******************************根据节点总数，生成对应数量的节点文件夹***************************************/

	// 2.为各个节点的文件夹填充相应文件
	dperInitDir.ConfigBooter()
	dperInitDir.ConfigDper()

	// 3.启动booter节点，生成booterUrl
	booterIndex := 0 // 随机选择一个booter节点的文件路径
	booter := dperInitDir.BooterDir[booterIndex]

	dir, _ := os.Getwd() // 保留切换前的工作目录

	execFile := "." + string(os.PathSeparator) + "p3Chain" + dperfile.ExecutableSuffix
	arg := "-mode=booter_init"
	var wg sync.WaitGroup

	wg.Add(1)
	go utils.RunExecFile(booter, execFile, arg, &wg)
	wg.Wait()

	// 4.分发booter节点的url
	os.Chdir(dir) // 返回到之前的工作目录

	booterUrlSrc := dperInitDir.BooterDir[booterIndex] + string(os.PathSeparator) + "booters" + string(os.PathSeparator) + "booter.txt"
	var booterUrlDest string
	// 先分发给全部的Dper节点
	for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
		booterUrlDest = dperInitDir.DperDir[i] + string(os.PathSeparator) + "booters" + string(os.PathSeparator) + "booter.txt"
		utils.CopyFile(booterUrlSrc, booterUrlDest)
	}
	// 再分发给全部的Observer节点
	for i := 0; i < dperInitDir.NetworkSetting.ObserverCount; i++ {
		booterUrlDest = dperInitDir.ObserverDir[i] + string(os.PathSeparator) + "booters" + string(os.PathSeparator) + "booter.txt"
		utils.CopyFile(booterUrlSrc, booterUrlDest)
	}
	fmt.Println("booter Url分发成功 ! ")

	// 5.重新改写booter节点的json配置文件
	for _, booterDir := range dperInitDir.BooterDir {
		settingsDirDest := booterDir + string(os.PathSeparator) + "settings"
		booterConfig := settingsDirDest + string(os.PathSeparator) + "booterConfig.json"
		jsonconfig.ResetBooterConfig(booterDir, booterConfig, BooterSetting)
	}

	// 6.启动全部节点的客户端(创建startAll和stopAll脚本)
	// 6.1 startAll的bat和shell脚本
	os.Chdir(dperRootDir) // 返回到之前的工作目录
	startScriptFile := "." + string(os.PathSeparator) + "startAll" + dperfile.AutoScriptSuffix
	stopScriptFile := "." + string(os.PathSeparator) + "stopAll" + dperfile.AutoScriptSuffix

	switch dperfile.System {
	case "windows":
		CreateStartAllWin(startScriptFile, dperInitDir)
		CreateStopAllWin(stopScriptFile, dperInitDir)
	default:
		CreateStartAllUnix(startScriptFile, dperInitDir, DperRunModeSetting.GnomeTerminalMode)
		CreateStopAllUnix(stopScriptFile, dperInitDir, DperRunModeSetting.GnomeTerminalMode)
		MacStartScript("./macstartAll.sh", dperInitDir)
		MacStopScript("./macstopAll.sh", dperInitDir)
	}

}

func CreateStartAllWin(file string, dperInitDir *dperfile.DperInit) {

	var startScriptCmd string = "@echo off\n"

	for i := 0; i < dperInitDir.NetworkSetting.BooterCount; i++ {
		cmd := "start /D \".\\dper_booter" + fmt.Sprintf("%d", i+1) + "\"  start.bat" + "\n"
		startScriptCmd += cmd
	}

	for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
		cmd := "start /D \".\\dper_dper" + fmt.Sprintf("%d", i+1) + "\"  start.bat" + "\n"
		startScriptCmd += cmd
	}
	for i := 0; i < dperInitDir.NetworkSetting.ObserverCount; i++ {
		cmd := "start /D \".\\dper_observer" + fmt.Sprintf("%d", i+1) + "\"  start.bat" + "\n"
		startScriptCmd += cmd
	}

	if dperInitDir.ContractSetting.ContractMode {
		startScriptCmd += "timeout /T 25\n"
		for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
			cmd := "start /D \".\\dper_dper" + fmt.Sprintf("%d", i+1) + "\"  run_contract.bat" + "\n"
			startScriptCmd += cmd
		}
	}

	utils.MkFile(file, startScriptCmd)
}

func CreateStopAllWin(file string, dperInitDir *dperfile.DperInit) {
	var stopScriptCmd string = "@echo off\n"

	for i := 0; i < dperInitDir.NetworkSetting.BooterCount; i++ {
		cmd := "start /D \".\\dper_booter" + fmt.Sprintf("%d", i+1) + "\" /b stop.bat" + "\n"
		stopScriptCmd += cmd
	}

	for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
		cmd := "start /D \".\\dper_dper" + fmt.Sprintf("%d", i+1) + "\" /b stop.bat" + "\n"
		stopScriptCmd += cmd
	}
	for i := 0; i < dperInitDir.NetworkSetting.ObserverCount; i++ {
		cmd := "start /D \".\\dper_observer" + fmt.Sprintf("%d", i+1) + "\" /b stop.bat" + "\n"
		stopScriptCmd += cmd
	}

	stopScriptCmd += "pause\n"

	utils.MkFile(file, stopScriptCmd)
}

// TODO:linux版本安装合约
func CreateStartAllUnix(file string, dperInitDir *dperfile.DperInit, isgnome bool) {
	var startScriptCmd string

	for i := 0; i < dperInitDir.NetworkSetting.BooterCount; i++ {
		cmd := "cd ./dper_booter" + fmt.Sprintf("%d", i+1) + "\n"

		if isgnome {
			cmd += "gnome-terminal -t '\"dper_booter" + fmt.Sprintf("%d", i+1) + "\"' -- bash -c \"./start.sh;exec bash\"\n"
		} else {
			cmd += "bash -c \"./start.sh &\"\n"
		}

		cmd += "cd ..\n"
		startScriptCmd += cmd
	}

	for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
		cmd := "cd ./dper_dper" + fmt.Sprintf("%d", i+1) + "\n"
		if isgnome {
			cmd += "gnome-terminal -t '\"dper_dper" + fmt.Sprintf("%d", i+1) + "\" ' -- bash -c \"./start.sh;exec bash\"\n"
		} else {
			cmd += "bash -c \"./start.sh &\"\n"
		}
		cmd += "cd ..\n"
		startScriptCmd += cmd
	}

	for i := 0; i < dperInitDir.NetworkSetting.ObserverCount; i++ {
		cmd := "cd ./dper_observer" + fmt.Sprintf("%d", i+1) + "\n"
		if isgnome {
			cmd += "gnome-terminal -t '\"dper_observer" + fmt.Sprintf("%d", i+1) + "\"' -- bash -c \"./start.sh;exec bash\"\n"
		} else {
			cmd += "bash -c \"./start.sh &\"\n"
		}
		cmd += "cd ..\n"
		startScriptCmd += cmd
	}

	if dperInitDir.ContractSetting.ContractMode {
		startScriptCmd += "sleep 25 \n"
		for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
			cmd := "cd ./dper_dper" + fmt.Sprintf("%d", i+1) + "\n"
			if isgnome {
				cmd += "gnome-terminal -t '\"dper_dper" + fmt.Sprintf("%d", i+1) + "\"' -- bash -c \"./run_contract.sh;exec bash\"\n"
			} else {
				cmd += "bash -c \"./run_contract.sh &\"\n"
			}
			cmd += "cd ..\n"
			startScriptCmd += cmd
		}
	}

	utils.MkFile(file, startScriptCmd)
}

func CreateStopAllUnix(file string, dperInitDir *dperfile.DperInit, isgnome bool) {
	var stopScriptCmd string
	for i := 0; i < dperInitDir.NetworkSetting.BooterCount; i++ {
		cmd := "cd ./dper_booter" + fmt.Sprintf("%d", i+1) + "\n"
		cmd += "bash -c \"./stop.sh &\"\n"
		cmd += "cd ..\n"
		stopScriptCmd += cmd
	}

	for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
		cmd := "cd ./dper_dper" + fmt.Sprintf("%d", i+1) + "\n"
		cmd += "bash -c \"./stop.sh &\"\n"
		cmd += "cd ..\n"
		stopScriptCmd += cmd
	}

	for i := 0; i < dperInitDir.NetworkSetting.ObserverCount; i++ {
		cmd := "cd ./dper_observer" + fmt.Sprintf("%d", i+1) + "\n"
		cmd += "bash -c \"./stop.sh &\"\n"
		cmd += "cd ..\n"
		stopScriptCmd += cmd
	}

	stopScriptCmd += "sleep 5\n"

	utils.MkFile(file, stopScriptCmd)
}


func MacStartScript(file string, dperInitDir *dperfile.DperInit) {
    var scriptCmd string

    // 指定根目录路径
    rootDir := "/Users/dengmingtao/Documents/GitHub/P3-Chain/dper/client/auto"

    // 添加启动 booter 节点的命令
    scriptCmd += "osascript -e 'tell application \"Terminal\" to do script \"cd " + rootDir + "/dper_booter1 && ./start.sh; exec bash\"'\n"

    // 遍历 dper 节点，启动它们
    for i := 1; i <= dperInitDir.NetworkSetting.DperCount; i++ {
        scriptCmd += fmt.Sprintf("osascript -e 'tell application \"Terminal\" to do script \"cd %s/dper_dper%d && ./start.sh; exec bash\"'\n", rootDir, i)
    }

    // 在运行合约之前等待 20 秒
    scriptCmd += "sleep 20\n"

    // 遍历 dper 节点，运行合约
    for i := 1; i <= dperInitDir.NetworkSetting.DperCount; i++ {
        scriptCmd += fmt.Sprintf("osascript -e 'tell application \"Terminal\" to do script \"cd %s/dper_dper%d && ./run_contract.sh; exec bash\"'\n", rootDir, i)
    }

    utils.MkFile(file, scriptCmd)
}
func MacStopScript(file string, dperInitDir *dperfile.DperInit) {
    var scriptCmd string

    // 指定根目录路径
    rootDir := "/Users/dengmingtao/Documents/GitHub/P3-Chain/dper/client/auto"

    // 添加停止 booter 节点的命令
    scriptCmd += "osascript -e 'tell application \"Terminal\" to do script \"cd " + rootDir + "/dper_booter1 && ./stop.sh &\"'\n"

    // 遍历 dper 节点，停止它们
    for i := 1; i <= dperInitDir.NetworkSetting.DperCount; i++ {
        scriptCmd += fmt.Sprintf("osascript -e 'tell application \"Terminal\" to do script \"cd %s/dper_dper%d && ./stop.sh &\"'\n", rootDir, i)
    }

    // 等待 5 秒
    scriptCmd += "sleep 5\n"

    utils.MkFile(file, scriptCmd)
}
