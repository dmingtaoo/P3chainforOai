# node.go
## Node{} struct
P2P网络中的最小元节点。
NodeID是公钥，共512比特，64字节。
sha common.Hash是用来方便测试的，指定一个节点的相对距离。

## newNode() func
Node的构造函数，默认将使用ipv4。

## Node.addr() func
根据Node信息返回一UDP地址，包括IP和端口号。

## Node.String() func
将Node的信息以url形式打包，默认的地址以TCP的IP地址与端口号表示，当发现Node的UDP端口与TCP端口不同时，指定discport（也就是discovery协议的端口号）为该UDP端口号。最终返回string类型。

## ParseNode() func
将一string类型的url重新解析排列生成Node，最终返回一Node类型。该url的scheme应该是dpnode（p3Chain node），故需要检查该url的构造形式是否符合dpnode。在构建Node前，需要依次检查：
* Scheme是否为“dpnode”。
* NodeID是否正确，不为空且能够通过HexID函数转换为64字节的NodeID。
* IP是否正确，能否转换为IPv4。
* 检查TCP端口号是否正确，将TCP端口号转换为16进制。
* 默认设置UDP端口号与TCP端口号相同，如果发现url有“discport”这一项则为UDP设置不同端口号。
最终调用newNode函数构造并返回Node

## MustParseNode() func
调用ParseNode函数返回Node，但是当不成功时直接引发中断。

## HexID() func
将string类型的十六进制信息（可能包含前缀0x）转换为NodeID。

## MustHexID() func
 调用HexID函数返回NodeID，但是当不成功时直接引发中断。

## PubkeyID() func
通过椭圆曲线加密算法的公钥生成NodeID。

## Node.Pubkey() func
通过Node的NodeID还原公钥，同时会检测该公钥对应的点在不在S256这一椭圆曲线上。

## recoverNodeID() func
根据所给的待签名信息与签名信息反推回相应的公钥，也就得到了需要的NodeID。

## distcmp() func 
简单比较两节点相对于一target节点（在函数中也为一NodeID）的距离远近。

## logdist() func 
Kad协议的重点，通过异或操作计算节点间的逻辑距离。通过对两节点的NodeID进行异或操作即可得到一逻辑距离值，返回该逻辑距离。两个节点的逻辑距离由进行XOR操作后的前置0个数表示。

## hashAtDistance() func
返回一个在与目标节点具有指定距离的哈希值。

# database.go
## nodeDB{} struct
每个节点创建一个nodeDB类存放所有已知的节点信息。包括一个levelDB数据库lvl，seeder是一数据库迭代器，self使用NodeID指代。runner为一次性锁，保证一个nodeDB之启动一个expirer进程。quit为一通道，用以通知expirer进程中断。

## newNodeDB() func
返回一个nodeDB，如果未指定存储路径则调用newMemoryNodeDB函数构建内存数据库，否则使用相关路径与版本号调用newPersistentNodeDB函数构建数据库。

## newPersistentNodeDB() func
使用所给的路径地址与版本号创建或打开一个levelDB数据库，并且初始化并更新其中的self信息。如果所给版本号更改，删除旧有数据库重新构造。

## newMemoryNodeDB() func
创建一个内存数据库

## makeKey() func
返回一个用于存储在levelDB的blob，在所给field前添加nodeDB前置符"n:"以及NodeID。

## splitKey() func
将一个key值拆分成NodeID和field。

## nodeDB.fetchInt64() func
根据所给key值从nodeDB中取一个整数，以int64存储。

## nodeDB.storeInt64() func 
将一个in64的数存入nodeDB中，一般是unix时间戳。

## nodeDB.node() func
根据所给NodeID值从一个数据库中取出node节点信息。

## nodeDB.updateNode() func
使用rlp库将node序列化，并存入nodeDB中，存入的放式是overwriting。

## nodeDB.deletNode() func
根据所给node节点的NodeID值，将nodeDB中的所有相关信息删除。

## nodeDB.ensureExpirer() func
确保expirer运行，如果已经运行会马上返回。

## nodeDB.expirer() func
启动expirer，用于循环调用expireNodes函数更新删除nodeDB中陈旧的节点信息。

## nodeDB.expireNodes() func
删除nodeDB中新鲜度超过阈值的节点信息。

## nodeDB.lastPing() func
根据所给NodeID值调用fetchInt64函数取回一节点上一次Ping的时间戳。

## nodeDB.lastPong() func
根据所给NodeID值调用fetchInt64函数取回一节点上一次Pong的时间戳。

## nodeDB.updateLastPing() func
根据所给NodeID调用makeKey与storeInt64函数实现节点上一次Ping信息的更新。

## nodeDB.updateLastPong() func
根据所给NodeID调用makeKey与storeInt64函数实现节点上一次Pong信息的更新。

## nodeDB.findFails() func
调用fetchInt64函数取出失败连接的信息。

## nodeDB.updateFindFails() func
根据所给NodeID调用makeKey与storeInt64函数实现对节点Fails信息的更新。

## nodeDB.querySeeds() func
从nodeDB数据库中取出一系列的节点作为种子节点用于辅助将节点一步步加入到网络中去。理想情况下我们会希望使用最新见到的节点作为种子节点，但是由于levelDB只支持迭代形式的查询，在这个地方使用nodeDB中标记为discover的节点（这些节点还从来没有被ping过）代替。

## nodeDB.close() func
终止nodeDB，分别检查迭代器、expirer、levelDB关闭与否。

# table.go 
## Table{} struct
Table结构体主要实现Kad协议中的K桶部分。mutex为同步锁，防止Table被多个进程同时写。buckets就是kad协议里存放不同距离的节点的多个K桶（Table中一共保存了512+1个K桶）。nursery是用于构建初期K桶的。bond都是一些用于建立通信连接的。

## bondproc{} struct
用于绑定一个正在连接的节点。

## transport{} interface
transport将会在udp.go中重载，对应了P2P网络构建需要用到的几个命令，包括ping，waitping，findnode。

## bucket{} struct
一个K桶，保存的是一定数量的与本节点一定距离的节点。lastLookup是用于表征上一次该K桶被查询的时间，代表了K桶中节点的新鲜度。

## newTable（） func
Table结构体的构造函数，需要输入transport，本节点的节点ID值，本地地址，本地nodeDB的地址用于打开或建立数据库（如果地址为空则建立内存数据库）。

## Table.Self() func
返回Table对应的本地节点。

## Table.ReadRandomNodes() func
从Table中随机读取一些节点信息返回。

## randUint() func
随机返回一个不大于所给数的uint32的数。

## Table.Close() func
关闭Table的网络连接与数据库。

## Table.Bootstrap() func
将所给的复数个节点添加进Table.nursery以便辅助节点一步步加入到网络中。

## Table.Lookup() func
根据所给目标节点ID（target）进行查询，Table通过向自身计算所得逻辑距离最小的节点查询，多跳返回。这里的实现是支持并发的。

## Table.refresh() func
通过查询一个随机生成的target值刷新Table中的K桶。如果K桶中没有节点，则使用seed节点引导进行刷新。

## nodeByDistance{} struct
辅助节点查询的结构体，保存的是目标值以及与目标值相近的节点信息。

## nodeByDistance.push() func
如果节点将指定的节点插入到nodeByDistance中（有顺序的），同时维持长度不超过最大值。

## Table.closest() func
从K桶中返回最接近target的几个节点。通过建立一个nodeByDistance实现，多次调用push函数。

## Table.len() func
返回所用K桶保存的节点信息总数。

## Table.bondall() func
输入一系列节点，与这些节点都建立连接。

## Table.bond() func
与远程的节点建立连接。

## Table.pingpong() func
跟一个给定节点打乒乓。

## Table.pingreplace() func
通过ping将一个有效节点的信息更新到指定的K桶中。

## Table.ping() func
ping一个远程节点，同时会更新nodeDB。

## Table.add() func
将一系列节点加入到Table对应的bucket中。

## Table.del() func 
将指定节点从Table中去除。

## bucket.bump() func
将一个节点从bucket中提前到最前面。

# udp.go
## ping{} struct
一个ping包，用以代表ping。

## pong{} struct
一个pong包，用以代表pong。

## findnode{} struct
一个findnode包，用以代表findnode指令。

## neighbors{} struct
用以回应findnode。

## rpcNode{} struct
用以表征一个远程节点。

## rpcEndpoint{} struct
用于RPC调用中指代节点，相较于rpcNode缺少NodeID这一属性。

## makeEndpoint() func
根据所给UDP信息、TCPport构造一个rpcEndpoint并返回。

## nodeFromRPC() func
根据所给rpcNode信息构造返回Node节点信息。

## nodeToRPC() func
根据所给节点信息返回rpcNode。

## packet{} interface
ping、pong、findnode、neighbors皆为该包接口，报应该实现函数handle用以对接受的包进行相应处理。

## conn{} interface
一个连接接口，用以表征一个连接。需要满足四个函数，ReadFromUDP从缓存中读取相应的信息，WriteToUDP在相应缓存写信息并等待发送，Close是关闭该连接，LocalAddr返回连接的相关信息。

## udp{} struct
建立一个UDP连接，该连接应该包括conn，节点的私钥，本节点的rpcEndpoint，addpending与gotreply都是用于排队技术控制并发的。closing是用于控制连接关闭的信号，nat是用于进行nat穿透相关的，同时绑定了一个匿名的Table结构体。

## pending{} struct
用以表征等待执行操作的结构体，注意其将回调函数存入其中。

## reply{} struct
用以存储待发送的reply。

## newUDP() func
根据所给信息构建一个UDP连接，并返回相应的UDP与Table。

## ListenUDP() func
根据所给信息实载一个UDP连接，返回的是一个table。

## udp.close() func
关闭一个UDP连接。

## udp.ping() func
调用UDP连接进行ping。

## udp.waitping() func
等待ping回复返回。

## udp.findnode() func
调用UDP连接发送一个findnode包。

## udp.pending() func
根据所给信息进行pending操作,将pending加入到udp.addpending中。

## udp.handleReply() func
对reply进行处理。

## udp.loop() func
一个独立的goroutine，循环检验各种通道的信息是否进入，并进行相应操作。

## init() func
主要是为maxNeighbors常量进行赋值，当从邻居处收到的包总大小不超过1280bytes时，此时的邻居数量即为maxNeighbors。

## udp.send() func
根据所给信息调用udp连接发送请求信息包。

## encodePacket() func
对请求包信息进行encode，同时会使用私钥进行签名。

## udp.readLoop() func
一个独立的goroutine，循环检测接收到的UDP报文并读取,调用udp.handlePacket函数对不同的报文包进行相应处理。

## udp.handlePacket() func
对包进行解码，并根据不同的包调用packet.handle函数进行处理。

## decodePacket() func
对报文包进行解码，返回packet，NodeID以及报文哈希。

## ping.handle() func
对ping包的相应处理。

## pong.handle() func
对pong包的相应处理。

## findnode.handle() func
对findnode包的相应处理。

## neighbors.handle() func 
对neighbors包的相应处理。








