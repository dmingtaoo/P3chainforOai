package nat

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
)

const soapRequestTimeout = 3 * time.Second

type upnp struct {
	dev     *goupnp.RootDevice //支持UPnP的根设备 ？？
	service string
	client  upnpClient //请求UPnP的客户端，也就是内网下的主机节点的客户端程序
}

type upnpClient interface {
	GetExternalIPAddress() (string, error)                                             //获取NAT设备对外的公网IP地址(多层NAT就不是公网IP地址)
	AddPortMapping(string, uint16, string, uint16, string, bool, string, uint32) error //添加端口映射
	DeletePortMapping(string, uint16, string) error                                    //删除端口映射
	GetNATRSIPStatus() (sip bool, nat bool, err error)
}

//获取内网主机上的客户端程序在NAT网管上的对外的映射IP地址
func (n *upnp) ExternalIP() (addr net.IP, err error) {
	ipString, err := n.client.GetExternalIPAddress() //获取内网主机的客户端程序的NAT映射IP地址
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(ipString) //对返回的字符串形式的映射IP地址进行解析(IPV4 or IPV6)
	if ip == nil {
		return nil, errors.New("bad IP in response")
	}
	return ip, nil
}

//在NAT网管上实现upnp协议的NAT地址映射
func (n *upnp) AddMapping(protocol string, extport, intport int, desc string, lifetime time.Duration) error {
	ip, err := n.internalAddress() //获取本主机节点的内网IP地址
	if err != nil {
		return nil
	}
	protocol = strings.ToUpper(protocol)
	lifetimeS := uint32(lifetime / time.Second)
	//调用以下方法在NAT网管上实现NAT地址映射(指定传输层协议类型，生存期，对外的端口等。。)
	return n.client.AddPortMapping("", uint16(extport), protocol, uint16(intport), ip.String(), true, desc, lifetimeS)
}

//获取内网主机的内网IP地址
func (n *upnp) internalAddress() (net.IP, error) {
	devaddr, err := net.ResolveUDPAddr("udp4", n.dev.URLBase.Host) //获取根设备指定的UDP地址
	if err != nil {
		return nil, err
	}
	ifaces, err := net.Interfaces() //获取本主机所有的网络接口(一台计算机一般有若干网络接口，IP的，蓝牙的。。。)
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs() //获取上述获得的所有网络接口的地址
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			switch x := addr.(type) { //类型断言，获取IP网络地址的网络接口
			case *net.IPNet:
				if x.Contains(devaddr.IP) { //查看这些IP地址的网络接口中是否存在根设备指定的UDP地址，若存在返回该地址
					return x.IP, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("could not find local address in same net as %v", devaddr)
}

//删除NAT网管上的指定映射记录
func (n *upnp) DeleteMapping(protocol string, extport, intport int) error {
	return n.client.DeletePortMapping("", uint16(extport), strings.ToUpper(protocol))
}

func (n *upnp) String() string {
	return "UPNP " + n.service
}

// discoverUPnP searches for Internet Gateway Devices
// and returns the first one it can find on the local network.
//本函数负责在局域网中搜寻支持UPNP的网管设备，并返回它在本地网络上可以找到的第一个设备。
//返回值是upnp结构体类型的c，该结构体实现了nat.go中的Interface接口
func discoverUPnP() Interface {
	found := make(chan *upnp, 2)
	// IGDv1   搜索支持UPnP IGD V1协议的路由器
	go discover(found, internetgateway1.URN_WANConnectionDevice_1, func(dev *goupnp.RootDevice, sc goupnp.ServiceClient) *upnp {
		switch sc.Service.ServiceType {
		case internetgateway1.URN_WANIPConnection_1:
			return &upnp{dev, "IGDv1-IP1", &internetgateway1.WANIPConnection1{sc}}
		case internetgateway1.URN_WANPPPConnection_1:
			return &upnp{dev, "IGDv1-PPP1", &internetgateway1.WANPPPConnection1{sc}}
		}
		return nil
	})
	// IGDv2  搜索支持UPnP IGD V2协议的路由器
	go discover(found, internetgateway2.URN_WANConnectionDevice_2, func(dev *goupnp.RootDevice, sc goupnp.ServiceClient) *upnp {
		switch sc.Service.ServiceType {
		case internetgateway2.URN_WANIPConnection_1:
			return &upnp{dev, "IGDv2-IP1", &internetgateway2.WANIPConnection1{sc}}
		case internetgateway2.URN_WANIPConnection_2:
			return &upnp{dev, "IGDv2-IP2", &internetgateway2.WANIPConnection2{sc}}
		case internetgateway2.URN_WANPPPConnection_1:
			return &upnp{dev, "IGDv2-PPP1", &internetgateway2.WANPPPConnection1{sc}}
		}
		return nil
	})
	for i := 0; i < cap(found); i++ {
		if c := <-found; c != nil {
			return c
		}
	}
	return nil
}

func discover(out chan<- *upnp, target string, matcher func(*goupnp.RootDevice, goupnp.ServiceClient) *upnp) {
	devs, err := goupnp.DiscoverDevices(target)
	if err != nil {
		return
	}
	found := false
	for i := 0; i < len(devs) && !found; i++ {
		if devs[i].Root == nil {
			continue
		}
		devs[i].Root.Device.VisitServices(func(service *goupnp.Service) {
			if found {
				return
			}
			// check for a matching IGD service
			sc := goupnp.ServiceClient{
				SOAPClient: service.NewSOAPClient(),
				RootDevice: devs[i].Root,
				Location:   devs[i].Location,
				Service:    service,
			}
			sc.SOAPClient.HTTPClient.Timeout = soapRequestTimeout
			upnp := matcher(devs[i].Root, sc)
			if upnp == nil {
				return
			}
			// check whether port mapping is enabled
			if _, nat, err := upnp.client.GetNATRSIPStatus(); err != nil || !nat {
				return
			}
			out <- upnp
			found = true
		})
	}
	if !found {
		out <- nil
	}
}
