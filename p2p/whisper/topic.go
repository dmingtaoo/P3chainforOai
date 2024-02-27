// Copyright 2015 The go-ethereum Authors
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

// Contains the Whisper protocol Topic element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#topics.

package whisper

import "p3Chain/crypto"

// Topic represents a cryptographically secure, probabilistic partial
// classifications of a message, determined as the first (left) 4 bytes of the
// SHA3 hash of some arbitrary data given by the original author of the message.

//Topic 中存储着消息的发送者用一些任意数据求出的的SHA3哈希的前4个字节。
type Topic [4]byte

// NewTopic creates a topic from the 4 byte prefix of the SHA3 hash of the data.
//
// Note, empty topics are considered the wildcard, and cannot be used in messages.

//将传入的数据(byte数组)的前4个字节取出并求SHA-3哈希作为Topic字段
func NewTopic(data []byte) Topic {
	prefix := [4]byte{}
	copy(prefix[:], crypto.Sha3(data)[:4])
	return Topic(prefix)
}

// NewTopics creates a list of topics from a list of binary data elements, by
// iteratively calling NewTopic on each of them.

//将一组传入的数据按照迭代循环的方式依次采用NewTopic()进行修改
func NewTopics(data ...[]byte) []Topic {
	topics := make([]Topic, len(data))
	for i, element := range data {
		topics[i] = NewTopic(element)
	}
	return topics
}

// NewTopicFromString creates a topic using the binary data contents of the
// specified string.

//把传入的字符串转换成byte数组，调用NewTopic()进行修改
func NewTopicFromString(data string) Topic {
	return NewTopic([]byte(data))
}

// NewTopicsFromStrings creates a list of topics from a list of textual data
// elements, by iteratively calling NewTopicFromString on each of them.

//把传入的字符串数组，按照迭代循环的方式调用NewTopicFromString()进行修改
func NewTopicsFromStrings(data ...string) []Topic {
	topics := make([]Topic, len(data))
	for i, element := range data {
		topics[i] = NewTopicFromString(element)
	}
	return topics
}

// String converts a topic byte array to a string representation.

//把Topic对象从4字节Byte数据转换成字符串形式
func (self *Topic) String() string {
	return string(self[:])
}

// topicMatcher is a filter expression to verify if a list of topics contained
// in an arriving message matches some topic conditions. The topic matcher is
// built up of a list of conditions, each of which must be satisfied by the
// corresponding topic in the message. Each condition may require: a) an exact
// topic match; b) a match from a set of topics; or c) a wild-card matching all.
//
// If a message contains more topics than required by the matcher, those beyond
// the condition count are ignored and assumed to match.
//
// Consider the following sample topic matcher:
//   sample := {
//     {TopicA1, TopicA2, TopicA3},
//     {TopicB},
//     nil,
//     {TopicD1, TopicD2}
//   }
// In order for a message to pass this filter, it should enumerate at least 4
// topics, the first any of [TopicA1, TopicA2, TopicA3], the second mandatory
// "TopicB", the third is ignored by the filter and the fourth either "TopicD1"
// or "TopicD2". If the message contains further topics, the filter will match
// them too.

//topicMatcher负责对接收到的消息进行条件过滤，具体来说这些消息必须有符合topicMatcher规定的topic字段
type topicMatcher struct {
	conditions []map[Topic]struct{} //存储本topicMatcher对象所规定的topic字段(topic表)
}

// newTopicMatcher create a topic matcher from a list of topic conditions.
//创建一个新的topicMatcher(根据传入的Topic切片)
func newTopicMatcher(topics ...[]Topic) *topicMatcher {
	matcher := make([]map[Topic]struct{}, len(topics)) //获取本topicMatcher需要包含的topic(创建map切片)
	for i, condition := range topics {
		matcher[i] = make(map[Topic]struct{}) //为每一个map开辟空间
		for _, topic := range condition {
			matcher[i][topic] = struct{}{} //matcher注册所有需要的topic
		}
	}
	return &topicMatcher{conditions: matcher} //返回一个新的包含了规定topic表的topicMatcher
}

// newTopicMatcherFromBinary create a topic matcher from a list of binary conditions.
//根据传入的切片数组(数组每个成员是一个切片)创建一个新的topicMatcher
func newTopicMatcherFromBinary(data ...[][]byte) *topicMatcher {
	topics := make([][]Topic, len(data))
	for i, condition := range data {
		topics[i] = NewTopics(condition...) //每个元素是一个Topic
	}
	return newTopicMatcher(topics...)
}

// newTopicMatcherFromStrings creates a topic matcher from a list of textual
// conditions.
//根据传入的字符串数组(数组每一个成员是一个字符串)创建一个新的topicMatcher
func newTopicMatcherFromStrings(data ...[]string) *topicMatcher {
	topics := make([][]Topic, len(data))
	for i, condition := range data {
		topics[i] = NewTopicsFromStrings(condition...) //每个元素是一个Topic
	}
	return newTopicMatcher(topics...)
}

// Matches checks if a list of topics matches this particular condition set.

//形参是从接收的消息中获取的Topic表，将其与本地创建的topicMatcher进行比较
//1. 如果Topic表中的topic数量不够，则不匹配
//2. 如果本地TopicMatcher有的，但传入Topic表没有，则不匹配
func (self *topicMatcher) Matches(topics []Topic) bool {
	// Mismatch if there aren't enough topics
	if len(self.conditions) > len(topics) {
		return false
	}
	// Check each topic condition for existence (skip wild-cards)
	for i := 0; i < len(topics) && i < len(self.conditions); i++ {
		if len(self.conditions[i]) > 0 {
			if _, ok := self.conditions[i][topics[i]]; !ok {
				return false
			}
		}
	}
	return true
}
