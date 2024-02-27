//go:build none
// +build none

// Contains a simple whisper peer setup and self messaging to allow playing
// around with the protocol and API without a fancy client implementation.

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"p3Chain/p2p/common"
	"p3Chain/p2p/crypto"
	"p3Chain/p2p/logger"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
)

func main() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))

	// Generate the peer identity
	key, err := crypto.GenerateKey() //server模块私钥
	if err != nil {
		fmt.Printf("Failed to generate peer key: %v.\n", err)
		os.Exit(-1)
	}
	name := common.MakeName("whisper-go", "1.0") //server模块名字
	shh := whisper.New()                         //初始化一个whisper客户端对象

	// Create a peer to communicate through
	server := server.Server{ //初始化一个server模块
		PrivateKey: key,
		MaxPeers:   10,
		Name:       name,
		Protocols:  []server.Protocol{shh.Protocol()},
		ListenAddr: ":30300",
		NAT:        nat.Any(),
	}
	fmt.Println("Starting peer...")
	if err := server.Start(); err != nil { //启动运行server模块(完成P2P网络的连接搭建)
		fmt.Printf("Failed to start peer: %v.\n", err)
		os.Exit(1)
	}

	// Send a message to self to check that something works
	payload := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", name, time.Now())
	if err := selfSend(shh, []byte(payload)); err != nil {
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}
}

// SendSelf wraps a payload into a Whisper envelope and forwards it to itself.
func selfSend(shh *whisper.Whisper, payload []byte) error {
	ok := make(chan struct{})

	// Start watching for self messages, output any arrivals
	id := shh.NewIdentity()   //whisper客户端通信使用的加密标识
	shh.Watch(whisper.Filter{ //whisper客户端安装消息处理程序
		To: &id.PublicKey, //envelop消息的接收方必须是当前whisper客户端
		Fn: func(msg *whisper.Message) { //对于过滤出来的消息调用此函数进行处理
			fmt.Printf("Message received: %s, signed with 0x%x.\n", string(msg.Payload), msg.Signature)
			close(ok) //关闭ok管道，作为读取消息完成的通知
		},
	})
	// Wrap the payload and encrypt it
	msg := whisper.NewMessage(payload)                             //利用payload初始化一个原生Message
	envelope, err := msg.Wrap(whisper.DefaultPoW, whisper.Options{ //将Message消息打包成envelop信封
		From: id,            //消息源--当前whisper客户端的私钥
		To:   &id.PublicKey, //消息目标--当前whisper客户端的的公钥
		TTL:  whisper.DefaultTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to seal message: %v", err)
	}
	// Dump the message into the system and wait for it to pop back out
	if err := shh.Send(envelope); err != nil { //发送envelop信封
		return fmt.Errorf("failed to send self-message: %v", err)
	}
	select {
	case <-ok: //等待接受处理成功
	case <-time.After(time.Second):
		return fmt.Errorf("failed to receive message in time")
	}
	return nil
}
