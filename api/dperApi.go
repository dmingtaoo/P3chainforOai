package api

import (
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/dper"
	"p3Chain/dper/client/commandLine"
	"p3Chain/dper/transactionCheck"
	"strings"
	"sync"
)

type DperService struct {
	dperCommand *commandLine.CommandLine
}

func NewDperService(dperCommand *commandLine.CommandLine) *DperService {
	return &DperService{
		dperCommand: dperCommand,
	}
}

// TODO: commandStr字符串必须以相应指令为前缀
func ParseString(commandStr string) []string {
	commandStr = strings.TrimSuffix(commandStr, "\n") //将输入指令最后的回车符删除
	arrCommandStr := strings.Fields(commandStr)       //以空白字符为间隔符,将输入指令进行分割

	if len(arrCommandStr) == 0 {
		return nil
	}

	return arrCommandStr
}

func (ds *DperService) BackListAccounts() (commandLine.AccountList, error) {

	return ds.dperCommand.BackAccountList()
}

func (ds *DperService) CreateNewAccount(commandStr string) (common.Address, error) {

	arrs := ParseString(commandStr)
	return ds.dperCommand.CreateNewAccount(arrs)

}

func (ds *DperService) UseAccount(commandStr string) error {
	arrs := ParseString(commandStr)
	return ds.dperCommand.UseAccount(arrs)

}

func (ds *DperService) CurrentAccount() string {

	return ds.dperCommand.CurrentAccount()

}

func (ds *DperService) OpenTxCheckMode(commandStr string) error {

	arrs := ParseString(commandStr)
	return ds.dperCommand.OpenTxCheckMode(arrs)

}

func (ds *DperService) SolidInvoke(commandStr string) (transactionCheck.CheckResult, error) {

	arrs := ParseString(commandStr)
	return ds.dperCommand.SolidInvoke(arrs)

}

func (ds *DperService) SolidCall(commandStr string) ([]string, error) {

	arrs := ParseString(commandStr)
	ds.dperCommand.SolidCall(arrs)

	return ds.dperCommand.SolidCall(arrs)

}

func (ds *DperService) SoftInvoke(commandStr string) (transactionCheck.CheckResult, error) {

	arrs := ParseString(commandStr)
	return ds.dperCommand.SoftInvoke(arrs)
}

func (ds *DperService) SoftInvokeQuery(commandStr string, wg sync.WaitGroup) {
	arrs := ParseString(commandStr)
	receipts, _ := ds.dperCommand.SoftInvokes(arrs)

	_ = receipts
}

func (ds *DperService) PublishTx(txs []*eles.Transaction) error {
	return ds.dperCommand.PublishTransaction(txs)
}

func (ds *DperService) SoftCall(commandStr string) ([]string, error) {

	arrs := ParseString(commandStr)
	return ds.dperCommand.SoftCall(arrs)
}

func (ds *DperService) BecomeBooter() error {
	return ds.dperCommand.BecomeBooter()
}

func (ds *DperService) BackViewNet() (dper.DPNetwork, error) {

	return ds.dperCommand.BackViewNet()
}

func (ds *DperService) HelpMenu() string {
	return ds.dperCommand.HelpMenu()
}

func (ds *DperService) Exit() {
	ds.dperCommand.Exit()
}

func (ds *DperService) BackContract() string {
	return ds.dperCommand.BackContract()
}

func (ds *DperService) QueryTx(txID common.Hash) *eles.TransactionEvent {
	if flag, txUpLinkTime := ds.dperCommand.BackDper().BackBlockChain().TxCheck(txID); flag {
		return txUpLinkTime
	} else {
		return nil
	}
}

func (ds *DperService) BackMTW() []string {
	return ds.dperCommand.BackMTW()
}
func (ds *DperService) SignMessage(msg string) (string,error) {
	return ds.dperCommand.SignMessage(msg)
}



// stamp
// func (ds *DperService) BackCredit(commandStr string) ([]string, error) {

// 	arrs := ParseString(commandStr)
// 	return ds.dperCommand.SoftCall(arrs)
// }

// func (ds *DperService) BackStampList(commandStr string) ([]string, error) {

// 	arrs := ParseString(commandStr)
// 	return ds.dperCommand.SoftCall(arrs)
// }

// func (ds *DperService) MintNewStamp(commandStr string) (transactionCheck.CheckResult, error) {

// 	arrs := ParseString(commandStr)
// 	return ds.dperCommand.SoftInvoke(arrs)
// }

// func (ds *DperService) TransStamp(commandStr string) (transactionCheck.CheckResult, error) {
// 	arrs := ParseString(commandStr)
// 	return ds.dperCommand.SoftInvoke(arrs)
// }
