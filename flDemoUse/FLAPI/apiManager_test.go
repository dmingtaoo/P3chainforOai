package FLAPI

import (
	"io/ioutil"
	"testing"
)

func TestFLManager_Evaluate(t *testing.T) {
	manager := NewFLManager(2)
	rawmodel1, err := ioutil.ReadFile("../testpara.pt")
	if err != nil {
		t.Fatalf("fail in read: %v", err)
	}
	//rawmodel2, err := ioutil.ReadFile("./gotpara.pt")
	//if err != nil {
	//	t.Fatalf("fail in read: %v",err)
	//}
	res, err := manager.Evaluate(rawmodel1)
	if err != nil {
		t.Fatalf("fail in evaluate: %v", err)
	}
	t.Logf("the evaluate res: %s", res)
}

func TestFLManager_ModelTrain(t *testing.T) {
	manager := NewFLManager(3)
	rawmodel1, err := ioutil.ReadFile("../testpara.pt")
	if err != nil {
		t.Fatalf("fail in read: %v", err)
	}
	newmodel, err := manager.ModelTrain(rawmodel1)
	if err != nil {
		t.Fatalf("fail in modeltrain: %v", err)
	}

	evalres, err := manager.Evaluate(newmodel)
	if err != nil {
		t.Fatalf("fail in evaluate: %v", err)
	}
	t.Logf("the evaluate res: %s", evalres)
}
