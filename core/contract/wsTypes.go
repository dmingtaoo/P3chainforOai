package contract

import (
	"p3Chain/common"
	"strconv"
)

const (
	type_domainInfo = byte(0x00)
	type_function   = byte(0x01)
	type_bytes      = byte(0x02)
	type_int64      = byte(0x03)
)

type worldStateValueType interface {
	backKind() byte
	toBytes() []byte
}
type WorldStateValueType interface {
	backKind() byte
	toBytes() []byte
	BackKind() byte
	ToBytes() []byte
}

// TODO: domain information type should help users to know how
// to use and manage a domain address, but now we just regard
// it as a description in plaintext
type domainInfo struct {
	plainValue []byte
}

func (di *domainInfo) backKind() byte {
	return type_domainInfo
}

func (di *domainInfo) toBytes() []byte {
	return di.plainValue
}

// TODO: in the future, function should support cascading calling
type contractFunction struct {
	do func(args [][]byte, self common.Address) ([][]byte, error)
}

func (cf *contractFunction) backKind() byte {
	return type_function
}

func (cf *contractFunction) toBytes() []byte {
	panic("to implement")
}

// type ContractFunction struct {
// 	do func(args [][]byte, self common.Address, pce *RemoteContractEngine) ([][]byte, error)
// }

// func (cf *ContractFunction) backKind() byte {
// 	return type_function
// }

// func (cf *ContractFunction) toBytes() []byte {
// 	panic("to implement")
// }

type commonBytes struct {
	value []byte
}

func (cb *commonBytes) backKind() byte {
	return type_bytes
}

func (cb *commonBytes) toBytes() []byte {
	return cb.value
}

type CommonBytes struct {
	Value []byte
}

func (cb *CommonBytes) BackKind() byte {
	return type_bytes
}

func (cb *CommonBytes) ToBytes() []byte {
	return cb.Value
}
func (cb *CommonBytes) backKind() byte {
	return type_bytes
}

func (cb *CommonBytes) toBytes() []byte {
	return cb.Value
}

type commonInt64 struct {
	value int64
}

func (c *commonInt64) backKind() byte {
	return type_int64
}

func (c *commonInt64) toBytes() []byte {
	str := strconv.Itoa(int(c.value))
	return []byte(str)
}
