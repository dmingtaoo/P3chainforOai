package commandLine

import (
	"bufio"
	"p3Chain/accounts"
	loglogrus "p3Chain/log_logrus"

	"errors"
	"fmt"
	"io"
	"os"
	"p3Chain/common"
	"p3Chain/dper"
)

const (
	dpername = "DPER"
	version  = "0.1"
)

var (
	lineHead = ">>> " + dpername + " v" + version + " <<<"
)

const (
	defaultDperConfigFilePath   = "./settings/dperConfig.json"
	defaultBooterConfigFilePath = "./settings/booterConfig.json"
	defaultRpcConfigFilePath    = "./settings/rpcConfig.json"
)

const (
	defaultBooterUrlSavePath = "./booters/booter.txt"
)

type CommandLine struct {
	serveType string

	dper           *dper.Dper
	running        bool
	accountManager *accounts.Manager
	defaultAccount accounts.Account
	txCheckMode    bool

	orderServiceServer *dper.OrderServiceServer
}

func (c *CommandLine) BackServeType() string {
	return c.serveType
}

func (c *CommandLine) BackContract() string {
	return c.dper.BackContract()
}

func (c *CommandLine) BackMTW() []string {
	return c.dper.BackMTW()
}

func (c *CommandLine) Run() {
	reader := bufio.NewReader(os.Stdin)
	c.serveType = "dper"
	c.welcome()
	for {
		c.printLineHead()
		cmdString, err := reader.ReadString('\n') // 读取回车以前的输入字符(指令)
		if err != nil {
			c.printError(err)
		}
		err = c.runCommand(cmdString) //执行指令
		if err != nil {
			c.printError(err)
		}
	}
}

func (c *CommandLine) Close() {
	c.dper.CloseDiskDB()
}

func (c *CommandLine) HttpRun(actionList string) error {
	if !common.FileExist(actionList) {
		loglogrus.Log.Errorf("Start Stage: Can't find actionList.txt in filePath:%s\n", actionList)
		return errors.New("error in autoAct: no such action list")
	}
	fi, err := os.Open(actionList)
	if err != nil {
		loglogrus.Log.Errorf("Start Stage: Can't Open actionList.txt , err:%v\n", err)
		return err
	}
	c.serveType = "dper"
	c.welcome()
	br := bufio.NewReader(fi)
	for {
		line, _, err := br.ReadLine() //read command frome txt line by line
		if err == io.EOF {
			break
		}
		err = c.runCommand(string(line)) //run command
		if err != nil {
			return err
		}
	}
	fi.Close()
	return nil
}

// Read Command from txt to complete the initialization of dper
func (c *CommandLine) AutoRun(actionList string) error {
	if !common.FileExist(actionList) {
		return errors.New("error in autoAct: no such action list")
	}
	fi, err := os.Open(actionList)
	if err != nil {
		return err
	}
	c.serveType = "dper"
	c.welcome()
	br := bufio.NewReader(fi)
	for {
		line, _, err := br.ReadLine() //read command frome txt line by line
		if err == io.EOF {
			break
		}
		err = c.runCommand(string(line)) //run command
		if err != nil {
			c.printError(err)
			return err
		}
	}
	fi.Close()
	reader := bufio.NewReader(os.Stdin)

	// ginHttp.NewGinRouter(blockService, dperService)

	for {
		c.printLineHead()
		cmdString, err := reader.ReadString('\n') // 读取回车以前的输入字符(指令)
		if err != nil {
			c.printError(err)
		}
		err = c.runCommand(cmdString) //执行指令
		if err != nil {
			c.printError(err)
		}
	}
}

func (c *CommandLine) BackDper() *dper.Dper {
	return c.dper
}
func (c *CommandLine) printLineHead() {
	fmt.Printf("\033[31;43m%s\033[0m $$ ", lineHead)
}

// TODO:需要增加休眠指令，用于同步各节点的初始化步骤流程
func (c *CommandLine) runCommand(commandStr string) error {
	switch c.serveType { //根据节点的类型采取不同指令执行策略
	case "dper":
		return c.runDperCommand(commandStr)
	case "booter":
		return c.runBooterCommand(commandStr)
	default:
		loglogrus.Log.Errorf("Start Stage: Unknown serve type , Please check whether the type is set for the current node (dper or booter)!\n")
		return fmt.Errorf("unknown serve type")
	}
}

func (c *CommandLine) printError(err error) {
	errr := fmt.Errorf("dper Error: %v", err)
	fmt.Fprintln(os.Stderr, errr)
}

func (c *CommandLine) welcome() {

	copyRightInfo := "Copyright © 2022, P3-Chain Authors. All Rights Reserved.\nThis software is part of the project P3-Chain.\n"
	p3ChainDescription := `P3-Chain is a consortium blockchain system that insists on supporting Byzantine fault-tolerance. It is an efficient, easy-use and safe blockchain platform, and will be well-known in the near future.
P3-Chain has good scalability and supports more nodes to run consensus method compared with current consortium blockchain system. This feature broadens the application scenarios where user is node of P3-Chain.
P3-Chain builds a native blockchain platform system with sharding management and support shards for collaborative interaction, which avoids the performance and security degradation caused by chain-cross facilities.
`

	bannerp3ChainInfo := `
===========================================================
____ _____        ____ _   _    _    ___ _   _ 
|  _ \___ /       / ___| | | |  / \  |_ _| \ | |
| |_) ||_ \ _____| |   | |_| | / _ \  | ||  \| |
|  __/___) |_____| |___|  _  |/ ___ \ | || |\  |
|_|  |____/       \____|_| |_/_/   \_\___|_| \_|
									                                                            
===========================================================
`
	dperDescription := `Welcome to use Dper.
Dper is a client terminal providing users with the basic way to construct and manage the P3-Chain.
Before using Dper to construct a P3-Chain, user should config the settings which is in path './settings/dperConfig.json'. P3-Chain provides users several ways to construct and initialize it, however,  the Dper now only provides users with the basic state-construct-start way. 
Version Features:
1. Support the three-stage (state-construct-start) P3-Chain construction method.
2. Support booter mode, and the default config path is in './settings/booterConfig.json'.
3. Support simple transaction invoke/call.
`
	bannerDperInfo := `
===========================================================
                                        __________
  _____                                / ___  ___ \
 |  __ \                              / / @ \/ @ \ \
 | |  | |  _ __     ___   _ __        \ \___/\___/ /\
 | |  | | | '_ \   / _ \ | '__|        \____\/____/||
 | |__| | | |_) | |  __/ | |           /     /\\\\\//
 |_____/  | .__/   \___| |_|           |     |\\\\\\
          | |                          \      \\\\\\
          |_|                           \______/\\\\
                                         _||_||_
===========================================================
`
	p3ChainInfo := copyRightInfo + bannerp3ChainInfo + p3ChainDescription
	fmt.Print(p3ChainInfo)
	dperInfo := copyRightInfo + bannerDperInfo + dperDescription
	fmt.Println()
	fmt.Print(dperInfo)
	startInfo := fmt.Sprintf("Client %s v%s starts!\n", dpername, version)
	fmt.Print(startInfo)
}

func (c *CommandLine) printByeBye() {
	byebye := `Dper is closed. 
Thanks for using Dper to manage P3-Chain ^_^ 
Hope to see you again.`
	fmt.Println(byebye)
}

func (c *CommandLine) printUsages() string {
	switch c.serveType {
	case "dper":
		commandsInfo := `Dper is a client terminal providing users with the basic way to construct and manage the P3-Chain.

To initialize the dper:
  init             Load the config, and initialize the dper.
  rpc			   start rpc service to run contract.

To join the P3-Chain, please go through the three-stage initialization:
  state            State the node self to the P3-Chain network (dpnet).
  construct        Use the received node state information to construct dpnet.
  start            Start the dper client after the construction of dpnet.

Common Commands:
  help             Display all the commands and so the usages.
  exit,quit        Close the dper client.
  listAccounts     List out all the accounts stored.
  info             Print out the current state of Dper.
  viewNet          Pint out the information about the dpnet the dper known. 
  createNewAccount Create a new account and set it as the default user account.
    usage:         createNewAccount 
                     -- Create an account with no password.
                   createNewAccount -p yourPassword		
                     -- Create an account with yourPassword. 				   
  useAccount       Use account with the password input and set it as default account.
    usage:         useAccount yourAccountAddress 
                     -- use yourAccountAddress to load user.
                   useAccount yourAccountAddress -p yourPassword
                     -- use yourAccountAddress and yourPassword to load user.
  txCheckMode      Set the transaction check mode (default is false). When this mode is on, invoke (both soft and solid) will wait no more than 10s to back the transaction results.
    usage:         txCheckMode on
                     -- set transaction check mode on
                   txCheckMode off
                     -- set transaction check mode off
  invoke           Use contract name and function name to invoke a transaction and broadcast it to the P3-Chain.
    usage:         invoke contractName functionName -args arg1 arg2 ...
                     -- contractName and functionName would be converted into address.
  call             Use contract name and function name to invoke a transaction locally and print the results got.
    usage:         call contractName functionName -args arg1 arg2 ...
                     -- contractName and functionName would be converted into address.
  solidInvoke      Use contract address and function address to invoke a transaction and broadcast it to the P3-Chain.
    usage:         solidInvoke contractAddr functionAddr -args arg1 arg2 ...
                     -- should specify the contract and function 20-bytes address.
  solidCall        Use contract address and function address to invoke a transaction locally and print the results got.
    usage:         solidCall contractAddr functionAddr -args arg1 arg2 ...
                     -- should specify the contract and function 20-bytes address.

Mode Change:
  beBooter         Change client mode from Dper mode to booter. Should be used before initiating Dper.
`
		fmt.Print(commandsInfo)
		return commandsInfo
	case "booter":
		commandsInfo := `Dper is a client terminal providing users with the basic way to construct and manage the P3-Chain.

Now Dper client has been changed to booter mode. 

To initialize the booter:
  init             Load the config, and initialize the booter.

To help nodes to construct P3-Chain and run upper-level consensus, booter also should go through the three-stage initialization:
  state            State the booter self to the P3-Chain network (dpnet).
  construct        Use the received node state information to construct dpnet.
  start            Start providing the upper block ordering service after the construction of dpnet.
  
Useful Commands:
  viewNet          Pint out the information about the dpnet the booter known.
  exit,quit        Close the booter (of say dper) client.
  saveurl          Save the booter self url which could be copied to other dper used as bootstrap node.
`
		fmt.Print(commandsInfo)
		return commandsInfo
	default:
		fmt.Println("Unknown serve type is got!")
		return ""
	}
}
