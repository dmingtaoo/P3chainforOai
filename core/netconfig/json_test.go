package netconfig

import (
	"fmt"
	"p3Chain/core/centre"
	"testing"
)

func TestJson(t *testing.T) {
	jsonTest := centre.CentralObject

	peerCount := jsonTest.PeerCount
	netCount := jsonTest.NetCount

	fmt.Println("中心节点部署子网个数: ", netCount)
	fmt.Println()

	for i := 0; i < peerCount; i++ {
		fmt.Println("当前节点NodeID:", jsonTest.Peers[i].NodeID)
		fmt.Println("当前节点Role:", jsonTest.Peers[i].Role)
		fmt.Println("当前节点子网ID:", jsonTest.Peers[i].NetID)
		fmt.Println()
	}

}
