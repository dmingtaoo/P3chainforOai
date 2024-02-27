package contract

import (
	"p3Chain/common"
	"testing"
)

func TestStringToContractAddress(t *testing.T) {
	contractAddr := stringToContractAddress("test")
	t.Logf("generated contract address: %x", contractAddr)
	if contractAddr != common.HexToAddress("0174657374000000000000000000000000000000") {
		t.Fatalf("not equal")
	}
}

func TestStringToStorageAddress(t *testing.T) {
	storageAddr := stringToStorageAddress("test")
	t.Logf("generated contract address: %x", storageAddr)
	if storageAddr != common.HexToAddress("0274657374000000000000000000000000000000") {
		t.Fatalf("not equal")
	}
}
