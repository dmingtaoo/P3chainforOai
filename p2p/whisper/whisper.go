// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisper

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"sync"
	"time"

	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/crypto/ecies"
	"p3Chain/filter"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"

	"p3Chain/p2p/server"

	"gopkg.in/fatih/set.v0"
)

const (
	statusCode   = 0x00
	messagesCode = 0x01

	protocolVersion uint64 = 0x02
	protocolName           = "shh"

	signatureFlag   = byte(1 << 7)
	signatureLength = 65

	expirationCycle   = 800 * time.Millisecond
	transmissionCycle = 300 * time.Millisecond
)

const (
	DefaultTTL = 50 * time.Second
	DefaultPoW = 50 * time.Millisecond
)

// 记录一条Message消息的全部信息
type MessageEvent struct {
	To      *ecdsa.PrivateKey
	From    *ecdsa.PublicKey
	Message *Message
}

// Whisper represents a dark communication interface through the
// network, using its very own P2P communication layer.
type Whisper struct {
	protocol server.Protocol //继承protocal类(需自主实现，尤其是Run函数)
	filters  *filter.Filters //消息过滤器

	keys map[string]*ecdsa.PrivateKey //存储本机所使用的所有公私钥对(公钥为index，私钥为value)

	messages    map[common.Hash]*Envelope // Pool of messages currently tracked by this node  当前节点envelope消息池
	expirations map[uint32]*set.SetNonTS  // 按照envelop的的有效时间(envelope.Expiry)存储所有envelope(有效时间为index,相同有效时间的节点位于同一集合)
	poolMu      sync.RWMutex              // Mutex to sync the message and expiration pools

	peers  map[*peer]struct{} // Set of currently active peers  所有与当前节点保持活跃连接的其他节点的连接控制器对象(包括conn连接套接字)
	peerMu sync.RWMutex       // Mutex to sync the active peer set

	quit chan struct{}
}

// New creates a Whisper client ready to communicate through the P2P
// network.
func New() *Whisper {
	whisper := &Whisper{
		filters:     filter.New(),
		keys:        make(map[string]*ecdsa.PrivateKey),
		messages:    make(map[common.Hash]*Envelope),
		expirations: make(map[uint32]*set.SetNonTS),
		peers:       make(map[*peer]struct{}),
		quit:        make(chan struct{}),
	}
	whisper.filters.Start()

	// server whisper sub protocol handler
	whisper.protocol = server.Protocol{
		Name:    protocolName,
		Version: uint(protocolVersion),
		Length:  2,
		Run:     whisper.handlePeer,
	}

	return whisper
}

// Protocol returns the whisper sub-protocol handler for this particular client.
func (w *Whisper) Protocol() server.Protocol {
	return w.protocol
}

// Version returns the whisper sub-protocols version number.
func (w *Whisper) Version() uint {
	return w.protocol.Version
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
// 为当前whisper客户端生成新的加密标识(一对公私钥)
func (w *Whisper) NewIdentity() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey() //生成公私钥对
	if err != nil {
		panic(err)
	}
	w.keys[string(crypto.FromECDSAPub(&key.PublicKey))] = key //公钥作为index，私钥作为value 存储在keys字段

	return key
}

// 根据传入的私钥，先获取其公钥，再将公私钥对注入到whisper对象的keys字段
func (w *Whisper) InjectIdentity(key *ecdsa.PrivateKey) {
	w.keys[string(crypto.FromECDSAPub(&(key.PublicKey)))] = key
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
// 判断当前whisper节点是否存在指定的公私钥对
func (w *Whisper) HasIdentity(key *ecdsa.PublicKey) bool {
	return w.keys[string(crypto.FromECDSAPub(key))] != nil
}

// GetIdentity retrieves the private key of the specified public identity.
// 根据传入的公钥获取其对应的私钥
func (w *Whisper) GetIdentity(key *ecdsa.PublicKey) *ecdsa.PrivateKey {
	return w.keys[string(crypto.FromECDSAPub(key))]
}

// Watch installs a new message handler to run in case a matching packet arrives
// from the whisper network.
// 安装一个新的消息处理程序，当接收到匹配的whisper消息时调用相应的回调函数fn
func (w *Whisper) Watch(options Filter) int {
	filter := filterer{
		to:      string(crypto.FromECDSAPub(options.To)),   //过滤消息的目标
		from:    string(crypto.FromECDSAPub(options.From)), //过滤消息的来源
		matcher: newTopicMatcher(options.Topics...),        //根据Envelop信封的topic表是否满足option要求进行过滤
		fn: func(data interface{}) {
			options.Fn(data.(*Message)) //对符合要求的Envelop消息将对其Message调用回调进行处理
		},
	}
	return w.filters.Install(filter) //为当前whisper节点注册添加该Envelop消息过滤器(有id标识)
}

// Unwatch removes an installed message handler.
// 根据过滤器的id，为whisper客户端解除一个Envelop消息过滤器
func (w *Whisper) Unwatch(id int) {
	w.filters.Uninstall(id)
}

// Send injects a message into the whisper send queue, to be distributed in the
// network in the coming cycles.
// 将需要进行发送的envelope信封添加到envelop消息池中
func (w *Whisper) Send(envelope *Envelope) error {
	return w.add(envelope)
}

// 开始更新当前节点消息池和expirations集合，删除过期消息
func (w *Whisper) Start() {
	loglogrus.Log.Infoln("P2P: Whisper started\n")
	go w.update()
}

// 停止update消息池和expirations集合
func (w *Whisper) Stop() {
	close(w.quit)
	loglogrus.Log.Infoln("P2P: Whisper stopped\n")
}

// Messages retrieves all the currently pooled messages matching a filter id.
// 在本地指定某一id的filter过滤器，接着在envelop消息池中寻找符合该过滤器过滤规则的envelop信封(准确说是解包后的message符合规则)
// 将符合规定的message消息集合返回
func (w *Whisper) Messages(id int) []*Message {
	messages := make([]*Message, 0)
	if filter := w.filters.Get(id); filter != nil { //找到指定id的filter过滤器
		for _, envelope := range w.messages { //遍历整个envelop消息池
			if message := w.open(envelope); message != nil { //将envelop信封解包为message消息
				if w.filters.Match(filter, createFilter(message, envelope.Topics)) { //判断能否在本地能找到此消息对应的filter过滤器
					messages = append(messages, message) //如果能找到，将这些message消息作为返回值
				}
			}
		}
	}
	return messages
}

// handlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
// handpeer用于完成whisper连接(握手)，由P2P层各节点进行调用(在实现P2P连接的前提下进行whisper协议的部署实现)
func (w *Whisper) handlePeer(peer *server.Peer, rw server.MsgReadWriter, ctx context.Context) error {
	// Create the new peer and start tracking it
	whisperPeer := newPeer(w, peer, rw) //基于与对端peer的socket连接，创建一个whisper节点

	w.peerMu.Lock()
	w.peers[whisperPeer] = struct{}{} //初始化时，尚不存在与本节点实现whisper通信的其他活跃节点
	w.peerMu.Unlock()

	loglogrus.Log.Infof("与节点 (%x) 进行 whisper handshake!\n", whisperPeer.peer.ID())

	defer func() {
		w.peerMu.Lock()
		delete(w.peers, whisperPeer) //因错误连接中断，将对端whisper节点从本地peers集合中删除
		w.peerMu.Unlock()
		loglogrus.Log.Infof("与节点 (%x) 断开 whisper handshake!\n", whisperPeer.peer.ID())
	}()

	// Run the peer handshake and state updates
	if err := whisperPeer.handshake(); err != nil { //向对端节点发送whisper协议初始化状态消息，如果验证也没有出错，进入下一步
		loglogrus.Log.Warnf("无法发送whisper初始化消息,err:%v\n", err)
		return err
	}
	whisperPeer.start()      //开始更新peer节点known列表，定期向对方节点发送envelop信封
	defer whisperPeer.stop() //停止start

	// Read and process inbound messages directly to merge into client-global state
	//循环读取对端whisper节点的消息
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Fetch the next packet and decode the contained envelopes
			packet, err := rw.ReadMsg() //读取对方envelop信封
			if err != nil {
				return err
			}
			var envelopes []*Envelope
			if err := packet.Decode(&envelopes); err != nil { //将收到的一批envelop信封进行rlp解码
				loglogrus.Log.Warnf("%v: failed to decode envelope: %v", peer, err)
				continue
			}
			//
			loglogrus.Log.Debugf("当前节点 (%s) 接收到来自节点 (%s) 的whisper消息,envelope数量:%d\n", whisperPeer.peer.LocalAddr().String(), whisperPeer.peer.RemoteAddr().String(), len(envelopes))

			// Inject all envelopes into the internal pool
			for _, envelope := range envelopes { //将本次收到的所有envelop信封添加到当前whisper节点的消息池中
				// loglogrus.Log.Infof("Envelope.Data = %v\n", envelope.Data)
				if err := w.add(envelope); err != nil {
					// TODO Punish peer here. Invalid envelope.
					loglogrus.Log.Warnf("%v: failed to pool envelope: %v", peer, err)
				}
				whisperPeer.mark(envelope) //这些收到的envelop信封对于当前whisper节点来说是已知的(知道来源)，将其加入到节点的known集合中
			}
		}
	}
}

// add inserts a new envelope into the message pool to be distributed within the
// whisper network. It also inserts the envelope into the expiration pool at the
// appropriate time-stamp.
// 负责将一个需要进行发送的envelop信封添加到当前whisper客户端节点的消息池中
// 同时还需将该envelop信封添加到expirations池中，并在本地的一个协程中不断完成该envelop的发送与接收
func (w *Whisper) add(envelope *Envelope) error {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	// Insert the message into the tracked pool
	hash := envelope.Hash()            //获取哈希值
	if _, ok := w.messages[hash]; ok { //在节点当前的消息池中查找是否已经存在该Envelop信封(若存在直接退出，不需要add)
		loglogrus.Log.Infof("whisper envelope already cached: %x\n", hash)
		return nil
	}
	w.messages[hash] = envelope //如果不存在，将其加入到自己的消息池中

	// Insert the message into the expiration pool for later removal
	//expirations池中添加一个新的集合
	if w.expirations[envelope.Expiry] == nil {
		w.expirations[envelope.Expiry] = set.New(set.NonThreadSafe).(*set.SetNonTS)
	}
	//将当前消息加入上述集合
	if !w.expirations[envelope.Expiry].Has(hash) {
		w.expirations[envelope.Expiry].Add(hash)
		// Notify the local node of a message arrival
		go w.postEvent(envelope) //一个协程在本地创建对应该envelop信封filter过滤器
	}
	//loglogrus.Log.Infof("cached whisper envelope %x\n", envelope)

	return nil
}

type ConfigMsg struct {
	MsgCode uint64 //消息码
	Payload []byte //载荷实体
}

// 将形参传入的rlp字节流消息进行反序列化，得到ConfigMsg结构的消息
func DeSerializeConfigMsg(serialized []byte) (*ConfigMsg, error) {
	cm := &ConfigMsg{}
	rdr := bytes.NewReader(serialized)
	if err := rlp.Decode(rdr, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// postEvent opens an envelope with the configured identities and delivers the
// message upstream from application processing.
// 对envelop信封解包为Message消息(看是否能成功)，然后在本地为此message对应的envelop创建一个过滤器
func (w *Whisper) postEvent(envelope *Envelope) {
	if message := w.open(envelope); message != nil { //对envelope成功解包获得的message进行处理(将接收方设置为自己)
		w.filters.Notify(createFilter(message, envelope.Topics), message) //在本地为此message对应的envelop创建一个过滤器
	}
}

// open tries to decrypt a whisper envelope with all the configured identities,
// returning the decrypted message and the key used to achieve it. If not keys
// are configured, open will return the payload as if non encrypted.
// 对传入的envelope信封进行解包，获取Message。同时对Message进行Payload的解密
// 成功返回message，失败返回nil
func (w *Whisper) open(envelope *Envelope) *Message {
	// Short circuit if no identity is set, and assume clear-text

	if len(w.keys) == 0 { //本地没有保存任何公私钥对
		if message, err := envelope.Open(nil); err == nil {
			return message
		}
	}
	// Iterate over the keys and try to decrypt the message
	for _, key := range w.keys {
		message, err := envelope.Open(key) //用本地存储的所有私钥对尝试对当前envelope解包、解密变成message
		if err == nil {                    //解包、解密都成功，让message消息的To(接收方)字段修改为对应私钥的公钥
			message.To = &key.PublicKey //消息的接收者修改为当前节点(本节点即是发送者，也是接收者)
			return message
		} else if err == ecies.ErrInvalidPublicKey { //message消息并没有加密，直接返回解包后的message
			return message
		}
	}
	// Failed to decrypt, don't return anything
	return nil
}

// createFilter creates a message filter to check against installed handlers.
// 根据message消息包(to跟from)跟topics表集合创建并返回一个envelop信封过滤器
func createFilter(message *Message, topics []Topic) filter.Filter {
	matcher := make([][]Topic, len(topics))
	for i, topic := range topics {
		matcher[i] = []Topic{topic} //matcher的每一个元素都是一个topic表
	}
	return filterer{
		to:      string(crypto.FromECDSAPub(message.To)),
		from:    string(crypto.FromECDSAPub(message.Recover())), //根据message和哈希和数字签名获取发送方的公钥
		matcher: newTopicMatcher(matcher...),
	}
}

// update loops until the lifetime of the whisper node, updating its internal
// state by expiring stale messages from the pool.
func (w *Whisper) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expirationCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C: //定期检测过期envelop信封，将其从消息池和expirations集合中删除
			w.expire()

		case <-w.quit: //whisper对象被关闭，结束update
			return
		}
	}
}

// expire iterates over all the expiration timestamps, removing all stale
// messages from the pools.
// 将expirations集合中所有超时的envelop信封删除(同时也将其从消息池中删除)
func (w *Whisper) expire() {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	now := uint32(time.Now().Unix())
	for then, hashSet := range w.expirations { //遍历whisper对象expirations集合中所有envelop信封
		// Short circuit if a future time
		if then > now { //如果信封还没有超时，继续检测下一条envelop
			continue
		}
		// Dump all expired messages and remove timestamp
		hashSet.Each(func(v interface{}) bool { //将所有过期envelop信封从消息池中删除
			delete(w.messages, v.(common.Hash))
			return true
		})
		w.expirations[then].Clear() //将过期envelop信封从expirations集合删除
	}
}

// envelopes retrieves all the messages currently pooled by the node.
// 获取当前whisper节点消息池中的所有envelope消息
func (w *Whisper) envelopes() []*Envelope {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	envelopes := make([]*Envelope, 0, len(w.messages))
	for _, envelope := range w.messages {
		envelopes = append(envelopes, envelope)
	}
	return envelopes
}
