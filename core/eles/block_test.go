package eles

import (
	"p3Chain/common"
	"testing"
)

func TestBlockHash(t *testing.T) {
	tblock := &Block{
		BlockID:      common.StringToHash("test"),
		Subnet:       []byte("test net"),
		Leader:       common.StringToAddress("test leader"),
		Version:      common.StringToHash("test version"),
		Nonce:        uint8(0),
		Transactions: make([]Transaction, 0),
		SubnetVotes:  make([]SubNetSignature, 0),
	}
	hash, err := tblock.Hash()
	if err != nil {
		t.Fatalf("fail in hash: %v", err)
	}
	if tblock.BlockID != common.StringToHash("test") {
		t.Fatalf("hash function changes the BlockID")
	}
	tblock.BlockID = common.StringToHash("changed")
	changedHash, err := tblock.Hash()
	if err != nil {
		t.Fatalf("fail in hash: %v", err)
	}
	if changedHash != hash {
		t.Fatalf("Changed BlockID result in different hash")
	}

}

func TestBlockReceiptID(t *testing.T) {
	tblock := &Block{
		BlockID:      common.StringToHash("test"),
		Subnet:       []byte("test net"),
		Leader:       common.StringToAddress("test leader"),
		Version:      common.StringToHash("test version"),
		Nonce:        uint8(0),
		Transactions: make([]Transaction, 0),
		SubnetVotes:  make([]SubNetSignature, 0),
		PrevBlock:    common.StringToHash("previous"),
	}
	tblock.ComputeBlockID()
	receiptID1, err := tblock.ComputeReceiptID()
	if err != nil {
		t.Fatalf("fail in compute receipt ID: %v", err)
	}
	receiptID2, err := tblock.BackReceiptID()
	if err != nil {
		t.Fatalf("fail in back receipt ID: %v", err)
	}
	if receiptID1 != receiptID2 {
		t.Fatalf("one block has more than one block receipt ID, receipt 1: %x, receipt 2: %x", receiptID1, receiptID2)
	}

	if receiptID2 != tblock.Receipt.ReceiptID {
		t.Fatalf("fail in setting receipt")
	}
}
