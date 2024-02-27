// maintain the bucket in kad
package discover

import (
	"encoding/binary"
	"math/rand"
	"net"
	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"sort"
	"sync"
	"time"
)

const (
	alpha      = 3  // Kademlia concurrency factor
	bucketSize = 16 // Kademlia bucket size
	hashBits   = len(common.Hash{}) * 8
	nBuckets   = hashBits + 1 // Number of buckets

	maxBondingPingPongs = 16
	maxFindnodeFailures = 5
)

type Table struct {
	mutex   sync.Mutex        // protects buckets, their content, and nursery
	buckets [nBuckets]*bucket // index of known nodes by distance
	nursery []*Node           // bootstrap nodes  引导节点
	db      *nodeDB           // database of known nodes

	bondmu    sync.Mutex
	bonding   map[NodeID]*bondproc
	bondslots chan struct{} // limits total number of active bonding processes

	nodeAddedHook func(*Node) // for testing

	net  transport
	self *Node // metadata of the local node
}

type bondproc struct {
	err  error
	n    *Node
	done chan struct{}
}

// transport is implemented by the UDP transport.
// it is an interface so we can test without opening lots of UDP
// sockets and without generating a private key.
type transport interface {
	ping(NodeID, *net.UDPAddr) error
	waitping(NodeID) error
	findnode(toid NodeID, addr *net.UDPAddr, target NodeID) ([]*Node, error)
	close()
}

// bucket contains nodes, ordered by their last activity.
// the entry that was most recently active is the last element
// in entries.
type bucket struct {
	lastLookup time.Time
	entries    []*Node //k-bucket 的真实队列  (最多放 16 个)  实时条目，按上次联系时间排序
}

// one node should maintain a table, thus the p2p network is kept.
// 初始化时创建一个Kad路由表
func newTable(t transport, ourID NodeID, ourAddr *net.UDPAddr, nodeDBPath string) *Table {
	// If no node database was given, use an in-memory one
	db, err := newNodeDB(nodeDBPath, Version, ourID) //打开leveldb数据库
	if err != nil {
		glog.V(logger.Warn).Infoln("Failed to open node database:", err)
		db, _ = newNodeDB("", Version, ourID)
	}

	tab := &Table{
		net:       t,
		db:        db,
		self:      newNode(ourID, ourAddr.IP, uint16(ourAddr.Port), uint16(ourAddr.Port)),
		bonding:   make(map[NodeID]*bondproc),
		bondslots: make(chan struct{}, maxBondingPingPongs),
	}

	for i := 0; i < cap(tab.bondslots); i++ {
		tab.bondslots <- struct{}{}
	}
	for i := range tab.buckets {
		tab.buckets[i] = new(bucket)
	}
	return tab
}

// Self returns the local node.
// The returned node should not be modified by the caller.
func (tab *Table) Self() *Node {
	return tab.self
}

// ReadRandomNodes fills the given slice with random nodes from the
// table. It will not write the same node more than once. The nodes in
// the slice are copies and can be modified by the caller.
func (tab *Table) ReadRandomNodes(buf []*Node) (n int) {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()
	// TODO: tree-based buckets would help here
	// Find all non-empty buckets and get a fresh slice of their entries.
	var buckets [][]*Node
	for _, b := range tab.buckets {
		if len(b.entries) > 0 {
			buckets = append(buckets, b.entries[:])
		}
	}
	if len(buckets) == 0 {
		return 0
	}
	// Shuffle the buckets.打乱整个buckets
	for i := uint32(len(buckets)) - 1; i > 0; i-- {
		j := randUint(i)
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}
	// Move head of each bucket into buf, removing buckets that become empty.
	var i, j int
	for ; i < len(buf); i, j = i+1, (j+1)%len(buckets) {
		b := buckets[j]
		buf[i] = &(*b[0])
		buckets[j] = b[1:]
		if len(b) == 1 {
			buckets = append(buckets[:j], buckets[j+1:]...)
		}
		if len(buckets) == 0 {
			break
		}
	}
	return i + 1
}

func randUint(max uint32) uint32 {
	if max == 0 {
		return 0
	}
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:]) % max
}

// Close terminates the network listener and flushes the node database.
func (tab *Table) Close() {
	tab.net.close()
	tab.db.close()
}

// Bootstrap sets the bootstrap nodes. These nodes are used to connect
// to the network if the table is empty. Bootstrap will also attempt to
// fill the table by performing random lookup operations on the
// network.
// 本方法用于设置引导节点，这些节点在本地table表为空时被用于连接到网络。
// 本方法会尝试在网络中执行随机的lookup操作来填充table表
func (tab *Table) Bootstrap(nodes []*Node) {
	tab.mutex.Lock()
	// TODO: maybe filter nodes with bad fields (nil, etc.) to avoid strange crashes
	tab.nursery = make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		cpy := *n
		cpy.sha = crypto.Sha3Hash(n.ID[:])
		tab.nursery = append(tab.nursery, &cpy)
	}
	tab.mutex.Unlock()
	tab.refresh()
}

// Lookup performs a network search for nodes close
// to the given target. It approaches the target by querying
// nodes that are closer to it on each iteration.
// The given target does not need to be an actual node
// identifier.

// 对给定的目标节点执行网络搜索。
// 它通过在每次迭代中查询离目标较近的节点来接近目标。给定的目标不需要是实际的节点标识符。
func (tab *Table) Lookup(targetID NodeID) []*Node {
	var (
		target         = crypto.Sha3Hash(targetID[:]) //对目标节点ID求哈希
		asked          = make(map[NodeID]bool)
		seen           = make(map[NodeID]bool)
		reply          = make(chan []*Node, alpha)
		pendingQueries = 0
	)
	// don't query further if we hit ourself.
	// unlikely to happen often in practice.
	asked[tab.self.ID] = true //表示自身是不需要被询问的(相当于一个已经被询问过的远端节点)

	tab.mutex.Lock()
	// update last lookup stamp (for refresh logic)
	tab.buckets[logdist(tab.self.sha, target)].lastLookup = time.Now() //更新本表与目标节点相同逻辑距离的那一行K-桶的最近一次更新时间
	// generate initial result set
	result := tab.closest(target, bucketSize) //返回路由表tab中距离target最近的bucketSize个节点
	tab.mutex.Unlock()

	// If the result set is empty, all nodes were dropped, refresh
	//result结果为空，说明路由表被删除，需要调用refresh刷新整个Kad路由表
	if len(result.entries) == 0 {
		tab.refresh()
		return nil
	}

	//result结果不为空
	for {
		// ask the alpha closest nodes that we haven't asked yet
		//遍历访问最接近目标节点的，同时也未被访问过的那些节点
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] { //确保该远端节点未被访问过
				asked[n.ID] = true
				pendingQueries++ //等待处理数+1
				go func() {
					// Find potential neighbors to bond with
					//r返回节点n的bucketSize个邻居节点
					r, err := tab.net.findnode(n.ID, n.addr(), targetID)
					if err != nil {
						// Bump the failure counter to detect and evacuate non-bonded entries
						fails := tab.db.findFails(n.ID) + 1
						tab.db.updateFindFails(n.ID, fails)
						glog.V(logger.Detail).Infof("Bumping failures for %x: %d", n.ID[:8], fails)

						if fails >= maxFindnodeFailures {
							glog.V(logger.Detail).Infof("Evacuating node %x: %d findnode failures", n.ID[:8], fails)
							tab.del(n)
						}
					}
					//与这些邻居节点建立bond，得到pong回复的传入reply管道
					reply <- tab.bondall(r)
				}()
			}
		}
		//只有当所有的远端节点都已处理完毕，才会退出循环
		if pendingQueries == 0 {
			// we have asked all closest nodes, stop the search
			break
		}
		// wait for the next reply
		//reply中可能存在重复的邻居节点，只将那些不重复的节点加入到result中
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, bucketSize) //result中存储着距离目标节点最近的bucketSize个不重复的邻居节点
			}

		}
		pendingQueries--
	}
	return result.entries //包含新的距离目标节点最近的 bucketSize 个节点
}

// refresh performs a lookup for a random target to keep buckets full, or seeds
// the table if it is empty (initial bootstrap or discarded faulty peers).
func (tab *Table) refresh() {
	seed := true

	// If the discovery table is empty, seed with previously known nodes
	//如果Kad路由表为空，使用以前已知的节点进行种子设定
	tab.mutex.Lock()
	for _, bucket := range tab.buckets {
		if len(bucket.entries) > 0 { //K-桶不为空
			seed = false
			break
		}
	}
	tab.mutex.Unlock()

	// If the table is not empty, try to refresh using the live entries
	//K-桶不为空，对K-桶进行刷新(最近联系的节点排到后面)
	if !seed {
		// The Kademlia paper specifies that the bucket refresh should
		// perform a refresh in the least recently used bucket. We cannot
		// adhere to this because the findnode target is a 512bit value
		// (not hash-sized) and it is not easily possible to generate a
		// sha3 preimage that falls into a chosen bucket.
		//
		// We perform a lookup with a random target instead.

		//生成一个随机的NodeID
		var target NodeID
		rand.Read(target[:])

		//使用Lookup方法对该随机的NodeID在网络中进行查询
		//它通过在每次迭代中查询离目标较近的节点来接近目标。给定的目标不需要是实际的节点标识符。
		result := tab.Lookup(target)
		if len(result) == 0 {
			// Lookup failed, seed after all
			seed = true
		}
	}
	//整个路由表为空
	if seed {
		// Pick a batch of previously know seeds to lookup with
		//选择一批以前已知的种子进行查找
		seeds := tab.db.querySeeds(10) //获取10个种子节点
		for _, seed := range seeds {
			glog.V(logger.Debug).Infoln("Seeding network with", seed)
		}
		nodes := append(tab.nursery, seeds...) //将这些种子节点追加到路由表tab的nursery切片中，然后再赋值给nodes

		// Bond with all the seed nodes (will pingpong only if failed recently)
		bonded := tab.bondall(nodes)
		if len(bonded) > 0 {
			tab.Lookup(tab.self.ID)
		}
		// TODO: the Kademlia paper says that we're supposed to perform
		// random lookups in all buckets further away than our closest neighbor.
	}
}

// nodesByDistance is a list of nodes, ordered by
// distance to target.
type nodesByDistance struct {
	entries []*Node
	target  common.Hash
}

// push adds the given node to the list, keeping the total size below maxElems.
// 将节点n插入到h的entries数组中，同时保证entries数组存储的距离h.target最近的maxElems个节点
func (h *nodesByDistance) push(n *Node, maxElems int) {

	//index := sort.Search(n int,f func(i int) bool) int
	//该函数使用二分查找的方法，会从[0, n)中取出一个值index，index为[0, n)中最小的使函数f(index)为True的值，并且f(index+1)也为True。
	//如果无法找到该index值，则该方法为返回n。

	//找出第一个比起新的节点n距离target更远的节点h.entries[i]的下标i(如果没找到则返回len(h.entries) --> 这是一个不断增长的值)
	ix := sort.Search(len(h.entries), func(i int) bool {
		return distcmp(h.target, h.entries[i].sha, n.sha) > 0
	})
	if len(h.entries) < maxElems { //只要h.entries未满maxElems，就将节点N追加到之上
		h.entries = append(h.entries, n)
	}
	//下标为len(h.entries)，说明未找到。也就是说传入的Node n相比之前距离目标节点更远
	if ix == len(h.entries) {
		// farther away than all nodes we already have.
		// if there was room for it, the node is now the last element.

		//h.entries[ix]是符合条件的第一个更远节点，因此需要将该节点以及之后的节点后移一位，插入新的距离目标最近的节点n
	} else {
		// slide existing entries down to make room
		// this will overwrite the entry we just appended.
		copy(h.entries[ix+1:], h.entries[ix:])
		h.entries[ix] = n
	}
}

// given id. The caller must hold tab.mutex.
// 查找Kad路由表中与目标节点ID的哈希最为接近的其他节点
func (tab *Table) closest(target common.Hash, nresults int) *nodesByDistance {
	// This is a very wasteful way to find the closest nodes but
	// obviously correct. I believe that tree-based buckets would make
	// this easier to implement efficiently.
	// TODO: Tree-based buckets
	close := &nodesByDistance{target: target}
	for _, b := range tab.buckets { //从路由表第一行到最后一行K-桶依次查询
		for _, n := range b.entries { //对每一行k-桶进行查询(n为Node类型变量，包含IP地址，端口号，NodeID四项数据)
			close.push(n, nresults) //调用push方法，使得close.entries[]中存储距离target最近的nresults个节点
		}
	}
	return close //返回这nresults个节点
}

// 计算整个Kad路由表保存的远端节点的数目
func (tab *Table) len() (n int) {
	for _, b := range tab.buckets {
		n += len(b.entries)
	}
	return n
}

// bondall bonds with all given nodes concurrently and returns
// those nodes for which bonding has probably succeeded.
// 多协程调用bond方法
func (tab *Table) bondall(nodes []*Node) (result []*Node) {
	rc := make(chan *Node, len(nodes))
	//遍历所有需要处理的节点，分别开启单独的协程负责让该节点与这些节点进行bond
	for i := range nodes {
		go func(n *Node) {
			nn, _ := tab.bond(false, n.ID, n.addr(), uint16(n.TCP)) //nn会返回那些成功建立bond的远端节点
			rc <- nn
		}(nodes[i])
	}
	for range nodes {
		if n := <-rc; n != nil {
			result = append(result, n)
		}
	}
	return result //返回与本节点成功建立bond的所有远端节点
}

// bond ensures the local node has a bond with the given remote node.
// It also attempts to insert the node into the table if bonding succeeds.
// The caller must not hold tab.mutex.
//bond方法会确保本地节点与给定的远端节点绑定。
//如果绑定成功还会将远端节点插入到本tab表中，但是调用者必须持有 tab.mutex
//
// A bond is must be established before sending findnode requests.
// Both sides must have completed a ping/pong exchange for a bond to
// exist. The total number of active bonding processes is limited in
// order to restrain network use.
//
//在本地节点向其他节点发送findnode请求之前必须先建立bond
//双方必须完成ping/pong交换才能建立bond
//为了限制网络的使用，活跃的bonding处理的数目是被限制的

// bond is meant to operate idempotently in that bonding with a remote
// node which still remembers a previously established bond will work.
// The remote node will simply not send a ping back, causing waitping
// to time out.
// bond意味着在与远程节点的绑定中以幂等方式操作，远程节点仍能记住先前建立的bond将起作用。
// 远程节点不会对ping进行回复时会导致ping超时
//
// If pinged is true, the remote node has just pinged us and one half
// of the process can be skipped.
// 如果pinged是true。 那么远端节点已经给我们发送了ping消息。这样一半的流程可以跳过。
func (tab *Table) bond(pinged bool, id NodeID, addr *net.UDPAddr, tcpPort uint16) (*Node, error) {
	// Retrieve a previously known node and any recent findnode failures
	node, fails := tab.db.node(id), 0 //在数据库中检索节点的id
	if node != nil {
		fails = tab.db.findFails(id)
	}
	// If the node is unknown (non-bonded) or failed (remotely unknown), bond from scratch
	var result error
	if node == nil || fails > 0 { //没有检索到，或者查询出错
		glog.V(logger.Detail).Infof("Bonding %x: known=%v, fails=%v", id[:8], node != nil, fails)

		tab.bondmu.Lock()
		w := tab.bonding[id] //w是map的value
		if w != nil {
			// Wait for an existing bonding process to complete.
			//阻塞等待一个现存的bonding处理结束
			tab.bondmu.Unlock()
			<-w.done
		} else {
			// Register a new bonding process.
			w = &bondproc{done: make(chan struct{})}
			tab.bonding[id] = w
			tab.bondmu.Unlock()
			// Do the ping/pong. The result goes into w.
			//利用新建的bond，向目标id的节点发送ping。然后等待对方的pong。成功收到pong后将对方节点信息存入w
			tab.pingpong(w, pinged, id, addr, tcpPort)

			// Unregister the process after it's done.
			//bond建立完成，目标源端节点的信息也已经存入本地。可以将该id的节点信息从bonding中删除(已经是bonded了)
			tab.bondmu.Lock()
			delete(tab.bonding, id)
			tab.bondmu.Unlock()
		}
		// Retrieve the bonding results
		result = w.err
		if result == nil {
			node = w.n //如果整个过程没有error，将获取目标id的节点的Node结构体，并作为返回值
		}
	}
	// Even if bonding temporarily failed, give the node a chance
	//在本地数据库中检索到对于该id的节点
	if node != nil {
		node.Sha() //获取节点id对于的哈希值
		tab.mutex.Lock()
		defer tab.mutex.Unlock()

		b := tab.buckets[logdist(tab.self.sha, node.sha)] //计算本节点与目标id节点的逻辑距离，并存入相应的K-桶中
		if !b.bump(node) {                                //本行K-桶中不存在node节点
			tab.pingreplace(node, b) //更新b对应的一行K-桶，根据情况决定是否插入node
		}
		tab.db.updateFindFails(id, 0)
	}
	return node, result
}

func (tab *Table) pingpong(w *bondproc, pinged bool, id NodeID, addr *net.UDPAddr, tcpPort uint16) {
	// Request a bonding slot to limit network usage
	<-tab.bondslots
	defer func() { tab.bondslots <- struct{}{} }()

	// Ping the remote side and wait for a pong
	//ping一个远程节点，并等待一个pong消息
	if w.err = tab.ping(id, addr); w.err != nil {
		close(w.done) //pingpong失败，return返回；否则继续执行
		return
	}
	if !pinged {
		// Give the remote node a chance to ping us before we start
		// sending findnode requests. If they still remember us,
		// waitping will simply time out.
		//若尚未收到过对方的ping消息，调用waitping()等待该节点的ping消息。如果已收到目标节点id的ping消息。waitping()就会退出
		tab.net.waitping(id)
	}
	// Bonding succeeded, update the node database
	//bond建立成功，将该目标节点的信息更新入本地数据库
	w.n = newNode(id, addr.IP, uint16(addr.Port), tcpPort)
	tab.db.updateNode(w.n)
	close(w.done)
}

// 更新b对应的一行K-桶，确定是否将new对应的远端节点插入该行K-桶中
func (tab *Table) pingreplace(new *Node, b *bucket) {
	if len(b.entries) == bucketSize { //本行K-桶已满
		oldest := b.entries[bucketSize-1] //获取队尾的节点(最早的远端节点)

		//向这个队尾节点发送ping消息，如果成功收到pong回复则不做任何操作，而是return结束本函数
		if err := tab.ping(oldest.ID, oldest.addr()); err == nil {
			// The node responded, we don't need to replace it.
			return
		}
	} else { //本行K-桶未满
		// Add a slot at the end so the last entry doesn't
		// fall off when adding the new node.
		b.entries = append(b.entries, nil)
	}
	//将本行K-桶节点依次往后移动，在队首插入新的远端节点new (两种可能：①K-桶未满 ②没有收到pong回复)
	copy(b.entries[1:], b.entries)
	b.entries[0] = new

	if tab.nodeAddedHook != nil {
		tab.nodeAddedHook(new)
	}
}

// ping a remote endpoint and wait for a reply, also updating the node database
// accordingly.
// 向源端节点发送ping消息并等待回复，同时更新本节点的数据库
func (tab *Table) ping(id NodeID, addr *net.UDPAddr) error {
	// Update the last ping and send the message
	tab.db.updateLastPing(id, time.Now())
	if err := tab.net.ping(id, addr); err != nil {
		return err
	}
	// Pong received, update the database and return
	tab.db.updateLastPong(id, time.Now())
	tab.db.ensureExpirer()

	return nil
}

// add puts the entries into the table if their corresponding
// bucket is not full. The caller must hold tab.mutex.
// add方法负责将entries队列中的节点放入Kad路由表相应的K-桶中(若该K-桶未满)
// 调用add者必须持有tab.mutex
func (tab *Table) add(entries []*Node) {
outer:
	for _, n := range entries {
		if n.ID == tab.self.ID {
			// don't add self. 不需要添加自身进入K-桶
			continue
		}
		//将entries切片中每一个节点元素按照与本节点的逻辑距离填充到相应行的K-桶中
		bucket := tab.buckets[logdist(tab.self.sha, n.sha)] //这是一个地址

		for i := range bucket.entries {
			if bucket.entries[i].ID == n.ID {
				// already in bucket
				continue outer
			}
		}
		if len(bucket.entries) < bucketSize {
			bucket.entries = append(bucket.entries, n) //在本行K-桶未满的情况下才会填充
			if tab.nodeAddedHook != nil {
				tab.nodeAddedHook(n)
			}
		}
	}
}

// del removes an entry from the node table (used to evacuate failed/non-bonded
// discovery peers).
// del方法负责将指定entry中的节点从Kad的路由表中删除(用于撤销错误的/无绑定的发现节点)
func (tab *Table) del(node *Node) {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	//获取对应行的K-桶
	bucket := tab.buckets[logdist(tab.self.sha, node.sha)] //这是一个地址
	for i := range bucket.entries {
		if bucket.entries[i].ID == node.ID {
			bucket.entries = append(bucket.entries[:i], bucket.entries[i+1:]...) //唯独没有包含第i的节点
			return
		}
	}
}

// 如果目标id的节点存在于该行K-桶中，将该节点移动到该行K-桶的最前方
// 移动一次就返回true;否则返回false
func (b *bucket) bump(n *Node) bool {
	//遍历本行的K-桶
	for i := range b.entries {
		//如果该id的节点已经存在于改行K-桶中，就将该id的节点移动到该行K-桶的最前方
		if b.entries[i].ID == n.ID {
			// move it to the front
			copy(b.entries[1:], b.entries[:i])
			b.entries[0] = n
			return true
		}
	}
	return false
}
