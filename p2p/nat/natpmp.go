package nat

import (
	"fmt"
	"net"
	"strings"
	"time"

	natpmp "github.com/jackpal/go-nat-pmp"
)

// natPMPClient adapts the NAT-PMP protocol implementation so it conforms to
// the common interface.
//pmp结构体，实现了Interface接口
type pmp struct {
	gw net.IP         //网管的IP地址
	c  *natpmp.Client //客户端
}

//获取网管IP地址的字符串形式
func (n *pmp) String() string {
	return fmt.Sprintf("NAT-PMP(%v)", n.gw)
}

//获取客户端在NAT网管上的映射地址
func (n *pmp) ExternalIP() (net.IP, error) {
	response, err := n.c.GetExternalAddress()
	if err != nil {
		return nil, err
	}
	return response.ExternalIPAddress[:], nil //成功的话返回映射的公网IP地址
}

//采用PMP协议实现在NAT设备上的端口映射(指定传输层协议、端口的映射关系，并设置生命周期)
func (n *pmp) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {
	if lifetime <= 0 {
		return fmt.Errorf("lifetime must not be <= 0")
	}
	// Note order of port arguments is switched between our
	// AddMapping and the client's AddPortMapping.
	_, err := n.c.AddPortMapping(strings.ToLower(protocol), intport, extport, int(lifetime/time.Second))
	return err
}

//删除NAT设备上指定的端口映射关系
func (n *pmp) DeleteMapping(protocol string, extport, intport int) (err error) {
	// To destroy a mapping, send an add-port with an internalPort of
	// the internal port to destroy, an external port of zero and a
	// time of zero.
	_, err = n.c.AddPortMapping(strings.ToLower(protocol), intport, 0, 0) //删除的方法比较特殊，就是让映射记录的声明周期变为0
	return err
}

//在局域网通过广播的方式查询支持NAT-PMP协议的网管路由器
func discoverPMP() Interface {
	// run external address lookups on all potential gateways
	gws := potentialGateways() //获取所有可能的局域网下NAT网管的IP地址(内网IP地址)
	found := make(chan *pmp, len(gws))
	for i := range gws {
		gw := gws[i]
		//由单独的协程负责向每一个可能的网管IP发送探测请求
		go func() {
			c := natpmp.NewClient(gw) //此函数向所有可能的IP地址发送请求，当真正的NAT网管收到此消息时，若支持NAT-PMP协议则成功建立连接
			if _, err := c.GetExternalAddress(); err != nil {
				found <- nil
			} else {
				found <- &pmp{gw, c} //成功与支持NAT-PMP协议的网管建立连接。返回该网管的内网IP和natpmp.Client结构体
			}
		}()
	}
	// return the one that responds first.
	// discovery needs to be quick, so we stop caring about
	// any responses after a very short timeout.
	timeout := time.NewTimer(1 * time.Second) //探测截止时间为1s
	defer timeout.Stop()
	for _ = range gws {
		select {
		case c := <-found:
			if c != nil {
				return c //返回该NAT网管的内网IP和能与其通信的natpmp.Client结构体
			}
		case <-timeout.C: //在1s内没有完成对NAT-PMP路由器的查询，返回nil
			return nil
		}
	}
	return nil
}

//三类内网IP地址范围
var (
	// LAN IP ranges
	_, lan10, _  = net.ParseCIDR("10.0.0.0/8")
	_, lan176, _ = net.ParseCIDR("172.16.0.0/12")
	_, lan192, _ = net.ParseCIDR("192.168.0.0/16")
)

// TODO: improve this. We currently assume that (on most networks)
// the router is X.X.X.1 in a local LAN range.
//获取所有可能的局域网中NAT网管的IP地址(在这里总是假设路由器的IP地址是每个网段下的1号主机地址)
func potentialGateways() (gws []net.IP) {
	ifaces, err := net.Interfaces() //查询本机的所有网络接口
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		ifaddrs, err := iface.Addrs() //获取本机所有网络接口的网络地址
		if err != nil {
			return gws
		}
		for _, addr := range ifaddrs {
			switch x := addr.(type) { //类型断言
			case *net.IPNet: //只取出IP类型网络地址的网络接口
				//只获取内网类型的IP地址
				if lan10.Contains(x.IP) || lan176.Contains(x.IP) || lan192.Contains(x.IP) {
					ip := x.IP.Mask(x.Mask).To4() //获取IP地址的网络地址(主机地址都设为0)
					if ip != nil {
						ip[3] = ip[3] | 0x01 //获取各网段下的1号主机的IP(通常情况下这些地址是网络中路由器的地址)
						gws = append(gws, ip)
					}
				}
			}
		}
	}
	return gws
}
