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

// Contains the Whisper protocol Message element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#messages.

package whisper

import (
	"crypto/ecdsa"
	"math/rand"
	"time"

	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/logger"
	"p3Chain/logger/glog"
)

// Message represents an end-user data packet to transmit through the Whisper
// protocol. These are wrapped into Envelopes that need not be understood by
// intermediate nodes, just forwarded.
// Message作为envelope消息的实际信息载荷部分
type Message struct {
	Flags     byte   // 此字节的第一位比特用于表示Message消息是否存在数字签名，其余比特位应是随机数
	Signature []byte //消息的数字签名
	Payload   []byte //实际的消息载荷

	Sent time.Time     // Time when the message was posted into the network  消息被发送到网络时的时间
	TTL  time.Duration // Maximum time to live allowed for the message

	To   *ecdsa.PublicKey // Message recipient (identity used to decode the message)  消息接收者的公钥
	Hash common.Hash      // Message envelope hash to act as a unique id   消息的哈希值
}

// Options specifies the exact way a message should be wrapped into an Envelope.
// 通过option参数控制Message消息发送时是否需要进行数字签名以及是否需要用公钥加密
type Options struct {
	From   *ecdsa.PrivateKey //发送方标注自己发送消息的私钥
	To     *ecdsa.PublicKey  //指明接收方身份(公钥)
	TTL    time.Duration
	Topics []Topic //Topic表
}

// NewMessage creates and initializes a non-signed, non-encrypted Whisper message.
// 初始化一条用于发送的Message(默认情况为无数字签名)
func NewMessage(payload []byte) *Message {
	// Construct an initial flag set: no signature, rest random
	flags := byte(rand.Intn(256))
	flags &= ^signatureFlag

	// Assemble and return the message
	return &Message{
		Flags:   flags,
		Payload: payload,
		Sent:    time.Now(),
	}
}

// Wrap bundles the message into an Envelope to transmit over the network.
//
// pow (Proof Of Work) controls how much time to spend on hashing the message,
// inherently controlling its priority through the network (smaller hash, bigger
// priority).
//
// The user can control the amount of identity, privacy and encryption through
// the options parameter as follows:
//   - options.From == nil && options.To == nil: anonymous broadcast  不签名，不加密，任何人都可以读取Message的内容
//   - options.From != nil && options.To == nil: signed broadcast (known sender)  有节点自身的数字签名，其他节点都可以接收，但无法伪造此节点身份
//   - options.From == nil && options.To != nil: encrypted anonymous message   使用接收方公钥对信息加密，则只有指定的接收方可以获取消息
//   - options.From != nil && options.To != nil: encrypted signed message  数字签名+公钥加密，消息的发送方和接收方都必须是固定的

// 负责将Message消息打包成envelope信封
func (self *Message) Wrap(pow time.Duration, options Options) (*Envelope, error) {
	// Use the default TTL if non was specified
	if options.TTL == 0 {
		options.TTL = DefaultTTL
	}
	self.TTL = options.TTL //确定Message消息的TTL

	// Sign and encrypt the message if requested
	//根据Message消息的flags字段确定是否需要进行数字签名
	if options.From != nil {
		if err := self.sign(options.From); err != nil { //用发送方私钥进行数字签名
			return nil, err
		}
	}
	//用接收方的公钥对Message消息的PayLoad字段进行加密
	if options.To != nil {
		if err := self.encrypt(options.To); err != nil { //用接收方公钥对Payload字段进行加密
			return nil, err
		}
	}

	// Wrap the processed message, seal it and return
	//利用当前Message消息和option合成一条envelope信封
	envelope := NewEnvelope(options.TTL, options.Topics, self)
	envelope.Seal(pow) //进行pow证明，envelope信封的哈希值受pow数值的影响，哈希值越小的envelope在网络中的优先级越高

	return envelope, nil
}

// sign calculates and sets the cryptographic signature for the message , also
// setting the sign flag.
// 判断Message消息是否需要进行数字签名，若需要则进行数字签名
func (self *Message) sign(key *ecdsa.PrivateKey) (err error) {
	self.Flags |= signatureFlag
	self.Signature, err = crypto.Sign(self.hash(), key)
	return
}

// Recover retrieves the public key of the message signer.
// 获取Message消息中的数字签名者的公钥(我们用签名者的私钥进行签名)
func (self *Message) Recover() *ecdsa.PublicKey {
	defer func() { recover() }() // in case of invalid signature

	// Short circuit if no signature is present
	if self.Signature == nil {
		return nil
	}
	// Otherwise try and recover the signature
	pub, err := crypto.SigToPub(self.hash(), self.Signature) //根据Message消息哈希值 跟 数字签名 获取公钥
	if err != nil {
		glog.V(logger.Error).Infof("Could not get public key from signature: %v", err)
		return nil
	}
	return pub //返回公钥
}

// encrypt encrypts a message payload with a public key.
// 使用Message接收方的公钥对发送的消息的PayLoad进行加密
func (self *Message) encrypt(key *ecdsa.PublicKey) (err error) {
	self.Payload, err = crypto.Encrypt(key, self.Payload)
	return
}

// decrypt decrypts an encrypted payload with a private key.
// 用传入的私钥对载荷部分进行解密
func (self *Message) decrypt(key *ecdsa.PrivateKey) error {
	cleartext, err := crypto.Decrypt(key, self.Payload)
	if err == nil {
		self.Payload = cleartext
	}
	return err
}

// hash calculates the SHA3 checksum of the message flags and payload.
// 计算Message消息 flags和payload合成字段的哈希值
func (self *Message) hash() []byte {
	return crypto.Sha3(append([]byte{self.Flags}, self.Payload...))
}

// bytes flattens the message contents (flags, signature and payload) into a
// single binary blob.
// 将Message消息的 flags, signature and payload 字段合入一个byte数组中
func (self *Message) bytes() []byte {
	return append([]byte{self.Flags}, append(self.Signature, self.Payload...)...)
}
