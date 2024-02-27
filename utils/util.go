// Package utils implements some useful functions.
package utils

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"p3Chain/common"
	"p3Chain/crypto"
	curve "p3Chain/crypto/secp256k1"
	loglogrus "p3Chain/log_logrus"
	"p3Chain/p2p/discover"
	"strings"
)

func SaveKey(key ecdsa.PrivateKey, path string) error {
	if common.FileExist(path) {
		return errors.New("error in Key save: key file has already exist")
	}
	var content bytes.Buffer
	gob.Register(curve.S256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(key)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, content.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}

func LoadKey(path string) (ecdsa.PrivateKey, error) {
	var key ecdsa.PrivateKey
	if !common.FileExist(path) {
		return key, errors.New("error in Key load: no such file")
	}
	gob.Register(curve.S256())
	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return key, err
	}
	decoder := gob.NewDecoder(bytes.NewBuffer(fileContent))
	err = decoder.Decode(&key)
	if err != nil {
		return key, err
	}
	key.Curve = curve.S256()
	return key, nil
}

func DecodePublicKey(pbkStr string) (ecdsa.PublicKey, error) {
	var pbKey ecdsa.PublicKey
	gob.Register(curve.S256())
	decoder := gob.NewDecoder(bytes.NewBuffer([]byte(pbkStr)))
	err := decoder.Decode(&pbKey)
	if err != nil {
		return pbKey, err
	}
	pbKey.Curve = curve.S256()

	return pbKey, nil
}

func EncodePublicKey(pbKey ecdsa.PublicKey) (string, error) {
	var pbkStr string
	var content bytes.Buffer
	gob.Register(curve.S256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(pbKey)
	if err != nil {
		return pbkStr, err
	}
	pbkStr = content.String()
	return pbkStr, nil
}

func SaveSelfPublicKey(pbKey ecdsa.PublicKey, path string) error {
	pbkString, err := EncodePublicKey(pbKey)
	if err != nil {
		return err
	}
	err = SaveString(pbkString, path)
	if err != nil {
		return err
	}
	return nil
}

func LoadPublicKey(path string) (ecdsa.PublicKey, error) {
	var key ecdsa.PublicKey
	if !common.FileExist(path) {
		return key, errors.New("error in Key load: no such file")
	}
	gob.Register(curve.S256())
	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return key, err
	}
	decoder := gob.NewDecoder(bytes.NewBuffer(fileContent))
	err = decoder.Decode(&key)
	if err != nil {
		return key, err
	}
	key.Curve = curve.S256()

	return key, nil
}

func LoadPublicKeyDir(dirPath string) ([]string, []ecdsa.PublicKey, error) {
	var names []string
	var pubkeys []ecdsa.PublicKey

	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return names, pubkeys, err
	}
	PthSep := string(os.PathSeparator)
	pbksuffix := ".pubkey"
	for _, fi := range dir {
		if fi.IsDir() {
			return names, pubkeys, errors.New("error in LoadPublicKey: has subdir in pubkey dir")
		} else {
			ok := strings.HasSuffix(fi.Name(), pbksuffix)
			if ok {
				filepath := dirPath + PthSep + fi.Name()
				pk, err := LoadPublicKey(filepath)
				if err != nil {
					return names, pubkeys, err
				}
				pubkeys = append(pubkeys, pk)
				names = append(names, strings.TrimSuffix(fi.Name(), pbksuffix))
			}
		}
	}

	return names, pubkeys, nil

}

func DecodePrivateKey(prvStr string) (ecdsa.PrivateKey, error) {
	var prvKey ecdsa.PrivateKey
	gob.Register(curve.S256())
	decoder := gob.NewDecoder(bytes.NewBuffer([]byte(prvStr)))
	err := decoder.Decode(&prvKey)
	if err != nil {
		return prvKey, err
	}
	prvKey.Curve = curve.S256()
	return prvKey, nil
}

// check whether the dictionary is exist or not, if not exist then create one
func DirCheck(dirPath string) error {
	if common.FileExist(dirPath) {
		return nil
	} else {
		err := os.MkdirAll(dirPath, 0755)
		return err
	}
}

func SaveSelfUrl(selfUrl string, path string) error {
	return SaveString(selfUrl, path)
}

func SaveString(str string, path string) error {
	if common.FileExist(path) {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}
	err := ioutil.WriteFile(path, []byte(str), 0644)
	return err
}

func ReadNodesUrl(path string) ([]*discover.Node, error) {
	nodes := []*discover.Node{}
	if !common.FileExist(path) {
		return nodes, nil
	}
	fi, err := os.Open(path)
	if err != nil {
		return nodes, err
	}
	defer fi.Close()

	br := bufio.NewReader(fi)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		n, err := discover.ParseNode(string(a))
		if err != nil {
			return nodes, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func GenerateNodeUrl(nodeID common.NodeID, laddr string) (string, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return "", err
	}

	node := &discover.Node{
		ID:  discover.NodeID(nodeID),
		IP:  addr.IP,
		TCP: uint16(addr.Port),
		UDP: uint16(addr.Port),
	}
	url := node.String()
	return url, nil
}

func GenerateNodeAddress(savePath string) (common.Address, *ecdsa.PrivateKey, error) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		loglogrus.Log.Debugf("Generate Private Key is failed:%v\n", err)
		return common.Address{}, nil, err
	}

	SaveSelfPublicKey(prvKey.PublicKey, savePath)
	address := crypto.PubkeyToAddress(prvKey.PublicKey)

	return address, prvKey, nil
}

func LoadNodeAddress(loadPath string) (common.Address, error) {

	pub, err := LoadPublicKey(loadPath)
	if err != nil {
		loglogrus.Log.Debugf("Load Public Key is failed:%v\n", err)
		return common.Address{}, err
	}

	address := crypto.PubkeyToAddress(pub)
	return address, nil
}
