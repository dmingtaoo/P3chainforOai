package jsonconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"p3Chain/dper/client/conf/iniObject"
)

// 针对Booter节点的
func BooterConfig(booterDir string, jsonFile string, booter *iniObject.Booter, http *iniObject.Http) error {

	result := make(map[string]interface{})

	//1.检查据dper_booter/booterKeyStore目录中是否存在已有的BooterKeyAddress
	filePath := booterDir + string(os.PathSeparator) + "booterKeyStore"
	_, err := os.Stat(filePath)
	//2.如果有,则读取该BooterKeyAddress(注意格式)填充到"BooterKeyAddress"项,同时修改"NewAddressMode"项为false
	if err == nil { //BooterKeyAddress存在
		fmt.Println("BooterKeyAddress存在")
		result["NewAddressMode"] = false
		result["BooterKeyAddress"] = backBooterKeyAddress(filePath)
	}
	//3.如果没有,则"NewAddressMode"项为true
	if os.IsNotExist(err) { //如果返回的错误类型使用os.isNotExist()判断为true，说明文件或者文件夹不存在
		fmt.Println("BooterKeyAddress不存在")
		result["NewAddressMode"] = true
		result["BooterKeyAddress"] = ""
	}

	result["KeyStoreDir"] = booter.KeyStoreDir
	result["ServerName"] = booter.ServerName
	result["ListenAddress"] = booter.ListenAddress
	result["InitBooterAddress"] = booter.InitBooterAddress
	result["NATKind"] = booter.NATKind
	result["BootstrapNodesFilePath"] = booter.BootstrapNodesFilePath
	result["MaxPeerNum"] = booter.MaxPeerNum
	result["IsDisturbBooterURL"] = true
	result["CentralConfigMode"] = booter.CentralConfigMode
	result["TotalDperCount"] = booter.TotalDperCount
	result["GroupCount"] = booter.GroupCount
	result["GroupLeader"] = booter.GroupLeader
	result["GroupFollower"] = booter.GroupFollower

	WriteJson(jsonFile, result)

	_, err = os.Stat(booterDir)
	if os.IsNotExist(err) { //如果返回的错误类型使用os.isNotExist()判断为true，说明文件或者文件夹不存在
		errStr := fmt.Sprintf("无法获取节点的settings配置文件目录,Dir:%s", booterDir)
		panic(errStr)
	}

	return nil
}

// Booter节点分发url之后,重新改写配置文件中的NewAddressMode和BooterKeyAddress
func ResetBooterConfig(booterDir string, jsonFile string, booter *iniObject.Booter) {
	result := make(map[string]interface{})

	//1.检查据dper_booter/booterKeyStore目录中是否存在已有的BooterKeyAddress
	filePath := booterDir + string(os.PathSeparator) + "booterKeyStore"
	_, err := os.Stat(filePath)
	//2.如果有,则读取该BooterKeyAddress(注意格式)填充到"BooterKeyAddress"项,同时修改"NewAddressMode"项为false
	if err == nil { //BooterKeyAddress存在
		fmt.Println("BooterKeyAddress存在")
		result["NewAddressMode"] = false
		result["BooterKeyAddress"] = backBooterKeyAddress(filePath)
	}
	//3.如果没有,则"NewAddressMode"项为true
	if os.IsNotExist(err) { //如果返回的错误类型使用os.isNotExist()判断为true，说明文件或者文件夹不存在
		fmt.Println("BooterKeyAddress不存在")
		result["NewAddressMode"] = true
		result["BooterKeyAddress"] = ""
	}

	result["KeyStoreDir"] = booter.KeyStoreDir
	result["ServerName"] = booter.ServerName
	result["ListenAddress"] = booter.ListenAddress
	result["InitBooterAddress"] = booter.InitBooterAddress
	result["NATKind"] = booter.NATKind
	result["BootstrapNodesFilePath"] = booter.BootstrapNodesFilePath
	result["MaxPeerNum"] = booter.MaxPeerNum
	result["IsDisturbBooterURL"] = false
	result["CentralConfigMode"] = booter.CentralConfigMode
	result["TotalDperCount"] = booter.TotalDperCount
	result["GroupCount"] = booter.GroupCount
	result["GroupLeader"] = booter.GroupLeader
	result["GroupFollower"] = booter.GroupFollower

	WriteJson(jsonFile, result)
}

// 针对Dper节点
func DperConfig(system string, settingDir string, jsonFile string, network *iniObject.Network, dper *iniObject.Dper, http *iniObject.Http, contract *iniObject.Contract) {
	result := make(map[string]interface{})

	result["NewAddressMode"] = dper.NewAddressMode
	result["DperKeyAddress"] = dper.DperKeyAddress
	result["DperPrivateKey"] = dper.DperPrivateKey

	result["AccountsDir"] = dper.AccountsDir
	result["ServerName"] = dper.ServerName
	result["ListenAddress"] = dper.ListenAddress
	result["InitBooterAddress"] = dper.InitBooterAddress
	result["NATKind"] = dper.NATKind
	result["BootstrapNodesFilePath"] = dper.BootstrapNodesFilePath
	result["MaxPeerNum"] = dper.MaxPeerNum
	result["MemoryDBMode"] = dper.MemoryDBMode
	result["BlockDBPath"] = dper.BlockDBPath
	result["StorageDBPath"] = dper.StorageDBPath
	result["StorageV2DBPath"] = dper.StorageV2DBPath
	result["DperRole"] = dper.DperRole
	result["SubNetName"] = dper.SubNetName
	result["CentralConfigMode"] = dper.CentralConfigMode
	result["GroupCount"] = dper.GroupCount
	result["GroupLeader"] = dper.GroupLeader
	result["GroupFollower"] = dper.GroupFollower
	result["ContractEngine"] = dper.ContractEngine
	result["MTWSchemes"] = dper.MTWSchemes

	if contract.ContractMode {
		result["ContractEngine"] = contract.ContractEngine
		if system == "windows" {
			result["RemoteSupportPipeName"] = contract.RemoteSupportPipeNameWin
			result["RemoteEnginePipeName"] = contract.RemoteEnginePipeNameWin
		} else {
			result["RemoteSupportPipeName"] = contract.RemoteSupportPipeNameUnix
			result["RemoteEnginePipeName"] = contract.RemoteEnginePipeNameUnix
		}

	}

	result["ViewChangeFunc"] = dper.ViewChangeFunc

	WriteJson(jsonFile, result)

	_, err := os.Stat(settingDir)
	if os.IsNotExist(err) { //如果返回的错误类型使用os.isNotExist()判断为true，说明文件或者文件夹不存在
		errStr := fmt.Sprintf("无法获取节点的settings配置文件目录,Dir:%s", settingDir)
		panic(errStr)
	}
}

// TODO:针对Observer
func ObserverConfig() {

}

// 读取json配置
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

// 重新改写json配置文件
func WriteJson(jsonFile string, result interface{}) {
	temp, _ := json.MarshalIndent(result, "", "")
	_ = os.WriteFile(jsonFile, temp, 0644)
}

func backBooterKeyAddress(path string) string {
	filepath, err := os.ReadDir(path) //读booterkey文件名
	if err != nil {
		panic(err)
	}
	fileName := path + string(os.PathSeparator) + filepath[0].Name() //取首个booterkey
	result := make(map[string]interface{})
	BooterKeyAdress, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(BooterKeyAdress, &result) //解析json k-v对
	if err != nil {
		panic(err)
	}
	str := fmt.Sprintf("%v", result["address"])
	fmt.Printf("获取到的booterKeyAddress:%s\n", str)
	return str //返回booterKeyAddress
}
