package main

import (
	"encoding/json"
	"fmt"
	"os"
	"p3Chain/dper/client/conf/utils"
	"strings"
)

type Machine struct {
	IP        string `json:"ip"`
	DirPath   string `json:"dirPath"`
	StartNode int    `json:"startNode"`
	EndNode   int    `json:"endNode"`
	HasBooter bool   `json:"hasBooter"`

	NetID         string `json:"netID"`
	LeaderCount   int    `json:"leaderCount"`
	FollowerCount int    `json:"followerCount"`
}

type MulMachine struct {
	MachineCount int       `json:"machineCount"`
	BooterIP     string    `json:"booterIP"`
	Machines     []Machine `json:"machines"`
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

func init() {
	SystemJudge()
}

type BooterConf struct {
	BooterKeyAddress       string
	BootstrapNodesFilePath string
	CentralConfigMode      bool
	GroupCount             int
	GroupFollower          int
	GroupLeader            int
	InitBooterAddress      string
	IsDisturbBooterURL     bool
	KeyStoreDir            string
	ListenAddress          string
	MaxPeerNum             int
	NATKind                string
	NewAddressMode         bool
	ServerName             string
	TotalDperCount         int
}

type DperConfig struct {
	AccountsDir            string
	BlockDBPath            string
	BootstrapNodesFilePath string
	CentralConfigMode      bool
	ContractEngine         string
	DperKeyAddress         string
	DperPrivateKey         string
	DperRole               string
	GroupCount             int
	GroupFollower          int
	GroupLeader            int
	InitBooterAddress      string
	ListenAddress          string
	MaxPeerNum             int
	MemoryDBMode           bool
	NATKind                string
	NewAddressMode         bool
	ServerName             string
	StorageDBPath          string
	SubNetName             string
	ViewChangeFunc         bool
}

func main() {

	filePath := "./mul.json"
	filePtr, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Can't Open mul.json, err:%v\n", err)
		return
	}
	defer filePtr.Close()
	var mulMachine MulMachine
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	if err = decoder.Decode(&mulMachine); err != nil {
		fmt.Printf("Can't Decode Multi-Machine Deploy from file(%s), err:%v\n", filePath, err)
		return
	}

	fmt.Println(mulMachine)

	dir, _ := os.Getwd()

	fmt.Printf("当前工作路径:%s\n", dir)

	autoPath := ".." + string(os.PathSeparator) + "auto"

	for _, machine := range mulMachine.Machines { // 创建多机文件夹
		machineFilePath := autoPath + string(os.PathSeparator) + machine.IP
		if err := utils.RemoveDirContents(machineFilePath); err != nil {
			fmt.Printf("删除文件(%s)失败,err:%v\n", machineFilePath, err)
		}

		utils.MkDir(machineFilePath)
	}

	for _, machine := range mulMachine.Machines { // 移动可执行文件到各个IP文件夹
		machineFilePath := autoPath + string(os.PathSeparator) + machine.IP
		p3ChainSrc := autoPath + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
		p3ChainDst := machineFilePath + string(os.PathSeparator) + "p3Chain" + ExecutableSuffix
		utils.CopyFile(p3ChainSrc, p3ChainDst)

		daemonSrc := autoPath + string(os.PathSeparator) + "daemon" + ExecutableSuffix
		daemonDst := machineFilePath + string(os.PathSeparator) + "daemon" + ExecutableSuffix
		utils.CopyFile(daemonSrc, daemonDst)

		daemonCloseSrc := autoPath + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
		daemonCloseDst := machineFilePath + string(os.PathSeparator) + "daemonClose" + ExecutableSuffix
		utils.CopyFile(daemonCloseSrc, daemonCloseDst)
	}

	for _, machine := range mulMachine.Machines { // 移动节点文件夹到对应IP文件夹（ 同时修改各个节点文件夹的IP/SubNetName/DperRole ）
		machineFilePath := autoPath + string(os.PathSeparator) + machine.IP
		if machine.HasBooter {
			booterDirPathSrc := autoPath + string(os.PathSeparator) + "dper_booter1"
			booterDirPathDst := machineFilePath + string(os.PathSeparator) + "dper_booter1"
			utils.CopyDir(booterDirPathSrc, booterDirPathDst)

			jsonFile := booterDirPathDst + string(os.PathSeparator) + "settings" + string(os.PathSeparator) + "booterConfig.json"
			rewriteBooterConfig(jsonFile, machine.IP)

		}
		leaderIndex := 1
		for i := machine.StartNode; i <= machine.EndNode; i++ {
			dperDirPathSrc := autoPath + string(os.PathSeparator) + "dper_dper" + fmt.Sprintf("%d", i)
			dperDirPathDst := machineFilePath + string(os.PathSeparator) + "dper_dper" + fmt.Sprintf("%d", i)
			utils.CopyDir(dperDirPathSrc, dperDirPathDst)

			jsonFile := dperDirPathDst + string(os.PathSeparator) + "settings" + string(os.PathSeparator) + "dperConfig.json"

			if leaderIndex <= machine.LeaderCount {
				rewriteDperConfig(jsonFile, mulMachine.BooterIP, machine.IP, machine.NetID, "Leader")
				leaderIndex++
			} else {
				rewriteDperConfig(jsonFile, mulMachine.BooterIP, machine.IP, machine.NetID, "Follower")
			}

		}
	}

	// 每个IP文件夹生成一个启动脚本,启动该IP文件夹内的所有节点
	for _, machine := range mulMachine.Machines { // 移动节点文件夹到对应IP文件夹
		machineFilePath := autoPath + string(os.PathSeparator) + machine.IP

		startScriptFile := machineFilePath + string(os.PathSeparator) + "startAll" + AutoScriptSuffix
		stopScriptFile := machineFilePath + string(os.PathSeparator) + "stopAll" + AutoScriptSuffix

		switch System {
		case "windows":
			CreateStartAllWin(machine.DirPath, startScriptFile, machine.HasBooter, machine.StartNode, machine.EndNode)
			CreateStopAllWin(machine.DirPath, stopScriptFile, machine.HasBooter, machine.StartNode, machine.EndNode)
		default:
			CreateStartAllUnix(machine.DirPath, startScriptFile, machine.HasBooter, machine.StartNode, machine.EndNode, true)
			CreateStopAllUnix(machine.DirPath, stopScriptFile, machine.HasBooter, machine.StartNode, machine.EndNode, true)
		}
	}

}

func rewriteBooterConfig(jsonFile string, ip string) {
	filePtr, err := os.Open(jsonFile)
	if err != nil {
		fmt.Printf("Can't Open %s, err:%v\n", jsonFile, err)
		return
	}
	defer filePtr.Close()
	var booter BooterConf
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	if err = decoder.Decode(&booter); err != nil {
		fmt.Printf("Can't Decode booter Deploy from file(%s), err:%v\n", jsonFile, err)
		return
	}

	initAddr := strings.Split(booter.InitBooterAddress, ":")
	booter.InitBooterAddress = ip + ":" + initAddr[1]

	listenAddr := strings.Split(booter.ListenAddress, ":")
	booter.ListenAddress = ip + ":" + listenAddr[1]

	temp, _ := json.MarshalIndent(booter, "", "")
	_ = os.WriteFile(jsonFile, temp, 0644)

}

func rewriteDperConfig(jsonFile string, booterIP string, selftIP string, netID string, role string) {
	filePtr, err := os.Open(jsonFile)
	if err != nil {
		fmt.Printf("Can't Open %s, err:%v\n", jsonFile, err)
		return
	}
	defer filePtr.Close()
	var dper DperConfig
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	if err = decoder.Decode(&dper); err != nil {
		fmt.Printf("Can't Decode booter Deploy from file(%s), err:%v\n", jsonFile, err)
		return
	}

	initAddr := strings.Split(dper.InitBooterAddress, ":")
	dper.InitBooterAddress = booterIP + ":" + initAddr[1]

	listenAddr := strings.Split(dper.ListenAddress, ":")
	dper.ListenAddress = selftIP + ":" + listenAddr[1]

	dper.SubNetName = netID
	dper.DperRole = role

	temp, _ := json.MarshalIndent(dper, "", "")
	_ = os.WriteFile(jsonFile, temp, 0644)

}

func CreateStartAllWin(dirPath string, file string, hasBooter bool, dperNumStart, dperNumEnd int) {

	var startScriptCmd string = "@echo off\n"

	if hasBooter {
		cmd := "start /D \".\\dper_booter" + fmt.Sprintf("%d", 1) + "\"  start.bat" + "\n"
		startScriptCmd += cmd
	}

	for i := dperNumStart; i <= dperNumEnd; i++ {
		cmd := "start /D \".\\dper_dper" + fmt.Sprintf("%d", i) + "\"  start.bat" + "\n"
		startScriptCmd += cmd
	}

	// TODO:合约安装
	// if dperInitDir.ContractSetting.ContractMode {
	// 	startScriptCmd += "timeout /T 25\n"
	// 	for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
	// 		cmd := "start /D \".\\dper_dper" + fmt.Sprintf("%d", i+1) + "\"  run_contract.bat" + "\n"
	// 		startScriptCmd += cmd
	// 	}
	// }

	utils.MkFile(file, startScriptCmd)
}

func CreateStopAllWin(dirPath string, file string, hasBooter bool, dperNumStart, dperNumEnd int) {
	var stopScriptCmd string = "@echo off\n"

	if hasBooter {
		cmd := "start /D \".\\dper_booter" + fmt.Sprintf("%d", 1) + "\"  start.bat" + "\n"
		stopScriptCmd += cmd
	}

	for i := dperNumStart; i <= dperNumEnd; i++ {
		cmd := "start /D \".\\dper_dper" + fmt.Sprintf("%d", i) + "\" /b stop.bat" + "\n"
		stopScriptCmd += cmd
	}

	stopScriptCmd += "pause\n"

	utils.MkFile(file, stopScriptCmd)
}

func CreateStartAllUnix(dirPath string, file string, hasBooter bool, dperNumStart, dperNumEnd int, isgnome bool) {
	var startScriptCmd string

	if hasBooter {
		cmd := "cd " + dirPath + "/dper_booter" + fmt.Sprintf("%d", 1) + "\n"
		if isgnome {
			cmd += "gnome-terminal -t '\"dper_booter" + fmt.Sprintf("%d", 1) + "\"' -- bash -c \"./start.sh;exec bash\"\n"
		} else {
			cmd += "bash -c \"./start.sh &\"\n"
		}
		cmd += "cd ..\n"
		startScriptCmd += cmd
	}

	for i := dperNumStart; i <= dperNumEnd; i++ {
		cmd := "cd " + dirPath + "/dper_dper" + fmt.Sprintf("%d", i) + "\n"
		if isgnome {
			cmd += "gnome-terminal -t '\"dper_dper" + fmt.Sprintf("%d", i) + "\" ' -- bash -c \"./start.sh;exec bash\"\n"
		} else {
			cmd += "bash -c \"./start.sh &\"\n"
		}
		cmd += "cd ..\n"
		startScriptCmd += cmd
	}

	// TODO:合约安装
	// if dperInitDir.ContractSetting.ContractMode {
	// 	startScriptCmd += "sleep 25 \n"
	// 	for i := 0; i < dperInitDir.NetworkSetting.DperCount; i++ {
	// 		cmd := "cd ./dper_dper" + fmt.Sprintf("%d", i+1) + "\n"
	// 		if isgnome {
	// 			cmd += "gnome-terminal -t '\"dper_dper" + fmt.Sprintf("%d", i+1) + "\"' -- bash -c \"./run_contract.sh;exec bash\"\n"
	// 		} else {
	// 			cmd += "bash -c \"./run_contract.sh &\"\n"
	// 		}
	// 		cmd += "cd ..\n"
	// 		startScriptCmd += cmd
	// 	}
	// }

	utils.MkFile(file, startScriptCmd)
}

func CreateStopAllUnix(dirPath string, file string, hasBooter bool, dperNumStart, dperNumEnd int, isgnome bool) {
	var stopScriptCmd string

	if hasBooter {
		cmd := "cd " + dirPath + "/dper_booter" + fmt.Sprintf("%d", 1) + "\n"
		cmd += "bash -c \"./stop.sh &\"\n"
		cmd += "cd ..\n"
		stopScriptCmd += cmd
	}

	for i := dperNumStart; i <= dperNumEnd; i++ {
		cmd := "cd " + dirPath + "/dper_dper" + fmt.Sprintf("%d", i) + "\n"
		cmd += "bash -c \"./stop.sh &\"\n"
		cmd += "cd ..\n"
		stopScriptCmd += cmd
	}

	stopScriptCmd += "sleep 5\n"

	utils.MkFile(file, stopScriptCmd)
}
