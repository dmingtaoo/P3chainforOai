package blockOrdering

import (
	"p3Chain/common"
	"testing"
)

func TestCircleTurns(t *testing.T) {
	testCircleTurns := NewCircleTurns()
	nodes := make([]common.NodeID, 10)
	for i := 0; i < len(nodes); i++ {
		temp := generateNodeID()
		nodes[i] = temp
		testCircleTurns.add(temp)
	}
	if testCircleTurns.length != 10 {
		t.Fatalf("fail in circleTurns add, length: %d", testCircleTurns.length)
	}
	acquire, err := testCircleTurns.getTurn()
	if err != nil {
		t.Fatalf("fail in circleTurns getTurn: %v", err)
	}
	if testCircleTurns.length != 10 {
		t.Fatalf("length changed, now is: %d", testCircleTurns.length)
	}
	if acquire != testCircleTurns.members[testCircleTurns.index-1] {
		t.Fatalf("fail in circleTurns get Turn")
	}
	for _, member := range nodes {
		testCircleTurns.remove(member)
	}
	if testCircleTurns.length != 0 {
		t.Fatalf("fail in remove")
	}
}
