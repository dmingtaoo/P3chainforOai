package contract

import (
	"crypto/ecdsa"
	"fmt"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/core/worldstate"
	"p3Chain/crypto"
	"p3Chain/rlp"
	"testing"
)

var (
	TEST_VERSION  = common.StringToHash("CONTRACT TEST")
	TEST_CONTRACT = stringToContractAddress("DEMO CONTRACT")

	// 0x00 of inner address is to specify the domain address
	TEST_CONTRACT_WSKEY       = common.AddressesCatenateWsKey(TEST_CONTRACT, domainInfoAddress)
	TEST_CONTRACT_DESCRIPTION = []byte("DEMO CONTRACT IS ON")

	TEST_FUNCTION_GET = stringToFunctionAddress("get")
	TEST_FUNCTION_SET = stringToFunctionAddress("set")

	TEST_FUNCTION_GROUP_CONFIG  = stringToFunctionAddress("group_config")
	TEST_FUNCTION_GROUP_SET     = stringToFunctionAddress("group_set")
	TEST_FUNCTION_GROUP_SEND    = stringToFunctionAddress("group_send")
	TEST_FUNCTION_GROUP_BALANCE = stringToFunctionAddress("group_balance")

	TEST_FUNCTION_SUPPLY_CREATE = stringToFunctionAddress("supply_create")

	TEST_FUNCTION_SUPPLY_SET_PURCHASEFORM = stringToFunctionAddress("supply_set_purchaseform")
	TEST_FUNCTION_SUPPLY_GET_PURCHASEFORM = stringToFunctionAddress("supply_get_purchaseform") //args:id

	TEST_FUNCTION_SUPPLY_SET_RECEIPTFORM = stringToFunctionAddress("supply_set_receipt")
	TEST_FUNCTION_SUPPLY_GET_RECEIPTFORM = stringToFunctionAddress("supply_get_receipt")

	TEST_FUNCTION_SUPPLY_GET_PROCESSFORM  = stringToFunctionAddress("supply_get_processform")
	TEST_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE = stringToFunctionAddress("supply_sign_credit_side")
	TEST_FUNCTION_SUPPLY_SIGN_PAY         = stringToFunctionAddress("supply_sign_pay")
	TEST_FUNCTION_SUPPLY_DELIVERY         = stringToFunctionAddress("supply_delivery")
	TEST_FUNCTION_SUPPLY_RECEIVE          = stringToFunctionAddress("supply_receive")
)

func generateDemoTX(contractAddr common.Address, functionAddr common.Address, args [][]byte, checkList []eles.CheckElement) eles.Transaction {
	senderPrvKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	tx := eles.Transaction{
		Sender:   crypto.PubkeyToAddress(senderPrvKey.PublicKey),
		Nonce:    0,
		Version:  TEST_VERSION,
		LifeTime: 0,

		Contract:  contractAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: checkList,
	}
	tx.SetTxID()
	signature, err := crypto.SignHash(tx.TxID, senderPrvKey)
	if err != nil {
		panic(err)
	}
	tx.Signature = signature
	return tx
}

// use the specified private key to generate a transaction
func generateDemoTX2(contractAddr common.Address, functionAddr common.Address, args [][]byte, checkList []eles.CheckElement, senderPrvKey *ecdsa.PrivateKey) eles.Transaction {
	tx := eles.Transaction{
		Sender:   crypto.PubkeyToAddress(senderPrvKey.PublicKey),
		Nonce:    0,
		Version:  TEST_VERSION,
		LifeTime: 0,

		Contract:  contractAddr,
		Function:  functionAddr,
		Args:      args,
		CheckList: checkList,
	}
	tx.SetTxID()
	signature, err := crypto.SignHash(tx.TxID, senderPrvKey)
	if err != nil {
		panic(err)
	}
	tx.Signature = signature
	return tx
}

var (
	contractValidCheckList = []eles.CheckElement{eles.GenerateCheckElement(TEST_CONTRACT_WSKEY, TEST_CONTRACT_DESCRIPTION, CHECK_TYPE_BYTES_EQUAL)}
)

var testTxs1 = []eles.Transaction{
	generateDemoTX(TEST_CONTRACT, TEST_FUNCTION_GET, [][]byte{[]byte("Leo")}, contractValidCheckList),
	generateDemoTX(TEST_CONTRACT, TEST_FUNCTION_SET, [][]byte{[]byte("Leo"), []byte("Awesome")}, contractValidCheckList),
	generateDemoTX(TEST_CONTRACT, TEST_FUNCTION_GET, [][]byte{[]byte("Leo")}, contractValidCheckList),
}

var testTxs2 = []eles.Transaction{
	generateDemoTX(TEST_CONTRACT, TEST_FUNCTION_GET, [][]byte{[]byte("Leo")}, []eles.CheckElement{}),
	generateDemoTX(
		TEST_CONTRACT, TEST_FUNCTION_SET,
		[][]byte{[]byte("Bob"), []byte("Handsome")},
		[]eles.CheckElement{eles.GenerateCheckElement(
			common.AddressesCatenateWsKey(BASE_STORAGE_ADDRESS, bytesToHashAddress([]byte("Leo"))),
			[]byte("Ugly"),
			CHECK_TYPE_BYTES_EQUAL_NOT)}),
	generateDemoTX(TEST_CONTRACT, TEST_FUNCTION_GET, [][]byte{[]byte("Bob")}, []eles.CheckElement{}),
	generateDemoTX(
		TEST_CONTRACT, TEST_FUNCTION_SET,
		[][]byte{[]byte("Bob::Weight"), []byte("100")},
		[]eles.CheckElement{eles.GenerateCheckElement(
			common.AddressesCatenateWsKey(BASE_STORAGE_ADDRESS, bytesToHashAddress([]byte("Bob"))),
			[]byte(""),
			CHECK_TYPE_COMMON_EXIST)}),
	generateDemoTX(
		TEST_CONTRACT, TEST_FUNCTION_SET,
		[][]byte{[]byte("Bob"), []byte("Thin")},
		[]eles.CheckElement{eles.GenerateCheckElement(
			common.AddressesCatenateWsKey(BASE_STORAGE_ADDRESS, bytesToHashAddress([]byte("Bob::Weight"))),
			[]byte("200"),
			CHECK_TYPE_BYTES_LESS)}),
	generateDemoTX(TEST_CONTRACT, TEST_FUNCTION_GET, [][]byte{[]byte("Bob")}, []eles.CheckElement{}),
}

var senderPrvKey1, _ = crypto.GenerateKey()

var senderPrvKey2, _ = crypto.GenerateKey()

var senderPrvKey3, _ = crypto.GenerateKey()

var testTxs3 = []eles.Transaction{
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_CONFIG, [][]byte{[]byte("group1")}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group1"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("10000")}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SEND, [][]byte{[]byte("group1"), crypto.PubkeyToAddress(senderPrvKey3.PublicKey).Bytes(), []byte("100")}, []eles.CheckElement{}, senderPrvKey2),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_BALANCE, [][]byte{[]byte("group1")}, []eles.CheckElement{}, senderPrvKey2),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_BALANCE, [][]byte{[]byte("group1")}, []eles.CheckElement{}, senderPrvKey3),

	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_CONFIG, [][]byte{[]byte("group2")}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group2"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("10000")}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SEND, [][]byte{[]byte("group2"), crypto.PubkeyToAddress(senderPrvKey3.PublicKey).Bytes(), []byte("100")}, []eles.CheckElement{}, senderPrvKey2),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_BALANCE, [][]byte{[]byte("group2")}, []eles.CheckElement{}, senderPrvKey2),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_BALANCE, [][]byte{[]byte("group2")}, []eles.CheckElement{}, senderPrvKey3),

	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group3"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("2")}, []eles.CheckElement{}, senderPrvKey2), //not root address set
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_CONFIG, [][]byte{[]byte("group3")}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_CONFIG, [][]byte{[]byte("group3")}, []eles.CheckElement{}, senderPrvKey2), //double config

	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group3"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("fuck")}, []eles.CheckElement{}, senderPrvKey2),                       //set not number
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group3"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("77787789223372036854775806")}, []eles.CheckElement{}, senderPrvKey1), //set overflow
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group3"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("2000000")}, []eles.CheckElement{}, senderPrvKey2),                    //not root address set

	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group3"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("2000000")}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SEND, [][]byte{[]byte("group3"), crypto.PubkeyToAddress(senderPrvKey3.PublicKey).Bytes(), []byte("100000000000")}, []eles.CheckElement{}, senderPrvKey2),       //account not enough
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET, [][]byte{[]byte("group3"), crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(), []byte("9223372036854775806")}, []eles.CheckElement{}, senderPrvKey1), //set overflow

	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_BALANCE, [][]byte{[]byte("group3")}, []eles.CheckElement{}, senderPrvKey2),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_GROUP_BALANCE, [][]byte{[]byte("group3")}, []eles.CheckElement{}, senderPrvKey3),
}

var testPurchaseForm = PurchaseForm{
	Process_id:          "hcuiahciuyfhineuorfbhiuoreafherqf",
	Contract_identifier: "test",
	Buyer:               "rand_buyer",
	Provider_name:       "provider_name",
	Goods_name:          "goods_name",
	Goods_type:          "goods_type",
	Goods_specification: "goods_specification",
	Goods_amount:        "goods_amount",
	Goods_unit:          "goods_unit",
	Goods_price:         "goods_price",
	Goods_total_price:   "goods_total_price",
	Due_and_amount:      "due_and_amount",
	Sign_date:           "sign_date",
}
var testReceiptForm = ReceiptForm{
	Process_id:          "hcuiahciuyfhineuorfbhiuoreafherqf",
	Contract_identifier: "test",
	Buyer:               "test",
	Provider_name:       "test",
	Goods_name:          "test",
	Goods_type:          "test",
	Goods_specification: "test",
	Goods_fahuo_amount:  "test",
	Goods_unit:          "test",
	Goods_price:         "test",
	Goods_total_price:   "test",
	Date:                "test",
}
var puchaseFormCode, _ = rlp.EncodeToBytes(testPurchaseForm)
var receiptFormCode, _ = rlp.EncodeToBytes(testReceiptForm)

var testTxs4 = []eles.Transaction{
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_CREATE,
		[][]byte{
			[]byte("hcuiahciuyfhineuorfbhiuoreafherqf"),
			crypto.PubkeyToAddress(senderPrvKey2.PublicKey).Bytes(),
			[]byte("debtor_name"),
			[]byte("credit_side_name"),
		}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SET_PURCHASEFORM, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf"), puchaseFormCode}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SET_RECEIPTFORM, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf"), receiptFormCode}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")}, []eles.CheckElement{}, senderPrvKey2),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SIGN_PAY, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")}, []eles.CheckElement{}, senderPrvKey1),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_DELIVERY, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")}, []eles.CheckElement{}, senderPrvKey2),
	generateDemoTX2(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_RECEIVE, [][]byte{[]byte("hcuiahciuyfhineuorfbhiuoreafherqf")}, []eles.CheckElement{}, senderPrvKey1),
}

func printReceipt(t *testing.T, receipt eles.TransactionReceipt) {
	res := fmt.Sprintf("transaction valid: %t, results:", receipt.Valid == byte(1))
	for i := 0; i < len(receipt.Result); i++ {
		s := fmt.Sprintf(" index: %d, value: %s", i, receipt.Result[i])
		res += s
	}
	t.Logf(res)
}

// use demo function compiler to test the precompile contract engine
func TestPreCompileCE(t *testing.T) {
	mockStateManager := worldstate.MockStateManager{}
	if err := mockStateManager.InitTestDataBase(); err != nil {
		t.Fatalf("fail in init database: %v", err)
	}

	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	mockPrecompileCE := NewPrecompileCE(&mockStateManager, fc, ci)

	funcGetWsKey := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_GET)
	funcSetWsKey := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SET)

	// in test, we directly set the key-value pairs to do contract installation
	installKeys := []common.WsKey{TEST_CONTRACT_WSKEY, funcGetWsKey, funcSetWsKey}
	installValues := [][]byte{TEST_CONTRACT_DESCRIPTION, DEMO_FUNCTION_GET, DEMO_FUNCTION_SET}

	if err := mockPrecompileCE.stateManager.WriteWorldStateValues(installKeys, installValues); err != nil {
		t.Fatalf("fail in install contract functions")
	}

	receipts1, writeSet1 := mockPrecompileCE.ExecuteTransactions(testTxs1)
	if len(receipts1) != len(testTxs1) {
		t.Fatalf("numbers of receipts and transactions are not matched")
	}

	if receipts1[0].Valid != byte(0) {
		t.Fatalf("check element interpreter work improperly")
	}

	for i := 0; i < len(receipts1); i++ {
		t.Logf("Receipt Index: %d of testTxs1:\n", i)
		printReceipt(t, receipts1[i])
	}

	if err := mockPrecompileCE.CommitWriteSet(writeSet1); err != nil {
		t.Fatalf("fail in commit write set: %v", err)
	}

	receipts2, writeSet2 := mockPrecompileCE.ExecuteTransactions(testTxs2)

	if len(receipts2) != len(testTxs2) {
		t.Fatalf("numbers of receipts and transactions are not matched")
	}

	if receipts2[0].Valid != byte(1) {
		t.Fatalf("write set is not committed properly")
	}

	for i := 0; i < len(receipts2); i++ {
		t.Logf("Receipt Index: %d of testTxs2:\n", i)
		printReceipt(t, receipts2[i])
	}

	if err := mockPrecompileCE.CommitWriteSet(writeSet2); err != nil {
		t.Fatalf("fail in commit write set: %v", err)
	}

}

func TestPreCompileCEGroup(t *testing.T) {
	mockStateManager := worldstate.MockStateManager{}
	if err := mockStateManager.InitTestDataBase(); err != nil {
		t.Fatalf("fail in init database: %v", err)
	}

	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	mockPrecompileCE := NewPrecompileCE(&mockStateManager, fc, ci)

	funcConfigWsKey := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_GROUP_CONFIG)
	funcSetWsKey := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_GROUP_SET)
	funcSendWsKey := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_GROUP_SEND)
	funcBalanceWsKey := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_GROUP_BALANCE)

	// in test, we directly set the key-value pairs to do contract installation
	installKeys := []common.WsKey{TEST_CONTRACT_WSKEY, funcConfigWsKey, funcSetWsKey, funcSendWsKey, funcBalanceWsKey}
	installValues := [][]byte{TEST_CONTRACT_DESCRIPTION, DEMO_FUNCTION_GROUP_CONFIG, DEMO_FUNCTION_GROUP_SET, DEMO_FUNCTION_GROUP_SEND, DEMO_FUNCTION_GROUP_BALANCE}

	if err := mockPrecompileCE.stateManager.WriteWorldStateValues(installKeys, installValues); err != nil {
		t.Fatalf("fail in install contract functions")
	}

	receipts5, writeSet5 := mockPrecompileCE.ExecuteTransactions(testTxs3)

	if len(receipts5) != len(testTxs3) {
		t.Fatalf("numbers of receipts and transactions are not matched")
	}

	if receipts5[0].Valid != byte(1) {
		t.Fatalf("write set is not committed properly")
	}

	for i := 0; i < len(receipts5); i++ {
		t.Logf("Receipt Index: %d of testTxs3:\n", i)
		printReceipt(t, receipts5[i])
	}

	if err := mockPrecompileCE.CommitWriteSet(writeSet5); err != nil {
		t.Fatalf("fail in commit write set: %v", err)
	}

}

func TestPreCompileCESupply(t *testing.T) {
	mockStateManager := worldstate.MockStateManager{}
	if err := mockStateManager.InitTestDataBase(); err != nil {
		t.Fatalf("fail in init database: %v", err)
	}

	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	mockPrecompileCE := NewPrecompileCE(&mockStateManager, fc, ci)

	funcCreateWsKey := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_CREATE)
	funcSetPurchaseform := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SET_PURCHASEFORM)
	funcGetPurchaseform := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_GET_PURCHASEFORM)
	funcSetReceipt := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SET_RECEIPTFORM)
	funcGetReceipt := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_GET_RECEIPTFORM)

	funcGetProcessform := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_GET_PROCESSFORM)
	funcGetSignCreditSide := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE)
	funcGetSignPay := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_SIGN_PAY)
	funcDelivery := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_DELIVERY)
	funcReceive := common.AddressesCatenateWsKey(TEST_CONTRACT, TEST_FUNCTION_SUPPLY_RECEIVE)

	// in test, we directly set the key-value pairs to do contract installation
	installKeys := []common.WsKey{
		TEST_CONTRACT_WSKEY,
		funcCreateWsKey,
		funcSetPurchaseform,
		funcGetPurchaseform,
		funcSetReceipt,
		funcGetReceipt,
		funcGetProcessform,
		funcGetSignCreditSide,
		funcGetSignPay,
		funcDelivery,
		funcReceive,
	}

	installValues := [][]byte{
		TEST_CONTRACT_DESCRIPTION,
		DEMO_FUNCTION_SUPPLY_CREATE,
		DEMO_FUNCTION_SUPPLY_SET_PURCHASEFORM,
		DEMO_FUNCTION_SUPPLY_GET_PURCHASEFORM,
		DEMO_FUNCTION_SUPPLY_SET_RECEIPTFORM,
		DEMO_FUNCTION_SUPPLY_GET_RECEIPTFORM,
		DEMO_FUNCTION_SUPPLY_GET_PROCESSFORM,
		DEMO_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE,
		DEMO_FUNCTION_SUPPLY_SIGN_PAY,
		DEMO_FUNCTION_SUPPLY_DELIVERY,
		DEMO_FUNCTION_SUPPLY_RECEIVE,
	}

	if err := mockPrecompileCE.stateManager.WriteWorldStateValues(installKeys, installValues); err != nil {
		t.Fatalf("fail in install contract functions")
	}

	receipts5, writeSet5 := mockPrecompileCE.ExecuteTransactions(testTxs4)

	if len(receipts5) != len(testTxs4) {
		t.Fatalf("numbers of receipts and transactions are not matched")
	}

	if receipts5[0].Valid != byte(1) {
		t.Fatalf("write set is not committed properly")
	}

	for i := 0; i < len(receipts5); i++ {
		t.Logf("Receipt Index: %d of testTxs3:\n", i)
		printReceipt(t, receipts5[i])
	}

	if err := mockPrecompileCE.CommitWriteSet(writeSet5); err != nil {
		t.Fatalf("fail in commit write set: %v", err)
	}

}
