package viewChange

import (
	"encoding/json"
	"os"
	loglogrus "p3Chain/log_logrus"
)

type DperConfig struct {
	AccountsDir            string
	BlockDBPath            string
	BootstrapNodesFilePath string
	CentralConfigMode      bool
	ContractEngine         string
	DperKeyAddress         string
	DperPrivateKey         string
	DperRole               string
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

var dperfile string

func ReloadConfigPath() {
	dir, _ := os.Getwd()
	loglogrus.Log.Infof("[View Change] 当前目录为:%s\n", dir)
	dperfile = dir + string(os.PathSeparator) + "settings" + string(os.PathSeparator) + "dperConfig.json"
}

func ReloadConfigFile() *DperConfig {
	dc := new(DperConfig)
	readJson(dperfile, dc)

	return dc
}

func UpdateConfigFile(dc *DperConfig, nodeRole string) {
	dc.DperRole = nodeRole

	writeJson(dperfile, dc)
}

// 读取json配置
func readJson(jsonFile string, result *DperConfig) error {
	byteValue, err := os.ReadFile(jsonFile) //读取json文件
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 读取Json文件失败,err:%v\n", err)
		return err
	}
	err = json.Unmarshal(byteValue, result) //解析json k-v对
	if err != nil {
		loglogrus.Log.Warnf("[View Change] 解析Json文件失败,err:%v\n", err)
		return err
	}
	return nil
}

// 重新改写json配置文件
func writeJson(jsonFile string, dc *DperConfig) {
	temp, _ := json.MarshalIndent(dc, "", "")
	if err := os.WriteFile(jsonFile, temp, 0644); err != nil {
		loglogrus.Log.Warnf("[View Change] 写入Json文件失败,err:%v\n", err)
	}
}
