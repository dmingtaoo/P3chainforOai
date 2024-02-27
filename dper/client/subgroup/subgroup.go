package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Group struct {
	StartNode     int    `json:"startNode"`
	EndNode       int    `json:"endNode"`
	NetID         string `json:"netID"`
	LeaderCount   int    `json:"leaderCount"`
	FollowerCount int    `json:"followerCount"`
}

type SubGroup struct {
	GroupCount int     `json:"groupCount"`
	Groups     []Group `json:"groups"`
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
	filePath := "./subgroup.json"
	filePtr, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Can't Open mul.json, err:%v\n", err)
		return
	}
	defer filePtr.Close()
	var subgroup SubGroup
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	if err = decoder.Decode(&subgroup); err != nil {
		fmt.Printf("Can't Decode Sub-group Deploy from file(%s), err:%v\n", filePath, err)
		return
	}

	fmt.Println(subgroup)

	dir, _ := os.Getwd()

	fmt.Printf("当前工作路径:%s\n", dir)

	autoPath := ".." + string(os.PathSeparator) + "auto"

	for _, group := range subgroup.Groups { // 修改各个节点文件夹的 SubNetName/DperRole

		leaderIndex := 1
		for i := group.StartNode; i <= group.EndNode; i++ {
			dperDirPathSrc := autoPath + string(os.PathSeparator) + "dper_dper" + fmt.Sprintf("%d", i)

			jsonFile := dperDirPathSrc + string(os.PathSeparator) + "settings" + string(os.PathSeparator) + "dperConfig.json"

			if leaderIndex <= group.LeaderCount {
				rewriteDperConfig(jsonFile, group.NetID, "Leader")
				leaderIndex++
			} else {
				rewriteDperConfig(jsonFile, group.NetID, "Follower")
			}

		}
	}

}

func rewriteDperConfig(jsonFile string, netID string, role string) {
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
		fmt.Printf("Can't Decode Dper Deploy from file(%s), err:%v\n", jsonFile, err)
		return
	}

	dper.SubNetName = netID
	dper.DperRole = role

	temp, _ := json.MarshalIndent(dper, "", "")
	_ = os.WriteFile(jsonFile, temp, 0644)

}
