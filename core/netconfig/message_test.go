package netconfig

import (
	"bytes"
	"crypto/ecdsa"
	"p3Chain/core/dpnet"
	"p3Chain/crypto"
	"p3Chain/rlp"
	"reflect"
	"testing"
)

var testKeys = []struct {
	prvKey *ecdsa.PrivateKey
}{
	{
		prvKey: generateKey(),
	},
	{
		prvKey: generateKey(),
	},
	{
		prvKey: generateKey(),
	},
}

func generateKey() *ecdsa.PrivateKey {
	key, _ := crypto.GenerateKey()
	return key
}

var initMsgConfigerGroup = []struct {
	msgCfg    InitMsgConfiger
	serilized []byte
}{
	{
		InitMsgConfiger{
			MsgCode:   selfNodeState,
			LifeTime:  50,
			Sender:    crypto.KeytoNodeID(&testKeys[0].prvKey.PublicKey),
			NetID:     "Test",
			NodeID:    crypto.KeytoNodeID(&testKeys[0].prvKey.PublicKey),
			Role:      dpnet.Follower,
			Signature: []byte{},
		},
		nil,
	},
	{
		InitMsgConfiger{
			MsgCode:   selfNodeState,
			LifeTime:  50,
			Sender:    crypto.KeytoNodeID(&testKeys[1].prvKey.PublicKey),
			NetID:     "Test",
			NodeID:    crypto.KeytoNodeID(&testKeys[1].prvKey.PublicKey),
			Role:      dpnet.Leader,
			Signature: []byte{},
		},
		nil,
	},
	{
		InitMsgConfiger{
			MsgCode:   selfNodeState,
			LifeTime:  50,
			Sender:    crypto.KeytoNodeID(&testKeys[2].prvKey.PublicKey),
			NetID:     "Test",
			NodeID:    crypto.KeytoNodeID(&testKeys[2].prvKey.PublicKey),
			Role:      dpnet.Booter,
			Signature: []byte{},
		},
		nil,
	},
}

var updateMsgConfigerGroup = []struct {
	msgCfg    UpdateMsgConfiger
	serilized []byte
}{
	{
		UpdateMsgConfiger{
			MsgCode:   DpNetInfo,
			LifeTime:  50,
			Sender:    crypto.KeytoNodeID(&testKeys[0].prvKey.PublicKey),
			NetID:     "Test",
			NodeID:    crypto.KeytoNodeID(&testKeys[0].prvKey.PublicKey),
			Role:      dpnet.Follower,
			Signature: []byte{},
			NodeList:  NodeListMsg{NodeList: make([]dpnet.Node, 0)},
		},
		nil,
	},
	{
		UpdateMsgConfiger{
			MsgCode:   DpNetInfo,
			LifeTime:  50,
			Sender:    crypto.KeytoNodeID(&testKeys[1].prvKey.PublicKey),
			NetID:     "Test",
			NodeID:    crypto.KeytoNodeID(&testKeys[1].prvKey.PublicKey),
			Role:      dpnet.Leader,
			Signature: []byte{},
			NodeList:  NodeListMsg{NodeList: make([]dpnet.Node, 0)},
		},
		nil,
	},
	{
		UpdateMsgConfiger{
			MsgCode:   DpNetInfo,
			LifeTime:  50,
			Sender:    crypto.KeytoNodeID(&testKeys[2].prvKey.PublicKey),
			NetID:     "Test",
			NodeID:    crypto.KeytoNodeID(&testKeys[2].prvKey.PublicKey),
			Role:      dpnet.Booter,
			Signature: []byte{},
			NodeList:  NodeListMsg{NodeList: make([]dpnet.Node, 0)},
		},
		nil,
	},
}

func TestSerialize(t *testing.T) {
	for _, ele := range initMsgConfigerGroup {
		serialized, err := rlp.EncodeToBytes(ele.msgCfg)
		if err != nil {
			t.Fatalf("fail in rlp encode:%v", err)
		}
		t.Logf("%q", serialized)
	}
}

func TestHash(t *testing.T) {
	for _, ele := range initMsgConfigerGroup {
		withoutSignature := ele.msgCfg.Hash()
		ele.msgCfg.Signature = []byte("test")
		withSignature := ele.msgCfg.Hash()
		if len(ele.msgCfg.Signature) != 0 {
		} else {
			t.Fatalf("Signature has been set as nil when using Hash func.")
		}
		if !bytes.Equal(withSignature, withoutSignature) {
			t.Fatalf("Hash changed when adding signature")
		}
	}
}

func TestInitMsgConfiger(t *testing.T) {
	for _, ele := range initMsgConfigerGroup {
		serialized := ele.msgCfg.BackPayload()
		deserialized, err := DeSerializeConfigMsg(serialized)
		if err != nil {
			t.Fatalf("fail in DeSerializeConfigMsg:%v", err)
		}
		if deserialized.MsgCode != ele.msgCfg.MsgCode {
			t.Fatalf("fail in DeSerializeConfigMsg: msgCode not fits")
		}
		deserializedImc, err := DeSerializeInitMsgConfiger(deserialized.Payload)
		if err != nil {
			t.Fatalf("fail in DeSerializeInitMsgConfiger:%v", err)
		}
		if !reflect.DeepEqual(*deserializedImc, ele.msgCfg) {
			t.Fatalf("Deserialized fails: value not fits")
		}
	}
}

func TestUpdateMsgConfiger(t *testing.T) {
	for _, ele := range updateMsgConfigerGroup {
		serialized := ele.msgCfg.BackPayload()
		deserialized, err := DeSerializeConfigMsg(serialized)
		if err != nil {
			t.Fatalf("fail in DeSerializeConfigMsg:%v", err)
		}
		if deserialized.MsgCode != ele.msgCfg.MsgCode {
			t.Fatalf("fail in DeSerializeConfigMsg: msgCode not fits")
		}
		deserializedImc, err := DeSerializeUpdateMsgConfiger(deserialized.Payload)
		if err != nil {
			t.Fatalf("fail in DeSerializeUpdateMsgConfiger:%v", err)
		}
		if !reflect.DeepEqual(*deserializedImc, ele.msgCfg) {
			t.Fatalf("Deserialized fails: value not fits")
		}
	}
}
