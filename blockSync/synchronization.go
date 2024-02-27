package blockSync

import (
	"context"
	"errors"
	"fmt"
	"p3Chain/blockSync/downloader"
	"p3Chain/coefficient"
	"p3Chain/p2p/server"
	"p3Chain/rlp"
	"sync"
	"time"

	"p3Chain/common"
	"p3Chain/core/dpnet"
	"p3Chain/core/eles"
	"p3Chain/core/validator"
	loglogrus "p3Chain/log_logrus"
)

const (
	StatusMsg                          = 0x00
	NewBlockHashesMsg                  = 0x01
	GetBlockHashesMsg                  = 0x02
	BlockHashesMsg                     = 0x03
	GetBlocksMsg                       = 0x04
	BlocksMsg                          = 0x05
	NewBlockMsg                        = 0x06
	GetBlockHashesFromNumberMsg        = 0x07
	ChainHeightMsg                     = 0x08
	protocolVersion             uint64 = 0x00 // ver 0.0

	protocolName = "sync"
)

var (
	errBadBlock = errors.New("The Block is Bad")
)

const (
	minDesiredPeerCount = 1 // Amount of peers desired to start syncing
	maxBlockCache       = 512
)

var (
	forceSyncCycle = coefficient.Synchronization_BlockSynchronizationCheckCycle // Time interval to force syncs, even if few peers are available
)

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type SyncProtocal struct {
	protocol        server.Protocol            //继承Protocal类，需自行实现Sync协议功能
	ValidateManager *validator.ValidateManager //区块验证类
	ChainManager    *eles.BlockChain           //继承worldstate.DpStateManager(用于在本地区块链上增删查补)

	broadcastSyncReq bool // 是否开启主动同步功能(主动发送区块同步请求)
	quit             chan struct{}

	dpNet    *dpnet.DpNet
	selfNode dpnet.Node

	downloader     *downloader.Downloader
	peers          *peerSet //保存所有与本节点完成handshake的peer节点
	chainHeightNum map[common.NodeID]uint64
	blockCache     map[common.NodeID][]eles.WrapBlock

	newPeerCh chan *peer

	chainHeightNumMutex sync.RWMutex

	leaderMode bool // the leader should not have to fetch blocks from others
}

func (sp *SyncProtocal) UseLeaderMode() {
	sp.leaderMode = true
}

func (sp *SyncProtocal) UseNormalMode() {
	sp.leaderMode = false
}

func New(chain *eles.BlockChain, selfNode dpnet.Node, dpNet *dpnet.DpNet) *SyncProtocal {
	sync := &SyncProtocal{
		selfNode:         selfNode,
		dpNet:            dpNet,
		ChainManager:     chain,
		peers:            newPeerSet(),
		quit:             make(chan struct{}),
		chainHeightNum:   make(map[common.NodeID]uint64),
		blockCache:       make(map[common.NodeID][]eles.WrapBlock),
		newPeerCh:        make(chan *peer, 5),
		leaderMode:       false,
		broadcastSyncReq: false,
	}
	sync.protocol = server.Protocol{
		Name:    protocolName,
		Version: uint(protocolVersion),
		Length:  9, // 0x00 to 0x08
		Run:     sync.handle,
	}
	// 注册downloader对象
	sync.downloader = downloader.New(sync.ChainManager.FindVersion, sync.ChainManager.GetBlockFromHash, sync.ChainManager.GetCurrentBlock, sync.ChainManager.InsertChains, sync.ChainManager.GetChainStatus, sync.removePeer)

	return sync
}

func (sp *SyncProtocal) Start() {

	ctx, cancel := context.WithCancel(context.Background())

	go sp.syncer(ctx) //进行与peer节点的区块链同步

	for {
		select {
		case <-sp.quit:
			cancel()
		default:
			time.Sleep(time.Second)
		}
	}

}

// 结束区块同步
func (sp *SyncProtocal) Stop() {
	sp.quit <- struct{}{}
}

// 设置区块同步器
func (sp *SyncProtocal) SetValidateManager(cv *validator.ValidateManager) {
	sp.ValidateManager = cv
}

// 获取区块同步器
func (sp *SyncProtocal) GetValidateManager() *validator.ValidateManager {
	return sp.ValidateManager
}

// 删除指定ID的Peer
func (sp *SyncProtocal) removePeer(nodeID common.NodeID) {
	// Short circuit if the peer was already removed
	peer := sp.peers.Peer(nodeID) //查询指定ID的子类协议peer节点
	if peer == nil {
		return
	}

	// Unregister the peer from the downloader and Ethereum peer set
	sp.downloader.UnregisterPeer(nodeID)
	if err := sp.peers.Unregister(nodeID); err != nil {
		loglogrus.Log.Warnf("Block synchronization failed: Unable to disconnect block synchronization p2p connection with node (%x)\n", nodeID)
		return
	}

	loglogrus.Log.Infof("Block synchronization: Block synchronization p2p connection with node (%x) is interrupted!\n", nodeID)
}

func (sp *SyncProtocal) Protocol() server.Protocol {
	return sp.protocol
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
// 作为协议Run函数,将在进行协议handshake时自动调用
func (sp *SyncProtocal) handle(p *server.Peer, rw server.MsgReadWriter, ctx context.Context) error {
	height, head, genesis := sp.ChainManager.GetChainStatus() //获取当前本地区块链的状态

	syncPeer := newPeer(sp, p, rw) //在p2p连接的基础上与对方peer建立sync协议

	loglogrus.Log.Infof("Block synchronization: Start block synchronization with node (%x)... LocalHeight (%d), LocalVersion (%x)\n",
		syncPeer.nodeID, height, head)

	if err := syncPeer.Handshake(height, head, genesis); err != nil { //完成与对方节点的协议handshake
		loglogrus.Log.Warnf("Block synchronization failed: Protocal handshake with node (%x) is failed, err:%v\n", syncPeer.nodeID, err)
		return err
	}

	if err := sp.peers.Register(syncPeer); err != nil { //完成了与对端peer的handshake,将其注册到本地的peers集合中
		loglogrus.Log.Infof("Block synchronization failed: Couldn't Register peer (%x) into peerSet, err:%v\n", syncPeer.nodeID, err)
		return err
	}
	defer sp.removePeer(syncPeer.nodeID)

	// 将此完成handshake的peer节点注册到downloader
	if err := sp.downloader.RegisterPeer(syncPeer.nodeID, syncPeer.version, syncPeer.Head(), syncPeer.RequestHashes, syncPeer.RequestHashesFromNumber, syncPeer.RequestBlocks, syncPeer.host.ChainManager.GetBlockByNumber); err != nil {
		loglogrus.Log.Infof("Block synchronization failed: Couldn't Register peer (%x) into downloader, err:%v\n", syncPeer.nodeID, err)
		return err
	}

	// 主循环,循环等待来自上述peer节点的message消息并完成处理
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			sp.handleMsg(syncPeer)
		}

	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (sp *SyncProtocal) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		loglogrus.Log.Warnf("Block synchronization failed: Couldn't Read the next message from the remote peer (%x), err:%v\n", p.nodeID, err)
		time.Sleep(time.Second)
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		loglogrus.Log.Warnf("Block synchronization failed: The number of bytes (%d) of synchronization message exceeds the upper limit (%d)\n", msg.Size, ProtocolMaxMsgSize)
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	switch msg.Code {
	case StatusMsg:
		loglogrus.Log.Warnf("Block synchronization failed: Status messages should never arrive after the handshake Stage!\n")
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	case GetBlockHashesFromNumberMsg: //2.对方peer请求获取从指定区块(request.Number)开始向后数的request.Amount个区块的哈希值
		var request getBlockHashesFromNumberData
		if err := msg.Decode(&request); err != nil {
			loglogrus.Log.Warnf("Block synchronization failed: Decode GetBlockHashesFromNumberMsg is failed, err:%v\n", err)
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		// 每次申请进行同步的区块数量是有限制的
		if request.Amount > uint64(downloader.MaxHashFetch) {
			request.Amount = uint64(downloader.MaxHashFetch)
		}
		loglogrus.Log.Infof("Block synchronization: Receive GetBlockHashesFromNumberMsg from remote peer (%x), Request block start number (%d), count (%d)\n",
			p.nodeID, request.Number, request.Amount)

		last := sp.ChainManager.GetBlockByNumber(request.Number + request.Amount - 1) //获取应该检索的最后一个区块
		if last == nil {                                                              //如果当前本地区块数量不够,重新设定Amount值,并设定最后一个区块为本地的当前区块
			last = sp.ChainManager.GetCurrentBlock()
			loglogrus.Log.Warnf("Block synchronization: The number (%d) of blocks requested by the remote node, and the actual number of blocks returned is (%d) and currentHeight (%d)\n",
				request.Amount, last.Number-request.Number+1, last.Number)
			request.Amount = last.Number - request.Number + 1
		}
		if request.Amount == 0 {
			loglogrus.Log.Warnf("Block synchronization failed: No blocks can be sent locally")
			time.Sleep(200 * time.Millisecond)
			return nil
		}

		if last.Number < request.Number { //请求的初始区块号Number不正确(超过当前区块链的最后一个区块)
			loglogrus.Log.Infof("Block synchronization failed: Illegal block start number (%d) requested by remote node, which is greater than the total number (%d) of current blockChain\n",
				request.Number, last.Number)
			return p.SendBlockHashes(nil)
		}
		// Retrieve the hashes from the last block backwards, reverse and return
		hash := last.RawBlock.BlockID
		hashes := make([]common.Hash, 0)

		hashes = append(hashes, sp.ChainManager.GetBlockHashesFromHash(hash, request.Amount)...) //获取从最后一个区块向上的amount个区块的哈希值

		for i := 0; i < len(hashes)/2; i++ { //完成上述hash集合的反转
			hashes[i], hashes[len(hashes)-1-i] = hashes[len(hashes)-1-i], hashes[i]
		}
		return p.SendBlockHashes(hashes) //向对方回复BlockHashesMsg消息,包含上述检索出的Amount个区块哈希值

	case BlockHashesMsg: //3.收到对端peer的回应,包含若干区块哈希值,将它们全部交给downloader完成排序
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))

		var hashes []common.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			loglogrus.Log.Warnf("Block synchronization failed: Decode BlockHashesMsg is failed, err:%v\n", err)
			break
		}
		loglogrus.Log.Infof("Block synchronization:  Receive BlockHashesMsg from remote peer (%x), the number of BlockHashes is (%d)\n", p.nodeID, len(hashes))
		err := sp.downloader.DeliverHashes(p.nodeID, hashes)
		if err != nil {
			loglogrus.Log.Errorf("Block synchronization failed: downloader has been stopped!\n")
		}

	case GetBlocksMsg: //4.对端peer请求获得指定hash的区块实体(若干区块)
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			loglogrus.Log.Warnf("Block synchronization failed: Decode GetBlocksMsg is failed, err:%v\n", err)
			return err
		}
		var (
			hash common.Hash
			//bytes  float64        //计算所有区块容量之和
			hashes []common.Hash     //存储所有待检索的区块的哈希值
			blocks []*eles.WrapBlock //存储所有检索到的区块(待发送)
		)
		for {
			err := msgStream.Decode(&hash)
			if err == rlp.EOL {
				break
			} else if err != nil {
				//loglogrus.Log.Warnf("Block synchronization failed: Decode GetBlocksMsg is failed, err:%v", err)
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			hashes = append(hashes, hash)

			// Retrieve the requested block, stopping if enough was found
			if block := sp.ChainManager.GetBlockFromHash(hash); block != nil { //从本地链中获取指定hash的区块
				blocks = append(blocks, block)
			}
		}
		loglogrus.Log.Infof("Block synchronization: Receive GetBlocksMsg from remote peer (%x), Total number (%d) of blocks to be acquired by remote peer, Total number (%d) of blocks returned\n",
			p.nodeID, len(hashes), len(blocks))
		return p.SendBlocks(blocks) //将区块集合回复给对端peer(BlocksMsg消息)

	case BlocksMsg: //5.收到对端peer回复的区块集合
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))

		var blocks []*eles.WrapBlock //存储回复消息中包含的所有区块
		if err := msgStream.Decode(&blocks); err != nil {
			loglogrus.Log.Warnf("Block synchronization failed: Decode BlocksMsg is failed, err:%v\n", err)
			blocks = nil
		}
		// 更新每一个区块的时间戳
		for _, block := range blocks {
			block.ReceivedAt = msg.ReceivedAt.Format("2006-01-02 15:04:05")
			if !sp.ValidateManager.LowerValidate(block.RawBlock) { //对区块进程lower channel层的验证
				loglogrus.Log.Warnf("Block synchronization failed: The received block (%x) from node (%x) cann't pass the local lower Validation!\n",
					block.RawBlock.BlockID, p.nodeID)
				err := fmt.Sprintf("Block synchronization failed: The received block (%x) from node (%x) cann't pass the local lower Validation!\n",
					block.RawBlock.BlockID, p.nodeID)
				return errors.New(err)
			}

			if !sp.ValidateManager.UpperValidate(block.RawBlock) { //对区块进程upper channel层的验证
				loglogrus.Log.Warnf("Block synchronization failed: The received block (%x) from node (%x) cann't pass the local upper Validation!\n",
					block.RawBlock.BlockID, p.nodeID)
				err := fmt.Sprintf("Block synchronization failed: The received block (%x) from node (%x) cann't pass the local upper Validation!\n",
					block.RawBlock.BlockID, p.nodeID)
				return errors.New(err)
			}

		}
		// 通过验证的区块交给downloader
		err := sp.downloader.DeliverBlocks(p.nodeID, blocks)
		if err != nil {
			loglogrus.Log.Errorf("Block synchronization failed: downloader has been stopped!\n")
		}
	case ChainHeightMsg: //收到对端peer回复的区块高度
		var receiveHeight chainHeightData
		if err := msg.Decode(&receiveHeight); err != nil {
			loglogrus.Log.Warnf("Block synchronization failed: Decode ChainHeightMsg is failed, err:%v\n", err)
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		sp.chainHeightNumMutex.Lock()
		sp.chainHeightNum[p.nodeID] = receiveHeight.Height //记录对方节点的Height
		p.SetHead(receiveHeight.Head)                      //更新peer对象中记录对端节点的head区块hash
		p.SetRemoteHeight(receiveHeight.Height)            //更新peer对象中记录的对端节点的区块高度
		sp.chainHeightNumMutex.Unlock()

		// fmt.Printf("收到对端peer (%x) 回复的区块高度 (%d)\n", p.nodeID, receiveHeight.Height)
		// loglogrus.Log.Infof("Block synchronization: Receive ChainHeightMsg from node (%x), The current block version (%x) and block height (%d) of this node ",
		// 	p.nodeID, receiveHeight.Head, receiveHeight.Height)

	default:
		loglogrus.Log.Warnf("Block synchronization failed: Unresolved message code : %v\n", msg.Code)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (sp *SyncProtocal) OpenBroadcastSyncReq() {
	sp.broadcastSyncReq = true
}

func (sp *SyncProtocal) CloseBroadcastSyncReq() {
	sp.broadcastSyncReq = false
}

// 节点向全网广播当前自己的块高
func (sp *SyncProtocal) BroadcastHeight(height uint64, head common.Hash, ctx context.Context) {

	if !sp.broadcastSyncReq {
		return
	}

	sameGroup := make([]common.NodeID, 0)
	sameGroupMap := make(map[common.NodeID]struct{})
	if sp.dpNet != nil || (sp.selfNode != dpnet.Node{}) {
		sameGroup = sp.dpNet.BackSubnetNodesID(sp.selfNode.NodeID, sp.selfNode.NetID)
		for _, nodeID := range sameGroup {
			sameGroupMap[nodeID] = struct{}{}
		}
	}

	for nodeID, peer := range sp.peers.peers { // 仅向同组的peer发送区块同步请求

		if _, ok := sameGroupMap[nodeID]; ok {
			peer.SendHeight(height, head) //将当前区块高度发送出去(ChainHeightMsg消息)
			// fmt.Printf("[Block Sync] 向节点(%x) 发送当前块高(%d)\n", nodeID, height)
		}

		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}

}

// 进行与远端peer节点的区块同步
func (sp *SyncProtocal) syncer(ctx context.Context) {

	//leaderSynStart := time.NewTicker(6 * forceSyncCycle)

	broadcastTick := time.NewTicker(2 * forceSyncCycle) //当forceSync计时器到达计时周期,即使没有足够的对等点，也强制同步
	selfSyncTick := time.NewTicker(4 * forceSyncCycle)  //当broadcastTick到达计时周期时,检查是否收到足够多的ChainHeightMsg消息,若足够则开始同步
	for {
		select {

		case <-selfSyncTick.C: // 通过接收到的其他节点的区块高度和头哈希来判断是否需要进行区块同步
			if sp.leaderMode {
				continue
			}
			// fmt.Println("可以进行区块同步,检测点1..........")
			// loglogrus.Log.Infof("可以进行区块同步,检测点1..........\n")
			if len(sp.chainHeightNum) >= minDesiredPeerCount { //确保有足够多的ChainHeightMsg消息可供选择,然后才能开始同步
				//遍历整个sp.chainHeightNum,找出具有最大Height的节点，申请与其进行区块同步
				sp.chainHeightNumMutex.RLock()
				maxHeight := uint64(0)
				var maxHash common.NodeID
				for hash, height := range sp.chainHeightNum {
					if height > maxHeight {
						maxHeight = height
						maxHash = hash
					}
				}
				sp.chainHeightNumMutex.RUnlock()
				//找到具有最大高度的对端节点，判断是否比本地节点的区块高度高.若比本地要大,则需要与对端peer进行同步
				if maxHeight > sp.ChainManager.Height {
					peer := sp.peers.Peer(maxHash) //根据节点NodeID找出peer对象

					loglogrus.Log.Infof("Block synchronization: Start block synchronization with node (%x)...  ---  local node currentHeight (%d), remote node currentHeight (%d)\n",
						peer.nodeID, sp.ChainManager.Height, peer.GetRemoteHeight())

					//synctx, _ := context.WithTimeout(ctx, 10*time.Second)
					synctx := context.Background()
					sp.downloader.Synchronise(synctx, peer.nodeID, peer.Head(), peer.GetRemoteHeight()) //进行同步
				}
			}

		case <-broadcastTick.C: //向其他节点广播自己的区块高度和头哈希
			height := sp.ChainManager.Height
			head := sp.ChainManager.CurrentHash
			go sp.BroadcastHeight(height, head, ctx)

		case <-ctx.Done():
			broadcastTick.Stop()
			selfSyncTick.Stop()
			return
		}
	}
}
