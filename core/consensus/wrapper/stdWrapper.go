package wrapper

import (
	"p3Chain/common"
	"p3Chain/core/contract"
	"p3Chain/core/eles"
	"p3Chain/core/worldstate"
)

type StdWrapper struct {
	stateManager worldstate.StateManager
}

func NewStdWrapper(sm worldstate.StateManager) *StdWrapper {
	return &StdWrapper{stateManager: sm}
}

type RWBox struct {
	sm        worldstate.StateManager
	originSet map[common.WsKey][]byte
	readSet   map[common.WsKey][]byte
	writeSet  map[common.WsKey][]byte
}

func NewRWBox(stateManger worldstate.StateManager) *RWBox {
	return &RWBox{
		sm:        stateManger,
		originSet: make(map[common.WsKey][]byte),
		readSet:   make(map[common.WsKey][]byte),
		writeSet:  make(map[common.WsKey][]byte),
	}
}

func (rb RWBox) Get(wsKey common.WsKey) ([]byte, error) {
	if ele, ok := rb.readSet[wsKey]; ok {
		return ele, nil
	}

	plainTexts, err := rb.sm.ReadWorldStateValues(wsKey)
	if err != nil {
		return nil, err
	}

	plainText := plainTexts[0]

	rb.originSet[wsKey] = plainText
	rb.readSet[wsKey] = plainText

	return plainText, nil
}

func (rb RWBox) Set(wsKey common.WsKey, value []byte) {
	rb.writeSet[wsKey] = value
	rb.readSet[wsKey] = value
}

func (rb *RWBox) ExportWriteList() []eles.WriteEle {
	writeSet := make([]eles.WriteEle, 0)
	for wsKey, ele := range rb.writeSet {
		we := eles.WriteEle{
			ValueAddress: wsKey,
			Value:        ele,
		}
		writeSet = append(writeSet, we)
	}
	writeSet = contract.SortWriteEles(writeSet)
	return writeSet
}

func (rb *RWBox) ExportReadList() []eles.ReadEle {
	readSet := make([]eles.ReadEle, 0)
	for wsKey, ele := range rb.originSet {
		re := eles.ReadEle{
			ValueAddress: wsKey,
			Value:        ele,
		}
		readSet = append(readSet, re)
	}
	readSet = contract.SortReadEles(readSet)
	return readSet
}

func (sw *StdWrapper) Execute(executeFunc func(rb *RWBox, args [][]byte) ([][]byte, error), args [][]byte) (ws []eles.WriteEle, rs []eles.ReadEle, res [][]byte, err error) {
	rb := NewRWBox(sw.stateManager)
	res, err = executeFunc(rb, args)
	if err != nil {
		return nil, nil, nil, err
	}
	ws = rb.ExportWriteList()
	rs = rb.ExportReadList()
	return
}
