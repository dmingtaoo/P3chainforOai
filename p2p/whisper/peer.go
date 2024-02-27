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
	"fmt"
	"time"

	"p3Chain/common"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/logger"
	"p3Chain/logger/glog"
	"p3Chain/rlp"

	"p3Chain/p2p/server"

	"gopkg.in/fatih/set.v0"
)

// peer represents a whisper protocol peer connection.
type peer struct {
	host *Whisper             //本地的whisper客户端主体对象
	peer *server.Peer         //与某一个对等节点的服务器连接对象
	ws   server.MsgReadWriter //与该对等节点的连接读写器

	known *set.Set // 存放所有whisper通信过程中接收过的envelop信封(也就是从host字段对应whisper节点处收到的envelop信封)

	quit chan struct{}
}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
// 根据与某一个对等节点的P2P连接，在本地创建一个whisper节点对象
func newPeer(host *Whisper, remote *server.Peer, rw server.MsgReadWriter) *peer {
	return &peer{
		host:  host,
		peer:  remote,
		ws:    rw,
		known: set.New(set.ThreadSafe).(*set.Set), //存放所有whisper通信过程中接收过的envelop信封(也就是从host字段对应whisper节点处收到的envelop信封)
		quit:  make(chan struct{}),
	}
}

// start initiates the peer updater, periodically broadcasting the whisper packets
// into the network.
func (self *peer) start() {
	go self.update() //定时更新known列表，定时向对端peer发送消息
	glog.V(logger.Debug).Infof("%v: whisper started", self.peer)
}

// stop terminates the peer updater, stopping message forwarding to it.
// 关闭本地的whisper对象
func (self *peer) stop() {
	close(self.quit) //结束self.update()所在协程，关闭whisper客户端
	glog.V(logger.Debug).Infof("%v: whisper stopped", self.peer)
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
// 向远端peer发送whisper协议初始化状态消息，同时验证远端peer的状态
func (self *peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		errc <- server.SendItems(self.ws, statusCode, protocolVersion) //向对方节点发送协议状态
	}()
	// Fetch the remote status packet and verify protocol match
	packet, err := self.ws.ReadMsg() //等待接收对方peer回复的packet数据包
	if err != nil {
		return err
	}
	//对回复的packet数据包进行验证，若没有错误返回nil,则表示协议通信成功
	if packet.Code != statusCode {
		return fmt.Errorf("peer sent %x before status packet", packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	if _, err := s.List(); err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	if peerVersion != protocolVersion {
		return fmt.Errorf("protocol version mismatch %d != %d", peerVersion, protocolVersion)
	}
	// Wait until out own status is consumed too
	if err := <-errc; err != nil {
		return fmt.Errorf("failed to send status packet: %v", err)
	}
	return nil
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (self *peer) update() {
	// Start the tickers for the updates
	expire := time.NewTicker(expirationCycle)
	transmit := time.NewTicker(transmissionCycle)

	// Loop and transmit until termination is requested
	for {
		select {
		case <-expire.C:
			self.expire() //定时将过期消息从known列表中删除

		case <-transmit.C: //定时向对方节点发送 在envelope消息池，但不在known列表中的 消息(不在known列表意味着对方节点从未有过该envelop信封)
			if err := self.broadcast(); err != nil {
				loglogrus.Log.Warnf("(%s) broadcast to (%s) failed: %v", self.peer.LocalAddr().String(), self.peer.RemoteAddr().String(), err)
				return
			}

		case <-self.quit: //whisper客户端关闭，不再进行update
			return
		}
	}
}

// mark marks an envelope known to the peer so that it won't be sent back.
// 将传入的消息的哈希值存入known列表中
func (self *peer) mark(envelope *Envelope) {
	self.known.Add(envelope.Hash())
}

// marked checks if an envelope is already known to the remote peer.
// 如果envelope消息的哈希出现在known列表中，则表明消息来源是已知的(也就是已经被marked)
func (self *peer) marked(envelope *Envelope) bool {
	return self.known.Has(envelope.Hash())
}

// expire iterates over all the known envelopes in the host and removes all
// expired (unknown) ones from the known list.
// 负责遍历当前节点所有已知消息，删除known列表中所有过期的消息(messages消息池有定时清除功能，known列表的即用此方法实现)
func (self *peer) expire() {
	// Assemble the list of available envelopes
	available := set.New(set.NonThreadSafe).(*set.SetNonTS)
	for _, envelope := range self.host.envelopes() { //获取当前whisper节点消息池中的所有envelope(消息池有定时清除过期消息的功能)
		available.Add(envelope.Hash())
	}
	// Cross reference availability with known status
	unmark := make(map[common.Hash]struct{}) //存储所有Messages消息池中过期的envelope消息
	self.known.Each(func(v interface{}) bool {
		if !available.Has(v.(common.Hash)) { //遍历找出所有在known中存在，但不在消息池中的消息(也就是在messages消息池中已经过期的消息)
			unmark[v.(common.Hash)] = struct{}{}
		}
		return true
	})
	// Dump all known but unavailable
	for hash, _ := range unmark { //这些尚未被mark的消息被视为过期消息，将其从known中移除
		self.known.Remove(hash)
	}
}

// broadcast iterates over the collection of envelopes and transmits yet unknown
// ones over the network.
// 将messages消息池中所有不在known列表中的消息发送给对端节点
func (self *peer) broadcast() error {
	// Fetch the envelopes and collect the unknown ones
	envelopes := self.host.envelopes()               //获取当前节点消息池中的所有消息
	transmit := make([]*Envelope, 0, len(envelopes)) //存储消息池中所有尚未被加入到known列表中的envelope
	for _, envelope := range envelopes {
		if !self.marked(envelope) { //检查当前envelope是否已经被加入known列表中(没有的话将其添加到known列表同时作为待发送数据)
			transmit = append(transmit, envelope) //存入到transmit
			self.mark(envelope)                   //加入到known列表

		}
	}
	// Transmit the unknown batch (potentially empty)
	//调用server模块，将所有envelope通过与对端节点的读写器发送出去
	if err := server.Send(self.ws, messagesCode, transmit); err != nil {
		return err
	}

	localAddr := self.peer.LocalAddr().String()
	remoteAddr := self.peer.RemoteAddr().String()

	loglogrus.Log.Debugf("whisper: 当前节点(%s)成功发送%d条envelope给目标(%s)\n", localAddr, len(transmit), remoteAddr)
	return nil
}
