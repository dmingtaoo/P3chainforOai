package rpc

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"p3Chain/accounts"
	"p3Chain/common"
	"p3Chain/core/contract"
	"p3Chain/dper"
	"p3Chain/rlp"
	"strings"
)

type RpcServiceConfig struct {
	NetWorkmethod string
	RpcIPAddress  string
}

type JRClass struct {
	dper *dper.Dper
}

// start a new contract of loan.
// request : "Debtor account;borrower account;processID;Debtor name;borrower name
// sample : ""
func (s *JRClass) CreateFinance(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[0])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FINANCE_FuncCreateAddr, [][]byte{
		[]byte(strSplit[2]),
		[]byte(strSplit[1]),
		[]byte(strSplit[3]),
		[]byte(strSplit[4])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("create loan contract ID:%s ok", strSplit[2])
	}
	return nil
}

// upload loan form
// request :
//
//		Process_id                string
//		Serial_no                 string
//		Amount                    string
//		Fin_amount                string
//		Loan_amount               string
//		Ware_no                   string
//		Contract_no               string
//		Mass_no                   string
//		Pledge_type               string
//		Debtor_name               string
//		Credit_side_name          string
//		Fin_start_date            string
//		Fin_end_date              string
//		Fin_days                  string
//		Service_price             string
//		Fund_prod_name            string
//		Fund_prod_int_rate        string
//		Fund_prod_service_price   string
//		Fund_prod_period          string
//		Payment_type              string
//		Bank_card_no              string
//		Bank_name                 string
//		Remark                    string
//	    debtor account 		      string
//
// split with ";"
// sample: ""
func (s *JRClass) UploadLoanForm(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[23])}
	var testLoanForm = contract.LoanForm{
		Process_id:              strSplit[0],
		Serial_no:               strSplit[1],
		Amount:                  strSplit[2],
		Fin_amount:              strSplit[3],
		Loan_amount:             strSplit[4],
		Ware_no:                 strSplit[5],
		Contract_no:             strSplit[6],
		Mass_no:                 strSplit[7],
		Pledge_type:             strSplit[8],
		Debtor_name:             strSplit[9],
		Credit_side_name:        strSplit[10],
		Fin_start_date:          strSplit[11],
		Fin_end_date:            strSplit[12],
		Fin_days:                strSplit[13],
		Service_price:           strSplit[14],
		Fund_prod_name:          strSplit[15],
		Fund_prod_int_rate:      strSplit[16],
		Fund_prod_service_price: strSplit[17],
		Fund_prod_period:        strSplit[18],
		Payment_type:            strSplit[19],
		Bank_card_no:            strSplit[20],
		Bank_name:               strSplit[21],
		Remark:                  strSplit[22],
	}
	var loanFormCode, _ = rlp.EncodeToBytes(testLoanForm)
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncSetLoanAddr, [][]byte{[]byte(strSplit[0]), loanFormCode})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("upload loan form ID:%s ok", strSplit[0])
	}
	return nil
}

// upload pay form
// request :
//
//		Process_id             string
//		Dk_no                  string
//		Dk_name                string
//		Dk_quota               string
//		Pay_account            string
//		Credit_start           string
//		Credit_end             string
//	    borrower account 	   string
//
// split with ";"
// sample: ""
func (s *JRClass) UploadPayForm(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[7])}
	var testPayForm = contract.PayForm{
		Process_id:   strSplit[0],
		Dk_no:        strSplit[1],
		Dk_name:      strSplit[2],
		Dk_quota:     strSplit[3],
		Pay_account:  strSplit[4],
		Credit_start: strSplit[5],
		Credit_end:   strSplit[6],
	}
	var payFormCode, _ = rlp.EncodeToBytes(testPayForm)
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncSetPayAddr, [][]byte{[]byte(strSplit[0]), payFormCode})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("upload pay form ID:%s ok", strSplit[0])
	}
	return nil
}

// upload repay form
// request :
//
//			Process_id             string
//	     Repay_no               string
//	     Repay_quota            string
//	     Repay_date             string
//		    debtor account 		   string
//
// split with ";"
// sample: ""
func (s *JRClass) UploadRepayForm(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[4])}
	var testRepayForm = contract.RepayForm{
		Process_id:  strSplit[0],
		Repay_no:    strSplit[1],
		Repay_quota: strSplit[2],
		Repay_date:  strSplit[3],
	}
	var repayFormCode, _ = rlp.EncodeToBytes(testRepayForm)
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncSetRepayAddr, [][]byte{[]byte(strSplit[0]), repayFormCode})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("upload repay form ID:%s ok", strSplit[0])
	}
	return nil
}

// query processNote
// requset: Process_id; receiver account
// sample
func (s *JRClass) QueryProcessNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	result := s.dper.SimpleInvokeTransactionLocally(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncGetProAddr, [][]byte{[]byte(strSplit[0])})
	if len(result) == 0 {
		*reply = "transaction is not executed"
		return nil
	}
	tmp := new(contract.FinanceProcessForm)
	err := rlp.DecodeBytes(result[0], tmp)

	if err != nil {
		*reply = err.Error()
	} else {
		*reply = tmp.Id + ";" +
			fmt.Sprintf("%x", tmp.Debtor_id) + ";" +
			tmp.Debtor_name + ";" +
			tmp.Debtor_status + ";" +
			fmt.Sprintf("%x", tmp.Credit_side_id) + ";" +
			tmp.Credit_side_name + ";" +
			tmp.Credit_side_status + ";" +
			tmp.Credit_pay + ";" +
			tmp.Debtor_confirm_shoukuan + ";" +
			tmp.Debtor_repay + ";" +
			tmp.Credit_confirm_huankuan + ";"
	}
	return nil
}

// query loanNote
// requset: Process_id; borrower account
// sample
func (s *JRClass) QueryLoanNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	result := s.dper.SimpleInvokeTransactionLocally(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncGetLoanAddr, [][]byte{[]byte(strSplit[0])})
	if len(result) == 0 {
		*reply = "transaction is not executed"
		return nil
	}
	tmp := new(contract.LoanForm)
	err := rlp.DecodeBytes(result[0], tmp)

	if err != nil {
		*reply = err.Error()
	} else {
		*reply =
			tmp.Process_id + ";" +
				tmp.Serial_no + ";" +
				tmp.Amount + ";" +
				tmp.Fin_amount + ";" +
				tmp.Loan_amount + ";" +
				tmp.Ware_no + ";" +
				tmp.Contract_no + ";" +
				tmp.Mass_no + ";" +
				tmp.Pledge_type + ";" +
				tmp.Debtor_name + ";" +
				tmp.Credit_side_name + ";" +
				tmp.Fin_start_date + ";" +
				tmp.Fin_end_date + ";" +
				tmp.Fin_days + ";" +
				tmp.Service_price + ";" +
				tmp.Fund_prod_name + ";" +
				tmp.Fund_prod_int_rate + ";" +
				tmp.Fund_prod_service_price + ";" +
				tmp.Fund_prod_period + ";" +
				tmp.Payment_type + ";" +
				tmp.Bank_card_no + ";" +
				tmp.Bank_name + ";" +
				tmp.Remark + ";"
	}
	return nil
}

// query payNote
// requset: Process_id;  borrower account
// sample
func (s *JRClass) QueryPayNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	result := s.dper.SimpleInvokeTransactionLocally(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncGetPayAddr, [][]byte{[]byte(strSplit[0])})
	if len(result) == 0 {
		*reply = "transaction is not executed"
		return nil
	}
	tmp := new(contract.PayForm)
	err := rlp.DecodeBytes(result[0], tmp)

	if err != nil {
		*reply = err.Error()
	} else {
		*reply =
			tmp.Process_id + ";" +
				tmp.Dk_no + ";" +
				tmp.Dk_name + ";" +
				tmp.Dk_quota + ";" +
				tmp.Pay_account + ";" +
				tmp.Credit_start + ";" +
				tmp.Credit_end + ";"
	}
	return nil
}

// query repayNote
// requset: Process_id; borrower account
// sample
func (s *JRClass) QueryRepayNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	result := s.dper.SimpleInvokeTransactionLocally(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncGetRepayAddr, [][]byte{[]byte(strSplit[0])})
	if len(result) == 0 {
		*reply = "transaction is not executed"
		return nil
	}
	tmp := new(contract.RepayForm)
	err := rlp.DecodeBytes(result[0], tmp)

	if err != nil {
		*reply = err.Error()
	} else {
		*reply =
			tmp.Process_id + ";" +
				tmp.Repay_no + ";" +
				tmp.Repay_quota + ";" +
				tmp.Repay_date + ";"
	}
	return nil
}

// change Credit_side_status to TRUE
// requset: Process_id; borrower account
// sample
func (s *JRClass) AcceptCreditSideNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncSignCreditAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("sign credit_side_status ID:%s ok", strSplit[0])
	}
	return nil

}

// change Credit_pay to TRUE
// requset: Process_id; borrower account
// sample
func (s *JRClass) AcceptPayNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncSignPayAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("sign pay ID:%s ok", strSplit[0])
	}
	return nil
}

// change Debtor_confirm_shoukuan to TRUE
// requset: Process_id; Debtor account
// sample
func (s *JRClass) ConfirmPayNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncConfirmPayAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("confirm pay ID:%s ok", strSplit[0])
	}
	return nil
}

// change Debtor_repay to TRUE
// requset: Process_id; Debtor account
// sample
func (s *JRClass) AcceptRepayNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncSignRepayAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("sign repay ID:%s ok", strSplit[0])
	}
	return nil
}

// change Credit_confirm_huankuan to TRUE
// requset: Process_id; borrower account
// sample
func (s *JRClass) ConfirmRepayNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FINANCE_FuncConfirmRepayAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("confirm repay ID:%s ok", strSplit[0])
	}
	return nil
}

type GYClass struct {
	dper *dper.Dper
}

// just check connection work available
func (s *GYClass) CheckConnection(request string, reply *string) error {
	*reply = "connection available"
	return nil
}

// request : account name
func (s *GYClass) CreateNewAccount(request string, reply *string) error {
	am := s.dper.BackAccountManager()
	account, err := am.NewAccount(request)
	if err != nil {
		*reply = err.Error()
		return nil
	}
	err = am.Unlock(account.Address, "")
	if err != nil {
		*reply = err.Error()
		return nil
	} else {
		*reply = fmt.Sprintf("%x", account.Address)
	}
	return nil
}

// start a new contract of purchase and send goods. here called supply
// request : "receiver account;sender account;processID;receiver name;sender name
// sample : ""
func (s *GYClass) CreateSupply(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[0])}

	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3, contract.DEMO_CONTRACT_3_FuncCreateAddr, [][]byte{
		[]byte(strSplit[2]),
		[]byte(strSplit[1]),
		[]byte(strSplit[3]),
		[]byte(strSplit[4])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("create supply contract ID:%s ok", strSplit[2])
	}
	return nil
}

// upload purchase form
// request :
//
//				Process_id          string
//				Contract_identifier string
//				Buyer               string
//				Provider_name       string
//				Goods_name          string
//				Goods_type          string
//				Goods_specification string
//				Goods_amount        string
//				Goods_unit          string
//				Goods_price         string
//				Goods_total_price   string
//				Due_and_amount      string
//				Sign_date           string
//	         	receiver account    string
//
// split with ";"
// sample: ""
func (s *GYClass) UploadPurchaseForm(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[13])}
	var testPurchaseForm = contract.PurchaseForm{
		Process_id:          strSplit[0],
		Contract_identifier: strSplit[1],
		Buyer:               strSplit[2],
		Provider_name:       strSplit[3],
		Goods_name:          strSplit[4],
		Goods_type:          strSplit[5],
		Goods_specification: strSplit[6],
		Goods_amount:        strSplit[7],
		Goods_unit:          strSplit[8],
		Goods_price:         strSplit[9],
		Goods_total_price:   strSplit[10],
		Due_and_amount:      strSplit[11],
		Sign_date:           strSplit[12],
	}
	var puchaseFormCode, _ = rlp.EncodeToBytes(testPurchaseForm)
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncSetPurAddr, [][]byte{[]byte(strSplit[0]), puchaseFormCode})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("upload purchase form ID:%s ok", strSplit[0])
	}
	return nil
}

// upload receipt form
// request :
//
//	 	Process_id          string
//		Contract_identifier string
//		Buyer               string
//		Provider_name       string
//		Goods_name          string
//		Goods_type          string
//		Goods_specification string
//		Goods_fahuo_amount  string
//		Goods_unit          string
//		Goods_price         string
//		Goods_total_price   string
//		Date                string
//	 	receiver account    string
//
// split with ";"
// sample: ""
func (s *GYClass) UploadReceiptForm(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[12])}
	var testReceiptForm = contract.ReceiptForm{
		Process_id:          strSplit[0],
		Contract_identifier: strSplit[1],
		Buyer:               strSplit[2],
		Provider_name:       strSplit[3],
		Goods_name:          strSplit[4],
		Goods_type:          strSplit[5],
		Goods_specification: strSplit[6],
		Goods_fahuo_amount:  strSplit[7],
		Goods_unit:          strSplit[8],
		Goods_price:         strSplit[9],
		Goods_total_price:   strSplit[10],
		Date:                strSplit[11],
	}
	var receiptFormCode, _ = rlp.EncodeToBytes(testReceiptForm)
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncSetReceiptAddr, [][]byte{[]byte(strSplit[0]), receiptFormCode})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("upload Receipt form ID:%s ok", strSplit[0])
	}
	return nil
}

// change Credit_side_status to TRUE
// requset: Process_id; receiver account
// sample
func (s *GYClass) AcceptPurchaseNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncSignCreditAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("sign credit_side_status ID:%s ok", strSplit[0])
	}
	return nil

}

// change Pay_status  to TRUE
// requset: Process_id; sender account
// sample
func (s *GYClass) AcceptPayment(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncSignPayAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("sign  Pay_status ID:%s ok", strSplit[0])
	}
	return nil

}

// change Delivery_status to TRUE
// requset: Process_id; receiver account
// sample
func (s *GYClass) AcceptSendGoods(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncDeliveryAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("sign Delivery_status ID:%s ok", strSplit[0])
	}
	return nil

}

// change Receive_status  to TRUE
// requset: Process_id;  sender  account
// sample
func (s *GYClass) AcceptReceive(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	_, err := s.dper.SimpleInvokeTransaction(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncReceiveAddr, [][]byte{[]byte(strSplit[0])})
	if err != nil {
		*reply = err.Error()
	} else {
		*reply = fmt.Sprintf("sign  Receive_status ID:%s ok", strSplit[0])
	}
	return nil

}

// query processNote
// requset: Process_id; receiver account
// sample
func (s *GYClass) QueryProcessNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	result := s.dper.SimpleInvokeTransactionLocally(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncGetProcessAddr, [][]byte{[]byte(strSplit[0])})
	if len(result) == 0 {
		*reply = "transaction is not executed"
		return nil
	}
	tmp := new(contract.ProcessForm)
	err := rlp.DecodeBytes(result[0], tmp)

	if err != nil {
		*reply = err.Error()
	} else {
		*reply = tmp.Id + ";" +
			fmt.Sprintf("%x", tmp.Debtor_id) + ";" +
			tmp.Debtor_name + ";" +
			tmp.Debtor_status + ";" +
			fmt.Sprintf("%x", tmp.Credit_side_id) + ";" +
			tmp.Credit_side_name + ";" +
			tmp.Credit_side_status + ";" +
			tmp.Pay_status + ";" +
			tmp.Delivery_status + ";" +
			tmp.Receive_status + ";"
	}
	return nil
}

// query PurchaseNote
// requset: Process_id; receiver account
// sample
func (s *GYClass) QueryPurchaseNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	result := s.dper.SimpleInvokeTransactionLocally(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncGetPurAddr, [][]byte{[]byte(strSplit[0])})
	if len(result) == 0 {
		*reply = "transaction is not executed"
		return nil
	}
	tmp := new(contract.PurchaseForm)
	err := rlp.DecodeBytes(result[0], tmp)

	if err != nil {
		*reply = err.Error()
	} else {
		*reply = tmp.Process_id + ";" +
			tmp.Contract_identifier + ";" +
			tmp.Buyer + ";" +
			tmp.Provider_name + ";" +
			tmp.Goods_name + ";" +
			tmp.Goods_type + ";" +
			tmp.Goods_specification + ";" +
			tmp.Goods_amount + ";" +
			tmp.Goods_unit + ";" +
			tmp.Goods_price + ";" +
			tmp.Goods_total_price + ";" +
			tmp.Due_and_amount + ";" +
			tmp.Sign_date + ";"
	}
	return nil
}

// query ReceiptNote
// requset: Process_id; receiver account
// sample
func (s *GYClass) QueryReceiptNote(request string, reply *string) error {
	var strSplit = strings.Split(request, ";")
	at1 := &accounts.Account{Address: common.HexToAddress(strSplit[1])}
	result := s.dper.SimpleInvokeTransactionLocally(*at1, contract.DEMO_CONTRACT_3,
		contract.DEMO_CONTRACT_3_FuncGetReceiptAddr, [][]byte{[]byte(strSplit[0])})
	if len(result) == 0 {
		*reply = "transaction is not executed"
		return nil
	}
	tmp := new(contract.ReceiptForm)
	err := rlp.DecodeBytes(result[0], tmp)

	if err != nil {
		*reply = err.Error()
	} else {
		*reply = tmp.Process_id + ";" +
			tmp.Contract_identifier + ";" +
			tmp.Buyer + ";" +
			tmp.Provider_name + ";" +
			tmp.Goods_name + ";" +
			tmp.Goods_type + ";" +
			tmp.Goods_specification + ";" +
			tmp.Goods_fahuo_amount + ";" +
			tmp.Goods_unit + ";" +
			tmp.Goods_price + ";" +
			tmp.Goods_total_price + ";" +
			tmp.Date + ";"
	}
	return nil
}

func RpcContractSupplyWith54front(dp *dper.Dper, netMethod string, ip string) error {

	GY := new(GYClass)
	GY.dper = dp
	JR := new(JRClass)
	JR.dper = dp

	rpc.RegisterName("GYClass", GY)
	rpc.RegisterName("JRClass", JR)
	listener, err := net.Listen(netMethod, ip)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
	}

}
