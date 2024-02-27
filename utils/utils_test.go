package utils

import (
	"os"
	"p3Chain/crypto"
	"testing"
)

const (
	PRV_TEST = "./prv.key"
	PUB_TEST = "./pub.pubkey"
)

func TestSaveLoadKeyPairs(t *testing.T) {
	os.Remove(PRV_TEST)
	os.Remove(PUB_TEST)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("error in generate key: %v\n", err)
	}
	err = SaveKey(*key, PRV_TEST)
	if err != nil {
		t.Fatalf("error in save prvkey: %v\n", err)
	}
	err = SaveSelfPublicKey(key.PublicKey, PUB_TEST)
	if err != nil {
		t.Fatalf("error in save pubkey: %v\n", err)
	}

	lprvk, err := LoadKey(PRV_TEST)
	if err != nil {
		t.Fatalf("error in load prvkey: %v\n", err)
	}

	lpubk, err := LoadPublicKey(PUB_TEST)
	if err != nil {
		t.Fatalf("error in load pubkey: %v\n", err)
	}

	if !(lprvk.Curve == key.Curve) {
		t.Fatalf("prvkey: load curve not right")
	}

	if !(lpubk.Curve == key.PublicKey.Curve) {
		t.Fatalf("pubey: load curve not right")
	}
}
