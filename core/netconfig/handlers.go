package netconfig

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/crypto"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/p2p/whisper"
	"reflect"
)

type NetConfigHandler interface {
	DoInitMsg(msg *whisper.Message)   //初始化构建dpnet网络时使用的消息处理方法
	DoUpdateMsg(msg *whisper.Message) //初始化阶段完成之后，更新dpnet网络时使用的消息处理方法
}

// 普通节点收到消息后的Handler
type CommonHandler struct {
	netManager *NetManager //继承
}

// 引导节点收到消息后的Handler
type BooterHandler struct {
	netManager *NetManager //继承
}

// 1. 对接收到的whisper协议Message数据包载荷部分反序列化，得到ConfigMsg消息
// 2. 再次进行反序列化，得到InitMsgConfiger消息(或CentralConfigMsg消息)
// 3. 根据对方数字签名验证发送者身份的合法性
// 4. InitMsgConfiger消息:获取发送者节点的身份信息(NetID NodeID Role),并将其加入到本地的P3-Chain网络中(更新添加到viewNet字段)
// 4. CentralConfigMsg消息:从DpNet中获取网络部署信息,更新到本地P3-Chain网络(更新到viewNet字段中)
func (ch *CommonHandler) DoInitMsg(msg *whisper.Message) {
	cfgMsg, err := DeSerializeConfigMsg(msg.Payload) //将获取的whisper协议Message数据包的载荷部分反序列化，得到ConfigMsg消息
	if err != nil {
		loglogrus.Log.Warnf("fail in DeSerializeConfigMsg: %v", err)
		return
	}
	switch cfgMsg.MsgCode {
	case selfNodeState:
		imc, err := DeSerializeInitMsgConfiger(cfgMsg.Payload) //再次进行反序列化，得到InitMsgConfiger消息
		if err != nil {
			loglogrus.Log.Warnf("fail in DeSerializeInitMsgConfiger: %v", err)
			return
		}

		if imc.Sender != imc.NodeID {
			loglogrus.Log.Warnf("incorrect sender and NodeID.")
			return
		}

		pubKey, err := crypto.SigToPub(imc.Hash(), imc.Signature) //根据消息的哈希值跟数字签名获取发送者的公钥
		signedSender := crypto.KeytoNodeID(pubKey)                //根据公钥计算发送者的节点ID
		if imc.Sender != signedSender {                           //判断消息是否真正由对应节点发送而来
			loglogrus.Log.Warnf("incorrect signature.")
			return
		}
		node := dpnet.Node{
			NetID:  imc.NetID,  //获取此消息的发送者所在的网络ID
			NodeID: imc.NodeID, //获取此消息的发送者的节点ID
			Role:   imc.Role,   //获取此消息的发送者在网络中的身份
		}
		//fmt.Printf("self is: %s, process: %s\n", ch.netManager.srv.Name, node.String())
		ch.netManager.netMutex.Lock()
		defer ch.netManager.netMutex.Unlock()
		ch.netManager.viewNet.AddNode(node) //将对方(消息的发送者)节点加入到本地的P3-Chain网络中(viewNet字段)
		return
	case centralConfig:
		ccf, err := DeSerializeCentralConfigMsg(cfgMsg.Payload) //再次进行反序列化，得到CentralConfigMsg消息
		if err != nil {
			loglogrus.Log.Warnf("fail in DeSerializeCentralConfigMsg: %v", err)
			return
		}
		if ccf.Sender != ccf.NodeID {
			loglogrus.Log.Warnf("incorrect sender and NodeID.")
			return
		}
		pubKey, err := crypto.SigToPub(ccf.Hash(), ccf.Signature) //根据消息的哈希值跟数字签名获取发送者的公钥
		signedSender := crypto.KeytoNodeID(pubKey)                //根据公钥计算发送者的节点ID
		if ccf.Sender != signedSender {                           //判断消息是否真正由对应节点发送而来
			loglogrus.Log.Warnf("incorrect signature.")
			return
		}
		ch.netManager.netMutex.Lock()
		defer ch.netManager.netMutex.Unlock()
		ch.netManager.Cen.DpNet = ccf.DpNet    //将CentralConfigMsg消息中的网络配置信息添加到Cen对象中
		ch.netManager.Cen.CentralizedNetConf() //Cen对象在配置信息中查询自身是否存在配置信息(否则需要重新获取配置信息)
		ch.netManager.CentralizedConf()        //根据Cen对象的DpNet更新viewNet对象
		return
	default:
		loglogrus.Log.Warnf("unknown msgCode: %v", cfgMsg.MsgCode)
		return
	}
}

func (ch *CommonHandler) DoUpdateMsg(msg *whisper.Message) {

	cfgMsg, err := DeSerializeConfigMsg(msg.Payload) //将获取的whisper协议Message数据包的载荷部分反序列化，得到ConfigMsg消息
	if err != nil {
		loglogrus.Log.Warnf("fail in DeSerializeConfigMsg: %v", err)
		return
	}
	umc, err := DeSerializeUpdateMsgConfiger(cfgMsg.Payload) //再次进行反序列化，得到UpdateMsgConfiger消息
	if err != nil {
		loglogrus.Log.Warnf("fail in DeSerializeUpdateMsgConfiger: %v", err)
		return
	}
	if umc.Sender != umc.NodeID {
		loglogrus.Log.Warnf("incorrect sender and NodeID.")
		return
	}
	pubKey, _ := crypto.SigToPub(umc.Hash(), umc.Signature) //根据消息的哈希值跟数字签名获取发送者的公钥
	signedSender := crypto.KeytoNodeID(pubKey)              //根据公钥计算发送者的节点ID
	if umc.Sender != signedSender {
		loglogrus.Log.Warnf("incorrect signature.")
		return
	}
	switch cfgMsg.MsgCode {
	case ReconnectState:
		loglogrus.Log.Infof("接收到更新消息,消息类型为:ReconnectState  ")

	case DpNetInfo:
		loglogrus.Log.Infof("接收到更新消息,消息类型为:DpNetInfo  ")

		// 根据umc.DpNet更新自己的netManger
		ch.netManager.netMutex.Lock()
		nodeList := umc.NodeList
		for _, node := range nodeList.NodeList {
			loglogrus.Log.Infof("当前节点：%x ,所属子网:%s, 节点角色:%d", node.NodeID, node.NetID, node.Role)
			ch.netManager.viewNet.AddNode(node) //将对方(消息的发送者)节点加入到本地的P3-Chain网络中(viewNet字段)
		}
		for _, booter := range nodeList.BooterList {
			loglogrus.Log.Infof("当前节点：%x ,所属子网:%s, 节点角色:%d", booter.NodeID, booter.NetID, booter.Role)
			ch.netManager.viewNet.AddNode(booter)
		}

		ch.netManager.Spr.Update(ch.netManager.srv, nil)

		defer ch.netManager.netMutex.Unlock()

	case UpdateNodeState:
		loglogrus.Log.Infof("接收到更新消息,消息类型为:UpdateNodeState  ")
		node := dpnet.Node{
			NetID:  umc.NetID,  //获取此消息的发送者所在的网络ID
			NodeID: umc.NodeID, //获取此消息的发送者的节点ID
			Role:   umc.Role,   //获取此消息的发送者在网络中的身份
		}
		if err := ch.netManager.UpdateLowerChannel(node, UpdateNodeState); err != nil { //根据新节点和上一轮viewNet完成子网组PeerGroup内部的更新
			loglogrus.Log.Warnf("更新lower channel层结构失败,err:%v\n", err)
			return
		}
		if err := ch.netManager.UpdateUpperChannel(node, UpdateNodeState); err != nil { //根据新节点和上一轮viewNet完成upper层PeerGroup的更新
			loglogrus.Log.Warnf("更新upper channel层结构失败,err:%v\n", err)
			return
		}
		ch.netManager.netMutex.Lock()
		if err := ch.netManager.viewNet.UpdateNode(node); err != nil { //完成viewNet的更新
			loglogrus.Log.Warnf("当前节点:%x 更新节点身份发生错误:%v\n", ch.netManager.SelfNode.NodeID, err)
			return
		}
		ch.netManager.netMutex.Unlock()
		return
	case AddNewNode:
		loglogrus.Log.Infof("接收到更新消息,消息类型为: addNewNode  ")
		node := dpnet.Node{
			NetID:  umc.NetID,  //获取此消息的发送者所在的网络ID
			NodeID: umc.NodeID, //获取此消息的发送者的节点ID
			Role:   umc.Role,   //获取此消息的发送者在网络中的身份
		}
		if err := ch.netManager.UpdateLowerChannel(node, AddNewNode); err != nil { //完成子网组内部的更新
			loglogrus.Log.Warnf("更新lower channel层结构失败,err:%v\n", err)
			return
		}
		if err := ch.netManager.UpdateUpperChannel(node, AddNewNode); err != nil { //完成upper层网络的更新
			loglogrus.Log.Warnf("更新upper channel层结构失败,err:%v\n", err)
			return
		}
		// fmt.Printf("当前节点:%x 完成所有配置\n", ch.netManager.SelfNode.NodeID)
		ch.netManager.netMutex.Lock()
		if err := ch.netManager.viewNet.AddNode(node); err != nil { //完成viewNet的更新
			loglogrus.Log.Warnf("当前节点:%x 添加到节点组发生错误:%v\n", ch.netManager.SelfNode.NodeID, err)
			return
		} else {
			loglogrus.Log.Warnf("当前节点:%x 成功添加到指定节点组\n", ch.netManager.SelfNode.NodeID)
		}
		ch.netManager.netMutex.Unlock()

		return
	case DelNode:
		loglogrus.Log.Infof("接收到更新消息,消息类型为: delNode  ")
		node := dpnet.Node{
			NetID:  umc.NetID,  //获取此消息的发送者所在的网络ID
			NodeID: umc.NodeID, //获取此消息的发送者的节点ID
			Role:   umc.Role,   //获取此消息的发送者在网络中的身份
		}
		if err := ch.netManager.UpdateLowerChannel(node, DelNode); err != nil { //根据viewNet完成子网组内部的更新
			loglogrus.Log.Infof("更新lower channel层结构失败,err:%v\n", err)
			return
		}
		if err := ch.netManager.UpdateUpperChannel(node, DelNode); err != nil { //根据viewNet完成upper层网络的更新
			loglogrus.Log.Infof("更新upper channel层结构失败,err:%v\n", err)
			return
		}
		ch.netManager.netMutex.Lock()
		if err := ch.netManager.viewNet.DelNode(node); err != nil { // 从当前节点组中删除指定的节点
			loglogrus.Log.Infof("当前节点:%x 从指定节点组删除发生错误:%v\n", ch.netManager.SelfNode.NodeID, err)
			return
		} else {
			loglogrus.Log.Infof("当前节点:%x 成功从指定节点组删除\n", ch.netManager.SelfNode.NodeID)
		}
		ch.netManager.netMutex.Unlock()
		return

	case LeaderChange:
		loglogrus.Log.Infof("接收到更新消息,消息类型为: LeaderChange ")

		// 1.验证此消息收集到的数字签名是否正确
		signValid := ch.netManager.LeaderChangeValidata(signedSender, umc.NetID, &umc.LeaderChange)

		if !signValid {
			loglogrus.Log.Warnf("Leader Change Msg 合法性验证失败\n")
			return
		}
		// 2.将原本的Leader节点降级为follower(upperNet层)
		// 3.将新的Leader节点上位(upperNet层)
		oldLeaderID := ch.netManager.GetSubNetLeader(umc.NetID) //获取变换节点组原本的LeaderID
		if (oldLeaderID != common.NodeID{}) {
			oldLeader := new(dpnet.Node)
			oldLeader.NetID = umc.NetID
			oldLeader.NodeID = oldLeaderID
			oldLeader.Role = dpnet.Follower //将其降级为Follower,也就是将其从upper PeerGroup中删除

			// if reflect.DeepEqual(oldLeader.NodeID, ch.netManager.SelfNode.NodeID) {
			// 	ch.netManager.SelfNode.Role = dpnet.Follower
			// }

			if err := ch.netManager.UpdateUpperChannel(*oldLeader, UpdateNodeState); err != nil {
				loglogrus.Log.Warnf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 无法将Leader(%x)在节点组(%s)中降级为Follower节点,err:%v\n",
					ch.netManager.SelfNode.NodeID, ch.netManager.SelfNode.NetID, ch.netManager.SelfNode.Role,
					oldLeader.NodeID, oldLeader.NetID, err)
			} else {
				loglogrus.Log.Infof("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 成功将Leader(%x)降级为节点组(%s)中的Follower节点\n",
					ch.netManager.SelfNode.NodeID, ch.netManager.SelfNode.NetID, ch.netManager.SelfNode.Role,
					oldLeader.NodeID, oldLeader.NetID)
			}

			newLeader := new(dpnet.Node)
			newLeader.NetID = umc.NetID
			newLeader.NodeID = umc.NodeID
			newLeader.Role = dpnet.Leader //新的Leader节点产生，加入到upper PeerGroup中
			if err := ch.netManager.UpdateUpperChannel(*newLeader, UpdateNodeState); err != nil {
				loglogrus.Log.Warnf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 无法将Follower(%x)在节点组(%s)中升级为Leader节点,err:%v\n",
					ch.netManager.SelfNode.NodeID, ch.netManager.SelfNode.NetID, ch.netManager.SelfNode.Role,
					newLeader.NodeID, newLeader.NetID, err)
			} else {
				loglogrus.Log.Infof("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 成功将Follower(%x)升级为节点组(%s)中的Leader节点\n",
					ch.netManager.SelfNode.NodeID, ch.netManager.SelfNode.NetID, ch.netManager.SelfNode.Role,
					newLeader.NodeID, newLeader.NetID)
			}

			if err := ch.netManager.BackViewNet().UpdateNode(*oldLeader); err != nil { //有顺序要求,先将原Leader降级,再让新Leader上位
				loglogrus.Log.Warnf("[Leader Change] 无法完成旧Leader在viewNet中的更新, err:%v\n", err)
			} else {
				loglogrus.Log.Infof("[Leader Change] 成功完成旧Leader在viewNet中的更新\n")
			}
			if err := ch.netManager.BackViewNet().UpdateNode(*newLeader); err != nil {
				loglogrus.Log.Warnf("[Leader Change] 无法完成新Leader在viewNet中的更新, err:%v\n", err)
			} else {
				loglogrus.Log.Infof("[Leader Change] 成功完成新Leader在viewNet中的更新\n")
			}

			if reflect.DeepEqual(newLeader.NodeID, ch.netManager.SelfNode.NodeID) {
				ch.netManager.Spr.Update(ch.netManager.srv, nil)
			}

		}
	default:
		loglogrus.Log.Warnf("unknown msgCode: %v", cfgMsg.MsgCode)
		return
	}

}

func (bh *BooterHandler) DoInitMsg(msg *whisper.Message) {
	cfgMsg, err := DeSerializeConfigMsg(msg.Payload)
	if err != nil {
		loglogrus.Log.Warnf("fail in DeSerializeConfigMsg: %v", err)
		return
	}
	switch cfgMsg.MsgCode {
	case selfNodeState:
		imc, err := DeSerializeInitMsgConfiger(cfgMsg.Payload)
		if err != nil {
			loglogrus.Log.Warnf("fail in DeSerializeInitMsgConfiger: %v", err)
			return
		}

		if imc.Sender != imc.NodeID {
			loglogrus.Log.Warnf("incorrect sender and NodeID.")
			return
		}

		pubKey, err := crypto.SigToPub(imc.Hash(), imc.Signature)
		signedSender := crypto.KeytoNodeID(pubKey)
		if imc.Sender != signedSender {
			loglogrus.Log.Warnf("incorrect signature.")
			return
		}

		node := dpnet.Node{
			NetID:  imc.NetID,
			NodeID: imc.NodeID,
			Role:   imc.Role,
		}
		//fmt.Printf("self is: %s, process: %s\n", bh.netManager.srv.Name, node.String())

		bh.netManager.netMutex.Lock()
		defer bh.netManager.netMutex.Unlock()
		bh.netManager.viewNet.AddNode(node)
		return
	case centralConfig:
		fmt.Println("Receive centralConfig msg .......")
		ccf, err := DeSerializeCentralConfigMsg(cfgMsg.Payload) //再次进行反序列化，得到CentralConfigMsg消息
		if err != nil {
			loglogrus.Log.Warnf("fail in DeSerializeCentralConfigMsg: %v", err)
			return
		}
		if ccf.Sender != ccf.NodeID {
			loglogrus.Log.Warnf("incorrect sender and NodeID.")
			return
		}
		pubKey, err := crypto.SigToPub(ccf.Hash(), ccf.Signature) //根据消息的哈希值跟数字签名获取发送者的公钥
		signedSender := crypto.KeytoNodeID(pubKey)                //根据公钥计算发送者的节点ID
		if ccf.Sender != signedSender {                           //判断消息是否真正由对应节点发送而来
			glog.V(logger.Warn).Infof("incorrect signature.")
			return
		}
		bh.netManager.netMutex.Lock()
		defer bh.netManager.netMutex.Unlock()
		bh.netManager.Cen.DpNet = ccf.DpNet    //将CentralConfigMsg消息中的网络配置信息添加到Cen对象中
		bh.netManager.Cen.CentralizedNetConf() //Cen对象在配置信息中查询自身是否存在配置信息(否则需要重新获取配置信息)
		bh.netManager.CentralizedConf()        //根据Cen对象的DpNet更新viewNet对象
		return
	default:
		loglogrus.Log.Warnf("unknown msgCode: %v", cfgMsg.MsgCode)
		return
	}
}

func (bh *BooterHandler) DoUpdateMsg(msg *whisper.Message) {
	cfgMsg, err := DeSerializeConfigMsg(msg.Payload)
	if err != nil {
		loglogrus.Log.Warnf("fail in DeSerializeConfigMsg: %v\n", err)
		return
	}
	umc, err := DeSerializeUpdateMsgConfiger(cfgMsg.Payload) //再次进行反序列化，得到UpdateMsgConfiger消息
	if err != nil {
		loglogrus.Log.Warnf("fail in DeSerializeUpdateMsgConfiger: %v\n", err)
		return
	}
	if umc.Sender != umc.NodeID {
		loglogrus.Log.Warnf("incorrect sender and NodeID.\n")
		return
	}
	pubKey, _ := crypto.SigToPub(umc.Hash(), umc.Signature) //根据消息的哈希值跟数字签名获取发送者的公钥
	signedSender := crypto.KeytoNodeID(pubKey)              //根据公钥计算发送者的节点ID
	if umc.Sender != signedSender {                         //判断消息是否真正由对应节点发送而来
		loglogrus.Log.Warnf("incorrect signature.\n")
		return
	}
	switch cfgMsg.MsgCode {
	case ReconnectState:
		loglogrus.Log.Infof("Receive reconnectState msg .......")
		// 向其回复整个分区网络的拓扑结构
		if err := bh.netManager.SendDpNetInfo(); err != nil {
			loglogrus.Log.Warnf("Can't Send DpNetInfo Msg to Disconnect Node ,err: %v\n", err)
		}

	case LeaderChange:
		loglogrus.Log.Infof("接收到更新消息,消息类型为: LeaderChange ")

		// 1.验证此消息收集到的数字签名是否正确
		signValid := bh.netManager.LeaderChangeValidata(signedSender, umc.NetID, &umc.LeaderChange)

		if !signValid {
			loglogrus.Log.Warnf("Leader Change Msg 合法性验证失败\n")
			return
		}
		// 2.将原本的Leader节点降级为follower(upperNet层)
		// 3.将新的Leader节点上位(upperNet层)
		oldLeaderID := bh.netManager.GetSubNetLeader(umc.NetID) //获取变换节点组原本的LeaderID
		if (oldLeaderID != common.NodeID{}) {
			oldLeader := new(dpnet.Node)
			oldLeader.NetID = umc.NetID
			oldLeader.NodeID = oldLeaderID
			oldLeader.Role = dpnet.Follower //将其降级为Follower,也就是将其从upper PeerGroup中删除
			if err := bh.netManager.UpdateUpperChannel(*oldLeader, UpdateNodeState); err != nil {
				fmt.Printf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 无法将Leader(%x)在节点组(%s)中降级为Follower节点,err:%v\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					oldLeader.NodeID, oldLeader.NetID, err)
				loglogrus.Log.Warnf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 无法将Leader(%x)在节点组(%s)中降级为Follower节点,err:%v\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					oldLeader.NodeID, oldLeader.NetID, err)
			} else {
				fmt.Printf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 成功将Leader(%x)降级为节点组(%s)中的Follower节点\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					oldLeader.NodeID, oldLeader.NetID)
				loglogrus.Log.Infof("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 成功将Leader(%x)降级为节点组(%s)中的Follower节点\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					oldLeader.NodeID, oldLeader.NetID)
			}

			newLeader := new(dpnet.Node)
			newLeader.NetID = umc.NetID
			newLeader.NodeID = umc.NodeID
			newLeader.Role = dpnet.Leader //新的Leader节点产生，加入到upper PeerGroup中
			if err := bh.netManager.UpdateUpperChannel(*newLeader, UpdateNodeState); err != nil {
				fmt.Printf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 无法将Follower(%x)在节点组(%s)中升级为Leader节点,err:%v\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					newLeader.NodeID, newLeader.NetID, err)
				loglogrus.Log.Warnf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 无法将Follower(%x)在节点组(%s)中升级为Leader节点,err:%v\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					newLeader.NodeID, newLeader.NetID, err)
			} else {
				fmt.Printf("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 成功将Follower(%x)升级为节点组(%s)中的Leader节点\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					newLeader.NodeID, newLeader.NetID)
				loglogrus.Log.Infof("[Leader Change] 当前节点(NodeID:%x netID:%s role:%d ) 成功将Follower(%x)升级为节点组(%s)中的Leader节点\n",
					bh.netManager.SelfNode.NodeID, bh.netManager.SelfNode.NetID, bh.netManager.SelfNode.Role,
					newLeader.NodeID, newLeader.NetID)
			}

			if err := bh.netManager.BackViewNet().UpdateNode(*oldLeader); err != nil { //有顺序要求,先将原Leader降级,再让新Leader上位
				fmt.Printf("[Leader Change] 无法完成旧Leader在viewNet中的更新, err:%v\n", err)
				loglogrus.Log.Warnf("[Leader Change] 无法完成旧Leader在viewNet中的更新, err:%v\n", err)
			} else {
				fmt.Printf("[Leader Change] 成功完成旧Leader在viewNet中的更新\n")
				loglogrus.Log.Infof("[Leader Change] 成功完成旧Leader在viewNet中的更新\n")
			}
			if err := bh.netManager.BackViewNet().UpdateNode(*newLeader); err != nil {
				fmt.Printf("[Leader Change] 无法完成新Leader在viewNet中的更新, err:%v\n", err)
				loglogrus.Log.Warnf("[Leader Change] 无法完成新Leader在viewNet中的更新, err:%v\n", err)
			} else {
				fmt.Printf("[Leader Change] 成功完成新Leader在viewNet中的更新\n")
				loglogrus.Log.Infof("[Leader Change] 成功完成新Leader在viewNet中的更新\n")
			}
		}
	}
}
