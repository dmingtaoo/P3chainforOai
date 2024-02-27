package downloader

import (
	"errors"
	"math/big"
	"sync"

	"p3Chain/common"
	"p3Chain/core/eles"
)

type relativeHashFetcherFn func(common.Hash) error
type absoluteHashFetcherFn func(uint64, int) error
type blockFetcherFn func([]common.Hash) error
type blockFetchByNumFn func(uint64) *eles.WrapBlock

var (
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

// peer represents an active peer from which hashes and blocks are retrieved.
type peer struct {
	nodeID                  common.NodeID         // Unique identifier of the peer   对等节点的标识符
	head                    common.Hash           // Hash of the peers latest known block  对等节点最新已知块的哈希
	RequestHashes           relativeHashFetcherFn // Method to retrieve a batch of hashes from an origin hash   根据给定的原始区块hash检索获取一批相关哈希值
	RequestHashesFromNumber absoluteHashFetcherFn // Method to retrieve a batch of hashes from an absolute position  根据给定的绝对位置检索获取一批区块哈希值
	RequestBlocks           blockFetcherFn        // Method to retrieve a batch of blocks   检索获取一批区块哈希值
	GetBlockByNumber        blockFetchByNumFn

	version int // Eth protocol version number to switch strategies

	SelfHeight   big.Int //存储本地节点的区块链高度
	RemoteHeight big.Int //存储对方节点的区块链高度
}

// newPeer create a new downloader peer, with specific hash and block retrieval
// mechanisms.
// 创建一个新的downloader peer对象,伴随有特殊的区块哈希和区块检索机制
func newPeer(id common.NodeID, version int, head common.Hash, RequestHashes relativeHashFetcherFn, RequestHashesFromNumber absoluteHashFetcherFn, RequestBlocks blockFetcherFn, getBlockByNum blockFetchByNumFn) *peer {
	return &peer{
		nodeID:                  id,
		head:                    head,
		RequestHashes:           RequestHashes,
		RequestHashesFromNumber: RequestHashesFromNumber,
		RequestBlocks:           RequestBlocks,
		GetBlockByNumber:        getBlockByNum,
		version:                 version,
	}
}

// peerSet represents the collection of active peer participating in the block
// download procedure.
// 包含参与块下载过程的活动对等peer的集合。
type peerSet struct {
	peers map[common.NodeID]*peer
	lock  sync.RWMutex
}

// newPeerSet creates a new peer set top track the active download sources.
// 创建一个新的peerSet集合对象
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[common.NodeID]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
// 将新的对等peer节点加入到本地ps.peers集合中(key为p.id,value为peer对象)
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
// 将指定p.id的peer节点从ps.peers集合中删除
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
// 从ps.peers集合检索并返回指定p.id的peer节点实体
func (ps *peerSet) Peer(id common.NodeID) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
// 返回ps.peers集合保存的peer节点的个数
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// AllPeers retrieves a flat list of all the peers within the set.
// 返回ps.peers集合中保存的所有peer节点对象
func (ps *peerSet) AllPeers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		list = append(list, p)
	}
	return list
}
