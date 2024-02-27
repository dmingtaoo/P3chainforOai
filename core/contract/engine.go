package contract

import (
	"p3Chain/common"
	"p3Chain/core/eles"
)

type ContractEngine interface {
	ExecuteTransactions(txs []eles.Transaction) ([]eles.TransactionReceipt, []eles.WriteEle)
	CommitWriteSet(writeSet []eles.WriteEle) error
}

var (
	// base contract is the contract installed with the initialization of the p3Chain system,
	// it specify the control strategy of the whole system, and also handles for other
	// contracts or storage installation and updating.
	BASE_CONTRACT_ADDRESS = common.HexToAddress("0100000000000000000000000000000000000000")

	// base storage is a default address to store bytes values
	BASE_STORAGE_ADDRESS          = common.HexToAddress("0200000000000000000000000000000000000000")
	DEMO_CONTRACT_STORAGE_1       = stringToStorageAddress("DEMO_CONTRACT_STORAGE_1")
	DEMO_CONTRACT_STORAGE_GROUP   = stringToStorageAddress("DEMO_CONTRACT_STORAGE_GROUP")
	DEMO_CONTRACT_STORAGE_SUPPLY  = stringToStorageAddress("DEMO_CONTRACT_STORAGE_SUPPLY")
	DEMO_CONTRACT_STORAGE_FINANCE = stringToStorageAddress("DEMO_CONTRACT_STORAGE_FINANCE")
)

var (
	DEMO_CONTRACT_STORAGE_FL = stringToStorageAddress("DEMO_CONTRACT_STORAGE_FL")
)
