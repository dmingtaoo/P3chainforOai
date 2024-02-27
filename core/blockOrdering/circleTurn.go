package blockOrdering

import (
	"fmt"
	"p3Chain/common"
)

// help for order service require selection strategy.
type circleTurns struct {
	members []common.NodeID
	index   int
	length  int
}

func (ct *circleTurns) add(member common.NodeID) {
	for i := 0; i < len(ct.members); i++ {
		if ct.members[i] == member {
			return
		}
	}
	ct.members = append(ct.members, member)
	ct.length = len(ct.members)
}

func (ct *circleTurns) remove(member common.NodeID) {
	index := 0
	flag := false
	for ; index < len(ct.members); index++ {
		if ct.members[index] == member {
			flag = true
			break
		}
	}
	if !flag {
		return
	}
	ct.members = append(ct.members[:index], ct.members[index+1:]...)
	ct.length = len(ct.members)
	if ct.length == 0 {
		ct.index = 0
	} else {
		ct.index = ct.index % ct.length
	}

}

func (ct *circleTurns) getTurn() (common.NodeID, error) {
	nowTurn := common.NodeID{}
	if len(ct.members) == 0 {
		return nowTurn, fmt.Errorf("no members in circleTurns")
	}
	nowTurn = ct.members[ct.index]
	ct.index = (ct.index + 1) % ct.length
	return nowTurn, nil
}

func NewCircleTurns() circleTurns {
	ct := circleTurns{
		members: make([]common.NodeID, 0),
		index:   0,
		length:  0,
	}
	return ct
}
