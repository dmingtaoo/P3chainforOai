package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"p3Chain/api"
	"p3Chain/core/worldstate"
	"p3Chain/dper/client/commandLine"
	"p3Chain/ginHttp"
	loglogrus "p3Chain/log_logrus"
	"time"
)

func main() {

	loglogrus.Log.Infof("[Dper] Dper Client running in PID: %d  PPID: %d \n", os.Getpid(), os.Getppid())
	root, _ := os.Getwd()
	pidFile := root + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "pid.log" // 保存dper Client 的pid

	// fmt.Println("PID file: ", pidFile)

	file, _ := os.OpenFile(pidFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	file.WriteString(fmt.Sprintf("[Dper] Dper Client running in PID: %d", os.Getpid()))
	file.Close()

	mode := flag.String("mode", "single", "Select a mode to start the project: (single/multi_manual/multi_http)")

	flag.Parse()

	switch *mode {
	case "booter_init":
		initActionList := "./prepare.txt"
		cli := new(commandLine.CommandLine)
		if err := cli.HttpRun(initActionList); err != nil {
			//loglogrus.Log.Errorf("Start Stage: Automation script execution failed , err:%v", err)
			return
		}
		defer cli.Close()

	case "single":
		cli := new(commandLine.CommandLine)
		cli.Run()
		defer cli.Close()
	case "multi_manual":
		actionList := "./action.txt"
		cli := new(commandLine.CommandLine)
		if err := cli.AutoRun(actionList); err != nil {
			//loglogrus.Log.Errorf("Start Stage: Automation script execution failed , err:%v", err)
			return
		}
		defer cli.Close()
	case "multi_http":
		actionList := "./action.txt"
		cli := new(commandLine.CommandLine)
		if err := cli.HttpRun(actionList); err != nil {
			loglogrus.Log.Errorf("Start Stage: Automation script execution failed , err:%v\n", err)
			return
		}

		dperService := api.NewDperService(cli)

		var blockService *api.BlockService = nil
		var DpStateManager *worldstate.DpStateManager = nil

		var networkService *api.NetWorkService = nil

		if cli.BackServeType() == "dper" {
			dper := cli.BackDper()
			if dper == nil {
				loglogrus.Log.Errorf("Start Stage: CommanLine has not set up dper.Dper!\n")
				return
			}
			sm := dper.BackStateManager()
			if sm == nil {
				loglogrus.Log.Errorf("Start Stage: Dper has not set up worldstate.StateManager!\n")
				return
			}
			dsm, ok := sm.(*worldstate.DpStateManager)
			if !ok {
				loglogrus.Log.Errorf("Start Stage: Can't assert an interface(worldstate.StateManager) as an object(worldstate.DpStateManager)\n")
				return
			} else {
				DpStateManager = dsm
				blockService = api.NewBlockService(DpStateManager)
			}

			nm := dper.BackNetManager()
			if nm == nil {
				loglogrus.Log.Errorf("Start Stage: Dper has not set up netconfig.NetManager!\n")
				return
			}
			networkService = api.NewNetWorkService(nm)
		}

		contractService := api.NewContractService(dperService)
		// 4.将服务器全部注册为http
		ginHttp.NewGinRouter(blockService, dperService, networkService, contractService)

		defer cli.Close()
		for {
			time.Sleep(time.Second)
		}

	default:
		cli := new(commandLine.CommandLine)
		cli.Run()
		defer cli.Close()
	}

}
