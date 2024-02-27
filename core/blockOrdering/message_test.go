package blockOrdering

import (
	"p3Chain/common"
	"p3Chain/rlp"
	"reflect"
	"testing"
)

func TestCreateBlockOrderMsg(t *testing.T) {
	initMsg := &InitMessage{
		Version: common.Hash{},
		Height:  2,
	}
	bom, err := CreateBlockOrderMsg(initMsg, common.NodeID{}, CODE_INIT)
	if err != nil {
		t.Fatalf("fail in test: %v", err)
	}
	val, err := RetrievePayload(bom)
	if err != nil {
		t.Fatalf("fail in retrieve payload: %v", err)
	}
	reIniMsg, ok := val.(*InitMessage)
	if !ok {
		t.Fatalf("type is wrong")
	}
	if !reflect.DeepEqual(initMsg, reIniMsg) {
		t.Fatalf("Not equal!")
	}
}

func TestDeserializeBlockOrderMsg(t *testing.T) {
	initMsg := &InitMessage{
		Version: common.Hash{},
		Height:  0,
	}
	bom, err := CreateBlockOrderMsg(initMsg, common.NodeID{}, CODE_INIT)
	if err != nil {
		t.Fatalf("fail in test: %v", err)
	}

	payload, err := rlp.EncodeToBytes(bom)
	if err != nil {
		t.Fatalf("fail in encode: %v", err)
	}

	bom_, err := DeserializeBlockOrderMsg(payload)
	if err != nil {
		t.Fatalf("fail in decode: %v", err)
	}
	if !reflect.DeepEqual(bom, bom_) {
		t.Fatalf("Not equal!")
	}
}
