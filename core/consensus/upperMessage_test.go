package consensus

import (
	"crypto/ecdsa"
	"fmt"
	"p3Chain/common"
	"p3Chain/core/separator"
	"p3Chain/crypto"
	"p3Chain/rlp"
	"testing"
)

func TestUpdateBlockValidator(t *testing.T) {
	// 1.生成4个私钥
	prvKeySet := make([]*ecdsa.PrivateKey, 0)
	for i := 0; i < 4; i++ {
		prv, _ := crypto.GenerateKey()
		prvKeySet = append(prvKeySet, prv)
	}
	// 2.生成4个公钥
	pubKeySet := make([]*ecdsa.PublicKey, 0)
	for i := 0; i < 4; i++ {
		pub := prvKeySet[i].Public().(*ecdsa.PublicKey)
		pubKeySet = append(pubKeySet, pub)
	}
	// 3.生成4个nodeID
	nodeIDSet := make([]common.NodeID, 0)
	for i := 0; i < 4; i++ {
		nodeID := crypto.KeytoNodeID(pubKeySet[i])
		nodeIDSet = append(nodeIDSet, nodeID)
	}

	// 4.获取4个对某字符串哈希的数字签名
	signatureSet := make([][]byte, 0)
	serialized, err := rlp.EncodeToBytes("helloWorld")
	if err != nil {
		fmt.Printf("字符串序列化失败,err:%v\n", err)
		return
	}
	msgHash := crypto.Sha3Hash(serialized)
	for i := 0; i < 4; i++ {
		selfSignature, err := crypto.Sign(msgHash.Bytes(), prvKeySet[i])
		if err != nil {
			fmt.Printf("第%d的私钥签名失败,err:%v\n", i, err)
			return
		}
		signatureSet = append(signatureSet, selfSignature)
	}
	// 5.生成UpdateBlockValidator Msg
	ubv, err := CreateUpperMsg_UpdateBlockValidator(nodeIDSet[0], "testnet1", common.Hash{}, signatureSet)
	if err != nil {
		fmt.Printf("生成UpdateBlockValidator Msg消息失败,err:%v\n", err)
		return
	}
	// 6.打包到separator.Message
	payload, err := rlp.EncodeToBytes(ubv)
	if err != nil {
		fmt.Printf("打包为separator.Message消息失败,err:%v\n", err)
		return
	}
	msg := &separator.Message{
		MsgCode: 0x03,
		NetID:   "testnet1",
		From:    nodeIDSet[0],
		PayLoad: payload,
	}
	_ = msg
	// 下面是解码

}
