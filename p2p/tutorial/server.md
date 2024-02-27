# server.go
## Server{} struct
Server结构体是用于表征服务器的，所有的连接都有Server结构体来维护。Server结构体的属性包括，PrivateKey：本节点的私钥；MaxPeers：维护的最大邻居数；MaxPendingPeers：维护的最大的在排队进行握手（handshake）时的连接数，默认设置为0；Discovery：为一标志符。Name：为服务器设置一个名称。BootstrapNodes：用以辅助节点跟网络中的其他节点建立连接。StaticNodes：用以预定义一些用以建立稳定连接与重连接的节点。TrustedNodes：类似于StaticNodes，但是是可以一定建立稳定连接的节点。NodeDatabase：建立节点数据库的路径。Protocols：为一列表，包含了服务器支持的所有协议。ListenAddr：监听地址。NAT：nat接口，用于实现地址转换。Dialer：用于对没有绑定的连接进行呼叫。NoDial：标识符，如果为真，服务器不会呼叫任何节点。lock、running：辅助并标志服务器启动。ntab：一个discoverTable；listener：一个监听类；ourHandshake：用以握手协议；lastLookup：上一次检查的时间。peerOp、peerOpDone：作为peer时调用服务器。quit，addstatic，posthandshake，addpeer，delpeer，loopWG：字面含义。

## conn{} struct
conn重载了network connection，以及收集在handshakes接收的相关信息。

## transport{} interface
一个transport接口，定义了两个用于二阶段握手的函数doEncHandshake以及doProtoHandshake。MsgReadWriter对通信缓存进行读写。

## conn.String() func
将一条连接打包成String形式。

## conn.is() func
返回连接是否建立（不管它是哪一种连接）。

## Server.Peers() func
返回服务器建立了连接的所有peer。

## Server.AddPeer() func
AddPeer连接至给定的节点然后维护该连接直至服务器终止。如果连接失败，服务器会尝试重新跟该peer建立连接。

## Server.Self() func
返回本地节点的endpoint信息（discover.Node）。

## Server.Stop() func
将服务器关闭，相应的所有连接也被关掉。

## Server.Start() func
启动服务器。

## Server.startListening() func
启动TCP监听。

## dialer{} interface
一个呼叫器（dialer）接口，共有三个函数，newTasks：建立新的任务；taskDone：结束一个task；addStatic：加入一个稳定的discover.Node。

## Server.run() func
指定一个dialer，运性服务器。服务器循环的询问dialer中是否有新的任务然后执行他们。

## Server.protoHandshakeChecks() func
对握手协议进行检测。首先会丢弃没有matching的连接，然后以再调用encHandshakeChecks()函数。

## Server.encHandshakeChecks() func
对握手协议进行检测。

## Server.listenLoop() func
该函数需要运行在一个单独的goroutine里，用以监听并接受已经绑定的连接的信息。

## Server.setupConn() func
执行握手协议并尝试建立连接，当一个新的连接建立或者握手协议失败时返回。该函数是实现Discovery协议的关键函数。

## Server.checkpoint() func
一个检查函数，接受一个conn对握手协议进行检测。`将一个连接addpeer`。

## Server.runPeer() func
runPeer需要单独运行在一个goroutine中，运行peer.run()然后删除peer。





