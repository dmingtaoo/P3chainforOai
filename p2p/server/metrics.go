// Contains the meters and timers used by the networking layer.

package server

import (
	"net"

	"p3Chain/metrics"
)

var (
	ingressConnectMeter = metrics.NewMeter("p2p/InboundConnects")
	ingressTrafficMeter = metrics.NewMeter("p2p/InboundTraffic")
	egressConnectMeter  = metrics.NewMeter("p2p/OutboundConnects")
	egressTrafficMeter  = metrics.NewMeter("p2p/OutboundTraffic")
)

// meteredConn is a wrapper around a network TCP connection that meters both the
// inbound and outbound network traffic.
// meteredConn是网络TCP连接的包装器，用于测量入站和出站网络流量。
type meteredConn struct {
	*net.TCPConn // Network connection to wrap with metering
}

// newMeteredConn creates a new metered connection, also bumping the ingress or
// egress connection meter.
// 指示一个conn socket 的流量是 ingress(入) 还是 egress(出) ，同时对流量进行计量
func newMeteredConn(conn net.Conn, ingress bool) net.Conn {
	if ingress { //入站流量
		ingressConnectMeter.Mark(1)
	} else { //出站流量
		egressConnectMeter.Mark(1)
	}
	return &meteredConn{conn.(*net.TCPConn)}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.TCPConn.Read(b)         //从TCP socket读取
	ingressTrafficMeter.Mark(int64(n)) //记录读取字节数(入站流量记录)
	return
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.TCPConn.Write(b)       //用TCP socket发送
	egressTrafficMeter.Mark(int64(n)) //记录发送字节数(出站流量记录)
	return
}
