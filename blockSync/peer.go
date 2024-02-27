package blockSync

import (
	"errors"
	"fmt"
	"p3Chain/blockSync/downloader"
	"p3Chain/common"
	"p3Chain/core/eles"
	"sync"

	"p3Chain/p2p/server"

	loglogrus "p3Chain/log_logrus"

	"gopkg.in/fatih/set.v0"
)

var (
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

type peer struct {
	peer *server.Peer
	rw   server.MsgReadWriter
	host *SyncProtocal

	version int    // Protocol version negotiated
	subNet  string // Network ID being on

	nodeID common.NodeID //对方节点的nodeID

	RemoteVersion common.Hash //
	lock          sync.RWMutex

	knownBlocks *set.Set //此对等方已知的区块集合(总是保存最新的maxKnownBlocks个区块)

	SelfHeight   uint64 //存储本地节点的区块链高度
	RemoteHeight uint64 //存储对方节点的区块链高度
}

func newPeer(host *SyncProtocal, remote *server.Peer, rw server.MsgReadWriter) *peer {
	return &peer{
		peer:        remote,
		rw:          rw,
		nodeID:      common.NodeID(remote.ID()),
		knownBlocks: set.New(set.ThreadSafe).(*set.Set),
		host:        host,
	}
}

// Head retrieves a copy of the current head (most recent) hash of the peer.
// 本方法负责检索对端peer当前的头部(最新的区块)哈希
func (p *peer) Head() (hash common.Hash) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.RemoteVersion[:])
	return hash
}

// SetHead updates the head (most recent) hash of the peer.
// 更新头部哈希
func (p *peer) SetHead(hash common.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.RemoteVersion[:], hash[:])
}

// 获取本地节点的区块高度
func (p *peer) GetSelfHeight() uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.SelfHeight
}

// 获取远端节点的区块高度
func (p *peer) GetRemoteHeight() uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.RemoteHeight
}

// 更新本地节点的区块高度
func (p *peer) SetSelfHeight(sh uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.SelfHeight = sh
}

// 更新远端节点的区块高度
func (p *peer) SetRemoteHeight(rh uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.RemoteHeight = rh
}

// SendBlockHashes sends a batch of known hashes to the remote peer.
// 向对方peer节点发送区块哈希(BlockHashesMsg消息)
func (p *peer) SendBlockHashes(hashes []common.Hash) error {
	return server.Send(p.rw, BlockHashesMsg, hashes)
}

// SendBlocks sends a batch of blocks to the remote peer.
// 向对方peer节点发送区块实体(BlocksMsg消息)
func (p *peer) SendBlocks(blocks []*eles.WrapBlock) error {
	return server.Send(p.rw, BlocksMsg, blocks)
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
// SendNewBlockHashes函数通过 hash值 通知宣布多个区块的可用性(向对端peer发送NewBlockHashesMsg消息)
func (p *peer) SendNewBlockHashes(hashes []common.Hash) error {
	for _, hash := range hashes { //在knownBlocks集合中更新形参指定的区块哈希(在与对方节点的peer中标记)
		p.knownBlocks.Add(hash)
	}
	return server.Send(p.rw, NewBlockHashesMsg, hashes) //向对方发送NewBlockHashesMsg消息,包含新的可用区块的哈希值
}

// SendNewBlock propagates an entire block to a remote peer.
// SendNewBlock将整个区块实体发送到对端peer(发送NewBlockMsg消息)
func (p *peer) SendNewBlock(block *eles.WrapBlock) error {

	hash, _ := block.RawBlock.Hash()
	p.knownBlocks.Add(hash)                                                       //将区块哈希添加到knownBlocks集合
	return server.Send(p.rw, NewBlockMsg, []interface{}{block, block.Number + 1}) //发送区块与对应td到对端peer
}

// SendHeight函数负责将当前块高度以及最新区块hash发送给对方节点(ChainHeightMsg消息)
func (p *peer) SendHeight(height uint64, head common.Hash) error {

	return server.Send(p.rw, ChainHeightMsg, chainHeightData{height, head}) //发送区块高度与最新区块hash到对端peer
}

// RequestHashes fetches a batch of hashes from a peer, starting at from, going
// towards the genesis block.
// 申请让对端节点检索指定的区块哈希值(发送GetBlockHashesMsg消息)
func (p *peer) RequestHashes(from common.Hash) error {
	return server.Send(p.rw, GetBlockHashesMsg, getBlockHashesData{from, uint64(downloader.MaxHashFetch)})
}

// RequestHashesFromNumber fetches a batch of hashes from a peer, starting at the
// requested block number, going upwards towards the genesis block.
// 申请从对端peer获取一批区块hash值,从请求的区块号from开始,向后累计count个(发送GetBlockHashesFromNumberMsg消息)
func (p *peer) RequestHashesFromNumber(from uint64, count int) error {
	return server.Send(p.rw, GetBlockHashesFromNumberMsg, getBlockHashesFromNumberData{from, uint64(count)})
}

// RequestBlocks fetches a batch of blocks corresponding to the specified hashes.
// 向对方节点申请获取形参指定hash值的一批区块(发送GetBlocksMsg消息)
func (p *peer) RequestBlocks(hashes []common.Hash) error {
	return server.Send(p.rw, GetBlocksMsg, hashes)
}

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
// 完成与对端peer的协议handshake
func (p *peer) Handshake(sh uint64, head common.Hash, genesis common.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 1)
	go func() {
		errc <- server.Send(p.rw, StatusMsg, &statusData{ //发送StatusMsg
			ProtocolVersion: uint32(p.version),
			SelfHeight:      sh,      //本地区块链高度
			CurrentBlock:    head,    //本地区块链的最新区块(即当前区块)哈希值
			GenesisBlock:    genesis, //本地区块链的创世区块哈希值
		})
	}()
	// In the mean time retrieve the remote status message
	msg, err := p.rw.ReadMsg() //等待接收对端peer的handshake回复
	if err != nil {
		return err
	}
	//验证回复消息的正确性
	if msg.Code != StatusMsg {
		err := errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
		loglogrus.Log.Errorf("Block synchronization failed: HandShake with peer (%x) is failed, err:%v\n", err)
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		err := errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
		loglogrus.Log.Errorf("Block synchronization failed: HandShake with peer (%x) is failed, err:%v\n", err)
		return err
	}
	// Decode the handshake and make sure everything matches
	var status statusData
	if err := msg.Decode(&status); err != nil {
		err := errResp(ErrDecode, "msg %v: %v", msg, err)
		loglogrus.Log.Errorf("Block synchronization failed: HandShake with peer (%x) is failed, err:%v\n", err)
		return err
	}
	if status.GenesisBlock != genesis { //创世区块必须为同一个
		err := errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock, genesis)
		loglogrus.Log.Errorf("Block synchronization failed: HandShake with peer (%x) is failed, err:%v\n", err)
		return err
	}
	if status.SubNet != p.subNet {
		err := errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock, genesis)
		loglogrus.Log.Errorf("Block synchronization failed: HandShake with peer (%x) is failed, err:%v\n", err)
		return errResp(ErrNetworkIdMismatch, "%s (!= %s)", status.SubNet, p.subNet)
	}
	if int(status.ProtocolVersion) != p.version { //子类协议版本必须相同
		err := errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
		loglogrus.Log.Errorf("Block synchronization failed: HandShake with peer (%x) is failed, err:%v\n", err)
		return errResp(ErrNetworkIdMismatch, "%s (!= %s)", status.SubNet, p.subNet)
	}
	// Configure the remote peer, and sanity check out handshake too
	p.RemoteHeight, p.RemoteVersion = status.SelfHeight, status.CurrentBlock //从回复消息中获取最新区块哈希和高低压,用于更新peer对象

	loglogrus.Log.Infof("Block synchronization: Get ResponseMsg from node (%x), RemoteHeight (%d)  RemoteVersion (%x)", p.nodeID, p.RemoteHeight, p.RemoteVersion)
	return <-errc
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%x]", p.nodeID,
		fmt.Sprintf("eth/%2d", p.version),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
// peerSet存储着当前参与以太坊子协议的活动对等体peer的集合。
type peerSet struct {
	peers map[common.NodeID]*peer //key为对端peer的id(p.id)
	lock  sync.RWMutex
}

// newPeerSet creates a new peer set to track the active participants.
// 创建一个新的peerSet对象
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[common.NodeID]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
// 向peerSet集合中注册一个新的peer节点(key == p.id , value == p)
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[p.nodeID]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.nodeID] = p
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
// 从peerSet集合删除一个指定id的peer节点
func (ps *peerSet) Unregister(nodeID common.NodeID) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[nodeID]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, nodeID)
	return nil
}

// Peer retrieves the registered peer with the given id.
// 根据给定的id检索出peer节点对象
func (ps *peerSet) Peer(nodeID common.NodeID) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[nodeID]
}

// Len returns if the current number of peers in the set.
// 计算peerSet集合容纳的peer节点数
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
// 检索出不包含给定区块哈希值的已连接peer组成的集合
func (ps *peerSet) PeersWithoutBlock(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownBlocks.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
// 在所有的已连接peer集合中检索出具有最大区块高度的peer节点作为返回值
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer   *peer
		bestHeight uint64
	)
	for _, p := range ps.peers { //遍历所有与本节点存在以太坊子类协议连接的peer节点对象
		if h := p.GetRemoteHeight(); bestPeer == nil || h > bestHeight { //找出具有最大区块高度的peer节点对象
			bestPeer, bestHeight = p, h
		}
	}
	return bestPeer
}
