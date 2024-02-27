package contract

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/crypto"
	"sort"
)

// read a function address and back its name
func readFunctionName(funcAddr common.Address) ([]byte, error) {
	name := make([]byte, len(funcAddr)-contract_ValueTypeCodeLength)
	if funcAddr[0] != contract_FunctionCode {
		return name, fmt.Errorf("not a function address")
	}
	copy(name, funcAddr[contract_ValueTypeCodeLength:])
	return name, nil
}

// parse a address with the given function name (first do hash)
func stringToFunctionAddress(name string) common.Address {
	nameHash := crypto.Sha3Hash([]byte(name))
	var funcAddr common.Address
	funcAddr[0] = contract_FunctionCode
	copy(funcAddr[contract_ValueTypeCodeLength:], nameHash[:])
	return funcAddr
}

// parse a address with the given function name (first do hash)
func StringToFunctionAddress(name string) common.Address {
	return stringToFunctionAddress(name)
}

// name should not over 19 bytes, or it will be cut off
func stringToContractAddress(name string) common.Address {
	nameBytes := []byte(name)
	var contractAddr common.Address
	contractAddr[0] = domain_ContractCode
	copy(contractAddr[domain_TypeCodeLength:], nameBytes)
	return contractAddr
}

// name should not over 19 bytes, or it will be cut off
func StringToContractAddress(name string) common.Address {
	return stringToContractAddress(name)
}

// name should not over 19 bytes, or it will be cut off
func stringToStorageAddress(name string) common.Address {
	nameBytes := []byte(name)
	var storageAddr common.Address
	storageAddr[0] = domain_StorageCode
	copy(storageAddr[domain_TypeCodeLength:], nameBytes)
	return storageAddr
}
func StringToStorageAddress(name string) common.Address {
	return stringToStorageAddress(name)
}

func bytesToHashAddress(aim []byte) common.Address {
	hash := crypto.Sha3Hash(aim)
	return common.HashToAddress(hash)
}

func BytesToHashAddress(aim []byte) common.Address {
	hash := crypto.Sha3Hash(aim)
	return common.HashToAddress(hash)
}

func SortWriteEles(writeSet []eles.WriteEle) []eles.WriteEle {
	sort.Slice(writeSet, func(i, j int) bool {
		return common.WsKeyLess(writeSet[i].ValueAddress, writeSet[j].ValueAddress)
	})
	return writeSet
}

func SortReadEles(readSet []eles.ReadEle) []eles.ReadEle {
	sort.Slice(readSet, func(i, j int) bool {
		return common.WsKeyLess(readSet[i].ValueAddress, readSet[j].ValueAddress)
	})
	return readSet
}
