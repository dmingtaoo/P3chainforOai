package cli

import (
	"bufio"
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/logger"
	"p3Chain/p2p/discover"
	"p3Chain/p2p/nat"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
	"p3Chain/utils"
	"runtime"
	"strings"
)

var (
	workpath    = common.GetExecutePath()  //返回当前可执行文件的目录的绝对路径
	os_seprator = string(os.PathSeparator) //操作系统指定的文件分隔符
)

var (
	CLI_VERSION         = "0.1"
	WHISPER_COMMANDLINE = "WPC"
	WHISPER_SERVER      = "WPS"
	KEY_DIR             = workpath + os_seprator + "keys"
	KEY_FILE            = KEY_DIR + os_seprator + "local.key"
	PUBKEY_DIR          = workpath + os_seprator + "pubkeys"
	SELFPUBKEY_FILE     = PUBKEY_DIR + os_seprator + "self.pubkey"
	LOG_DIR             = workpath + os_seprator + "log"
	LOG_FILE            = LOG_DIR + os_seprator + "wpc.log"
	NODE_DIR            = workpath + os_seprator + "bootnodes"
	NODE_FILE           = NODE_DIR + os_seprator + "nodes.txt"
	SELF_FILE           = NODE_DIR + os_seprator + "self.txt"
	LOCAL_ADDRESS       = "127.0.0.1:30300"
	ACTION_DIR          = workpath + os_seprator + "action"
	ACTION_FILE         = ACTION_DIR + os_seprator + "actionlist.txt"

	EX_IP_ADDRESS_FILE = workpath + os_seprator + "ExIP.txt" //保留本机公网IP地址的文件的绝对路径
)

var WPS_LOGGER, WPC_LOGGER *logger.Logger

type CommandLine struct {
	registerPubKey map[string]ecdsa.PublicKey
	registerPrvKey map[string]ecdsa.PrivateKey
}

func NewCommandLine() *CommandLine {
	cmd := CommandLine{}
	cmd.registerPubKey = make(map[string]ecdsa.PublicKey)
	cmd.registerPrvKey = make(map[string]ecdsa.PrivateKey)
	return &cmd
}

func (c *CommandLine) startServer(method string, servername string, discoverFlag bool, maxPeer int, boostNodes []*discover.Node, pro server.Protocol, addr string, key *ecdsa.PrivateKey) *server.Server {
	name := common.MakeName(servername, CLI_VERSION)
	var wps server.Server
	if !discoverFlag {
		wps = server.Server{
			PrivateKey: key,
			MaxPeers:   maxPeer,
			Name:       name,
			Protocols:  []server.Protocol{pro},
			ListenAddr: addr,
			//NAT:        nat.Any(),
			//NAT: nat.NatHole(net.IP{117, 50, 178, 228}, 8080),
			NAT: nat.StaticReflect(EX_IP_ADDRESS_FILE),
		}
	} else {
		wps = server.Server{
			PrivateKey:     key,
			MaxPeers:       maxPeer,
			Discovery:      true,
			BootstrapNodes: boostNodes,
			Name:           name,
			Protocols:      []server.Protocol{pro},
			//NAT:            nat.Any(),
			//NAT:        nat.NatHole(net.IP{117, 50, 178, 228}, 8080),
			NAT:        nat.StaticReflect(EX_IP_ADDRESS_FILE),
			ListenAddr: addr,
		}
	}
	//选择本地节点获取自身公网IP地址的方式
	switch method {
	case "any":
		wps.NAT = nat.Any()
	case "server":
		wps.NAT = nat.NatHole(net.IP{117, 50, 178, 228}, 8080)
	case "file":
		wps.NAT = nat.StaticReflect(EX_IP_ADDRESS_FILE)
	default:
		wps.NAT = nat.Any()
	}

	WPS_LOGGER.Infof("start p2p server\n")
	wps.Start()
	return &wps
}

func (c *CommandLine) Run() {

	fmt.Println("This is a tiny whisper client ot test the P2P module.")
	fmt.Println("Version: ", CLI_VERSION)

	logfile, _ := os.OpenFile(LOG_FILE, os.O_RDWR|os.O_CREATE, os.ModePerm) //创建log日志文件
	defer logfile.Close()

	logger.AddLogSystem(logger.NewStdLogSystem(logfile, log.LstdFlags, logger.InfoLevel))
	logger.AddLogSystem(logger.NewStdLogSystem(logfile, log.LstdFlags, logger.WarnLevel))
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.WarnLevel))
	WPC_LOGGER = logger.NewLogger(WHISPER_COMMANDLINE)
	WPS_LOGGER = logger.NewLogger(WHISPER_SERVER)
	WPC_LOGGER.Infoln("whisper client commandline start")
	logger.Flush()

	if err := utils.DirCheck(KEY_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(LOG_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(NODE_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(ACTION_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(PUBKEY_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}

	//创建一个新的名为 clientstart 的FlagSet(自定义的flag对象)
	clientStartCmd := flag.NewFlagSet("clientstart", flag.ExitOnError)
	clientName := clientStartCmd.String("name", "whisper-client", "the name of the whisper client")
	keyPath := clientStartCmd.String("keypath", KEY_FILE, "the file path of the node's private key, if nil will try to find key in the default path")
	maxPeer := clientStartCmd.Int("peernum", 10, "the max number of peers to connect")
	useDiscovery := clientStartCmd.Bool("discovery", false, "whether to use discovery")
	listenAddr := clientStartCmd.String("addr", LOCAL_ADDRESS, "the listen address of the whisper client")
	nodePath := clientStartCmd.String("nodepath", NODE_FILE, "the file path of nodes used to bootstrap, if nil will try the default path")
	autoMode := clientStartCmd.Bool("automode", false, "if automode, wpc will do actions in action list autonomously")
	actionPath := clientStartCmd.String("actionpath", ACTION_FILE, "used in automode, which stores the actions")
	//下一个参数确定本地节点采用何种方式获取自身公网IP地址(有 any 、 server、file 三种选择，分别意味着用nat.Any(), nat.NatHole(), nat.StaticReflect()三种方法)
	getIPMethod := clientStartCmd.String("method", "any", "the method of getting external IP address")

	//创建一个新的名为 keygenerate 的FlagSet(自定义的flag对象)
	keyGenerateCmd := flag.NewFlagSet("keygenerate", flag.ExitOnError)
	keyStorePath := keyGenerateCmd.String("keypath", KEY_FILE, "the file path of the generated key, if nil will use default path")

	//创建一个新的名为 urlsave 的FlagSet(自定义的flag对象)
	urlSaveCmd := flag.NewFlagSet("urlsave", flag.ExitOnError)
	urlAddr := urlSaveCmd.String("addr", LOCAL_ADDRESS, "the listen address of the whisper client")
	urlSavePath := urlSaveCmd.String("savepath", SELF_FILE, "the file path to save url, if nil will use default path")
	urlKeyPath := urlSaveCmd.String("keypath", KEY_FILE, "the file path of the node's private key, if nil will try to find key in the default path")

	printUsage := func() {
		fmt.Println("Usage is as follows:")
		fmt.Println("clientstart")
		clientStartCmd.PrintDefaults() //向标准错误输出写入当前flag对象(clientStartCmd)的所有注册的命令行参数以及其说明
		fmt.Println("keygenerate")
		keyGenerateCmd.PrintDefaults()
		fmt.Println("urlsave")
		urlSaveCmd.PrintDefaults()
	}

	if len(os.Args) == 1 { //如果没有输入任何命令行参数(只有对可执行文件的执行命令)
		printUsage() //打印错误信息
	} else {
		switch os.Args[1] { //分析到底采用的哪一个flag对象的命令行
		case "clientstart":
			WPC_LOGGER.Infoln("parse command: clientstart")
			err := clientStartCmd.Parse(os.Args[2:]) //解析余下的命令行参数，解析后填充到注册时绑定的各个变量中
			if err != nil {
				WPC_LOGGER.Infof("error: %v.\n", err)
				runtime.Goexit()
			}
		case "keygenerate":
			WPC_LOGGER.Infoln("parse command: keygenerate")
			err := keyGenerateCmd.Parse(os.Args[2:])
			if err != nil {
				WPC_LOGGER.Infof("error: %v.\n", err)
				runtime.Goexit()
			}
		case "urlsave":
			WPC_LOGGER.Infoln("parse command: urlsave")
			err := urlSaveCmd.Parse(os.Args[2:])
			if err != nil {
				WPC_LOGGER.Infof("error: %v.\n", err)
				runtime.Goexit()
			}

		default:
			printUsage()
		}
	}
	//clientStartCmd.Parsed()可以查看clientStartCmd.Parse()是否已经被调用过，是的话返回true
	if clientStartCmd.Parsed() {
		WPC_LOGGER.Infof("start whisper client")
		shh := whisper.New()
		defer shh.Stop()
		selfKey, err := utils.LoadKey(*keyPath)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
			runtime.Goexit()
		}
		bootNodes, err := utils.ReadNodesUrl(*nodePath)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
			runtime.Goexit()
		}
		wps := c.startServer(*getIPMethod, *clientName, *useDiscovery, *maxPeer, bootNodes, shh.Protocol(), *listenAddr, &selfKey)
		defer wps.Stop()

		if *autoMode {
			WPC_LOGGER.Infof("auto mode start")
			err := c.autoAct(*actionPath, wps, shh)
			if err != nil {
				WPS_LOGGER.Infof("error: %v.\n", err)
			}
			WPC_LOGGER.Infof("auto mode end")
		} else {
			WPC_LOGGER.Infof("action interpreter start")
			c.actionInterpreter(wps, shh)
			WPC_LOGGER.Infof("action interpreter end")
		}
	}

	if keyGenerateCmd.Parsed() {
		WPC_LOGGER.Infof("generate and save key")
		c.createKey(*keyStorePath)
	}

	if urlSaveCmd.Parsed() {
		WPC_LOGGER.Infof("generate URL and save")
		c.saveUrl(*urlAddr, *urlKeyPath, *urlSavePath)
	}

	WPC_LOGGER.Infoln("whisper client commandline stop")
	logger.Flush()
	runtime.Goexit()
}

func (c *CommandLine) createKey(path string) {
	key, err := crypto.GenerateKey()
	if err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	err = utils.SaveKey(*key, path)
	if err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	err = utils.SaveSelfPublicKey(key.PublicKey, SELFPUBKEY_FILE)
	if err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
}

func (c *CommandLine) autoAct(actionList string, wps *server.Server, shh *whisper.Whisper) error {
	if !common.FileExist(actionList) {
		return errors.New("error in autoAct: no such action list")
	}
	fi, err := os.Open(actionList)
	if err != nil {
		return err
	}
	defer fi.Close()

	br := bufio.NewReader(fi)
	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}
		err = c.doAction(string(line), wps, shh)
		if err != nil {
			return err
		}
	}
	return nil

}

func (c *CommandLine) actionInterpreter(wps *server.Server, shh *whisper.Whisper) {
	printActionUsage()
	reader := bufio.NewReader(os.Stdin)
scanline:
	for {
		logger.Flush()
		var line string
		fmt.Println("Please input action:")
		line, _ = reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "Exit" || line == "exit" || line == "EXIT" || line == "quit" || line == "Quit" || line == "QUIT" || line == "q" || line == "Q" {
			break scanline
		}
		err := c.doAction(line, wps, shh)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	}

}

func (c *CommandLine) saveUrl(listenaddr, keypath, savepath string) {
	selfKey, err := utils.LoadKey(keypath)
	if err != nil {
		WPS_LOGGER.Infof("error: %v.\n", err)
		runtime.Goexit()
	}
	parsedNode, err := discover.ParseUDP(&selfKey, listenaddr)
	if err != nil {
		WPS_LOGGER.Infof("error: %v.\n", err)
		runtime.Goexit()
	}
	url := parsedNode.String()
	err = utils.SaveSelfUrl(url, savepath)
	if err != nil {
		WPS_LOGGER.Infof("error: %v.\n", err)
		runtime.Goexit()
	}
}
