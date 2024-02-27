package downloader

import (
	"context"
	"errors"
	"p3Chain/common"
	"p3Chain/core/eles"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"sync"
	"sync/atomic"
	"time"
)

var (
	MaxHashFetch  = 30  // Amount of hashes to be fetched per retrieval request
	MaxBlockFetch = 128 // Amount of blocks to be fetched per retrieval request

	hashTTL  = 5 * time.Second // Time it takes for a hash request to time out
	blockTTL = 6 * time.Second // Time it takes for a block request to time out
)

var (
	errBusy             = errors.New("busy")
	errUnknownPeer      = errors.New("peer is unknown or unhealthy")
	errBadPeer          = errors.New("action from bad peer ignored")
	errStallingPeer     = errors.New("peer is stalling")
	errBannedHead       = errors.New("peer head hash already banned")
	errTimeout          = errors.New("timeout")
	errEmptyHashSet     = errors.New("empty hash set by peer")
	errEmptyBlockSet    = errors.New("empty block set by peer")
	errPeersUnavailable = errors.New("no peers available or all peers tried for block download process")
	errInvalidChain     = errors.New("retrieved hash chain is invalid")
	errCrossCheckFailed = errors.New("block cross-check failed")
	errCancelHashFetch  = errors.New("hash fetching canceled (requested)")
	errCancelBlockFetch = errors.New("block fetching canceled (requested)")
	errNoSyncActive     = errors.New("no sync active")

	errNotFindAncestorBlock = errors.New("not find common ancestor block")
	errInvalidAncestorBlock = errors.New("the common ancestor block number obtained is greater than the total number of local blockChain")
)

// hashCheckFn is a callback type for verifying a hash's presence in the local chain.
// 验证某一区块hash是否存在于本地区块链
type hashCheckFn func(common.Hash) bool

// blockRetrievalFn is a callback type for retrieving a block from the local chain.
// 在本地区块链上检索某一区块
type blockRetrievalFn func(common.Hash) *eles.WrapBlock

// headRetrievalFn is a callback type for retrieving the head block from the local chain.
// 检索获取本地区块链的最新区块
type headRetrievalFn func() *eles.WrapBlock

// chainInsertFn is a callback type to insert a batch of blocks into the local chain.
// 向本地区块链中插入一批新的区块
type chainInsertFn func(*eles.WrapBlocks) (int, error)

// 读取当前本地区块链状态
type chainStatusFn func() (Height uint64, head common.Hash, genesis common.Hash)

// peerDropFn is a callback type for dropping a peer detected as malicious.
// 删除被检测为恶意节点的peer
type peerDropFn func(id common.NodeID)

// 记录从对端peer获取的区块集合
type blockPack struct {
	peerId common.NodeID     //peer节点标识符
	blocks []*eles.WrapBlock //获取的区块集合
}

// 记录从对端peer获取的区块hash集合
type hashPack struct {
	peerId common.NodeID //peer节点标识符
	hashes []common.Hash //获取的区块哈希集合
}

type Downloader struct {
	peers *peerSet // Set of active peers from which download can proceed  可以下载区块的活跃对等节点组成的集合

	interrupt int32 // Atomic boolean to signal termination

	// Callbacks
	hasBlock    hashCheckFn      // Checks if a block is present in the chain   检查区块是否存在于本地区块链
	getBlock    blockRetrievalFn // Retrieves a block from the chain   从本地区块链获取指定hash的区块
	headBlock   headRetrievalFn  // Retrieves the head block from the chain   获取本地区块链的最新区块
	insertChain chainInsertFn    // Injects a batch of blocks into the chain   向本地区块链中插入一批新区块
	chainStatus chainStatusFn
	dropPeer    peerDropFn // Drops a peer for misbehaving    删除指定p.id的恶意节点

	// Status
	synchronising int32 //是否正在进行区块同步的标志位(==1表示正在同步)

	// Channels
	newPeerCh  chan *peer
	hashCh     chan hashPack  // Channel receiving inbound hashes
	blockCh    chan blockPack // Channel receiving inbound blocks
	processCh  chan bool      // Channel to signal the block fetcher of new or finished work
	cancelCh   chan struct{}  // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex   // Lock to protect the cancel channel in delivers
}

func New(hasBlock hashCheckFn, getBlock blockRetrievalFn, headBlock headRetrievalFn, insertChain chainInsertFn, chainStatus chainStatusFn, dropPeer peerDropFn) *Downloader {
	downloader := &Downloader{
		peers:       newPeerSet(),
		hasBlock:    hasBlock,
		getBlock:    getBlock,
		headBlock:   headBlock,
		insertChain: insertChain,
		chainStatus: chainStatus,
		dropPeer:    dropPeer,
		newPeerCh:   make(chan *peer, 1),
		hashCh:      make(chan hashPack, 1),
		blockCh:     make(chan blockPack, 1),
		processCh:   make(chan bool, 1),
	}

	return downloader
}

// 检查downloader是否正在检索区块中
func (d *Downloader) Synchronising() bool {
	return atomic.LoadInt32(&d.synchronising) > 0
}

// 将新的peer节点注入到peerSet集合中
func (d *Downloader) RegisterPeer(id common.NodeID, version int, head common.Hash, RequestHashes relativeHashFetcherFn, RequestHashesFromNumber absoluteHashFetcherFn, RequestBlocks blockFetcherFn, GetBlockByNumber blockFetchByNumFn) error {
	if err := d.peers.Register(newPeer(id, version, head, RequestHashes, RequestHashesFromNumber, RequestBlocks, GetBlockByNumber)); err != nil {
		loglogrus.Log.Debugf("Block synchronization Downloader failed: Couldn't Register peer (%x) into Downloader peerSet, err:%v\n", id, err)
		return err
	}
	loglogrus.Log.Infof("Block synchronization Downloader: Register peer (%x) into Downloader peerSet successfully!\n", id)

	return nil
}

// 将指定p.id的peer节点从peerSet集合中删除
func (d *Downloader) UnregisterPeer(nodeID common.NodeID) error {
	glog.V(logger.Detail).Infoln("Unregistering peer", nodeID)
	if err := d.peers.Unregister(nodeID); err != nil {
		glog.V(logger.Error).Infoln("Unregister failed:", err)
		return err
	}
	return nil
}

// 尝试将本地区块链与远程peer节点进行同步
func (d *Downloader) Synchronise(ctx context.Context, id common.NodeID, head common.Hash, height uint64) {
	loglogrus.Log.Infof("Block synchronization downloader: Attempting synchronise with node (%x)... RemoteCurrentVersion (%x) RemoteCurrentHeight (%d)\n", id, head, height)

	switch err := d.synchronise(ctx, id, head, height); err { //进行同步(三个实参分别是:1.远程peer的的p.id 2.远程peer的头区块 3.远程peer的区块高度)
	case nil:
		loglogrus.Log.Infof("Block synchronization downloader: Block Synchronisation with node (%x) has been completed!\n", id)
	case errBusy:
		loglogrus.Log.Warnf("Block synchronization downloader: The current node is in block synchronization, please wait...\n")

	case errTimeout, errBadPeer, errStallingPeer, errBannedHead, errEmptyHashSet, errPeersUnavailable, errInvalidChain, errCrossCheckFailed:
		loglogrus.Log.Infof("Block synchronization downloader: Block synchronization with node (%x) is failed, err: %v\n", id, err)
		//d.dropPeer(id)

	default:
		loglogrus.Log.Infof("Block synchronization downloader failed: Block synchronization with node (%x) is failed, err: %v\n", id, err)
	}
}

func (d *Downloader) synchronise(ctx context.Context, id common.NodeID, hash common.Hash, height uint64) error {
	if !atomic.CompareAndSwapInt32(&d.synchronising, 0, 1) { //确保一次最多只有一个goroutine在进行区块同步(执行当前synchronise方法)
		return errBusy
	}
	defer atomic.StoreInt32(&d.synchronising, 0) //完成同步后,将d.synchronising重新设置为0

	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelLock.Unlock()

	p := d.peers.Peer(id) //在peerSet集合中检索目标peer节点
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeer(ctx, p, hash, height) //与该peer节点完成区块链同步

}

// 基于头区块hash和指定的远程peer,与目标节点进行区块同步
func (d *Downloader) syncWithPeer(ctx context.Context, p *peer, hash common.Hash, height uint64) (err error) {
	defer func() {
		// reset on error
		if err != nil {
			d.cancel()
		}
	}()

	number, getHashes, err := d.FindAncestor(ctx, p) //获取本地与p节点的公共祖先区块的编号以及从从公共祖先区块开始的若干个区块hash值
	if err != nil {
		loglogrus.Log.Warnf("Block synchronization downloader failed: Couldn't find common ancestor block between local node and remote node (%x), err:%v\n", p.nodeID, err)
		return err
	}
	loglogrus.Log.Infof("Block synchronization downloader: Fetch common ancestor block is succeed, ancestor blockNumber (%d), Number of blocks (%d) waiting for synchronization\n",
		number, len(getHashes))

	d.fetchBlocks(ctx, p, number, getHashes)

	return nil
}

// 关闭d.cancelCh管道，停止downloader的所有操作,同时重置downloader的queue队列
func (d *Downloader) cancel() {
	d.cancelLock.Lock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
		default:
			close(d.cancelCh)
		}
	}
	d.cancelLock.Unlock()

}

// 终端当前downloader,取消当前同步操作
func (d *Downloader) Terminate() {
	atomic.StoreInt32(&d.interrupt, 1) //d.interrupt置位1,终止同步操作(终止process())
	d.cancel()                         //重置清空queue队列
}

// 尝试定位本地链与远程对等peer的区块链的共同祖先区块
func (d *Downloader) FindAncestor(ctx context.Context, p *peer) (uint64, []common.Hash, error) {
	loglogrus.Log.Infof("Block synchronization downloader: Attempting to query the common ancestor block with node (%x) ...\n", p.nodeID)

	headBlock := d.headBlock()
	var head uint64
	var chainIsEmpty bool = false
	if *headBlock == (eles.WrapBlock{}) { //如果本地数据库中不存在任何区块
		chainIsEmpty = true
	} else { //
		chainIsEmpty = false
		head = headBlock.Number //获取本地区块链最新区块的区块编号
	}
	var from uint64

	round := head / uint64(MaxHashFetch-1)

	if round >= 1 { // 说明已经至少生成过 MaxHashFetch 个区块
		from = round * uint64(MaxHashFetch-1)
	} else { // 说明本地还未生成 MaxHashFetch 个区块
		from = 0
	}

	go p.RequestHashesFromNumber(from, MaxHashFetch) // 向对方peer申请获取从from开始的MaxHashFetch个区块的hash值(发送GetBlockHashesFromNumberMsg消息)

	startNumber, startHash := uint64(0), common.Hash{} //记录(双方区块链)的共同祖先区块的编号与哈希值
	getHashes := make([]common.Hash, 0)                //保存从公共祖先区块开始的区块hash集合
	timeout := time.After(hashTTL)

	for finished := false; !finished; {
		select {

		case <-ctx.Done():
			return 0, []common.Hash{}, errCancelHashFetch

		case <-d.cancelCh:
			return 0, []common.Hash{}, errCancelHashFetch

		case hashPack := <-d.hashCh: //得到了对端节点对GetBlockHashesFromNumberMsg消息的回应hash
			if hashPack.peerId != p.nodeID {
				loglogrus.Log.Warnf("Block synchronization downloader failed: Received hashes from incorrect peer, expect (%x) infact (%x)\n", p.nodeID, hashPack.peerId)
				break
			}
			hashes := hashPack.hashes
			if len(hashes) == 0 { // 没有获取任何响应的区块hash
				loglogrus.Log.Infof("Block synchronization downloader failed: Empty GetBlockHashesFromNumberMsg from node (%x), Does not contain any block hash!\n", p.nodeID)
				return 0, []common.Hash{}, errEmptyHashSet
			}
			// 根据对端peer返回的hash集合确定两条链是否存在共同祖先区块
			finished = true

			if chainIsEmpty { // 本地区块数目为0，全量同步
				getHashes = append(getHashes, hashes...)

			} else { // 本地区块数目不为0，从公共祖先区块处进行同步
				for i := 0; i < len(hashes); i++ { // hashes集合存储着按照区块编号从小到大的区块的hash值
					if d.hasBlock(hashes[i]) { // 本地已有的区块,不需要进行同步
						continue
					} else {
						if i == 0 {
							// TODO:正常来说从收到的hash回复中，前几个hash的区块在本地必然是存在的。但如果出现hashes[0]代表的区块在本地就不存在，可能需要重新调整from的值，再次进行申请
							panic("区块同步异常,可能需要重新调整from!!!!!!")
						}
						startNumber = from + uint64(i-1) // 前一个区块是通信双方的最后一个公共祖先区块
						startHash = hashes[i-1]
						getHashes = append(getHashes, hashes[i:]...) // 等待去同步的区块的hash
						break
					}
				}
			}
			_ = startHash
			return startNumber, getHashes, nil
		case <-timeout:
			loglogrus.Log.Warnf("Block synchronization downloader failed: Timeout! Failed to receive GetBlockHashesFromNumberMsg from other nodes within the limited time (%v s)\n", hashTTL.Seconds())
			return 0, []common.Hash{}, errTimeout
		}
	}
	//若查询到了共同祖先区块,可以直接返回
	// if !common.EmptyHash(hash) {
	// 	loglogrus.Log.Infof("Block synchronization downloader: The common ancestor block between local node and node (%x) has been found: blockNum (%d),blockHash (%x)\n",
	// 		p.nodeID, number, hash)
	// 	return number, getHashes, nil
	// }
	return 0, nil, errNotFindAncestorBlock
}

// 只负责向指定的远程节点发送 GetBlocksMsg 请求 (无需等待接收)
func (d *Downloader) fetchBlocks(ctx context.Context, p *peer, from uint64, hashes []common.Hash) error {

	//1.公共祖先区块编号必须小于等于本地链的currentBlock区块编号
	if from > d.headBlock().Number {
		return errInvalidAncestorBlock
	}

	loglogrus.Log.Infof("Block synchronization downloader: Start to fetch (%d) blocks by remote node (%x) from block (%d) ...\n", len(hashes), p.nodeID, from)
	//向对方请求从公共祖先区块开始的所有区块实体
	go p.RequestBlocks(hashes)

	// 等待从DeliverBlocks获取区块(从 d.blockCh 管道)
	timeout := time.After(blockTTL)
	for finished := false; !finished; {
		select {
		case <-ctx.Done():
			return errCancelBlockFetch
		case <-d.cancelCh:
			return errCancelBlockFetch

		case <-d.hashCh: //得到了对端节点对GetBlockHashesFromNumberMsg消息的回应hash

		case blockPack := <-d.blockCh:
			// Out of bounds blocks received, ignore them
			if blockPack.peerId != p.nodeID {
				loglogrus.Log.Warnf("Block synchronization downloader failed: Received hashes from incorrect peer, expect (%x) infact (%x)\n", p.nodeID, blockPack.peerId)
				break
			}
			// Make sure the peer actually gave something valid
			blocks := eles.WrapBlocks{}
			blocks = append(blocks, blockPack.blocks...)
			if len(blocks) == 0 {
				loglogrus.Log.Infof("Block synchronization downloader failed: Empty GetBlocksMsg from node (%x), Does not contain any block hash!\n", p.nodeID)
				return errEmptyBlockSet
			}

			finished = true

			// 将区块存储到本地
			if _, err := d.insertChain(&blocks); err != nil {
				loglogrus.Log.Warnf("Block synchronization downloader failed: Commit Block into Local BlockChain is failed, err:%v\n", err)
			}

			height, version, _ := d.chainStatus()
			loglogrus.Log.Infof("Block synchronization downloader: Block synchronization succeeded, Current local blockchain status: CurrentHeight (%d), CurrentVersion (%x)\n",
				height, version)

		case <-timeout:
			loglogrus.Log.Warnf("Block synchronization downloader failed: Timeout! Failed to receive GetBlocksMsg from other nodes within the limited time (%v s)\n", blockTTL.Seconds())
			return errTimeout
		}
	}

	return nil
}

// DeliverBlocks injects a new batch of blocks received from a remote node.
// This is usually invoked through the BlocksMsg by the protocol handler.
// 将从远端peer处获取的区块block集合与对端节点的p.id一起,组装成 blockPack(每次只选取两个区块进行组装,也就是一组父子区块),输入到d.blockCh管道中
// 通常是在收到对端peer节点的BlocksMsg消息后,由本地的由协议处理程序调用
func (d *Downloader) DeliverBlocks(id common.NodeID, blocks []*eles.WrapBlock) error {
	if atomic.LoadInt32(&d.synchronising) == 0 { //确保downloader正处于活跃状态(正在与peer节点进行区块同步)
		return errNoSyncActive
	}
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()

	select {
	case d.blockCh <- blockPack{id, blocks}: //将从对端peer处获取的区块实体集合连同对端节点的p.id共同打包成blockPack输入到d.blockCh管道
		return nil

	case <-cancel:
		return errNoSyncActive
	}
}

// DeliverHashes injects a new batch of hashes received from a remote node into
// the download schedule. This is usually invoked through the BlockHashesMsg by
// the protocol handler.
// 将从远端peer处获取的区块hash集合与对端节点的p.id一起,组装成 hashPack,输入到d.hashCh管道中
// 通常是在收到对端peer节点的BlockHashesMsg消息后,由本地的由协议处理程序调用
func (d *Downloader) DeliverHashes(id common.NodeID, hashes []common.Hash) error {
	if atomic.LoadInt32(&d.synchronising) == 0 { //确保downloader正处于活跃状态(也就是正处于与peer的同步状态中)
		return errNoSyncActive
	}

	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()

	select {
	case d.hashCh <- hashPack{id, hashes}:
		return nil

	case <-cancel:
		return errNoSyncActive
	}
}
