// Contains a simple whisper peer setup and self messaging to allow playing
// around with the protocol and API without a fancy client implementation.
// using the discovery

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/logger"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
)

func main() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))
	// logger.NewLogger("TAG").Infoln("Start Whisper Server...")

	shh1 := whisper.New() //创建第一个whisper客户端对象
	shh2 := whisper.New()
	shh3 := whisper.New()

	//ssh1.Protocol()使得server模块需要支持whisper协议(完成whisper协议的handshake，执行Run函数)
	server1 := createServer("serv1", nil, shh1.Protocol(), "127.0.0.1:30300") //创建第一个server对象,同时启动该server对象(支持whisper协议)
	url1 := server1.Self().String()                                           //返回server的Node信息
	n, err := discover.ParseNode(url1)                                        //对url字符串进行解析，取出内容
	if err != nil {
		fmt.Printf("Error: Bootstrap URL %s: %v\n", url1, err)
		os.Exit(-1)
	}
	server2 := createServer("serv2", []*discover.Node{n}, shh2.Protocol(), "127.0.0.1:30301") //创建其后的server对象,皆以1号server的Node作为引导节点
	server3 := createServer("serv3", []*discover.Node{n}, shh3.Protocol(), "127.0.0.1:30302")
	url2 := server2.Self().String()
	url3 := server3.Self().String()

	fmt.Println("The URL of server1:", url1)
	fmt.Println("The URL of server2:", url2)
	fmt.Println("The URL of server3:", url3)

	// Send a message to self to check that something works
	payload1 := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", server2.Name, time.Now())
	if err := selfSend(shh1, shh2, []byte(payload1)); err != nil { //whisper2发消息给whisper1
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}
	payload2 := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", server1.Name, time.Now())
	if err := selfSend(shh2, shh1, []byte(payload2)); err != nil { //whisper1发消息给whisper2
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}

	payload3 := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", server3.Name, time.Now())
	if err := selfSend(shh2, shh3, []byte(payload3)); err != nil { //whisper3发消息给whisper2
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}

	fmt.Println(server1.ReadRandomNodes(5)) //whisper1随机显示5个已知节点信息(不足5个显示对应个数)
	fmt.Println(server2.ReadRandomNodes(5)) //whisper2随机显示5个已知节点信息
	fmt.Println(server3.ReadRandomNodes(5)) //whisper3随机显示5个已知节点信息
	server1.Stop()
	server2.Stop()
	server3.Stop()
}

// 创建server对象,然后调用server.Start()启动该server模块
func createServer(servername string, boostNodes []*discover.Node, pro server.Protocol, addr string) *server.Server {
	key, err := crypto.GenerateKey() //生成私钥
	if err != nil {
		fmt.Printf("Failed to generate peer key: %v.\n", err)
		os.Exit(-1)
	}
	name := common.MakeName("whisper-go-"+servername, "1.0") //server模块的名称

	server := server.Server{ //创建server对象
		PrivateKey:     key, //私钥
		MaxPeers:       10,
		Discovery:      true,
		BootstrapNodes: boostNodes, //引导节点
		Name:           name,
		Protocols:      []server.Protocol{pro}, //支持的协议
		NAT:            nat.Any(),
		ListenAddr:     addr, //监听地址
	}

	fmt.Println("Starting peer...")
	server.Start() //启动服务器模块(完成P2P网络节点之间的连接搭建)

	return &server
}

// SendSelf wraps a payload into a Whisper envelope and forwards it to itself.
func selfSend(shh1, shh2 *whisper.Whisper, payload []byte) error {
	ok := make(chan struct{})

	// Start watching for self messages, output any arrivals
	id1 := shh1.NewIdentity()  //whisper1生成的加密标识私钥
	id2 := shh2.NewIdentity()  //whisper2生成的加密标识私钥
	shh1.Watch(whisper.Filter{ //whisper1安装消息处理程序
		To: &id1.PublicKey, //消息的接收方必须是whisper1
		Fn: func(msg *whisper.Message) { //对于过滤出来的消息调用此函数进行处理
			fmt.Printf("Message received: %s, signed with 0x%x.\n", string(msg.Payload), msg.Signature)
			close(ok) //whisper1收到消息后关闭ok管道,通知已经收到whisper2的envelop
		},
	})
	// Wrap the payload and encrypt it
	msg := whisper.NewMessage(payload)                             //利用payload初始化一个原生Message
	envelope, err := msg.Wrap(whisper.DefaultPoW, whisper.Options{ //将Message消息打包成envelop信封
		From: id2,            //私钥
		To:   &id1.PublicKey, //公钥
		TTL:  whisper.DefaultTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to seal message: %v", err)
	}
	// Dump the message into the system and wait for it to pop back out
	if err := shh2.Send(envelope); err != nil { //由whisper2发送此envelop消息到整个P2P网络(只有whisper1安装了对应的过滤器能够收到消息)
		return fmt.Errorf("failed to send self-message: %v", err)
	}
	select {
	case <-ok: //whisper1收到了whisper2的envelop消息
	case <-time.After(time.Second * 100):
		return fmt.Errorf("failed to receive message in time")
	}
	return nil
}
