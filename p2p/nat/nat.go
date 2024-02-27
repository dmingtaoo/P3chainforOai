// Package nat provides access to common network port mapping protocols.
package nat

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"p3Chain/logger"
	"p3Chain/logger/glog"

	natpmp "github.com/jackpal/go-nat-pmp"
)

// An implementation of nat.Interface can map local ports to ports
// accessible from the Internet.
// 本接口的函数能够实现本地内网地址到公网地址的映射
type Interface interface {
	// These methods manage a mapping between a port on the local
	// machine to a port that can be connected to from the internet.
	//
	// protocol is "UDP" or "TCP". Some implementations allow setting
	// a display name for the mapping. The mapping may be removed by
	// the gateway when its lifetime ends.

	//形参protocol可以是UDP也可是TCP. 形参name表示在映射时可以设置映射的显示名称。
	//lifetime表示映射存在生存期(当生存期结束时，映射关系会被NAT设备删除)
	AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error
	DeleteMapping(protocol string, extport, intport int) error

	// This method should return the external (Internet-facing)
	// address of the gateway device.
	//本方法负责返回NAT设备的外网地址
	ExternalIP() (net.IP, error)

	// Should return name of the method. This is used for logging.
	String() string
}

// Parse parses a NAT interface description.
// The following formats are currently accepted.
// Note that mechanism names are not case-sensitive.
// 对描述NAT的字符串进行解析，下面是当前会被接收的格式，不区分映射机制的大小写
//
//	"" or "none"         return nil
//	"extip:77.12.33.4"   will assume the local machine is reachable on the given IP  假定NAT对外映射地址为此IP，外部网络可以此访问内网
//	"any"                uses the first auto-detected mechanism
//	"upnp"               uses the Universal Plug and Play protocol  使用UPnP协议
//	"pmp"                uses NAT-PMP with an auto-detected gateway address  使用PMP协议，
//	"pmp:192.168.0.1"    uses NAT-PMP with the given gateway address
func Parse(spec string) (Interface, error) {
	var (
		parts = strings.SplitN(spec, ":", 2) //如果有字符":"，以其为边界将字符串划分为两部分
		mech  = strings.ToLower(parts[0])    //将第一部分，也就是映射机制小写
		ip    net.IP
	)
	if len(parts) > 1 {
		ip = net.ParseIP(parts[1])
		if ip == nil {
			return nil, errors.New("invalid IP address")
		}
	}
	switch mech {
	case "", "none", "off":
		return nil, nil
	case "any", "auto", "on":
		return Any(), nil
	case "extip", "ip":
		if ip == nil {
			return nil, errors.New("missing IP address")
		}
		return ExtIP(ip), nil
	case "upnp":
		return UPnP(), nil
	case "pmp", "natpmp", "nat-pmp":
		return PMP(ip), nil
	default:
		return nil, fmt.Errorf("unknown mechanism %q", parts[0])
	}
}

const (
	mapTimeout        = 20 * time.Minute //NAT映射持续时间
	mapUpdateInterval = 15 * time.Minute //更新时间
)

// Map adds a port mapping on m and keeps it alive until c is closed.
// This function is typically invoked in its own goroutine.
// Map函数为m添加一个端口映射，并保持此映射关系直到管道C被关闭 。通常由一个协程单独运行此函数
func Map(m Interface, c chan struct{}, protocol string, extport, intport int, name string) {
	refresh := time.NewTimer(mapUpdateInterval)
	defer func() {
		refresh.Stop()
		glog.V(logger.Debug).Infof("deleting port mapping: %s %d -> %d (%s) using %s\n", protocol, extport, intport, name, m)
		m.DeleteMapping(protocol, extport, intport) //函数结束时删除NAT映射
	}()
	if err := m.AddMapping(protocol, intport, extport, name, mapTimeout); err != nil { //添加映射关系
		glog.V(logger.Debug).Infof("network port %s:%d could not be mapped: %v\n", protocol, intport, err)
	} else {
		glog.V(logger.Info).Infof("mapped network port %s:%d -> %d (%s) using %s\n", protocol, extport, intport, name, m)
	}
	for {
		select {
		case _, ok := <-c: //当管道c被关闭时，ok == false ，本函数才会被结束(NAT映射被删除)
			if !ok {
				return
			}
		case <-refresh.C: //到达了计时器规定的时间，重新为m添加NAT映射(刷新)，保证NAT映射不会失效
			glog.V(logger.Detail).Infof("refresh port mapping %s:%d -> %d (%s) using %s\n", protocol, extport, intport, name, m)
			if err := m.AddMapping(protocol, intport, extport, name, mapTimeout); err != nil {
				glog.V(logger.Debug).Infof("network port %s:%d could not be mapped: %v\n", protocol, intport, err)
			}
			refresh.Reset(mapUpdateInterval)
		}
	}
}

// ExtIP assumes that the local machine is reachable on the given
// external IP address, and that any required ports were mapped manually.
// Mapping operations will not return an error but won't actually do anything.
// 把 net.IP类型的IP地址转化为 extIP类型
func ExtIP(ip net.IP) Interface {
	if ip == nil {
		panic("IP must not be nil")
	}
	return extIP(ip) //将形参ip强制类型转化为extIP类型(本类型有各种方法)
}

type extIP net.IP //实现了Interface类型的接口

func (n extIP) ExternalIP() (net.IP, error) { return net.IP(n), nil }                      //返回net.IP类型的IP地址
func (n extIP) String() string              { return fmt.Sprintf("ExtIP(%v)", net.IP(n)) } //返回字符串类型的IP地址

// These do nothing.
func (extIP) AddMapping(string, int, int, string, time.Duration) error { return nil }
func (extIP) DeleteMapping(string, int, int) error                     { return nil }

// Any returns a port mapper that tries to discover any supported
// mechanism on the local network.
// 本函数负责在NAT设备上搜索任意的支持实现NAT映射的协议(UPnP或PMP),返回第一个查询到的端口映射器
func Any() Interface {
	// TODO: attempt to discover whether the local machine has an
	// Internet-class address. Return ExtIP in this case.
	portMap := startautodisc("UPnP or NAT-PMP", func() Interface {
		found := make(chan Interface, 2)
		go func() { found <- discoverUPnP() }() //一个协程负责检查NAT设备是否支持UPnP协议，是的话返回UPnP端口映射器
		go func() { found <- discoverPMP() }()  //一个协程负责检查NAT设备是否支持PMP协议，是的话返回PMP端口映射器
		for i := 0; i < cap(found); i++ {
			if c := <-found; c != nil {
				return c //返回第一个获取到的端口映射器
			}
		}
		return nil //NAT设备不支持UPnP和PMP，返回nil
	})
	return portMap
}

// state out the external and internal addr pair directly
func StaticReflect(filePath string) Interface {
	sr := new(addrReflector)
	sr.filePath = filePath
	return sr
}

func NatHole(holeIP net.IP, holePort int) Interface {
	nh := new(NatHoleMap)
	nh.serverIP = holeIP
	nh.serverPort = holePort
	return nh
}

// UPnP returns a port mapper that uses UPnP. It will attempt to
// discover the address of your router using UDP broadcasts.
// 查询NAT设备是否支持UPnP协议，若支持返回一个使用UPnP协议的端口映射器
// 查询过程将以UDP广播的方式发送到整个局域网的设备上
func UPnP() Interface {
	return startautodisc("UPnP", discoverUPnP)
}

// PMP returns a port mapper that uses NAT-PMP. The provided gateway
// address should be the IP of your router. If the given gateway
// address is nil, PMP will attempt to auto-discover the router.
// 查询NAT设备是否支持PMP协议，若支持返回一个使用PMP协议的端口映射器
// 参数中提供的网关地址应该是局域网中的路由器的IP地址，如果给定的网关地址为零，PMP将尝试自动发现路由器。
func PMP(gateway net.IP) Interface {
	if gateway != nil {
		return &pmp{gw: gateway, c: natpmp.NewClient(gateway)} //有网关地址，直接向NAT网管发送请求
	}
	return startautodisc("NAT-PMP", discoverPMP) //无网关地址，需要通过广播方式进行查询
}

// autodisc represents a port mapping mechanism that is still being
// auto-discovered. Calls to the Interface methods on this type will
// wait until the discovery is done and then call the method on the
// discovered mechanism.
//
// This type is useful because discovery can take a while but we
// want return an Interface value from UPnP, PMP and Auto immediately.
type autodisc struct {
	what string           // type of interface being autodiscovered  需要被检测的协议类型
	once sync.Once        //sync.Once 也是 Go 官方的一并发辅助对象，它能够让函数方法只执行一次，达到类似 init 函数的效果
	doit func() Interface //在等待期间需要调用的用于进行协议检测的函数

	mu    sync.Mutex
	found Interface //存放搜索获得的端口映射器(采用UPnP协议或者NAT-PMP协议)
}

// 开启单独协程负责执行指定的NAT映射协议搜索函数
// 返回值为autodisc类型结构体，包含了搜索获得的端口映射器(采用UPnP协议或者NAT-PMP协议)，实现了Interface接口
func startautodisc(what string, doit func() Interface) Interface {
	// TODO: monitor network configuration and rerun doit when it changes.
	ad := &autodisc{what: what, doit: doit}
	// Start the auto discovery as early as possible so it is already
	// in progress when the rest of the stack calls the methods.
	go ad.wait() //由一个单独的协程负责执行协议搜索函数doit
	return ad
}

// 实现指定的端口映射
func (n *autodisc) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {

	//如果是直接采用natMap方法，无需调用n.wait()方法
	if n.what == "natMap" {
		return n.found.AddMapping(protocol, extport, intport, name, lifetime)
	}

	if err := n.wait(); err != nil {
		return err
	}
	return n.found.AddMapping(protocol, extport, intport, name, lifetime)
}

// 取消指定的端口映射
func (n *autodisc) DeleteMapping(protocol string, extport, intport int) error {

	//如果是直接采用natMap方法，无需调用n.wait()方法
	if n.what == "natMap" {
		return n.found.DeleteMapping(protocol, extport, intport)
	}

	if err := n.wait(); err != nil {
		return err
	}
	return n.found.DeleteMapping(protocol, extport, intport)
}

// 获取本主机在NAT设备上的对外映射IP地址
func (n *autodisc) ExternalIP() (net.IP, error) {

	//如果是直接采用natMap方法，无需调用n.wait()方法
	if n.what == "natMap" {
		return n.found.ExternalIP()
	}

	if err := n.wait(); err != nil {
		return nil, err
	}
	return n.found.ExternalIP()
}

// 以字符串形式返回使用的NAT映射协议(UPnP协议和PMP协议会返回不同的字符串)
func (n *autodisc) String() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.found == nil {
		return n.what
	} else {
		return n.found.String()
	}
}

// wait blocks until auto-discovery has been performed.
func (n *autodisc) wait() error {
	n.once.Do(func() { //本匿名函数将只执行一次
		n.mu.Lock()
		n.found = n.doit() //执行搜索函数
		n.mu.Unlock()
	})
	if n.found == nil {
		return fmt.Errorf("no %s router discovered", n.what)
	}
	return nil
}
