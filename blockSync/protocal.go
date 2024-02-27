package blockSync

import (
	"p3Chain/common"
	"p3Chain/core/eles"
)

const (
	NetworkId          = 1
	ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message
)

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
)

// statusData is the network packet for the status message.
type statusData struct {
	ProtocolVersion uint32
	SubNet          string
	SelfHeight      uint64
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
}

// getBlockHashesData is the network packet for the hash based block retrieval
// message.
type getBlockHashesData struct {
	Hash   common.Hash
	Amount uint64
}

// getBlockHashesFromNumberData is the network packet for the number based block
// retrieval message.
type getBlockHashesFromNumberData struct {
	Number uint64
	Amount uint64
}

// newBlockData is the network packet for the block propagation message.
type newBlockData struct {
	Block        *eles.WrapBlock
	RemoteHeight uint64
}

type chainHeightData struct {
	Height uint64
	Head   common.Hash
}
