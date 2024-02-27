package contract

import (
	"bytes"
	"fmt"
	"p3Chain/common"
	"p3Chain/rlp"
	"strconv"
	"strings"
)

var (
	ERROR_FUNCTION_ARGS     = fmt.Errorf("unmatched arguments")
	ERROR_FUNCTION_COMPILE  = fmt.Errorf("cannot compile this function")
	ERROR_FUNCTION_VALUEGET = fmt.Errorf("cannot get value")
	ERROR_INNER             = fmt.Errorf("contract function inner error")

	ERROR_FUNCTION_BOSSNAME   = fmt.Errorf("money set boss name unmatch")
	ERROR_FUNCTION_INVALIDNUM = fmt.Errorf("balance is not a number")
	ERROR_FUNCTION_OVERFLOW   = fmt.Errorf("balance number overflow")
	ERROR_FUNCTION_POVERTY    = fmt.Errorf("sender balance not enough")

	ERROR_FUNCTION_CONFIGDOUBLE = fmt.Errorf("config double write")
)

type functionCompiler interface {
	compileFunc(pce *PrecompileContractEngine, plainText []byte) (func(args [][]byte, self common.Address) ([][]byte, error), error)
}

type baseFunctionCompiler struct{}

func (bfc *baseFunctionCompiler) compileFunc(pce *PrecompileContractEngine, plainText []byte) (func(args [][]byte, self common.Address) ([][]byte, error), error) {
	panic("to implement")
}

var (
	DEMO_FUNCTION_SET = []byte("DEMO::FUNCTION::SET")
	DEMO_FUNCTION_GET = []byte("DEMO::FUNCTION::GET")

	DEMO_FUNCTION_GROUP_CONFIG  = []byte("DEMO::FUNCTION::GROUP_CONFIG")
	DEMO_FUNCTION_GROUP_SET     = []byte("DEMO::FUNCTION::GROUP_SET")
	DEMO_FUNCTION_GROUP_SEND    = []byte("DEMO::FUNCTION::GROUP_SEND")
	DEMO_FUNCTION_GROUP_BALANCE = []byte("DEMO::FUNCTION::GROUP_BALANCE")

	DEMO_FUNCTION_SUPPLY_CREATE = []byte("DEMO::FUNCTION::SUPPLY::CREATE")

	DEMO_FUNCTION_SUPPLY_SET_PURCHASEFORM = []byte("DEMO::FUNCTION::SUPPLY::SET_PURCHASEFORM")
	DEMO_FUNCTION_SUPPLY_GET_PURCHASEFORM = []byte("DEMO::FUNCTION::SUPPLY::GET_PURCHASEFORM") //args:id

	DEMO_FUNCTION_SUPPLY_SET_RECEIPTFORM = []byte("DEMO::FUNCTION::SUPPLY::SET_RECEIPTFORM")
	DEMO_FUNCTION_SUPPLY_GET_RECEIPTFORM = []byte("DEMO::FUNCTION::SUPPLY::GET_RECEIPTFORM")

	DEMO_FUNCTION_SUPPLY_GET_PROCESSFORM  = []byte("DEMO::FUNCTION::SUPPLY::GET_PROCESSFORM")
	DEMO_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE = []byte("DEMO::FUNCTION::SUPPLY::SIGN_CREDIT_SIDE")
	DEMO_FUNCTION_SUPPLY_SIGN_PAY         = []byte("DEMO::FUNCTION::SUPPLY::SIGN_PAY")
	DEMO_FUNCTION_SUPPLY_DELIVERY         = []byte("DEMO::FUNCTION::SUPPLY::DELIVERY")
	DEMO_FUNCTION_SUPPLY_RECEIVE          = []byte("DEMO::FUNCTION::SUPPLY::RECEIVE")

	DEMO_STORAGE_SUPPLY_PURCHASE_ID = []byte("DEMO::STORAGE::SUPPLY::PURCHASE::ID")
	DEMO_STORAGE_SUPPLY_SUPPLY_ID   = []byte("DEMO::STORAGE::SUPPLY::SUPPLY::ID")
	DEMO_STORAGE_SUPPLY_PROCESSFORM = []byte("DEMO::STORAGE::SUPPLY::PROCESSFORM")
	//********************************************************************************************
	DEMO_FUNCTION_FINANCE_CREATE       = []byte("DEMO::FUNCTION::FINANCE::CREATE")
	DEMO_FUNCTION_FINANCE_SET_LOANFORM = []byte("DEMO::FUNCTION::FINANCE::SET_LOANFORM")
	DEMO_FUNCTION_FINANCE_GET_LOANFORM = []byte("DEMO::FUNCTION::FINANCE::GET_LOANFORM")

	DEMO_FUNCTION_FINANCE_SET_PAYFORM = []byte("DEMO::FUNCTION::FINANCE::SET_PAYFORM")
	DEMO_FUNCTION_FINANCE_GET_PAYFORM = []byte("DEMO::FUNCTION::FINANCE::GET_PAYFORM")

	DEMO_FUNCTION_FINANCE_SET_REPAYFORM = []byte("DEMO::FUNCTION::FINANCE::SET_REPAYFORM")
	DEMO_FUNCTION_FINANCE_GET_REPAYFORM = []byte("DEMO::FUNCTION::FINANCE::GET_REPAYFORM")

	DEMO_FUNCTION_FINANCE_GET_PROCESSFORM  = []byte("DEMO::FUNCTION::FINANCE::GET_PROCESSFORM")
	DEMO_FUNCTION_FINANCE_SIGN_CREDIT_SIDE = []byte("DEMO::FUNCTION::FINANCE::SIGN_CREDIT_SIDE")
	DEMO_FUNCTION_FINANCE_SIGN_PAY         = []byte("DEMO::FUNCTION::FINANCE::SIGN_PAY")
	DEMO_FUNCTION_FINANCE_CONFIRM_PAY      = []byte("DEMO::FUNCTION::FINANCE::CONFIRM_PAY")
	DEMO_FUNCTION_FINANCE_SIGN_REPAY       = []byte("DEMO::FUNCTION::FINANCE::SIGN_REPAY")
	DEMO_FUNCTION_FINANCE_CONFIRM_REPAY    = []byte("DEMO::FUNCTION::FINANCE::CONFIRM_REPAY")

	DEMO_STORAGE_FINANCE_DEBTOR_ID   = []byte("DEMO::STORAGE::FINANCE::DEBTOR::ID")
	DEMO_STORAGE_FINANCE_BORROWER_ID = []byte("DEMO::STORAGE::FINANCE::BORROWER::ID")
	DEMO_STORAGE_FINANCE_PROCESSFORM = []byte("DEMO::STORAGE::FINANCE::PROCESSFORM")

	//********************************************************************************************
	DEMO_FUNCTION_FL_INIT      = []byte("DEMO::FUNCTION::FL::INIT")
	DEMO_FUNCTION_FL_GETHEIGHT = []byte("DEMO::FUNCTION::FL::GETHEIGHT")

	DEMO_FUNCTION_FL_STOREMODEL = []byte("DEMO::FUNCTION::FL::STOREMODEL")
	DEMO_FUNCTION_FL_GETMODEL   = []byte("DEMO::FUNCTION::FL::GETMODEL")

	DEMO_FUNCTION_FL_GETCURRENT = []byte("DEMO::FUNCTION::FL::GETCURRENT")

	DEMO_STORAGE_FL_HEIGHT = []byte("DEMO::STORAGE::FL::HEIGHT")

	HEIGHT_WSKEY = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FL, bytesToHashAddress(DEMO_STORAGE_FL_HEIGHT))

	//********************************************************************************************
	DEMO_FUNCTION_ADD_INIT = []byte("DEMO::FUNCTION::ADD::INIT")
	DEMO_FUNCTION_ADD_ADD  = []byte("DEMO::FUNCTION::ADD::ADD")
	DEMO_FUNCTION_ADD_SET  = []byte("DEMO::FUNCTION::ADD::SET")
	DEMO_FUNCTION_ADD_GET  = []byte("DEMO::FUNCTION::ADD::GET")
	DEMO_FUNCTION_ADD_SUM  = []byte("DEMO::FUNCTION::ADD::SUM")
)

type ProcessForm struct {
	Id                 string
	Debtor_id          string
	Debtor_name        string
	Debtor_status      string
	Credit_side_id     string
	Credit_side_name   string
	Credit_side_status string
	Pay_status         string
	Delivery_status    string
	Receive_status     string
}

type PurchaseForm struct {
	Process_id          string
	Contract_identifier string
	Buyer               string
	Provider_name       string
	Goods_name          string
	Goods_type          string
	Goods_specification string
	Goods_amount        string
	Goods_unit          string
	Goods_price         string
	Goods_total_price   string
	Due_and_amount      string
	Sign_date           string
}

type ReceiptForm struct {
	Process_id          string
	Contract_identifier string
	Buyer               string
	Provider_name       string
	Goods_name          string
	Goods_type          string
	Goods_specification string
	Goods_fahuo_amount  string
	Goods_unit          string
	Goods_price         string
	Goods_total_price   string
	Date                string
}

type FinanceProcessForm struct {
	Id                      string
	Debtor_id               string
	Debtor_name             string
	Debtor_status           string
	Credit_side_id          string
	Credit_side_name        string
	Credit_side_status      string
	Credit_pay              string
	Debtor_confirm_shoukuan string
	Debtor_repay            string
	Credit_confirm_huankuan string
}

type LoanForm struct {
	Process_id              string
	Serial_no               string
	Amount                  string
	Fin_amount              string
	Loan_amount             string
	Ware_no                 string
	Contract_no             string
	Mass_no                 string
	Pledge_type             string
	Debtor_name             string
	Credit_side_name        string
	Fin_start_date          string
	Fin_end_date            string
	Fin_days                string
	Service_price           string
	Fund_prod_name          string
	Fund_prod_int_rate      string
	Fund_prod_service_price string
	Fund_prod_period        string
	Payment_type            string
	Bank_card_no            string
	Bank_name               string
	Remark                  string
}
type PayForm struct {
	Process_id   string
	Dk_no        string
	Dk_name      string
	Dk_quota     string
	Pay_account  string
	Credit_start string
	Credit_end   string
}
type RepayForm struct {
	Process_id  string
	Repay_no    string
	Repay_quota string
	Repay_date  string
}

type demoFunctionCompiler struct{}

func (dfc *demoFunctionCompiler) compileFunc(pce *PrecompileContractEngine, plainText []byte) (func(args [][]byte, self common.Address) ([][]byte, error), error) {
	if bytes.Equal(plainText, DEMO_FUNCTION_SET) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 2 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerAddr := bytesToHashAddress(args[0])
			value := &commonBytes{
				value: args[1],
			}
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_1, innerAddr)
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("set succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_GET) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerAddr := bytesToHashAddress(args[0])
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_1, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_GROUP_CONFIG) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerAddr := bytesToHashAddress(args[0])
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_GROUP, innerAddr)
			_, err := pce.Get(wsKey)
			if err == nil {
				return nil, ERROR_FUNCTION_CONFIGDOUBLE
			}
			setValue := &commonBytes{
				value: self.Bytes(),
			}
			pce.Set(wsKey, setValue)
			result := [][]byte{[]byte("group root config succeed")}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_GROUP_SET) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 3 {
				return nil, ERROR_FUNCTION_ARGS
			}
			senderAddr := bytesToHashAddress(args[0])
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_GROUP, senderAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}
			set_num, err := strconv.ParseInt(string(args[2]), 10, 64)
			if err != nil {
				return nil, ERROR_FUNCTION_INVALIDNUM
			}
			aimAddr := common.HexToAddress(string(args[1]))
			innerBuffer := append(aimAddr[:], args[0]...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_GROUP, innerAddr)
			var balance int64
			value, err = pce.Get(wsKey)
			if err == nil {
				balance, err = strconv.ParseInt(string(value.toBytes()), 10, 64)
				if err != nil {
					return nil, ERROR_FUNCTION_INVALIDNUM
				}
				if (balance+set_num < balance) || (balance+set_num < set_num) {
					return nil, ERROR_FUNCTION_OVERFLOW
				}
			} else {
				balance = 0
			}
			NewBalance := strconv.FormatInt(balance+set_num, 10)
			setValue := &commonBytes{
				value: []byte(NewBalance),
			}
			pce.Set(wsKey, setValue)
			result := [][]byte{[]byte("group money set succeed")}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_GROUP_SEND) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 3 {
				return nil, ERROR_FUNCTION_ARGS
			}

			sendMoney, err := strconv.ParseInt(string(args[2]), 10, 64)
			if err != nil {
				return nil, ERROR_FUNCTION_INVALIDNUM
			}
			senderBuffer := append(self.Bytes(), args[0]...)
			senderAddr := bytesToHashAddress(senderBuffer)
			swsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_GROUP, senderAddr)
			value, err := pce.Get(swsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			senderBalance, err := strconv.ParseInt(string(value.toBytes()), 10, 64)
			if err != nil {
				return nil, ERROR_FUNCTION_INVALIDNUM
			}
			if senderBalance < sendMoney {
				return nil, ERROR_FUNCTION_POVERTY
			}
			aimAddr := common.HexToAddress(string(args[1]))
			innerBuffer := append(aimAddr[:], args[0]...)
			innerAddr := bytesToHashAddress(innerBuffer)
			rwsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_GROUP, innerAddr)
			var receiverBalance int64
			value, err = pce.Get(rwsKey)
			if err == nil {
				receiverBalance, err = strconv.ParseInt(string(value.toBytes()), 10, 64)
				if err != nil {
					return nil, ERROR_FUNCTION_INVALIDNUM
				}
				if (receiverBalance+sendMoney < receiverBalance) || (receiverBalance+sendMoney < sendMoney) {
					return nil, ERROR_FUNCTION_OVERFLOW
				}
			} else {
				receiverBalance = 0
			}
			NewBalance := strconv.FormatInt(senderBalance-sendMoney, 10)
			setValue := &commonBytes{
				value: []byte(NewBalance),
			}
			pce.Set(swsKey, setValue)
			NewBalance = strconv.FormatInt(receiverBalance+sendMoney, 10)
			setValue = &commonBytes{
				value: []byte(NewBalance),
			}
			pce.Set(rwsKey, setValue)
			result := [][]byte{[]byte("separate money set succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_GROUP_BALANCE) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(self.Bytes(), args[0]...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_GROUP, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_CREATE) { //process id  ,supply pubkey
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 4 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_PURCHASE_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			rwsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(rwsKey)
			if err == nil {
				return nil, ERROR_FUNCTION_CONFIGDOUBLE
			}
			supplyBuffer := append(args[0], DEMO_STORAGE_SUPPLY_SUPPLY_ID...)
			senderAddr := bytesToHashAddress(supplyBuffer)
			swsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, senderAddr)
			value, err = pce.Get(swsKey)
			if err == nil {
				return nil, ERROR_FUNCTION_CONFIGDOUBLE
			}
			value = &commonBytes{
				value: self.Bytes(),
			}
			pce.Set(rwsKey, value)
			value = &commonBytes{
				value: args[1],
			}
			pce.Set(swsKey, value)

			pf := ProcessForm{
				Id:                 string(args[0]),
				Debtor_id:          self.Str(),
				Debtor_name:        string(args[2]),
				Debtor_status:      "false",
				Credit_side_id:     string(args[1]),
				Credit_side_name:   string(args[3]),
				Credit_side_status: "false",
				Pay_status:         "false",
				Delivery_status:    "false",
				Receive_status:     "false",
			}
			processFormArgs := append(args[0], DEMO_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			processFormArgs = append(args[0], DEMO_FUNCTION_SUPPLY_SIGN_PAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			processFormArgs = append(args[0], DEMO_FUNCTION_SUPPLY_DELIVERY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			processFormArgs = append(args[0], DEMO_FUNCTION_SUPPLY_RECEIVE...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			pf_rlp, _ := rlp.EncodeToBytes(pf)
			processFormArgs = append(args[0], DEMO_STORAGE_SUPPLY_PROCESSFORM...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: pf_rlp,
			}
			pce.Set(wsKey, value)

			result := [][]byte{[]byte("supply create success")}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_SET_PURCHASEFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 2 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_PURCHASE_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}
			innerBuffer = append(args[0], DEMO_FUNCTION_SUPPLY_SET_PURCHASEFORM...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: args[1],
			}
			pce.Set(wsKey, value)

			result := [][]byte{[]byte("set PURCHASEFORM succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_GET_PURCHASEFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerBuffer := append(args[0], DEMO_FUNCTION_SUPPLY_SET_PURCHASEFORM...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_SET_RECEIPTFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 2 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_SUPPLY_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}
			innerBuffer = append(args[0], DEMO_FUNCTION_SUPPLY_SET_RECEIPTFORM...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: args[1],
			}
			pce.Set(wsKey, value)

			result := [][]byte{[]byte("set RECEIPTFORM succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_GET_RECEIPTFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerBuffer := append(args[0], DEMO_FUNCTION_SUPPLY_SET_RECEIPTFORM...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_GET_PROCESSFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_PROCESSFORM...)
			innerAddr := bytesToHashAddress(innerBuffer)
			pwsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(pwsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}

			pf := new(ProcessForm)
			rlp.DecodeBytes(value.toBytes(), pf)

			pf.Debtor_status = "true"
			processFormArgs := append(args[0], DEMO_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Credit_side_status = string(value.toBytes())
			if strings.Compare("false", pf.Credit_side_status) == 0 {
				pf.Debtor_status = "false"
			}

			processFormArgs = append(args[0], DEMO_FUNCTION_SUPPLY_SIGN_PAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Pay_status = string(value.toBytes())
			if strings.Compare("false", pf.Pay_status) == 0 {
				pf.Debtor_status = "false"
			}

			processFormArgs = append(args[0], DEMO_FUNCTION_SUPPLY_DELIVERY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Delivery_status = string(value.toBytes())
			if strings.Compare("false", pf.Delivery_status) == 0 {
				pf.Debtor_status = "false"
			}

			processFormArgs = append(args[0], DEMO_FUNCTION_SUPPLY_RECEIVE...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Receive_status = string(value.toBytes())
			if strings.Compare("false", pf.Receive_status) == 0 {
				pf.Debtor_status = "false"
			}

			pf_code, _ := rlp.EncodeToBytes(pf)
			value = &commonBytes{
				value: pf_code,
			}
			pce.Set(pwsKey, value)
			result := [][]byte{pf_code}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_SUPPLY_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_SUPPLY_SIGN_CREDIT_SIDE...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign credit side succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_SIGN_PAY) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_PURCHASE_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_SUPPLY_SIGN_PAY...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign pay side succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_DELIVERY) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_SUPPLY_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_SUPPLY_DELIVERY...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign delivery side succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_SUPPLY_RECEIVE) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_SUPPLY_PURCHASE_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_SUPPLY_RECEIVE...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_SUPPLY, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign receive side succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_CREATE) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 4 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_BORROWER_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			rwsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(rwsKey)
			if err == nil {
				return nil, ERROR_FUNCTION_CONFIGDOUBLE
			}
			repayBuffer := append(args[0], DEMO_STORAGE_FINANCE_DEBTOR_ID...)
			senderAddr := bytesToHashAddress(repayBuffer)
			swsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, senderAddr)
			value, err = pce.Get(swsKey)
			if err == nil {
				return nil, ERROR_FUNCTION_CONFIGDOUBLE
			}
			value = &commonBytes{
				value: self.Bytes(),
			}
			pce.Set(swsKey, value)
			aimAddr := common.HexToAddress(string(args[1]))
			value = &commonBytes{
				value: aimAddr[:],
			}
			pce.Set(rwsKey, value)

			pf := FinanceProcessForm{
				Id:                      string(args[0]),
				Debtor_id:               self.Str(),
				Debtor_name:             string(args[2]),
				Debtor_status:           "false",
				Credit_side_id:          string(aimAddr[:]),
				Credit_side_name:        string(args[3]),
				Credit_side_status:      "false",
				Credit_pay:              "false",
				Debtor_confirm_shoukuan: "false",
				Debtor_repay:            "false",
				Credit_confirm_huankuan: "false",
			}
			processFormArgs := append(args[0], DEMO_FUNCTION_FINANCE_SIGN_CREDIT_SIDE...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_SIGN_PAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_CONFIRM_PAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_SIGN_REPAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_CONFIRM_REPAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("false"),
			}
			pce.Set(wsKey, value)

			pf_rlp, _ := rlp.EncodeToBytes(pf)
			processFormArgs = append(args[0], DEMO_STORAGE_FINANCE_PROCESSFORM...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: pf_rlp,
			}
			pce.Set(wsKey, value)

			result := [][]byte{[]byte("finance create success")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_SET_LOANFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 2 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_BORROWER_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}
			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_SET_LOANFORM...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: args[1],
			}
			pce.Set(wsKey, value)

			result := [][]byte{[]byte("set LOANFORM succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_GET_LOANFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerBuffer := append(args[0], DEMO_FUNCTION_FINANCE_SET_LOANFORM...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_SET_PAYFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 2 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_BORROWER_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}
			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_SET_PAYFORM...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: args[1],
			}
			pce.Set(wsKey, value)

			result := [][]byte{[]byte("set PAYFORM succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_GET_PAYFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerBuffer := append(args[0], DEMO_FUNCTION_FINANCE_SET_PAYFORM...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_SET_REPAYFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 2 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_DEBTOR_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}
			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_SET_REPAYFORM...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: args[1],
			}
			pce.Set(wsKey, value)

			result := [][]byte{[]byte("set PAYFORM succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_GET_REPAYFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerBuffer := append(args[0], DEMO_FUNCTION_FINANCE_SET_REPAYFORM...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_GET_PROCESSFORM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_PROCESSFORM...)
			innerAddr := bytesToHashAddress(innerBuffer)
			pwsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(pwsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}

			pf := new(FinanceProcessForm)
			rlp.DecodeBytes(value.toBytes(), pf)

			pf.Debtor_status = "true"
			processFormArgs := append(args[0], DEMO_FUNCTION_FINANCE_SIGN_CREDIT_SIDE...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Credit_side_status = string(value.toBytes())
			if strings.Compare("false", pf.Credit_side_status) == 0 {
				pf.Debtor_status = "false"
			}

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_SIGN_PAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Credit_pay = string(value.toBytes())
			if strings.Compare("false", pf.Credit_pay) == 0 {
				pf.Debtor_status = "false"
			}

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_CONFIRM_PAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Debtor_confirm_shoukuan = string(value.toBytes())
			if strings.Compare("false", pf.Debtor_confirm_shoukuan) == 0 {
				pf.Debtor_status = "false"
			}

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_SIGN_REPAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Debtor_repay = string(value.toBytes())
			if strings.Compare("false", pf.Debtor_repay) == 0 {
				pf.Debtor_status = "false"
			}

			processFormArgs = append(args[0], DEMO_FUNCTION_FINANCE_CONFIRM_REPAY...)
			innerAddr = bytesToHashAddress(processFormArgs)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err = pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			pf.Credit_confirm_huankuan = string(value.toBytes())
			if strings.Compare("false", pf.Credit_confirm_huankuan) == 0 {
				pf.Debtor_status = "false"
			}

			pf_code, _ := rlp.EncodeToBytes(pf)
			value = &commonBytes{
				value: pf_code,
			}
			pce.Set(pwsKey, value)
			result := [][]byte{pf_code}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_SIGN_CREDIT_SIDE) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_BORROWER_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_SIGN_CREDIT_SIDE...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign credit side succeed")}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_SIGN_PAY) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_BORROWER_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_SIGN_PAY...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign pay succeed")}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_CONFIRM_PAY) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_DEBTOR_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_CONFIRM_PAY...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign confirm pay succeed")}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_SIGN_REPAY) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_DEBTOR_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_SIGN_REPAY...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign repay succeed")}
			return result, nil
		}
		return fn, nil
	}
	if bytes.Equal(plainText, DEMO_FUNCTION_FINANCE_CONFIRM_REPAY) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerBuffer := append(args[0], DEMO_STORAGE_FINANCE_BORROWER_ID...)
			innerAddr := bytesToHashAddress(innerBuffer)
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			if bytes.Compare(value.toBytes(), self.Bytes()) != 0 {
				return nil, ERROR_FUNCTION_BOSSNAME
			}

			innerBuffer = append(args[0], DEMO_FUNCTION_FINANCE_CONFIRM_REPAY...)
			innerAddr = bytesToHashAddress(innerBuffer)
			wsKey = common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FINANCE, innerAddr)
			value = &commonBytes{
				value: []byte("true"),
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("sign repay succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FL_INIT) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 0 {
				return nil, ERROR_FUNCTION_ARGS
			}

			height := strconv.Itoa(0)
			value := &commonBytes{
				value: []byte(height),
			}

			pce.Set(HEIGHT_WSKEY, value)
			result := [][]byte{[]byte("FL contract init succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FL_STOREMODEL) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}

			heightvalue, err := pce.Get(HEIGHT_WSKEY)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}

			height, err := strconv.Atoi(string(heightvalue.toBytes()))

			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}

			height += 1

			modelName := "model" + strconv.Itoa(height)

			innerAddr := bytesToHashAddress([]byte(modelName))
			value := &commonBytes{
				value: args[0],
			}
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FL, innerAddr)

			heightStr := strconv.Itoa(height)
			heightValue := &commonBytes{
				value: []byte(heightStr),
			}

			pce.Set(wsKey, value)
			pce.Set(HEIGHT_WSKEY, heightValue)

			result := [][]byte{[]byte("set succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FL_GETMODEL) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			num, err := strconv.Atoi(string(args[0]))
			if err != nil {
				return nil, ERROR_INNER
			}
			modelName := "model" + strconv.Itoa(num)
			innerAddr := bytesToHashAddress([]byte(modelName))
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FL, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FL_GETCURRENT) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 0 {
				return nil, ERROR_FUNCTION_ARGS
			}

			heightValue, err := pce.Get(HEIGHT_WSKEY)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}

			height, err := strconv.Atoi(string(heightValue.toBytes()))
			if err != nil {
				return nil, ERROR_INNER
			}

			modelName := "model" + strconv.Itoa(height)

			innerAddr := bytesToHashAddress([]byte(modelName))

			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_FL, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_FL_GETHEIGHT) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 0 {
				return nil, ERROR_FUNCTION_ARGS
			}

			heightValue, err := pce.Get(HEIGHT_WSKEY)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{heightValue.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	return nil, ERROR_FUNCTION_COMPILE
}

// ===========================================================================
// for contract ADD
type addFunctionCompiler struct{}

func (afc *addFunctionCompiler) compileFunc(pce *PrecompileContractEngine, plainText []byte) (func(args [][]byte, self common.Address) ([][]byte, error), error) {
	if bytes.Equal(plainText, DEMO_FUNCTION_ADD_INIT) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			iniVal := []byte(strconv.Itoa(1))
			for i := 0; i < 40001; i++ {
				strNo := strconv.Itoa(i)
				innerAddr := bytesToHashAddress([]byte(strNo))
				value := &commonBytes{
					value: iniVal,
				}
				wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_1, innerAddr)
				pce.Set(wsKey, value)
			}
			result := [][]byte{[]byte("init succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_ADD_SET) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 2 {
				return nil, ERROR_FUNCTION_ARGS
			}
			wsKey := common.BytesToWsKey(args[0])
			value := &commonBytes{
				value: args[1],
			}
			pce.Set(wsKey, value)
			result := [][]byte{[]byte("set succeed")}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_ADD_GET) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 1 {
				return nil, ERROR_FUNCTION_ARGS
			}
			innerAddr := bytesToHashAddress(args[0])
			wsKey := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_1, innerAddr)
			value, err := pce.Get(wsKey)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}
			result := [][]byte{value.toBytes()}
			return result, nil
		}
		return fn, nil
	}

	if bytes.Equal(plainText, DEMO_FUNCTION_ADD_SUM) {
		fn := func(args [][]byte, self common.Address) ([][]byte, error) {
			if len(args) != 3 {
				return nil, ERROR_FUNCTION_ARGS
			}

			innerAddr1 := BytesToHashAddress(args[0])
			wsKey1 := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_1, innerAddr1)

			innerAddr2 := BytesToHashAddress(args[1])
			wsKey2 := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_1, innerAddr2)

			innerAddr3 := BytesToHashAddress(args[2])
			wsKey3 := common.AddressesCatenateWsKey(DEMO_CONTRACT_STORAGE_1, innerAddr3)

			value1, err := pce.Get(wsKey1)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}

			value2, err := pce.Get(wsKey2)
			if err != nil {
				return nil, ERROR_FUNCTION_VALUEGET
			}

			val1, err := strconv.Atoi(string(value1.toBytes()))
			if err != nil {
				return nil, ERROR_INNER
			}

			val2, err := strconv.Atoi(string(value2.toBytes()))
			if err != nil {
				return nil, ERROR_INNER
			}

			val := (val1 + val2) % 10000

			val_byte := []byte(strconv.Itoa(val))

			value := &commonBytes{
				value: val_byte,
			}

			pce.Set(wsKey3, value)

			result := [][]byte{[]byte("add succeed")}
			return result, nil
		}
		return fn, nil

	}

	return nil, ERROR_FUNCTION_COMPILE
}
