package eles

func CreateInvalidTransactionReceipt() TransactionReceipt {
	invalidTxRpt := TransactionReceipt{
		Valid:  byte(0),
		Result: make([][]byte, 0),
	}
	return invalidTxRpt
}

func CreateValidTransactionReceipt(result [][]byte) TransactionReceipt {
	validTxRpt := TransactionReceipt{
		Valid:  byte(1),
		Result: result,
	}
	return validTxRpt
}
