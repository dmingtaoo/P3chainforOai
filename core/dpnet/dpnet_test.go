package dpnet

import (
	"encoding/json"
	"fmt"
	"p3Chain/crypto"
	"testing"
)

func TestJsonString(t *testing.T) {

	prvKey1, err := crypto.GenerateKey()
	if err != nil {
		return
	}
	NodeID1 := crypto.KeytoNodeID(&prvKey1.PublicKey)
	node1 := NewNode(NodeID1, int(Follower), "net1")

	prvKey2, err := crypto.GenerateKey()
	if err != nil {
		return
	}
	NodeID2 := crypto.KeytoNodeID(&prvKey2.PublicKey)
	node2 := NewNode(NodeID2, int(Follower), "net1")

	testMap := make(map[string]Node)
	testMap["192.168.1.1"] = *node1
	testMap["192.168.1.2"] = *node2

	if res, err := json.Marshal(testMap); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(res))
	}
}

func TestDperJson(t *testing.T) {
	prvKey1, err := crypto.GenerateKey()
	if err != nil {
		return
	}
	NodeID1 := crypto.KeytoNodeID(&prvKey1.PublicKey)
	node1 := NewNode(NodeID1, int(Follower), "net1")

	bytes, err := json.Marshal(node1)
	if err != nil {
		fmt.Printf("json 编码失败,err:%v\n", err)
	}

	newNode := &Node{}
	if err := json.Unmarshal(bytes, newNode); err != nil {
		fmt.Printf("json 解码失败,err:%v\n", err)
	} else {
		fmt.Println(*newNode)
	}

}
