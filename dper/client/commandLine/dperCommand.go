package commandLine

import (
	"errors"
	"fmt"
	"os"
	"p3Chain/accounts"
	"p3Chain/common"
	"p3Chain/core/consensus"
	"p3Chain/core/contract"
	"p3Chain/core/eles"
	"p3Chain/core/viewChange"
	"p3Chain/dper"
	"p3Chain/dper/configer"
	"p3Chain/dper/rpc"
	"p3Chain/crypto"
	"p3Chain/dper/transactionCheck"
	loglogrus "p3Chain/log_logrus"
	"strings"
	"time"
	"encoding/hex"
)

const (
	MaxTry int = 3 // 重连网络尝试次数
)

type AccountList struct {
	Accounts []string `json:"accounts"`
}

func (c *CommandLine) runDperCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n") //将输入指令最后的回车符删除
	arrCommandStr := strings.Fields(commandStr)       //以空白字符为间隔符,将输入指令进行分割
	if len(arrCommandStr) == 0 {
		loglogrus.Log.Warnf("Command is empty, please re-enter!\n")
		return nil
	}
	switch arrCommandStr[0] { //查看分割后的首部指令,采取不同执行策略
	case "exit", "Exit", "EXIT", "quit", "Quit", "QUIT":
		c.Close()
		c.printByeBye()
		os.Exit(0)
	case "init", "Init", "INIT":
		loglogrus.Log.Infof("Dper Init: Loading config...,configFile Path:%s\n", defaultDperConfigFilePath)
		dpcfg := configer.LoadToDperCfg(defaultDperConfigFilePath) //加载dper节点的配置文件,返回一个dper.DperConfig配置对象
		dp := dper.NewDper(dpcfg)                                  //根据dper.DperConfig配置对象创建一个dper节点

		c.dper = dp
		c.accountManager = dp.BackAccountManager()
	case "stateSelf", "state":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper State: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		err := c.dper.StateSelf(false)
		if err != nil {
			loglogrus.Log.Warnf("Dper State failed: Unable to send StateMsg to other Node,err:%v\n", err)
		}
	case "construct", "constructDpnet":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper Construct: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		err := c.dper.ConstructDpnet()
		if err != nil {
			for i := 0; i < MaxTry; i++ {
				c.dper.StateSelf(true)
				time.Sleep(sleepTime)
				netInfo := c.dper.BackViewNetInfo()
				fmt.Print(netInfo)
				time.Sleep(sleepTime)
				if err = c.dper.ConstructDpnet(); err == nil {
					return nil
				}
			}
			loglogrus.Log.Warnf("Dper Construct failed,err:%v\n", err)
			return err
		}
	case "start", "Start", "START":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper Start: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		err := c.dper.Start()
		if err != nil {
			loglogrus.Log.Warnf("Dper Start failed, err:%v\n", err)
			return err
		}
		c.running = true

	case "viewChange":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper Start: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		selfNode := c.dper.BackSelfNode()
		nodeViewChange := viewChange.NewViewChange(c.dper.BackConsensusPromoter(), c.dper.BackSubNetLeader(selfNode.NetID),
			c.dper.BackConsensusPromoter().BackTargetTPBFT(consensus.COMMON_TPBFT_NAME), c.dper.BackSyn())
		go nodeViewChange.ViewChangeStart()

	case "listAccounts":
		if c.accountManager == nil {
			loglogrus.Log.Warnf("Dper listAccounts failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		accounts, err := c.accountManager.Accounts()
		if err != nil {
			return err
		}
		c.printAccounts(accounts)
	case "createNewAccount":
		if c.accountManager == nil {
			loglogrus.Log.Warnf("Dper createNewAccount failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		_, err := c.handleCreateNewAccounts(arrCommandStr)
		if err != nil {
			return err
		}
		loglogrus.Log.Infof("Dper createNewAccount: Create new account successfully!\n")
	case "useAccount":
		if c.accountManager == nil {
			loglogrus.Log.Warnf("Dper useAccount failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		err := c.handleUseAccount(arrCommandStr)
		if err != nil {
			return err
		}
	case "info", "Info", "INFO":
		c.PrintSelfInfo()

	case "txCheckMode":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper txCheckMode failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		if !c.running {
			loglogrus.Log.Warnf("Dper txCheckMode failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
			return fmt.Errorf("should first start dper!")
		}
		err := c.handleTxCheckMode(arrCommandStr)
		if err != nil {
			return err
		}
	case "solidInvoke":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper solidInvoke failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		if !c.running {
			loglogrus.Log.Warnf("Dper solidInvoke failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
			return fmt.Errorf("should first start dper!")
		}
		receipt, err := c.solidInvokeTransaction(arrCommandStr)

		_ = receipt
		if err != nil {
			return err
		}
	case "solidCall":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper solidCall failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		if !c.running {
			loglogrus.Log.Warnf("Dper solidCall failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
			return fmt.Errorf("should first start dper!")
		}
		res, err := c.solidCallTransaction(arrCommandStr)
		if err != nil {
			return err
		}
		c.printTransactionResults(res)

	case "invoke", "Invoke", "INVOKE", "softInvoke":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper invoke failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		if !c.running {
			loglogrus.Log.Warnf("Dper invoke failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
			return fmt.Errorf("should first start dper!")
		}
		res, err := c.softInvokeTransaction(arrCommandStr)
		_ = res
		if err != nil {
			return err
		}

	case "call", "Call", "CALL", "softCall":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper call failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		if !c.running {
			loglogrus.Log.Warnf("Dper call failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
			return fmt.Errorf("should first start dper!")
		}
		res, err := c.softCallTransaction(arrCommandStr)
		if err != nil {
			return err
		}
		c.printTransactionResults(res)

	case "beBooter":
		err := c.changeServeTypeToBooter()
		if err != nil {
			return err
		}
		fmt.Println("Now is booter!")
	case "viewNet":
		if c.dper == nil {
			loglogrus.Log.Warnf("Dper viewNet failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
			return nil
		}
		netInfo := c.dper.BackViewNetInfo()
		fmt.Print(netInfo)
	case "rpc", "RPC", "Rpc":
		fmt.Println("Loading rpc config...")
		cfg, err := configer.LoadRpcCfg(defaultRpcConfigFilePath)
		if err != nil {
			return err
		}
		go rpc.RpcContractSupplyWith54front(c.dper, cfg.NetWorkmethod, cfg.RpcIPAddress)
		fmt.Println("rpc has started! ip:", cfg.RpcIPAddress)
	case "help":
		c.printUsages()
	case "sleep":
		time.Sleep(sleepTime)
	case "sleeplong":
		time.Sleep(sleepLongTime)

	default:
		loglogrus.Log.Warnf("Dper: Unknown command is input. Pint \"help\" to see all commands\n")
		return nil
	}

	return nil
}

func (c *CommandLine) printAccounts(accounts []accounts.Account) string {
	info := fmt.Sprintf("Got %d accounts:\n", len(accounts))
	for i := 0; i < len(accounts); i++ {
		info += fmt.Sprintf("No.%d, Address: %x\n", i, accounts[i].Address)
	}
	fmt.Print(info)
	return info
}

func (c *CommandLine) handleCreateNewAccounts(arrs []string) (common.Address, error) {
	if len(arrs) == 1 {
		account, err := c.accountManager.NewAccount("")
		if err != nil {
			return common.Address{}, err
		}
		c.defaultAccount = account
		c.accountManager.Unlock(account.Address, "")
		return account.Address, nil
	}
	if len(arrs) >= 2 {
		switch arrs[1] {
		case "-p", "-password":
		default:
			loglogrus.Log.Info("Dper: Please note -- You have not set a password for your account!\n")
			return common.Address{}, fmt.Errorf("should input password")
		}
		if len(arrs) == 2 {
			account, err := c.accountManager.NewAccount("")
			if err != nil {
				return common.Address{}, err
			}
			c.defaultAccount = account
			c.accountManager.Unlock(account.Address, "")
			loglogrus.Log.Info("Dper: Please note -- The password you set for the account (%x) is an empty string!\n", account.Address)
			return account.Address, nil
		}
		if len(arrs) == 3 {
			account, err := c.accountManager.NewAccount(arrs[2])
			if err != nil {
				return common.Address{}, err
			}
			c.defaultAccount = account
			err = c.accountManager.Unlock(account.Address, arrs[2])
			if err != nil {
				loglogrus.Log.Warnf("Dper failed: handleCreateNewAccounts can't Unlock specify account (%x)\n", account.Address)
				return common.Address{}, err
			}
			loglogrus.Log.Infof("Dper: New account (%x) created successfully!", account.Address)
			return account.Address, nil
		}
	}
	loglogrus.Log.Warnf("Dper: Create New Account is failed, because can't resolve the arguments!\n")
	return common.Address{}, fmt.Errorf("cannot resolve the arguments")
}

func (c *CommandLine) handleUseAccount(arrs []string) error {
	if len(arrs) < 2 {
		loglogrus.Log.Warnf("Dper failed: handleUseAccount unmatched arguments, You should specify the address of the account!\n")
		return fmt.Errorf("unmatched arguments")
	}
	accountAddr := common.HexToAddress(arrs[1])
	account, err := c.accountManager.GetAccount(accountAddr)
	if err != nil {
		return fmt.Errorf("account with address: %s cannot load, %v", arrs[1], err)
	}
	if len(arrs) == 2 {
		errr := c.accountManager.Unlock(account.Address, "")
		if errr != nil {
			loglogrus.Log.Warnf("Dper failed: handleUseAccount can't Unlock specify account (%x)\n", account.Address)
			return errr
		}
		c.defaultAccount = account
		return nil
	}
	if len(arrs) >= 3 {
		switch arrs[2] {
		case "-p", "-password":
		default:
			loglogrus.Log.Warnf("Dper failed: Please note -- You should input password to Login account(%x)!\n", accountAddr)
			return fmt.Errorf("unmatched arguments")
		}
		if len(arrs) == 3 {
			errr := c.accountManager.Unlock(account.Address, "")
			if errr != nil {
				loglogrus.Log.Warnf("Dper failed: handleUseAccount can't Unlock specify account (%x)\n", account.Address)
				return errr
			}
			c.defaultAccount = account
			loglogrus.Log.Infof("Dper: Successfully switched users. The current user is (%x)\n", c.defaultAccount)
			return nil
		}
		if len(arrs) == 4 {
			errr := c.accountManager.Unlock(account.Address, arrs[3])
			if errr != nil {
				loglogrus.Log.Warnf("Dper failed: handleUseAccount can't Unlock specify account (%x)\n", account.Address)
				return errr
			}
			c.defaultAccount = account
			loglogrus.Log.Infof("Dper: Successfully switched users. The current user is (%x)\n", c.defaultAccount)
			return nil
		}
	}
	loglogrus.Log.Warnf("Dper: Switch current user account is failed, because can't resolve the arguments!\n")
	return fmt.Errorf("cannot resolve the arguments")
}

func (c *CommandLine) PrintSelfInfo() string {
	info := ""
	if c.dper == nil {
		info += "Dper has not been initialized yet.\n"
	} else {
		info += "Dper has initialized already.\n"
	}
	info += fmt.Sprintf("Now used account address: %x\n", c.defaultAccount.Address)
	fmt.Print(info)

	return info
}

func (c *CommandLine) handleTxCheckMode(arrs []string) error {
	if len(arrs) != 2 {
		loglogrus.Log.Warnf("Dper failed: handleTxCheckMode unmatched arguments!\n")
		return fmt.Errorf("unmatched arrguments is input")
	}
	switch arrs[1] {
	case "on", "start":
		if c.txCheckMode {
			loglogrus.Log.Infof("Dper: Tx Check Mode has been started already!\n")
			return fmt.Errorf("txCheckMode has been already on")
		}
		c.dper.StartCheckTransaction()
		c.txCheckMode = true
	case "off", "close", "stop":
		if !c.txCheckMode {
			loglogrus.Log.Infof("Dper: Tx Check Mode has been shutdown already!\n")
			return fmt.Errorf("txCheckMode has been already off")
		}
		c.dper.CloseCheckTransaction()
		c.txCheckMode = false
	default:
		loglogrus.Log.Warnf("Dper failed: handleTxCheckMode unkown argument %s is input\n", arrs[1])
		return fmt.Errorf("unkown argument %s is input", arrs[1])
	}
	loglogrus.Log.Infof("Dper: handleTxCheckMode runs successfully, The current Tx check mode is %s\n", arrs[1])
	return nil
}

// invoke contractAddr functionAddr -args arg1 arg2 arg3...
func (c *CommandLine) solidInvokeTransaction(arrs []string) (transactionCheck.CheckResult, error) {
	if len(arrs) < 4 {
		loglogrus.Log.Warnf("Dper failed: solidInvokeTransaction unmatched arguments, Correct input format: invoke contractAddr functionAddr -args arg1 arg2 arg3...\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("unmatched arguments")
	}
	if arrs[3] != "-args" {
		loglogrus.Log.Warnf("Dper failed: solidInvokeTransaction unmatched arguments, Correct input format: invoke contractAddr functionAddr -args arg1 arg2 arg3...\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("unmatched arguments")
	}
	contractAddr := common.HexToAddress(arrs[1])
	functionAddr := common.HexToAddress(arrs[2])
	args := make([][]byte, 0)
	for i := 4; i < len(arrs); i++ {
		args = append(args, []byte(arrs[i]))
	}
	var err error
	var res transactionCheck.CheckResult
	if c.txCheckMode {
		res, err = c.dper.SimplePublishTransaction(c.defaultAccount, contractAddr, functionAddr, args)
		if err != nil {
			return transactionCheck.CheckResult{}, err
		}
		c.printTransactionCheckResults(res)
	} else {
		txID, err := c.dper.SimpleInvokeTransaction(c.defaultAccount, contractAddr, functionAddr, args)
		if err != nil {
			return transactionCheck.CheckResult{}, err
		}
		res.TransactionID = txID
	}
	loglogrus.Log.Infof("Dper: solidInvokeTransaction runs successfully, transaction has been upload blockChain!\n")
	return res, nil
}

// call contractAddr functionAddr -args arg1 arg2 arg3...
func (c *CommandLine) solidCallTransaction(arrs []string) ([][]byte, error) {
	res := make([][]byte, 0)
	if len(arrs) < 4 {
		loglogrus.Log.Warnf("Dper failed: solidCallTransaction unmatched arguments, Correct input format: call contractAddr functionAddr -args arg1 arg2 arg3...\n")
		return res, fmt.Errorf("unmatched arguments")
	}
	if arrs[3] != "-args" {
		loglogrus.Log.Warnf("Dper failed: solidCallTransaction unmatched arguments, Correct input format: call contractAddr functionAddr -args arg1 arg2 arg3...\n")
		return res, fmt.Errorf("unmatched arguments")
	}
	contractAddr := common.HexToAddress(arrs[1])
	functionAddr := common.HexToAddress(arrs[2])
	args := make([][]byte, 0)
	for i := 4; i < len(arrs); i++ {
		args = append(args, []byte(arrs[i]))
	}
	res = c.dper.SimpleInvokeTransactionLocally(c.defaultAccount, contractAddr, functionAddr, args)
	loglogrus.Log.Infof("Dper: solidCallTransaction runs successfully!")
	return res, nil
}

func (c *CommandLine) softInvokeTransaction(arrs []string) (transactionCheck.CheckResult, error) {
	if len(arrs) < 4 {
		loglogrus.Log.Warnf("Dper failed: softInvokeTransaction unmatched arguments, Correct input format: call contractName functionName -args arg1 arg2 arg3...\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("unmatched arguments")
	}
	if arrs[3] != "-args" {
		loglogrus.Log.Warnf("Dper failed: softInvokeTransaction unmatched arguments, Correct input format: call contractName functionName -args arg1 arg2 arg3...\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("unmatched arguments")
	}
	contractAddr := contract.StringToContractAddress(arrs[1])
	functionAddr := contract.StringToFunctionAddress(arrs[2])
	args := make([][]byte, 0)
	for i := 4; i < len(arrs); i++ {
		args = append(args, []byte(arrs[i]))
	}
	var err error
	var res transactionCheck.CheckResult
	if c.txCheckMode {

		res, err = c.dper.SimplePublishTransaction(c.defaultAccount, contractAddr, functionAddr, args)
		if err != nil {
			return transactionCheck.CheckResult{}, err
		}
		c.printTransactionCheckResults(res)

	} else {
		txID, err := c.dper.SimpleInvokeTransaction(c.defaultAccount, contractAddr, functionAddr, args)
		if err != nil {
			return transactionCheck.CheckResult{}, err
		}
		res.TransactionID = txID
		// c.dper.BackBlockChain().TxCheck(txID)
	}
	return res, err
}

func (c *CommandLine) PublishTransaction(txList []*eles.Transaction) error {
	if txList == nil || len(txList) == 0 {
		return errors.New("Dper failed: The Number of Transaction is void!")
	}

	c.dper.PublishTransactions(txList)
	return nil
}

func (c *CommandLine) softInvokeTransactions(arrs []string) ([]*transactionCheck.CheckResult, error) {
	if len(arrs) < 4 {
		loglogrus.Log.Warnf("Dper failed: softInvokeTransaction unmatched arguments, Correct input format: call contractName functionName -args arg1 arg2 arg3...\n")
		return []*transactionCheck.CheckResult{}, fmt.Errorf("unmatched arguments")
	}
	if arrs[3] != "-args" {
		loglogrus.Log.Warnf("Dper failed: softInvokeTransaction unmatched arguments, Correct input format: call contractName functionName -args arg1 arg2 arg3...\n")
		return []*transactionCheck.CheckResult{}, fmt.Errorf("unmatched arguments")
	}
	contractAddr := contract.StringToContractAddress(arrs[1])
	functionAddr := contract.StringToFunctionAddress(arrs[2])
	args := make([][]byte, 0)
	for i := 4; i < len(arrs); i++ {
		args = append(args, []byte(arrs[i]))
	}
	var err error
	var res []*transactionCheck.CheckResult
	// if c.txCheckMode {
	// 	go func() {
	// 		res, err = c.dper.SimplePublishTransactions(c.defaultAccount, contractAddr, functionAddr, args)
	// 		if err != nil {
	// 			return
	// 		}
	// 		for _, receipt := range res {
	// 			c.printTransactionCheckResults(*receipt)
	// 		}
	// 	}()

	// } else {
	// 	err = c.dper.SimpleInvokeTransactions(c.defaultAccount, contractAddr, functionAddr, args)
	// 	if err != nil {
	// 		return []*transactionCheck.CheckResult{}, err
	// 	}
	// }
	// TODO:压力测试暂时不支持开启交易检测(节点开启交易检测功能后，将变为串行)
	err = c.dper.SimpleInvokeTransactions(c.defaultAccount, contractAddr, functionAddr, args)
	if err != nil {
		return []*transactionCheck.CheckResult{}, err
	}
	return res, err
}

// invoke contractAddr functionAddr -args arg1 arg2 arg3...
func (c *CommandLine) softCallTransaction(arrs []string) ([][]byte, error) {
	res := make([][]byte, 0)
	if len(arrs) < 4 {
		loglogrus.Log.Warnf("Dper failed: softCallTransaction unmatched arguments, Correct input format: call contractName functionName -args arg1 arg2 arg3...\n")
		return res, fmt.Errorf("unmatched arguments")
	}
	if arrs[3] != "-args" {
		loglogrus.Log.Warnf("Dper failed: softCallTransaction unmatched arguments, Correct input format: call contractName functionName -args arg1 arg2 arg3...\n")
		return res, fmt.Errorf("unmatched arguments")
	}
	contractAddr := contract.StringToContractAddress(arrs[1])
	functionAddr := contract.StringToFunctionAddress(arrs[2])
	args := make([][]byte, 0)
	for i := 4; i < len(arrs); i++ {
		args = append(args, []byte(arrs[i]))
	}
	res = c.dper.SimpleInvokeTransactionLocally(c.defaultAccount, contractAddr, functionAddr, args)
	return res, nil
}

func (c *CommandLine) printTransactionResults(res [][]byte) string {
	info := "Got transaction results:\n"
	for i := 0; i < len(res); i++ {
		info += fmt.Sprintf("index%d:%s\n", i, res[i])
	}
	fmt.Print(info)
	return info
}

func (c *CommandLine) changeServeTypeToBooter() error {
	if c.dper != nil || c.running {
		return fmt.Errorf("cannot change to booter serve type, dper has been initialized!")
	}
	c.serveType = "booter"
	lineHead = ">>> BOOTER <<<"
	return nil
}

func (c *CommandLine) printTransactionCheckResults(res transactionCheck.CheckResult) {
	loglogrus.Log.Infof("Transaction ID: %x\n", res.TransactionID)
	loglogrus.Log.Infof("Valid: %v\n", res.Valid)
	loglogrus.Log.Infof("Transaction Results: %s\n", res.Result)
	loglogrus.Log.Infof("Consensus Delay: %d ms\n", res.Interval.Milliseconds())
}

func (c *CommandLine) BackAccountList() (AccountList, error) {
	if c.accountManager == nil {
		loglogrus.Log.Warnf("Dper listAccounts failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return AccountList{}, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	accounts, err := c.accountManager.Accounts()
	if err != nil {
		return AccountList{}, err
	}

	accountList := AccountList{
		Accounts: make([]string, 0),
	}
	for _, account := range accounts {
		accountStr := fmt.Sprintf("%x", account)
		accountList.Accounts = append(accountList.Accounts, accountStr)
	}

	return accountList, nil
}

func (c *CommandLine) CreateNewAccount(arrCommandStr []string) (common.Address, error) {
	if c.accountManager == nil {
		loglogrus.Log.Warnf("Dper createNewAccount failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return common.Address{}, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	account, err := c.handleCreateNewAccounts(arrCommandStr)
	if err != nil {
		return common.Address{}, err
	}
	loglogrus.Log.Infof("Dper createNewAccount: Create new account successfully!\n")
	return account, nil
}

func (c *CommandLine) UseAccount(arrCommandStr []string) error {
	if c.accountManager == nil {
		loglogrus.Log.Warnf("Dper useAccount failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	err := c.handleUseAccount(arrCommandStr)
	if err != nil {
		return err
	}
	//fmt.Println("Use account: succeed.")
	return nil
}

func (c *CommandLine) CurrentAccount() string {

	if c.dper == nil {
		return ""
	} else {
		return fmt.Sprintf("%x", c.defaultAccount.Address)
	}
}

func (c *CommandLine) OpenTxCheckMode(arrCommandStr []string) error {
	if c.dper == nil {
		loglogrus.Log.Warnf("Dper txCheckMode failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	if !c.running {
		loglogrus.Log.Warnf("Dper txCheckMode failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
		return fmt.Errorf("should first start dper!")
	}
	err := c.handleTxCheckMode(arrCommandStr)
	if err != nil {
		return err
	}
	//fmt.Println("Set txCheckMode: succeed.")
	return nil
}

func (c *CommandLine) SolidInvoke(arrCommandStr []string) (transactionCheck.CheckResult, error) {
	if c.dper == nil {
		loglogrus.Log.Warnf("Dper solidInvoke failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	if !c.running {
		loglogrus.Log.Warnf("Dper solidInvoke failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("should first start dper!")
	}
	receipt, err := c.solidInvokeTransaction(arrCommandStr)
	if err != nil {
		return transactionCheck.CheckResult{}, err
	}
	return receipt, nil
}

func (c *CommandLine) SolidCall(arrCommandStr []string) ([]string, error) {
	if c.dper == nil {
		loglogrus.Log.Warnf("Dper solidCall failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return nil, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	if !c.running {
		loglogrus.Log.Warnf("Dper solidCall failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
		return nil, fmt.Errorf("should first start dper!")
	}
	res, err := c.solidCallTransaction(arrCommandStr)
	if err != nil {
		return nil, err
	}
	resStr := make([]string, 0)
	for _, val := range res {
		resStr = append(resStr, fmt.Sprintf("%s", val))
	}

	return resStr, nil
}

func (c *CommandLine) SoftInvoke(arrCommandStr []string) (transactionCheck.CheckResult, error) {
	if c.dper == nil {
		loglogrus.Log.Warnf("Dper invoke failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	if !c.running {
		loglogrus.Log.Warnf("Dper invoke failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
		return transactionCheck.CheckResult{}, fmt.Errorf("should first start dper!")
	}
	receipt, err := c.softInvokeTransaction(arrCommandStr)
	if err != nil {
		return transactionCheck.CheckResult{}, err
	}
	return receipt, nil
}

func (c *CommandLine) SoftInvokes(arrCommandStr []string) ([]*transactionCheck.CheckResult, error) {
	if c.dper == nil {
		loglogrus.Log.Warnf("Dper invoke failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return []*transactionCheck.CheckResult{}, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	if !c.running {
		loglogrus.Log.Warnf("Dper invoke failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
		return []*transactionCheck.CheckResult{}, fmt.Errorf("should first start dper!")
	}
	receipts, err := c.softInvokeTransactions(arrCommandStr)
	if err != nil {
		return []*transactionCheck.CheckResult{}, err
	}
	return receipts, nil
}

func (c *CommandLine) SoftCall(arrCommandStr []string) ([]string, error) {
	if c.dper == nil {
		loglogrus.Log.Warnf("Dper call failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return nil, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}
	if !c.running {
		loglogrus.Log.Warnf("Dper call failed: Dper hasn't be running, You should use \"start\" to make Dper be running!\n")
		return nil, fmt.Errorf("should first start dper!")
	}
	res, err := c.softCallTransaction(arrCommandStr)
	if err != nil {
		return nil, err
	}
	resStr := make([]string, 0)
	for _, val := range res {
		resStr = append(resStr, fmt.Sprintf("%s", val))
	}

	return resStr, nil
}

func (c *CommandLine) BecomeBooter() error {
	err := c.changeServeTypeToBooter()
	if err != nil {
		return err
	}
	//fmt.Println("Now is booter!")
	return nil
}

func (c *CommandLine) BackViewNet() (dper.DPNetwork, error) {
	if c.dper == nil {
		loglogrus.Log.Warnf("Dper viewNet failed: Dper is nil, should first use command \"init\" to initialize the dper!\n")
		return dper.DPNetwork{}, fmt.Errorf("should first use command \"init\" to initialize the dper!")
	}

	net := c.dper.BackViewNet()
	return net, nil

}

func (c *CommandLine) HelpMenu() string {
	return c.printUsages()
}

func (c *CommandLine) Exit() {
	c.Close()
	c.printByeBye()
	os.Exit(0)
}
func (c *CommandLine) SignMessage(msg string) (string,error) {
	keyStore := c.accountManager.BackKeystore()
	keyAddr:=c.defaultAccount.Address
	key, err := keyStore.GetKey(keyAddr, "")
	if err != nil {
		return "", fmt.Errorf("get key err: %v", err)
	}
	prvKey := key.PrivateKey
	// 生成签名
	h := crypto.Sha3Hash([]byte(msg))
	sig, err := crypto.SignHash(h, prvKey)
	if err != nil {
		return "", fmt.Errorf("sign err: %v", err)
	}
	// 返回签名的十六进制字符串
	return hex.EncodeToString(sig), nil
	
}

