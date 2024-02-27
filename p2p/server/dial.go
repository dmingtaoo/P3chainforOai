package server

import (
	"container/heap"
	"crypto/rand"
	"fmt"
	"net"
	"time"

	loglogrus "p3Chain/log_logrus"
	"p3Chain/p2p/discover"
)

const (
	// This is the amount of time spent waiting in between
	// redialing a certain node.
	dialHistoryExpiration = 30 * time.Second //与节点的重连等待时间

	// Discovery lookups are throttled and can only run
	// once every few seconds.
	lookupInterval = 4 * time.Second //lookup进行的时间间隔
)

// dialstate schedules dials and discovery lookups.
// it get's a chance to compute new tasks on every iteration
// of the main loop in Server.run.
type dialstate struct {
	maxDynDials int           //当前节点最大连接数
	ntab        discoverTable //Kad路由表相关的功能接口

	lookupRunning bool //是否已调用lookup填充lookupBuf
	bootstrapped  bool //是否已调用bootstrap更新kad路由表

	dialing     map[discover.NodeID]connFlag       //等待连接队列(动态连接、静态连接、可信连接、入站连接)
	lookupBuf   []*discover.Node                   // current discovery lookup results   Lookup获取的结果
	randomNodes []*discover.Node                   // filled from Table		ReadRandomNodes获取的结果
	static      map[discover.NodeID]*discover.Node //保存所有需要静态连接节点的信息
	hist        *dialHistory                       //保存与所有节点的连接记录(对方节点ID、连接的时间点)
}

type discoverTable interface {
	Self() *discover.Node                           //在本地的Kad路由表中查询自身的Node信息
	Close()                                         //关闭Kad路由表和leveldb数据库
	Bootstrap([]*discover.Node)                     //设置引导节点，利用这些引导节点更新Kad路由表
	Lookup(target discover.NodeID) []*discover.Node //对给定的目标节点执行网络搜索。返回距离目标节点最近的bucketSize 个节点
	ReadRandomNodes([]*discover.Node) int           //返回Kad路由表中的随机的若干节点
}

// the dial history remembers recent dials.
type dialHistory []pastDial

// pastDial is an entry in the dial history.
type pastDial struct {
	id  discover.NodeID
	exp time.Time
}

// 一个接口，不同结构体对其的实现会使执行不同的任务
type task interface {
	Do(*Server)
}

// A dialTask is generated for each node that is dialed.
// 第一种类型的任务，为每一个需要进行连接的节点生成一个dialTask结构体
type dialTask struct {
	flags connFlag
	dest  *discover.Node
}

// discoverTask runs discovery table operations.
// Only one discoverTask is active at any time.
//
// If bootstrap is true, the task runs Table.Bootstrap,
// otherwise it performs a random lookup and leaves the
// results in the task.
// 第二种类型的任务，discoverTask负责运行Kad路由表的具体操作，同一时刻只有一个discoverTask处于活跃状态
type discoverTask struct {
	bootstrap bool //为true时，当前节点执行Table.Bootstrap;为false时，执行Table.Lookup , 此时results就是Lookup的返回值
	results   []*discover.Node
}

// A waitExpireTask is generated if there are no other tasks
// to keep the loop in Server.run ticking.
// 第三种类型的任务
type waitExpireTask struct {
	time.Duration
}

// 根据 需要静态连接的连接列表 , 本地保存的Kad路由表 初始化一个dialstate对象
func newDialState(static []*discover.Node, ntab discoverTable, maxdyn int) *dialstate {
	s := &dialstate{
		maxDynDials: maxdyn,
		ntab:        ntab,
		static:      make(map[discover.NodeID]*discover.Node),
		dialing:     make(map[discover.NodeID]connFlag),
		randomNodes: make([]*discover.Node, maxdyn/2),
		hist:        new(dialHistory),
	}
	for _, n := range static { //根据所有需静态连接peer的信息填充static字段
		s.static[n.ID] = n
	}
	return s
}

// 为dialstate对象(连接管理器)添加一个新静态连接节点信息
func (s *dialstate) addStatic(n *discover.Node) {
	s.static[n.ID] = n
}

// 从dialstate对象(连接管理器)删除一个静态连接节点信息
func (s *dialstate) delStatic(n *discover.Node) {
	delete(s.static, n.ID)
}

// 返回一个任务队列，需要调用server模块完成这些任务(有dialTask任务、discoverTask任务、waitExpireTask任务)
func (s *dialstate) newTasks(nRunning int, peers map[discover.NodeID]*Peer, now time.Time) []task {
	var newtasks []task //一个任务队列，存储所有当前节点需要执行的任务
	//负责为节点n添加连接标志位(确定需要实现怎样的连接)
	addDial := func(flag connFlag, n *discover.Node) bool {
		_, dialing := s.dialing[n.ID] //查看指定节点n是否存在于dialing集合中(位于等待连接队列)
		if dialing || peers[n.ID] != nil || s.hist.contains(n.ID) {
			return false
		}
		s.dialing[n.ID] = flag
		newtasks = append(newtasks, &dialTask{flags: flag, dest: n}) //任务队列添加新任务，完成本节点与这些节点的连接
		return true
	}
	//根据已连接节点的身份调用addDial，添加到s.dialing的同时为其添加dialTask任务

	// Compute number of dynamic dials necessary at this point.
	needDynDials := s.maxDynDials //必要的动态连接节点数 == 系统最大动态连接数
	for _, p := range peers {     //遍历所有新连接节点的连接标志位，如果是动态连接needDynDials--
		if p.rw.is(dynDialedConn) {
			needDynDials--
		}
	}
	for _, flag := range s.dialing { //遍历所有已连接节点连接标志位，如果是动态连接needDynDials--
		if flag&dynDialedConn != 0 {
			needDynDials--
		}
	}

	// Expire the dial history on every invocation.
	//对hist连接历史记录队列进行更新，删除所有now时刻之前的历史记录
	s.hist.expire(now)

	// Create dials for static nodes if they are not connected.
	//为所有 static 节点添加静态连接标志位，同时为其添加任务
	for _, n := range s.static {
		addDial(staticDialedConn, n)
	}

	// Use random nodes from the table for half of the necessary
	// dynamic dials.
	randomCandidates := needDynDials / 2 //必要的动态连接节点数其一半来自于Kad路由表随机获取的节点
	if randomCandidates > 0 && s.bootstrapped {
		n := s.ntab.ReadRandomNodes(s.randomNodes)       //随机获取本地Kad路由表中若干节点
		for i := 0; i < randomCandidates && i < n; i++ { //将这些节点作为动态连接节点
			if addDial(dynDialedConn, s.randomNodes[i]) {
				needDynDials--
			}
		}
	}
	// Create dynamic dials from random lookup results, removing tried
	// items from the result buffer.
	// 如果 needDynDials 还未归零(动态节点数不够)，将通过lookup获取的节点作为动态连接节点
	i := 0
	for ; i < len(s.lookupBuf) && needDynDials > 0; i++ {
		if addDial(dynDialedConn, s.lookupBuf[i]) {
			needDynDials--
		}
	}
	s.lookupBuf = s.lookupBuf[:copy(s.lookupBuf, s.lookupBuf[i:])] //从lookupBuf中删除上述已经设置的节点
	// Launch a discovery lookup if more candidates are needed. The
	// first discoverTask bootstraps the table and won't return any
	// results.
	// lookupBuf 未能成功填满 needDynDials 同时发现并未执行lookup操作填充 lookupBuf
	// lookupRunning = true 将启动lookup操作
	if len(s.lookupBuf) < needDynDials && !s.lookupRunning {
		s.lookupRunning = true
		newtasks = append(newtasks, &discoverTask{bootstrap: !s.bootstrapped}) //表示需要执行lookup操作(任务)
	}

	// Launch a timer to wait for the next node to expire if all
	// candidates have been tried and no task is currently active.
	// This should prevent cases where the dialer logic is not ticked
	// because there are no pending events.
	// nRunning == 0 表示节点整体没有任何task;len(newtasks) == 0 不需要进行lookup任务和节点的连接任务
	if nRunning == 0 && len(newtasks) == 0 && s.hist.Len() > 0 {
		t := &waitExpireTask{s.hist.min().exp.Sub(now)}
		newtasks = append(newtasks, t) //任务队列添加一个等待任务(等待时间为h[0]-now)
	}
	return newtasks
}

// task任务完成之后，根据完成的结果更新dialstate对象
func (s *dialstate) taskDone(t task, now time.Time) {
	switch t := t.(type) { //判断任务的类型
	case *dialTask: //是连接任务(与对应节点的连接已经被完成了，这里无需再次进行连接)
		s.hist.add(t.dest.ID, now.Add(dialHistoryExpiration)) //该任务指定的被连接节点添加到历史记录列表，过期时间为当前时刻dialHistoryExpiration之后
		delete(s.dialing, t.dest.ID)                          //已经完成连接，从dialing等待队列中删除
	case *discoverTask: //是节点发现任务
		if t.bootstrap { //true表示完成的是bootstrap操作，Kad路由表已更新
			s.bootstrapped = true //更新bootstrapped字段
		}
		s.lookupRunning = false                         //否则的话，表示完成的是lookup操作
		s.lookupBuf = append(s.lookupBuf, t.results...) //task对象t将返回lookup获取的若干个Node，将其添加到lookupBuf
	}
}

// dialTask结构体实现的Do接口,用于完成与对端peer指定方式的连接
func (t *dialTask) Do(srv *Server) {
	addr := &net.TCPAddr{IP: t.dest.IP, Port: int(t.dest.TCP)} //对端节点的TCP地址
	fd, err := srv.Dialer.Dial("tcp", addr.String())           //tcp方式完成与对端节点的连接
	if err != nil {
		loglogrus.Log.Errorf("dial error: %v", err)
		return
	}
	mfd := newMeteredConn(fd, false) //本socket是流量出口(false)

	srv.setupConn(mfd, t.flags, t.dest)
}

func (t *dialTask) String() string {
	return fmt.Sprintf("%v %x %v:%d", t.flags, t.dest.ID[:8], t.dest.IP, t.dest.TCP)
}

// discoverTask结构体实现的Do接口,用于lookup发现节点或bootstrap更新Kad路由表
func (t *discoverTask) Do(srv *Server) {
	if t.bootstrap { //为true则执行Bootstrap，利用引导节点srv.BootstrapNodes更新Kad路由表
		srv.ntab.Bootstrap(srv.BootstrapNodes)
		return
	}
	// newTasks generates a lookup task whenever dynamic dials are
	// necessary. Lookups need to take some time, otherwise the
	// event loop spins too fast.

	//否则执行Lookup，将查找获取的节点组更新t.results
	next := srv.lastLookup.Add(lookupInterval)
	if now := time.Now(); now.Before(next) {
		time.Sleep(next.Sub(now))
	}
	srv.lastLookup = time.Now()
	var target discover.NodeID
	rand.Read(target[:])
	t.results = srv.ntab.Lookup(target) //执行lookup操作
}

func (t *discoverTask) String() (s string) {
	if t.bootstrap {
		s = "discovery bootstrap"
	} else {
		s = "discovery lookup"
	}
	if len(t.results) > 0 {
		s += fmt.Sprintf(" (%d results)", len(t.results))
	}
	return s
}

// waitExpireTask结构体实现的Do接口,单纯的等待指定时间段结束
func (t waitExpireTask) Do(*Server) {
	time.Sleep(t.Duration)
}
func (t waitExpireTask) String() string {
	return fmt.Sprintf("wait for dial hist expire (%v)", t.Duration)
}

// Use only these methods to access or modify dialHistory.
// 返回最早与当前节点连接的节点的连接时间信息(dialHistory以队列形式记录所有节点的连接时间信息,h[0]就是最早的)
func (h dialHistory) min() pastDial {
	return h[0]
}

// 将新节点与其连接时间信息加入到dialHistory队列
func (h *dialHistory) add(id discover.NodeID, exp time.Time) {
	heap.Push(h, pastDial{id, exp})
}

// 查看指定id的节点是否存在于dialHistory队列(即是否与本节点存在连接历史记录)
func (h dialHistory) contains(id discover.NodeID) bool {
	for _, v := range h {
		if v.id == id {
			return true
		}
	}
	return false
}

// 根据形参传入的时间点,判断最早节点的连接时刻是否在此时间点之前，是就将其移除队列(删除所有过期连接记录)
func (h *dialHistory) expire(now time.Time) {
	for h.Len() > 0 && h.min().exp.Before(now) { //循环判断队首元素是否需要移除
		heap.Pop(h)
	}
}

// heap.Interface boilerplate
func (h dialHistory) Len() int           { return len(h) }
func (h dialHistory) Less(i, j int) bool { return h[i].exp.Before(h[j].exp) }
func (h dialHistory) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *dialHistory) Push(x interface{}) {
	*h = append(*h, x.(pastDial))
}
func (h *dialHistory) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
