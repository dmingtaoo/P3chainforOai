package netconfig

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/whisper"
	"p3Chain/rlp"
	"time"
)

type ConfigMsg struct {
	MsgCode uint64 //消息码
	Payload []byte //载荷实体
}

// 把ConfigMsg消息进行rlp序列化,变为rlp字节流
func (cm *ConfigMsg) Serialize() ([]byte, error) {
	bfr := bytes.NewBuffer(nil)
	if err := rlp.Encode(bfr, &cm); err != nil {
		return nil, err
	}
	return bfr.Bytes(), nil
}

// 将形参传入的rlp字节流消息进行反序列化，得到ConfigMsg结构的消息
func DeSerializeConfigMsg(serialized []byte) (*ConfigMsg, error) {
	cm := &ConfigMsg{}
	rdr := bytes.NewReader(serialized)
	if err := rlp.Decode(rdr, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

type MsgConfiger interface {
	BackFrom() *ecdsa.PrivateKey //获取发送节点发送消息时使用的私钥
	BackTo() *ecdsa.PublicKey    //获取发送节点发送消息时指定的接收方的公钥
	BackLifeTime() time.Duration //获取消息的生存时间
	BackTopic() whisper.Topic    //whisper协议中Message消息的Topic表
	BackPayload() []byte         //获取消息的载荷实体
}

// implements MsgConfiger
type InitMsgConfiger struct {
	MsgCode   uint64         //消息码
	LifeTime  uint64         //消息的生存时间
	Sender    common.NodeID  //消息的发送节点ID
	NetID     string         //消息的发送节点所处的网络
	NodeID    common.NodeID  //消息的发送节点ID
	Role      dpnet.RoleType //消息的发送节点在P3-Chain网络中的身份
	Signature []byte         //消息的数字签名
}

// 将形参指定的payload载荷字节流进行rlp反序列化，存入InitMsgConfiger对象中(初始化一个InitMsgConfiger对象)
func DeSerializeInitMsgConfiger(payload []byte) (*InitMsgConfiger, error) {
	imc := &InitMsgConfiger{}
	err := rlp.DecodeBytes(payload, imc)
	if err != nil {
		return nil, err
	}
	return imc, nil
}

// 将InitMsgConfiger对象消息进行rlp序列化并用SHA-3进行加密(求哈希)
func (imc *InitMsgConfiger) Hash() []byte {
	imc_copy := *imc
	imc_copy.Signature = nil
	plainImc, err := rlp.EncodeToBytes(imc_copy)
	if err != nil {
		glog.V(logger.Error).Infof("fail in encoding InitMsgConfiger: %v", err)
		return nil
	}
	return crypto.Sha3(plainImc)
}

func (imc *InitMsgConfiger) BackLifeTime() time.Duration {
	return time.Duration(imc.LifeTime) * time.Second
}

func (imc *InitMsgConfiger) BackFrom() *ecdsa.PrivateKey {
	return nil
}

func (imc *InitMsgConfiger) BackTo() *ecdsa.PublicKey {
	return nil
}

func (imc *InitMsgConfiger) BackTopic() whisper.Topic {
	return whisper.NewTopicFromString(stringConfigInit)
}

// 将InitMsgConfiger消息进行序列化，作为ConfigMsg消息的载荷部分，然后序列化得到ConfigMsg消息
func (imc *InitMsgConfiger) BackPayload() []byte {
	cm := ConfigMsg{}
	switch imc.MsgCode {
	case selfNodeState:
		cm.MsgCode = selfNodeState
		encodedMsg, err := rlp.EncodeToBytes(imc) //对InitMsgConfiger消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的InitMsgConfiger消息
	default:
		err := fmt.Errorf("no such msgCode %v in topic %v", imc.MsgCode, stringConfigInit)
		glog.V(logger.Debug).Infof("fail in encoding nodestate: %v", err)
		return nil
	}
	encodedConfigMsg, err := rlp.EncodeToBytes(cm) //将ConfigMsg消息再进行rlp序列化
	if err != nil {
		glog.V(logger.Error).Infof("fail in BackPayload: %v", err)
		return nil
	}
	return encodedConfigMsg //返回序列化之后的ConfigMsg消息
}

type NodeListMsg struct {
	NodeList   []dpnet.Node
	BooterList []dpnet.Node
}

type LeaderChangeMsg struct {
	ViewChangeHash common.Hash
	Signatures     [][]byte
}

type UpdateMsgConfiger struct {
	MsgCode   uint64         //消息码
	LifeTime  uint64         //消息的生存时间
	Sender    common.NodeID  //消息的发送节点ID
	NetID     string         //消息的发送节点所处的网络
	NodeID    common.NodeID  //消息的发送节点ID
	Role      dpnet.RoleType //消息的发送节点在P3-Chain网络中的身份
	Signature []byte         //消息的数字签名

	NodeList NodeListMsg // 所有的节点信息

	LeaderChange LeaderChangeMsg //专门用于viewChange更换Leader节点
}

// 将形参指定的payload载荷字节流进行rlp反序列化，存入UpdateMsgConfiger对象中
func DeSerializeUpdateMsgConfiger(payload []byte) (*UpdateMsgConfiger, error) {
	umc := &UpdateMsgConfiger{}
	err := rlp.DecodeBytes(payload, umc)
	if err != nil {
		return nil, err
	}
	return umc, nil
}

// 将UpdateMsgConfiger对象消息进行rlp序列化并用SHA-3进行加密(求哈希)
func (umc *UpdateMsgConfiger) Hash() []byte {
	umc_copy := *umc
	umc_copy.Signature = nil
	plainUmc, err := rlp.EncodeToBytes(umc_copy)
	if err != nil {
		loglogrus.Log.Infof("fail in encoding InitMsgConfiger: %v\n", err)
		return nil
	}
	return crypto.Sha3(plainUmc)
}

func (umc *UpdateMsgConfiger) BackFrom() *ecdsa.PrivateKey {
	return nil
}

func (umc *UpdateMsgConfiger) BackTo() *ecdsa.PublicKey {
	return nil
}

func (umc *UpdateMsgConfiger) BackLifeTime() time.Duration {
	return time.Duration(umc.LifeTime) * time.Second
}

func (umc *UpdateMsgConfiger) BackTopic() whisper.Topic {
	return whisper.NewTopicFromString(stringConfigUpdate)
}

func (umc *UpdateMsgConfiger) BackPayload() []byte {
	cm := ConfigMsg{}
	switch umc.MsgCode {
	case DpNetInfo:
		cm.MsgCode = DpNetInfo
		encodedMsg, err := rlp.EncodeToBytes(umc) //对UpdateMsgConfiger消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的UpdateMsgConfiger消息

	case ReconnectState:
		cm.MsgCode = ReconnectState
		encodedMsg, err := rlp.EncodeToBytes(umc) //对UpdateMsgConfiger消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的UpdateMsgConfiger消息

	case UpdateNodeState:
		cm.MsgCode = UpdateNodeState
		encodedMsg, err := rlp.EncodeToBytes(umc) //对UpdateMsgConfiger消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的UpdateMsgConfiger消息
	case AddNewNode:
		cm.MsgCode = AddNewNode
		encodedMsg, err := rlp.EncodeToBytes(umc) //对UpdateMsgConfiger消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的UpdateMsgConfiger消息
	case DelNode:
		cm.MsgCode = DelNode
		encodedMsg, err := rlp.EncodeToBytes(umc) //对UpdateMsgConfiger消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的UpdateMsgConfiger消息
	case LeaderChange:
		cm.MsgCode = LeaderChange
		encodedMsg, err := rlp.EncodeToBytes(umc) //对UpdateMsgConfiger消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的UpdateMsgConfiger消息
	default:
		err := fmt.Errorf("no such msgCode %v in topic %v", umc.MsgCode, stringConfigInit)
		glog.V(logger.Debug).Infof("fail in encoding nodestate: %v", err)
		return nil
	}
	encodedConfigMsg, err := rlp.EncodeToBytes(cm) //将ConfigMsg消息再进行rlp序列化
	if err != nil {
		glog.V(logger.Error).Infof("fail in BackPayload: %v", err)
		return nil
	}
	return encodedConfigMsg //返回序列化之后的ConfigMsg消息
}

type CentralConfigMsg struct {
	MsgCode   uint64        //消息码
	LifeTime  uint64        //消息的生存时间
	Sender    common.NodeID //消息包中包含发送者的nodeID
	DpNet     dpnet.DpNet   //整个DpNet网络的部署情况(由中心节点发送给其他节点)
	NodeID    common.NodeID //消息的发送节点ID
	Signature []byte        //消息的数字签名
}

// 将形参指定的payload载荷字节流进行rlp反序列化，存入CentralConfigMsg对象中(初始化一个CentralConfigMsg对象)
func DeSerializeCentralConfigMsg(payload []byte) (*CentralConfigMsg, error) {
	ccf := &CentralConfigMsg{}
	err := rlp.DecodeBytes(payload, ccf)
	if err != nil {
		return nil, err
	}
	return ccf, nil
}

// 将CentralConfigMsg对象消息进行rlp序列化并用SHA-3进行加密(求哈希)
func (ccf *CentralConfigMsg) Hash() []byte {
	ccf_copy := *ccf
	ccf_copy.Signature = nil
	plainCcf, err := rlp.EncodeToBytes(ccf_copy)
	if err != nil {
		glog.V(logger.Error).Infof("fail in encoding CentralConfigMsg: %v", err)
		return nil
	}
	return crypto.Sha3(plainCcf)
}

func (ccf *CentralConfigMsg) BackLifeTime() time.Duration {
	return time.Duration(ccf.LifeTime) * time.Second
}

func (ccf *CentralConfigMsg) BackFrom() *ecdsa.PrivateKey {
	return nil
}

func (ccf *CentralConfigMsg) BackTo() *ecdsa.PublicKey {
	return nil
}

func (ccf *CentralConfigMsg) BackTopic() whisper.Topic {
	return whisper.NewTopicFromString(stringConfigInit)
}

// 将CentralConfigMsg消息进行序列化，作为ConfigMsg消息的载荷部分，然后序列化得到ConfigMsg消息
func (ccf *CentralConfigMsg) BackPayload() []byte {
	cm := ConfigMsg{}
	switch ccf.MsgCode {
	case centralConfig:
		cm.MsgCode = centralConfig
		encodedMsg, err := rlp.EncodeToBytes(ccf) //对CentralConfigMsg消息进行rlp序列化,变为字节流
		if err != nil {
			glog.V(logger.Error).Infof("fail in encoding nodestate: %v", err)
			return nil
		}
		cm.Payload = encodedMsg //让ConfigMsg消息的载荷部分获取该序列化后的CentralConfigMsg消息
	default:
		err := fmt.Errorf("no such msgCode %v in topic %v", ccf.MsgCode, stringConfigInit)
		glog.V(logger.Debug).Infof("fail in encoding nodestate: %v", err)
		return nil
	}
	encodedConfigMsg, err := rlp.EncodeToBytes(cm) //将ConfigMsg消息再进行rlp序列化
	if err != nil {
		glog.V(logger.Error).Infof("fail in BackPayload: %v", err)
		return nil
	}
	return encodedConfigMsg //返回序列化之后的ConfigMsg消息
}
