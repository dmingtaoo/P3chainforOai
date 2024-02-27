package nat

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NatHoleMap struct {
	serverIP   net.IP //服务器的IP地址
	serverPort int    //服务器的Port

	selfIPEx      net.IP //自身的对外IP
	udpSelfPortEx int    //自身的UDP对外Port
	tcpSelfPortEx int    //自身的TCP对外Port

	selfPortIn int //内网主机本地使用的Port

	ipMutex   sync.WaitGroup //对selfIPEx进行保护，AddMapping()方法让等待组+-1，ExternalIP()方法等待等待组=0
	IPChannel chan bool      //对selfIPEx进行保护，同一时刻只有一个协程可以操作该字段
}

//本方法可以获取自身的对外公网IP地址
func (client *NatHoleMap) ExternalIP() (addr net.IP, err error) {

	client.ipMutex.Wait()
	//如果已经通过AddMapping方法获取了自己的公网IP地址,不再需要向第三方服务器获取
	if client.selfIPEx != nil {
		fmt.Printf("External IP is existed: %s\n", client.selfIPEx)
		return client.selfIPEx, nil
	}

	//client.IPChannel <- true //任何协程的ExternalIP()和AddMapping()方法与第三方服务器通信时，需要用管道(通信前写入，通信完成后读出)
	//1.连接到服务器UDP端口(golang中，udp也需要"连接")
	socket, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   client.serverIP,
		Port: client.serverPort,
	})
	if err != nil {
		fmt.Println("连接UDP服务器失败,err: ", err)
		return nil, err
	}
	defer socket.Close()
	//2.向服务器发送指令
	sendData := []byte("GET ExIP:") //获取自身公网IP
	_, err = socket.Write(sendData) // 发送数据
	if err != nil {
		fmt.Println("发送数据失败,err: ", err)
		return nil, err
	}
	//3.读取从服务器接收的数据
	data := make([]byte, 4096)
	socket.SetReadDeadline(time.Now().Add(1000 * time.Millisecond)) //设置等待时间为1s
	n, remoteAddr, err := socket.ReadFromUDP(data)                  // 接收数据
	if err != nil {
		fmt.Println("接收数据失败或超时, err: ", err)
		return nil, err
	}
	fmt.Printf("freedom intport : selfUDPAddr:%v\tserverAddr:%v\tbyteCount:%v\n", string(data[:n]), remoteAddr, n)

	ExAddr := strings.Split(string(data[:n]), ":") //以分隔符：为界限，分别获取自身对外的公网IP地址和端口号

	client.selfIPEx = net.ParseIP(ExAddr[0])          //获取自身对外公网IP
	client.udpSelfPortEx, _ = strconv.Atoi(ExAddr[1]) //获取自身对外公网Port(未采用指定内网端口的对外映射端口)

	//<-client.IPChannel
	return client.selfIPEx, nil
}

func (client *NatHoleMap) String() string {
	return "NatHoleMap: " + client.serverIP.String()
}

//本方法实现获取内网端口映射后的对外端口号，仅针对于锥型NAT来说，每一个内网主机上的客户端应用对外端口号是固定的，
//因此返回向服务器发送消息时所用的端口号即可
func (client *NatHoleMap) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {

	//client.IPChannel <- true
	client.ipMutex.Add(1)
	//必须获取用指定本地端口号intport时的对外端口号
	if strings.Contains(protocol, "udp") { //udp-NAT打洞时对外端口号
		//向服务器UDP地址发送消息
		socket, err := SendMessage("udp", intport, client.serverIP, client.serverPort, []byte("GET ExIP:"))
		if err != nil {
			fmt.Println(err)
			return err
		}
		//接收来自服务器的回复(自身的公网UDP地址)
		udpSocket := socket.(*net.UDPConn) //类型断言
		data := make([]byte, 4096)
		//udpSocket.SetReadDeadline(time.Now().Add(1000 * time.Millisecond)) //设置等待时间为1s
		n, remoteAddr, err := udpSocket.ReadFromUDP(data) // 接收数据
		if err != nil {
			fmt.Println("接收数据失败, err: ", err)
			return err
		}
		fmt.Printf("Specify port: selfUDPAddr:%v\tserverAddr:%v\tbyteCount:%v\n", string(data[:n]), remoteAddr, n)
		ExAddr := strings.Split(string(data[:n]), ":") //以分隔符：为界限，分别获取自身对外的公网IP地址和端口号

		client.selfIPEx = net.ParseIP(ExAddr[0])          //获取自身对外公网IP
		client.udpSelfPortEx, _ = strconv.Atoi(ExAddr[1]) //获取自身对外公网Port(采用指定内网端口的对外映射端口)

		//<-client.IPChannel
		defer udpSocket.Close()

	} else if strings.Contains(protocol, "tcp") { //tcp-NAT打洞时对外端口号

		socket, err := SendMessage("tcp", intport, client.serverIP, client.serverPort, []byte("TCP GET ExIP:"))
		if err != nil {
			fmt.Println(err)
			return err
		}
		//接收来自服务器的回复(自身的公网TCP地址)
		tcpSocket := socket.(*net.TCPConn)
		data := make([]byte, 4096)
		//tcpSocket.SetReadDeadline(time.Now().Add(1000 * time.Millisecond)) //设置等待时间为1s
		n, err := tcpSocket.Read(data) // 接收数据
		if err != nil {
			fmt.Println("接收数据失败, err: ", err)
			return err
		}
		fmt.Printf("Specify port: selfTCPAddr:%v\tserverAddr:%v\tbyteCount:%v\n", string(data[:n]), tcpSocket.RemoteAddr().String(), n)
		ExAddr := strings.Split(string(data[:n]), ":") //以分隔符：为界限，分别获取自身对外的公网IP地址和端口号

		client.selfIPEx = net.ParseIP(ExAddr[0])          //获取自身对外公网IP
		client.tcpSelfPortEx, _ = strconv.Atoi(ExAddr[1]) //获取自身对外公网Port(采用指定内网端口的对外映射端口)

		//<-client.IPChannel
		defer tcpSocket.Close()
	}
	client.ipMutex.Done()
	return nil
}

//采用指定的协议，指定的本地网络地址，向着指定的目标网络地址发送指定内容的消息
func SendMessage(protocal string, inport int, dstIP net.IP, dstPort int, msg []byte) (interface{}, error) {

	if protocal == "udp" {
		//本地将使用的网络地址
		laddr := &net.UDPAddr{
			//IP:   client.internalAddress(),
			Port: inport,
		}
		//用指定的内网地址连接指定的目标网络地址
		socket, err := net.DialUDP("udp4", laddr, &net.UDPAddr{
			IP:   dstIP,
			Port: dstPort,
		})
		if err != nil {
			fmt.Println("连接UDP服务器失败,err: ", err)
			return nil, err
		}

		//发送指定内容的消息
		sendData := msg
		_, err = socket.Write(sendData) // 发送数据
		if err != nil {
			fmt.Println("发送数据失败,err: ", err)
			return nil, err
		}
		return socket, nil //返回UDP socket

	} else if protocal == "tcp" {
		//本地将使用的网络地址
		laddr := &net.TCPAddr{
			//IP:   client.internalAddress(),
			Port: inport,
		}
		//用指定的内网地址连接指定的目标网络地址
		socket, err := net.DialTCP("tcp", laddr, &net.TCPAddr{
			IP:   dstIP,
			Port: dstPort,
		})
		if err != nil {
			fmt.Println("连接TCP服务器失败,err: ", err)
			return nil, err
		}
		//发送指定内容的消息
		sendData := msg
		_, err = socket.Write(sendData) // 发送数据
		if err != nil {
			fmt.Println("发送数据失败,err: ", err)
			return nil, err
		}
		return socket, nil //返回TCP socket
	}
	return nil, nil
}

func (client *NatHoleMap) DeleteMapping(protocol string, extport, intport int) (err error) {

	//udp可以通过停止发送心跳包结束映射
	//tcp通过断开tcp连接即可结束映射
	return nil
}
