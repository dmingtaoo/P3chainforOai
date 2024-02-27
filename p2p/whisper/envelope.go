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

// Contains the Whisper protocol Envelope element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#envelopes.

package whisper

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/crypto/ecies"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/rlp"
)

// Envelope represents a clear-text data packet to transmit through the Whisper
// network. Its contents may or may not be encrypted and signed.
// Envelope表示通过Whisper网络传输的明文数据包。其内容可能有也可能没有进行加密和签名。
type Envelope struct {
	Expiry uint32  // 消息的有效时间
	TTL    uint32  // TTL
	Topics []Topic //消息中含有的Topic表
	Data   []byte  //发送的数据，存储[]byte话的Message消息
	Nonce  uint32  //随机数

	hash common.Hash // Cached hash of the envelope to avoid rehashing every time
}

// NewEnvelope wraps a Whisper message with expiration and destination data
// included into an envelope for network forwarding.

// 将要发送的message消息打包成一个Envelope(消息实体msg , topic表)
func NewEnvelope(ttl time.Duration, topics []Topic, msg *Message) *Envelope {
	return &Envelope{
		Expiry: uint32(time.Now().Add(ttl).Unix()),
		TTL:    uint32(ttl.Seconds()),
		Topics: topics,
		Data:   msg.bytes(),
		Nonce:  0,
	}
}

// Seal closes the envelope by spending the requested amount of time as a proof
// of work on hashing the data.
// 通过花费自身的计算能力，获取符合条件的nonce值
// envelope信封的哈希值受pow数值的影响，哈希值越小的envelope在网络中的优先级越高
func (self *Envelope) Seal(pow time.Duration) {
	d := make([]byte, 64) //存储envelope的rlp编码
	copy(d[:32], self.rlpWithoutNonce())

	finish, bestBit := time.Now().Add(pow).UnixNano(), 0 //在pow规定的时间内，不断尝试nonce值
	for nonce := uint32(0); time.Now().UnixNano() < finish; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint32(d[60:], nonce)

			firstBit := common.FirstBitSet(common.BigD(crypto.Sha3(d)))
			if firstBit > bestBit {
				self.Nonce, bestBit = nonce, firstBit //当计算负荷条件时，更改envelope的nonce字段
			}
			nonce++
		}
	}
}

// rlpWithoutNonce returns the RLP encoded envelope contents, except the nonce.
// 将envelope进行rlp编码(除了nonce字段)后返回
func (self *Envelope) rlpWithoutNonce() []byte {
	enc, _ := rlp.EncodeToBytes([]interface{}{self.Expiry, self.TTL, self.Topics, self.Data})
	return enc
}

// Open extracts the message contained within a potentially encrypted envelope.
// 将envelope信封解包成Message数据包(用自己的私钥对message消息的payload进行解密)
func (self *Envelope) Open(key *ecdsa.PrivateKey) (msg *Message, err error) {
	// Split open the payload into a message construct
	data := self.Data //message消息合成envelope信封时，是按照相应顺序将所有字段转化为[]byte(顺序为：Flags、Signature、Payload)

	message := &Message{
		Flags: data[0], //第一个字节正是message的Falgs字段
		Sent:  time.Unix(int64(self.Expiry-self.TTL), 0),
		TTL:   time.Duration(self.TTL) * time.Second,
		Hash:  self.Hash(),
	}
	data = data[1:] //Signature和Payload

	if message.Flags&signatureFlag == signatureFlag { //确定Message数据包是否包含了数字签名
		if len(data) < signatureLength {
			loglogrus.Log.Warnf("Whisper Open Envelope:  unable to open envelope. First bit set but len(data) < len(signature)\n")
			return nil, fmt.Errorf("unable to open envelope. First bit set but len(data) < len(signature)")
		}
		message.Signature, data = data[:signatureLength], data[signatureLength:] //前半部分是数字签名，后半部分是payload
	}
	message.Payload = data //没有数字签名时，剩下的都是payload

	if key == nil {
		return message, nil //如果没有传入私钥，直接将解包后的message返回
	}
	err = message.decrypt(key) //否则，用传入的私钥对message的Payload进行解密
	switch err {
	case nil:
		return message, nil //解密成功的话，返回message

	case ecies.ErrInvalidPublicKey: //说明Payload实际上并没有被加密
		return message, err

	default: //解密失败，说明私钥不匹配
		return nil, fmt.Errorf("unable to open envelope, decrypt failed: %v", err)
	}
}

// Hash returns the SHA3 hash of the envelope, calculating it if not yet done.
// 计算Envelope rlp加密之后的SHA-3哈希值，存放到Envelope的hash字段
func (self *Envelope) Hash() common.Hash {
	if (self.hash == common.Hash{}) {
		enc, _ := rlp.EncodeToBytes(self)
		self.hash = crypto.Sha3Hash(enc)
	}
	return self.hash
}

// DecodeRLP decodes an Envelope from an RLP data stream.
func (self *Envelope) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw() //从rlp编码流中读取数据
	if err != nil {
		return err
	}
	// The decoding of Envelope uses the struct fields but also needs
	// to compute the hash of the whole RLP-encoded envelope. This
	// type has the same structure as Envelope but is not an
	// rlp.Decoder so we can reuse the Envelope struct definition.
	type rlpenv Envelope
	if err := rlp.DecodeBytes(raw, (*rlpenv)(self)); err != nil { //对rlp编码流进行解码，解密后的数据存到self(当前Envelop对象)
		return err
	}
	self.hash = crypto.Sha3Hash(raw) //self的哈希值需要用rlp编码流进行计算
	return nil
}
