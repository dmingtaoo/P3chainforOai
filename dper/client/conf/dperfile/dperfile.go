package dperfile

import (
	"fmt"
	"os"
	"p3Chain/dper/client/conf/iniObject"
	iniobject "p3Chain/dper/client/conf/iniObject"
	jsonconfig "p3Chain/dper/client/conf/jsonConfig"
	"p3Chain/dper/client/conf/utils"
	"strconv"
	"strings"
	"sync"

	"github.com/go-ini/ini"
)

type DperInit struct {
	RootDir     string
	BooterDir   []string
	DperDir     []string
	ObserverDir []string

	NetworkSetting     *iniObject.Network
	NetworkInitSetting *iniObject.NetworkInit
	BooterSetting      *iniObject.Booter
	DperSetting        *iniObject.Dper

	DperRunModeSetting *iniObject.DperRunMode

	ContractSetting *iniObject.Contract

	LogSetting        *iniObject.Log
	HttpSetting       *iniObject.Http
	DataAchiveSetting *iniObject.DataAchive
}

var System string
var ExecutableSuffix string
var AutoScriptSuffix string

func SystemJudge() {
	system := utils.JudgeOperatingSystem()
	switch system {
	case "macOS":
		System = "macOS"
		ExecutableSuffix = ""
		AutoScriptSuffix = ".sh"
	case "linux":
		System = "linux"
		ExecutableSuffix = ""
		AutoScriptSuffix = ".sh"
	case "windows":
		System = "windows"
		ExecutableSuffix = ".exe"
		AutoScriptSuffix = ".bat"
	default:
		ExecutableSuffix = ""
	}
}

func NewDperInit(NetworkSetting *iniobject.Network, NetworkInit *iniObject.NetworkInit, BooterSetting *iniobject.Booter, DperSetting *iniobject.Dper, HttpSetting *iniobject.Http,
	DperRunModeSetting *iniObject.DperRunMode, DataAchiveSetting *iniObject.DataAchive, ContractSetting *iniObject.Contract, LogSetting *iniObject.Log) *DperInit {
	didp := new(DperInit)
	didp.NetworkSetting = NetworkSetting
	didp.NetworkInitSetting = NetworkInit
	didp.BooterSetting = BooterSetting
	didp.DperSetting = DperSetting
	didp.DperRunModeSetting = DperRunModeSetting

	didp.ContractSetting = ContractSetting
	didp.LogSetting = LogSetting
	didp.HttpSetting = HttpSetting
	didp.DataAchiveSetting = DataAchiveSetting
	SystemJudge()
	return didp
}

func (didp *DperInit) CreateRootDir(rootDir string) {

	if _, err := os.Stat(rootDir); err == nil {
		// 如果文件夹已存在，删除它
		os.RemoveAll(rootDir)
	}
	// 创建文件夹
	if err := os.Mkdir(rootDir, 0755); err != nil {
		panic(err)
	}
	didp.RootDir = rootDir

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取当前目录失败:", err)
		return
	}

	parentDir := utils.GetParentDirectory(currentDir)
	p3ChainSrc := parentDir + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
	p3ChainDest := rootDir + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
	utils.CopyFile(p3ChainSrc, p3ChainDest)

	daemonSrc := parentDir + string(os.PathSeparator) + "daemon" + ExecutableSuffix
	daemonDest := rootDir + string(os.PathSeparator) + "daemon" + ExecutableSuffix
	utils.CopyFile(daemonSrc, daemonDest)

	daemonCloseSrc := parentDir + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
	daemonCloseDest := rootDir + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
	utils.CopyFile(daemonCloseSrc, daemonCloseDest)

}

func (didp *DperInit) CreateNodeDir(booter int, dper int, observer int) {
	// 必须保证节点根目录存在
	if _, err := os.Stat(didp.RootDir); err != nil {
		panic(err)
	}
	for i := 0; i < booter; i++ {
		booterDir := didp.RootDir + string(os.PathSeparator) + fmt.Sprintf("dper_booter%d", i+1)
		if err := os.Mkdir(booterDir, 0755); err != nil {
			panic(err)
		}
		didp.BooterDir = append(didp.BooterDir, booterDir)
	}
	for i := 0; i < dper; i++ {
		dperDir := didp.RootDir + string(os.PathSeparator) + fmt.Sprintf("dper_dper%d", i+1)
		if err := os.Mkdir(dperDir, 0755); err != nil {
			panic(err)
		}
		didp.DperDir = append(didp.DperDir, dperDir)
	}
	for i := 0; i < observer; i++ {
		observerDir := didp.RootDir + string(os.PathSeparator) + fmt.Sprintf("dper_observer%d", i+1)
		if err := os.Mkdir(observerDir, 0755); err != nil {
			panic(err)
		}
		didp.ObserverDir = append(didp.ObserverDir, observerDir)
	}
}

// 配置所有Booter节点
func (didp *DperInit) ConfigBooter() {

	for index, booterDir := range didp.BooterDir {
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Println("获取当前目录失败:", err)
			return
		}
		// 1. 复制settings目录及其下所有文件到各个Booter节点的目录下
		settingsDirSrc := currentDir + string(os.PathSeparator) + "settings"
		settingsDirDest := booterDir + string(os.PathSeparator) + "settings"
		utils.CopyDir(settingsDirSrc, settingsDirDest)
		// 2.修改json配置文件和http信息(根据从ini文件中读取的配置信息)
		booterConfig := settingsDirDest + string(os.PathSeparator) + "booterConfig.json" // 节点文件夹配置文件所在路径
		addr := strings.Split(didp.BooterSetting.ListenAddress, ":")                     // 拆分为IP和Port两部分（dper节点之间的通信地址）

		port, _ := strconv.Atoi(addr[1])
		addr[1] = fmt.Sprintf("%d", port+index)
		didp.BooterSetting.ListenAddress = addr[0] + ":" + addr[1] // 重新修改dper通信端口

		didp.HttpSetting.HttpPort += index // 重新修改http端口

		jsonconfig.BooterConfig(booterDir, booterConfig, didp.BooterSetting, didp.HttpSetting)

		// 3.修改各节点settings目录下扩展功能的配置文件extended.ini
		didp.SetExtendedIniFile(settingsDirDest)

		// 4.在各个Dper节点文件夹下创建日志目录
		logDirDest := booterDir + string(os.PathSeparator) + "log"
		utils.MkDir(logDirDest)
		consensusLogDirDest := logDirDest + string(os.PathSeparator) + "consensusLog"
		errLogDirDest := logDirDest + string(os.PathSeparator) + "errLog"
		logLogDirDest := logDirDest + string(os.PathSeparator) + "log"
		utils.MkDir(consensusLogDirDest)
		utils.MkDir(errLogDirDest)
		utils.MkDir(logLogDirDest)

		// 5.拷贝各种可执行文件到当前节点目录下
		dir, _ := os.Getwd() // 保留切换前的工作目录
		targetDir := ".." + string(os.PathSeparator)
		execFile := "." + string(os.PathSeparator) + "build" + AutoScriptSuffix
		arg := ""
		var wg sync.WaitGroup
		wg.Add(1)
		go utils.RunExecFile(targetDir, execFile, arg, &wg)
		wg.Wait()

		os.Chdir(dir) // 返回到之前的工作目录

		parentDir := utils.GetParentDirectory(currentDir)
		p3ChainSrc := parentDir + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
		p3ChainDest := booterDir + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
		utils.CopyFile(p3ChainSrc, p3ChainDest)

		daemonSrc := parentDir + string(os.PathSeparator) + "daemon" + ExecutableSuffix
		daemonDest := booterDir + string(os.PathSeparator) + "daemon" + ExecutableSuffix
		utils.CopyFile(daemonSrc, daemonDest)

		daemonCloseSrc := parentDir + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
		daemonCloseDest := booterDir + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
		utils.CopyFile(daemonCloseSrc, daemonCloseDest)

		// 合约程序
		if didp.ContractSetting.ContractMode {
			contractExecSrc := parentDir + string(os.PathSeparator) + didp.ContractSetting.ContractExecFileName + ExecutableSuffix
			contractExecDest := booterDir + string(os.PathSeparator) + didp.ContractSetting.ContractExecFileName + ExecutableSuffix
			utils.CopyFile(contractExecSrc, contractExecDest)
		}

		// 6.booter创建prepare.txt / action.txt
		prepareFileSrc := currentDir + string(os.PathSeparator) + "prepare.txt"
		prepareFileDest := booterDir + string(os.PathSeparator) + "prepare.txt"
		utils.CopyFile(prepareFileSrc, prepareFileDest)

		actionFileSrc := currentDir + string(os.PathSeparator) + "booterAction.txt"
		actionFileDest := booterDir + string(os.PathSeparator) + "action.txt"
		utils.CopyFile(actionFileSrc, actionFileDest)

		// 7.创建booter文件夹(存放booter节点的url)
		booterDirDest := booterDir + string(os.PathSeparator) + "booters"
		utils.MkDir(booterDirDest)

		// 8. 生成start.bat(start.sh)和stop.bat(stop.sh)脚本
		var startCmd string
		if didp.DperRunModeSetting.DaemonMode {
			startCmd = "." + string(os.PathSeparator) + "daemon" + ExecutableSuffix + " -prog " + "\"" +
				didp.DperRunModeSetting.ExecFileName + ExecutableSuffix + " " + didp.DperRunModeSetting.RunMode + "\"" + " -daemon" + "\n"
		} else {
			startCmd = "." + string(os.PathSeparator) + didp.DperRunModeSetting.ExecFileName + ExecutableSuffix + " " + didp.DperRunModeSetting.RunMode + "\n"
		}

		stopCmd := "." + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix + "\n"
		var pauseCmd string
		if System == "windows" {
			pauseCmd = "pause\n"
		} else {
			pauseCmd = "sleep 5 \n"
		}

		startScriptFile := booterDir + string(os.PathSeparator) + "start" + AutoScriptSuffix
		stopScriptFile := booterDir + string(os.PathSeparator) + "stop" + AutoScriptSuffix
		utils.MkFile(startScriptFile, startCmd+pauseCmd)
		utils.MkFile(stopScriptFile, stopCmd+pauseCmd)
	}
}

// 配置所有Dper节点
func (didp *DperInit) ConfigDper() {

	startHttpPort := didp.HttpSetting.HttpPort + didp.NetworkSetting.BooterCount
	remoteSupportPipeNameWin := didp.ContractSetting.RemoteSupportPipeNameWin
	remoteEnginePipeNameWin := didp.ContractSetting.RemoteEnginePipeNameWin

	remoteSupportPipeNameUnix := didp.ContractSetting.RemoteSupportPipeNameUnix
	remoteEnginePipeNameUnix := didp.ContractSetting.RemoteEnginePipeNameUnix

	for index, dperDir := range didp.DperDir {
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Println("获取当前目录失败:", err)
			return
		}
		// 1. 复制settings目录及其下所有文件到各个Dper节点的目录下
		settingsDirSrc := currentDir + string(os.PathSeparator) + "settings"
		settingsDirDest := dperDir + string(os.PathSeparator) + "settings"
		utils.CopyDir(settingsDirSrc, settingsDirDest)

		// 2.修改json配置文件和http信息(根据从ini文件中读取的配置信息)
		dperConfig := settingsDirDest + string(os.PathSeparator) + "dperConfig.json" // 节点文件夹配置文件所在路径
		addr := strings.Split(didp.BooterSetting.ListenAddress, ":")                 // 拆分为IP和Port两部分（dper节点之间的通信地址）
		port, _ := strconv.Atoi(addr[1])
		addr[1] = fmt.Sprintf("%d", port+didp.NetworkSetting.BooterCount+index) // dper节点占用的端口都在booter之后
		didp.DperSetting.ListenAddress = addr[0] + ":" + addr[1]                // 重新修改dper通信端口

		didp.HttpSetting.HttpPort = startHttpPort + index // 重新修改http端口

		didp.ContractSetting.RemoteSupportPipeNameWin = remoteSupportPipeNameWin + fmt.Sprintf("%d", 20230+index)
		didp.ContractSetting.RemoteEnginePipeNameWin = remoteEnginePipeNameWin + fmt.Sprintf("%d", 20330+index)

		didp.ContractSetting.RemoteSupportPipeNameUnix = remoteSupportPipeNameUnix + fmt.Sprintf("%d.sock", 20230+index)
		didp.ContractSetting.RemoteEnginePipeNameUnix = remoteEnginePipeNameUnix + fmt.Sprintf("%d.sock", 20330+index)

		// 重点：划分子网，各Dper节点分配角色
		didp.GroupingStrategy(index, didp.NetworkSetting, didp.DperSetting)

		jsonconfig.DperConfig(System, settingsDirDest, dperConfig, didp.NetworkSetting, didp.DperSetting, didp.HttpSetting, didp.ContractSetting)

		// 3.修改各节点settings目录下扩展功能的配置文件extended.ini
		didp.SetExtendedIniFile(settingsDirDest)

		// 4.在各个Dper节点文件夹下创建日志目录
		logDirDest := dperDir + string(os.PathSeparator) + "log"
		utils.MkDir(logDirDest)
		consensusLogDirDest := logDirDest + string(os.PathSeparator) + "consensusLog"
		errLogDirDest := logDirDest + string(os.PathSeparator) + "errLog"
		logLogDirDest := logDirDest + string(os.PathSeparator) + "log"
		utils.MkDir(consensusLogDirDest)
		utils.MkDir(errLogDirDest)
		utils.MkDir(logLogDirDest)

		// 5.拷贝各种可执行文件到当前节点目录下
		parentDir := utils.GetParentDirectory(currentDir)
		//p3ChainSrc := parentDir + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
		// p3ChainDest := dperDir + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
		// utils.CopyFile(p3ChainSrc, p3ChainDest)

		//daemonSrc := parentDir + string(os.PathSeparator) + "daemon" + ExecutableSuffix
		// daemonDest := dperDir + string(os.PathSeparator) + "daemon" + ExecutableSuffix
		// utils.CopyFile(daemonSrc, daemonDest)

		//daemonCloseSrc := parentDir + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
		// daemonCloseDest := dperDir + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
		// utils.CopyFile(daemonCloseSrc, daemonCloseDest)

		// 合约程序
		if didp.ContractSetting.ContractMode {
			contractExecSrc := parentDir + string(os.PathSeparator) + didp.ContractSetting.ContractExecFileName + ExecutableSuffix
			contractExecDest := dperDir + string(os.PathSeparator) + didp.ContractSetting.ContractExecFileName + ExecutableSuffix
			utils.CopyFile(contractExecSrc, contractExecDest)
		}

		// 6.dper创建 action.txt
		actionFileSrc := currentDir + string(os.PathSeparator) + "dperAction.txt"
		actionFileDest := dperDir + string(os.PathSeparator) + "action.txt"
		utils.CopyFile(actionFileSrc, actionFileDest)
		if didp.DperSetting.ViewChangeFunc {
			utils.AppendContext(actionFileDest, "viewChange")
		}

		// 7.创建booter文件夹(存放booter节点的url)
		booterDirDest := dperDir + string(os.PathSeparator) + "booters"
		utils.MkDir(booterDirDest)

		// 8.生成start.bat(start.sh)和stop.bat(stop.sh)脚本
		var startCmd string
		if didp.DperRunModeSetting.DaemonMode {
			// startCmd = parentDir + string(os.PathSeparator) + "daemon" + ExecutableSuffix + " -prog " + "\"" +
			// 	didp.DperRunModeSetting.ExecFileName + ExecutableSuffix + " " + didp.DperRunModeSetting.RunMode + "\"" + " -daemon" + "\n"
			startCmd = ".." + string(os.PathSeparator) + "daemon" + ExecutableSuffix + " -prog " + "\"" +
				didp.DperRunModeSetting.ExecFileName + ExecutableSuffix + " " + didp.DperRunModeSetting.RunMode + "\"" + " -daemon" + "\n"

		} else {
			// startCmd = parentDir + string(os.PathSeparator) +
			// 	didp.DperRunModeSetting.ExecFileName + ExecutableSuffix + " " + didp.DperRunModeSetting.RunMode + "\n"
			startCmd = ".." + string(os.PathSeparator) +
				didp.DperRunModeSetting.ExecFileName + ExecutableSuffix + " " + didp.DperRunModeSetting.RunMode + "\n"
		}
		// stopCmd := parentDir + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix + "\n"
		stopCmd := ".." + string(os.PathSeparator) + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix + "\n"
		var pauseCmd string
		if System == "windows" {
			pauseCmd = "pause\n"
		} else {
			startCmd = "." + string(os.PathSeparator) + startCmd
			stopCmd = "." + string(os.PathSeparator) + stopCmd
			pauseCmd = "sleep 5 \n"
		}
		var sockRm string // unix删除/tmp目录下之前的.sock文件
		if System == "windows" {

		} else {
			sockRm = "rm -rf " + didp.ContractSetting.RemoteSupportPipeNameUnix + "\n"
			sockRm += "rm -rf " + didp.ContractSetting.RemoteEnginePipeNameUnix + "\n"
		}

		startScriptFile := dperDir + string(os.PathSeparator) + "start" + AutoScriptSuffix
		stopScriptFile := dperDir + string(os.PathSeparator) + "stop" + AutoScriptSuffix
		utils.MkFile(startScriptFile, sockRm+startCmd+pauseCmd)
		utils.MkFile(stopScriptFile, stopCmd+pauseCmd)

		// 9.生成合约的脚本 run_contract.bat(run_contract.sh)
		if didp.ContractSetting.ContractMode {
			contractScriptFile := dperDir + string(os.PathSeparator) + "run_contract" + AutoScriptSuffix
			execFile := "." + string(os.PathSeparator) + didp.ContractSetting.ContractExecFileName + ExecutableSuffix
			var arg1 string
			var arg2 string

			if System == "windows" {
				arg1 = " -local_pipe=" + didp.ContractSetting.RemoteEnginePipeNameWin
				arg2 = " -dper_pipe=" + didp.ContractSetting.RemoteSupportPipeNameWin
			} else {

				arg1 = " -local_pipe=" + didp.ContractSetting.RemoteEnginePipeNameUnix
				arg2 = " -dper_pipe=" + didp.ContractSetting.RemoteSupportPipeNameUnix
			}

			contractCmd := execFile + arg1 + arg2 + "\n"

			utils.MkFile(contractScriptFile, contractCmd+pauseCmd)
		}

	}
}

// Dper节点的分组策略
func (didp *DperInit) GroupingStrategy(index int, networkSetting *iniObject.Network, dperSetting *iniObject.Dper) {
	// 当前策略：前GroupCount个节点作为各分区的Leader，后续的节点作为各分区的Follower
	if index < networkSetting.GroupCount {
		dperSetting.DperRole = "Leader"
		dperSetting.SubNetName = fmt.Sprintf("net%d", index)
	} else {
		dperSetting.DperRole = "Follower"
		dperSetting.SubNetName = fmt.Sprintf("net%d", index%networkSetting.GroupCount)
	}

}

// 配置所有Observer节点
func (didp *DperInit) ConfigObserver() {

}

func (didp *DperInit) SetExtendedIniFile(settingsDirDest string) {
	extendedFile := settingsDirDest + string(os.PathSeparator) + "extended.ini"
	cfg, err := ini.Load(extendedFile)
	if err != nil {
		fmt.Printf("Fail to parse '%s': %v\n", extendedFile, err)
		return
	}
	// http restful配置信息
	cfg.Section("Http").Key("RunMode").SetValue(didp.HttpSetting.RunMode)
	cfg.Section("Http").Key("IP").SetValue(didp.HttpSetting.IP)
	cfg.Section("Http").Key("HttpPort").SetValue(fmt.Sprintf("%d", didp.HttpSetting.HttpPort))
	cfg.Section("Http").Key("ReadTimeout").SetValue(fmt.Sprintf("%d", int(didp.HttpSetting.ReadTimeout.Seconds())))
	cfg.Section("Http").Key("WriteTimeout").SetValue(fmt.Sprintf("%d", int(didp.HttpSetting.WriteTimeout.Seconds())))

	// 数据归档配置信息
	cfg.Section("DataArchive").Key("ArchiveMode").SetValue(fmt.Sprintf("%v", didp.DataAchiveSetting.ArchiveMode))
	cfg.Section("DataArchive").Key("ArchiveScanInterval").SetValue(fmt.Sprintf("%d", int(didp.DataAchiveSetting.ArchiveScanInterval.Seconds())))
	cfg.Section("DataArchive").Key("ArchiveThreshold").SetValue(fmt.Sprintf("%d", didp.DataAchiveSetting.ArchiveThreshold))

	// 日志相关
	cfg.Section("Log").Key("LogLevel").SetValue(didp.LogSetting.LogLevel)

	// 网络初始化相关
	cfg.Section("NetworkInit").Key("WaittingViewNet").SetValue(fmt.Sprintf("%d", didp.NetworkInitSetting.WaittingViewNet))
	cfg.Section("NetworkInit").Key("SleepTime").SetValue(fmt.Sprintf("%d", didp.NetworkInitSetting.SleepTime))
	cfg.Section("NetworkInit").Key("SleepLongTime").SetValue(fmt.Sprintf("%d", didp.NetworkInitSetting.SleepLongTime))

	cfg.SaveTo(extendedFile)
}
