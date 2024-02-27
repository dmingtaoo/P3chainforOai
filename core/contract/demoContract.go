package contract

import (
	"p3Chain/common"
	"p3Chain/core/worldstate"
	loglogrus "p3Chain/log_logrus"
)

var (
	// demo contract FL is an example for FL
	DEMO_CONTRACT_FL_NAME        = "DEMO_CONTRACT_FL"
	DEMO_CONTRACT_FL             = stringToContractAddress("FL")
	DEMO_CONTRACT_FL_WSKEY       = common.AddressesCatenateWsKey(DEMO_CONTRACT_FL, domainInfoAddress)
	DEMO_CONTRACT_FL_DESCRIPTION = []byte("DEMO CONTRACT FL IS ON")

	DEMO_CONTRACT_FL_FuncInitAddr       = stringToFunctionAddress("init")
	DEMO_CONTRACT_FL_FuncGetHeightAddr  = stringToFunctionAddress("getHeight")
	DEMO_CONTRACT_FL_FuncStoreModelAddr = stringToFunctionAddress("storeModel")
	DEMO_CONTRACT_FL_FuncGetModelAddr   = stringToFunctionAddress("getModel")
	DEMO_CONTRACT_FL_FuncGetCurrentAddr = stringToFunctionAddress("getCurrent")

	DEMO_CONTRACT_FL_FuncInitWsKey       = common.AddressesCatenateWsKey(DEMO_CONTRACT_FL, DEMO_CONTRACT_FL_FuncInitAddr)
	DEMO_CONTRACT_FL_FuncGetHeightWsKey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_FL, DEMO_CONTRACT_FL_FuncGetHeightAddr)
	DEMO_CONTRACT_FL_FuncStoreModelWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_FL, DEMO_CONTRACT_FL_FuncStoreModelAddr)
	DEMO_CONTRACT_FL_FuncGetModelWsKey   = common.AddressesCatenateWsKey(DEMO_CONTRACT_FL, DEMO_CONTRACT_FL_FuncGetModelAddr)
	DEMO_CONTRACT_FL_FuncGetCurrentWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_FL, DEMO_CONTRACT_FL_FuncGetCurrentAddr)
)

var (
	DEMO_CONTRACT_ADD_NAME        = "DEMO_CONTRACT_ADD"
	DEMO_CONTRACT_ADD             = stringToContractAddress("ADD")
	DEMO_CONTRACT_ADD_WSKEY       = common.AddressesCatenateWsKey(DEMO_CONTRACT_ADD, domainInfoAddress)
	DEMO_CONTRACT_ADD_DESCRIPTION = []byte("DEMO CONTRACT ADD IS ON")

	DEMO_CONTRACT_ADD_FuncInitAddr = stringToFunctionAddress("init")
	DEMO_CONTRACT_ADD_FuncAddAddr  = stringToFunctionAddress("add")
	DEMO_CONTRACT_ADD_FuncGetAddr  = stringToFunctionAddress("get")
	DEMO_CONTRACT_ADD_FuncSetAddr  = stringToFunctionAddress("set")
	DEMO_CONTRACT_ADD_FuncSumAddr  = stringToFunctionAddress("sum")

	DEMO_CONTRACT_ADD_FuncInitWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_ADD, DEMO_CONTRACT_ADD_FuncInitAddr)
	DEMO_CONTRACT_ADD_FuncAddWsKey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_ADD, DEMO_CONTRACT_ADD_FuncAddAddr)
	DEMO_CONTRACT_ADD_FuncGetWsKey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_ADD, DEMO_CONTRACT_ADD_FuncGetAddr)
	DEMO_CONTRACT_ADD_FuncSetWsKey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_ADD, DEMO_CONTRACT_ADD_FuncSetAddr)
	DEMO_CONTRACT_ADD_FuncSumWsKey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_ADD, DEMO_CONTRACT_ADD_FuncSumAddr)
)

var (
	// demo contract 1 is an example for get and set
	DEMO_CONTRACT_1_NAME         = "DEMO_CONTRACT_1"
	DEMO_CONTRACT_1              = stringToContractAddress("DEMO1")
	DEMO_CONTRACT_1_WSKEY        = common.AddressesCatenateWsKey(DEMO_CONTRACT_1, domainInfoAddress)
	DEMO_CONTRACT_1_DESCRIPTION  = []byte("DEMO CONTRACT 1 IS ON")
	DEMO_CONTRACT_1_FuncGetAddr  = stringToFunctionAddress("get")
	DEMO_CONTRACT_1_FuncSetAddr  = stringToFunctionAddress("set")
	DEMO_CONTRACT_1_FuncGetWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_1, DEMO_CONTRACT_1_FuncGetAddr)
	DEMO_CONTRACT_1_FuncSetWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_1, DEMO_CONTRACT_1_FuncSetAddr)
)
var (
	// demo contract 2 is an example for get and set
	DEMO_CONTRACT_2_NAME            = "DEMO_CONTRACT_2"
	DEMO_CONTRACT_2                 = stringToContractAddress("DEMO2")
	DEMO_CONTRACT_2_WSKEY           = common.AddressesCatenateWsKey(DEMO_CONTRACT_2, domainInfoAddress)
	DEMO_CONTRACT_2_DESCRIPTION     = []byte("DEMO CONTRACT 2 IS ON")
	DEMO_CONTRACT_2_FuncConfigAddr  = stringToFunctionAddress("group_config")
	DEMO_CONTRACT_2_FuncSetAddr     = stringToFunctionAddress("group_set")
	DEMO_CONTRACT_2_FuncSendAddr    = stringToFunctionAddress("group_send")
	DEMO_CONTRACT_2_FuncBalanceAddr = stringToFunctionAddress("group_balance")

	DEMO_CONTRACT_2_FuncConfigWsKey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_2, DEMO_CONTRACT_2_FuncConfigAddr)
	DEMO_CONTRACT_2_FuncSetWsKey     = common.AddressesCatenateWsKey(DEMO_CONTRACT_2, DEMO_CONTRACT_2_FuncSetAddr)
	DEMO_CONTRACT_2_FuncSendWsKey    = common.AddressesCatenateWsKey(DEMO_CONTRACT_2, DEMO_CONTRACT_2_FuncSendAddr)
	DEMO_CONTRACT_2_FuncBalanceWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_2, DEMO_CONTRACT_2_FuncBalanceAddr)
)
var (
	// demo contract 3 is an example for get and set
	DEMO_CONTRACT_3_NAME               = "DEMO_CONTRACT_3"
	DEMO_CONTRACT_3                    = stringToContractAddress("DEMO3")
	DEMO_CONTRACT_3_WSKEY              = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, domainInfoAddress)
	DEMO_CONTRACT_3_DESCRIPTION        = []byte("DEMO CONTRACT 3 IS ON")
	DEMO_CONTRACT_3_FuncCreateAddr     = stringToFunctionAddress("supply_create")
	DEMO_CONTRACT_3_FuncSetPurAddr     = stringToFunctionAddress("supply_set_purchaseform")
	DEMO_CONTRACT_3_FuncGetPurAddr     = stringToFunctionAddress("supply_get_purchaseform")
	DEMO_CONTRACT_3_FuncSetReceiptAddr = stringToFunctionAddress("supply_set_receipt")
	DEMO_CONTRACT_3_FuncGetReceiptAddr = stringToFunctionAddress("supply_get_receipt")
	DEMO_CONTRACT_3_FuncGetProcessAddr = stringToFunctionAddress("supply_get_processform")

	DEMO_CONTRACT_3_FuncSignCreditAddr = stringToFunctionAddress("supply_sign_credit_side")
	DEMO_CONTRACT_3_FuncSignPayAddr    = stringToFunctionAddress("supply_sign_pay")
	DEMO_CONTRACT_3_FuncDeliveryAddr   = stringToFunctionAddress("supply_delivery")
	DEMO_CONTRACT_3_FuncReceiveAddr    = stringToFunctionAddress("supply_receive")

	DEMO_CONTRACT_3_FuncCreateWsKey     = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncCreateAddr)
	DEMO_CONTRACT_3_FuncSetPurWsKey     = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncSetPurAddr)
	DEMO_CONTRACT_3_FuncGetPurWsKey     = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncGetPurAddr)
	DEMO_CONTRACT_3_FuncSetReceiptWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncSetReceiptAddr)
	DEMO_CONTRACT_3_FuncGetReceiptWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncGetReceiptAddr)
	DEMO_CONTRACT_3_FuncGetProcessWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncGetProcessAddr)
	DEMO_CONTRACT_3_FuncSignCreditWsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncSignCreditAddr)
	DEMO_CONTRACT_3_FuncSignPayWsKey    = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncSignPayAddr)
	DEMO_CONTRACT_3_FuncDeliveryWsKey   = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncDeliveryAddr)
	DEMO_CONTRACT_3_FuncReceiveWsKey    = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FuncReceiveAddr)

	DEMO_CONTRACT_3_FINANCE_FuncCreateAddr   = stringToFunctionAddress("finance_create")
	DEMO_CONTRACT_3_FINANCE_FuncSetLoanAddr  = stringToContractAddress("finance_set_loanform")
	DEMO_CONTRACT_3_FINANCE_FuncGetLoanAddr  = stringToContractAddress("finance_get_loanform")
	DEMO_CONTRACT_3_FINANCE_FuncSetPayAddr   = stringToContractAddress("finance_set_payform")
	DEMO_CONTRACT_3_FINANCE_FuncGetPayAddr   = stringToContractAddress("finance_get_payform")
	DEMO_CONTRACT_3_FINANCE_FuncSetRepayAddr = stringToContractAddress("finance_set_repayform")
	DEMO_CONTRACT_3_FINANCE_FuncGetRepayAddr = stringToContractAddress("finance_get_repayform")

	DEMO_CONTRACT_3_FINANCE_FuncGetProAddr       = stringToContractAddress("finance_get_processform")
	DEMO_CONTRACT_3_FINANCE_FuncSignCreditAddr   = stringToContractAddress("finance_sign_credit_side")
	DEMO_CONTRACT_3_FINANCE_FuncSignPayAddr      = stringToContractAddress("finance_sign_pay")
	DEMO_CONTRACT_3_FINANCE_FuncConfirmPayAddr   = stringToContractAddress("finance_confirm_pay")
	DEMO_CONTRACT_3_FINANCE_FuncSignRepayAddr    = stringToContractAddress("finance_sign_repay")
	DEMO_CONTRACT_3_FINANCE_FuncConfirmRepayAddr = stringToContractAddress("finance_comfirm_repay")

	DEMO_CONTRACT_3_FINANCE_FuncCreateWsKey   = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncCreateAddr)
	DEMO_CONTRACT_3_FINANCE_FuncSetLoanWskey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncSetLoanAddr)
	DEMO_CONTRACT_3_FINANCE_FuncGetLoanWskey  = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncGetLoanAddr)
	DEMO_CONTRACT_3_FINANCE_FuncSetPayWskey   = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncSetPayAddr)
	DEMO_CONTRACT_3_FINANCE_FuncGetPayWskey   = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncGetPayAddr)
	DEMO_CONTRACT_3_FINANCE_FuncSetRepayWskey = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncSetRepayAddr)
	DEMO_CONTRACT_3_FINANCE_FuncGetRepayWskey = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncGetRepayAddr)

	DEMO_CONTRACT_3_FINANCE_FuncGetProWskey       = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncGetProAddr)
	DEMO_CONTRACT_3_FINANCE_FuncSignCreditWskey   = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncSignCreditAddr)
	DEMO_CONTRACT_3_FINANCE_FuncSignPayWskey      = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncSignPayAddr)
	DEMO_CONTRACT_3_FINANCE_FuncConfirmPayWskey   = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncConfirmPayAddr)
	DEMO_CONTRACT_3_FINANCE_FuncSignRepayWskey    = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncSignRepayAddr)
	DEMO_CONTRACT_3_FINANCE_FuncConfirmRepayWskey = common.AddressesCatenateWsKey(DEMO_CONTRACT_3, DEMO_CONTRACT_3_FINANCE_FuncConfirmRepayAddr)
)

var (
	DEMO_CONTRACT_MIX123_NAME = "DEMO_CONTRACT_MIX123"
	//DEMO_REMOTE_CONTRACT_NAME = "DEMO_REMOTE_CONTRACT_NAME"
	DEMO_PIPE_CONTRACT_NAME = "DEMO_PIPE_CONTRACT_NAME"
)

func CreateDemo_Contract_ADD(sm worldstate.StateManager) (*PrecompileContractEngine, error) {
	fc := &addFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	precompileCE := NewPrecompileCE(sm, fc, ci)
	installKeys := []common.WsKey{DEMO_CONTRACT_ADD_WSKEY, DEMO_CONTRACT_ADD_FuncInitWsKey, DEMO_CONTRACT_ADD_FuncAddWsKey, DEMO_CONTRACT_ADD_FuncSetWsKey, DEMO_CONTRACT_ADD_FuncGetWsKey, DEMO_CONTRACT_ADD_FuncSumWsKey}
	installValues := [][]byte{DEMO_CONTRACT_ADD_DESCRIPTION, DEMO_FUNCTION_ADD_INIT, DEMO_FUNCTION_ADD_ADD, DEMO_FUNCTION_ADD_SET, DEMO_FUNCTION_ADD_GET, DEMO_FUNCTION_ADD_SUM}
	err := precompileCE.stateManager.WriteWorldStateValues(installKeys, installValues)
	if err != nil {
		loglogrus.Log.Errorf("Install PrecompileContractEngine:FL is failed,err:%v\n", err)
		return nil, err
	}
	return precompileCE, nil
}

func CreateDemo_Contract_FL(sm worldstate.StateManager) (*PrecompileContractEngine, error) {
	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	precompileCE := NewPrecompileCE(sm, fc, ci)
	installKeys := []common.WsKey{DEMO_CONTRACT_FL_WSKEY, DEMO_CONTRACT_FL_FuncInitWsKey, DEMO_CONTRACT_FL_FuncGetHeightWsKey, DEMO_CONTRACT_FL_FuncStoreModelWsKey, DEMO_CONTRACT_FL_FuncGetModelWsKey, DEMO_CONTRACT_FL_FuncGetCurrentWsKey}
	installValues := [][]byte{DEMO_CONTRACT_FL_DESCRIPTION, DEMO_FUNCTION_FL_INIT, DEMO_FUNCTION_FL_GETHEIGHT, DEMO_FUNCTION_FL_STOREMODEL, DEMO_FUNCTION_FL_GETMODEL, DEMO_FUNCTION_FL_GETCURRENT}
	err := precompileCE.stateManager.WriteWorldStateValues(installKeys, installValues)
	if err != nil {
		loglogrus.Log.Errorf("Install PrecompileContractEngine:FL is failed,err:%v\n", err)
		return nil, err
	}
	return precompileCE, nil
}

// CreateDemo_Contract_1 creates a precompiled contract engine which runs the
// basic demo contract which has get & set functions.
func CreateDemo_Contract_1(sm worldstate.StateManager) (*PrecompileContractEngine, error) {
	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	precompileCE := NewPrecompileCE(sm, fc, ci)
	installKeys := []common.WsKey{DEMO_CONTRACT_1_WSKEY, DEMO_CONTRACT_1_FuncGetWsKey, DEMO_CONTRACT_1_FuncSetWsKey}
	installValues := [][]byte{DEMO_CONTRACT_1_DESCRIPTION, DEMO_FUNCTION_GET, DEMO_FUNCTION_SET}
	err := precompileCE.stateManager.WriteWorldStateValues(installKeys, installValues)
	if err != nil {
		loglogrus.Log.Errorf("Install PrecompileContractEngine:DEMO1 is failed,err:%v\n", err)
		return nil, err
	}
	return precompileCE, nil
}

// func CreateRemoteContract(sm worldstate.StateManager, remote_ip string) (*RemoteContractEngine, error) {
// 	ci := &commonCheckElementInterpreter{}
// 	precompileCE := NewRemoteCE(sm, ci, remote_ip)

//		return precompileCE, nil
//	}
func CreatePipeContract(sm worldstate.StateManager, remote_ip string) *PipeContractEngine {
	ci := &commonCheckElementInterpreter{}
	precompileCE := NewPipeCE(sm, ci, remote_ip)

	return precompileCE
}
func CreateDemo_Contract_2(sm worldstate.StateManager) (*PrecompileContractEngine, error) {
	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	precompileCE := NewPrecompileCE(sm, fc, ci)
	installKeys := []common.WsKey{DEMO_CONTRACT_2_WSKEY,
		DEMO_CONTRACT_2_FuncConfigWsKey,
		DEMO_CONTRACT_2_FuncSetWsKey,
		DEMO_CONTRACT_2_FuncSendWsKey,
		DEMO_CONTRACT_2_FuncBalanceWsKey,
	}
	installValues := [][]byte{DEMO_CONTRACT_2_DESCRIPTION,
		DEMO_FUNCTION_GROUP_CONFIG,
		DEMO_FUNCTION_GROUP_SET,
		DEMO_FUNCTION_GROUP_SEND,
		DEMO_FUNCTION_GROUP_BALANCE,
	}
	err := precompileCE.stateManager.WriteWorldStateValues(installKeys, installValues)
	if err != nil {
		loglogrus.Log.Errorf("Install PrecompileContractEngine:DEMO2 is failed,err:%v\n", err)
		return nil, err
	}
	return precompileCE, nil
}
func CreateDemo_Contract_3(sm worldstate.StateManager) (*PrecompileContractEngine, error) {
	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	precompileCE := NewPrecompileCE(sm, fc, ci)
	installKeys := []common.WsKey{DEMO_CONTRACT_3_WSKEY,
		DEMO_CONTRACT_3_FuncCreateWsKey,
		DEMO_CONTRACT_3_FuncSetPurWsKey,
		DEMO_CONTRACT_3_FuncGetPurWsKey,
		DEMO_CONTRACT_3_FuncSetReceiptWsKey,
		DEMO_CONTRACT_3_FuncGetReceiptWsKey,
		DEMO_CONTRACT_3_FuncGetProcessWsKey,
		DEMO_CONTRACT_3_FuncSignCreditWsKey,
		DEMO_CONTRACT_3_FuncSignPayWsKey,
		DEMO_CONTRACT_3_FuncDeliveryWsKey,
		DEMO_CONTRACT_3_FuncReceiveWsKey,

		DEMO_CONTRACT_3_FINANCE_FuncCreateWsKey,
		DEMO_CONTRACT_3_FINANCE_FuncSetLoanWskey,
		DEMO_CONTRACT_3_FINANCE_FuncGetLoanWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSetPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncGetPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSetRepayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncGetRepayWskey,

		DEMO_CONTRACT_3_FINANCE_FuncGetProWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSignCreditWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSignPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncConfirmPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSignRepayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncConfirmRepayWskey,
	}
	installValues := [][]byte{DEMO_CONTRACT_3_DESCRIPTION,
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

		DEMO_FUNCTION_FINANCE_CREATE,
		DEMO_FUNCTION_FINANCE_SET_LOANFORM,
		DEMO_FUNCTION_FINANCE_GET_LOANFORM,
		DEMO_FUNCTION_FINANCE_SET_PAYFORM,
		DEMO_FUNCTION_FINANCE_GET_PAYFORM,
		DEMO_FUNCTION_FINANCE_SET_REPAYFORM,
		DEMO_FUNCTION_FINANCE_GET_REPAYFORM,

		DEMO_FUNCTION_FINANCE_GET_PROCESSFORM,
		DEMO_FUNCTION_FINANCE_SIGN_CREDIT_SIDE,
		DEMO_FUNCTION_FINANCE_SIGN_PAY,
		DEMO_FUNCTION_FINANCE_CONFIRM_PAY,
		DEMO_FUNCTION_FINANCE_SIGN_REPAY,
		DEMO_FUNCTION_FINANCE_CONFIRM_REPAY,
	}
	err := precompileCE.stateManager.WriteWorldStateValues(installKeys, installValues)
	if err != nil {
		loglogrus.Log.Errorf("Install PrecompileContractEngine:DEMO3 is failed,err:%v\n", err)
		return nil, err
	}
	return precompileCE, nil
}

// contain all contract
func CreateDemo_Contract_MIX123(sm worldstate.StateManager) (*PrecompileContractEngine, error) {
	fc := &demoFunctionCompiler{}
	ci := &commonCheckElementInterpreter{}
	precompileCE := NewPrecompileCE(sm, fc, ci)
	installKeys := []common.WsKey{
		DEMO_CONTRACT_1_WSKEY,
		DEMO_CONTRACT_1_FuncGetWsKey,
		DEMO_CONTRACT_1_FuncSetWsKey,

		DEMO_CONTRACT_2_WSKEY,
		DEMO_CONTRACT_2_FuncConfigWsKey,
		DEMO_CONTRACT_2_FuncSetWsKey,
		DEMO_CONTRACT_2_FuncSendWsKey,
		DEMO_CONTRACT_2_FuncBalanceWsKey,

		DEMO_CONTRACT_3_WSKEY,
		DEMO_CONTRACT_3_FuncCreateWsKey,
		DEMO_CONTRACT_3_FuncSetPurWsKey,
		DEMO_CONTRACT_3_FuncGetPurWsKey,
		DEMO_CONTRACT_3_FuncSetReceiptWsKey,
		DEMO_CONTRACT_3_FuncGetReceiptWsKey,
		DEMO_CONTRACT_3_FuncGetProcessWsKey,
		DEMO_CONTRACT_3_FuncSignCreditWsKey,
		DEMO_CONTRACT_3_FuncSignPayWsKey,
		DEMO_CONTRACT_3_FuncDeliveryWsKey,
		DEMO_CONTRACT_3_FuncReceiveWsKey,

		DEMO_CONTRACT_3_FINANCE_FuncCreateWsKey,
		DEMO_CONTRACT_3_FINANCE_FuncSetLoanWskey,
		DEMO_CONTRACT_3_FINANCE_FuncGetLoanWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSetPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncGetPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSetRepayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncGetRepayWskey,

		DEMO_CONTRACT_3_FINANCE_FuncGetProWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSignCreditWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSignPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncConfirmPayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncSignRepayWskey,
		DEMO_CONTRACT_3_FINANCE_FuncConfirmRepayWskey,
	}
	installValues := [][]byte{
		DEMO_CONTRACT_1_DESCRIPTION,
		DEMO_FUNCTION_GET,
		DEMO_FUNCTION_SET,

		DEMO_CONTRACT_2_DESCRIPTION,
		DEMO_FUNCTION_GROUP_CONFIG,
		DEMO_FUNCTION_GROUP_SET,
		DEMO_FUNCTION_GROUP_SEND,
		DEMO_FUNCTION_GROUP_BALANCE,

		DEMO_CONTRACT_3_DESCRIPTION,
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
		DEMO_FUNCTION_FINANCE_CREATE,
		DEMO_FUNCTION_FINANCE_SET_LOANFORM,
		DEMO_FUNCTION_FINANCE_GET_LOANFORM,
		DEMO_FUNCTION_FINANCE_SET_PAYFORM,
		DEMO_FUNCTION_FINANCE_GET_PAYFORM,
		DEMO_FUNCTION_FINANCE_SET_REPAYFORM,
		DEMO_FUNCTION_FINANCE_GET_REPAYFORM,

		DEMO_FUNCTION_FINANCE_GET_PROCESSFORM,
		DEMO_FUNCTION_FINANCE_SIGN_CREDIT_SIDE,
		DEMO_FUNCTION_FINANCE_SIGN_PAY,
		DEMO_FUNCTION_FINANCE_CONFIRM_PAY,
		DEMO_FUNCTION_FINANCE_SIGN_REPAY,
		DEMO_FUNCTION_FINANCE_CONFIRM_REPAY,
	}
	err := precompileCE.stateManager.WriteWorldStateValues(installKeys, installValues)
	if err != nil {
		loglogrus.Log.Errorf("Install PrecompileContractEngine:DEMO_MAX123 is failed,err:%v\n", err)
		return nil, err
	}
	return precompileCE, nil
}
