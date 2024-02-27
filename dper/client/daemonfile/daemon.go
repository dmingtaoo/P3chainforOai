package main

import (
	"flag"
	"os"
	"os/exec"
	loglogrus "p3Chain/log_logrus"
	"runtime"
)

const (
	PROGRAM = "prog"
	DAEMON  = "daemon"
	FOREVER = "forever"
)

func StripSlice(slice []string, element string) []string {
	for i := 0; i < len(slice); {
		if slice[i] == element && i != len(slice)-1 {
			slice = append(slice[:i], slice[i+1:]...)
			break
		} else if slice[i] == element && i == len(slice)-1 {
			slice = slice[:i]
			break
		} else {
			i++
		}
	}
	return slice
}

func SubProcess(args []string, shell bool) *exec.Cmd {
	var cmd *exec.Cmd
	if shell {
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command(os.Getenv("SHELL"), "-c", args[0])
			break
		case "linux":
			cmd = exec.Command(os.Getenv("SHELL"), "-c", args[0])
			break
		case "windows":
			cmd = exec.Command("cmd", "/C", args[0])
			break
		default:
			os.Exit(1)
			break
		}
	} else {
		cmd = exec.Command(args[0], args[1:]...)
	}

	err := cmd.Start()

	// if shell {
	// 	loglogrus.Log.Infof("[Daemon] Program Subprocess (%s) running in PID: %d \n", args[0], cmd.Process.Pid)
	// 	root, _ := os.Getwd()
	// 	pidFile := root + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "pid.log" // 保存program Subprocess 的pid
	// 	file, _ := os.OpenFile(pidFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	// 	file.WriteString(fmt.Sprintf("[Daemon] Program Subprocess (%s) running in PID: %d", args[0], cmd.Process.Pid))
	// 	file.Close()
	// }

	if err != nil {
		loglogrus.Log.Errorf("[Daemon] Error: %s\n", err)
	}
	os.Exit(0)
	return cmd
}

func main() {
	var cmd *exec.Cmd
	program := flag.String(PROGRAM, "", "run program")
	daemon := flag.Bool(DAEMON, false, "run in daemon")
	forever := flag.Bool(FOREVER, false, "run forever")
	flag.Parse()
	loglogrus.Log.Infof("[Daemon] Master Process running in PID: %d PPID: %d ARG: %s PROG:\"%s\"\n", os.Getpid(), os.Getppid(), os.Args, *program)
	if *program == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *daemon { // Step1: 创建一个子守护进程，然后主进程退出
		SubProcess(StripSlice(os.Args, "-"+DAEMON), false)
		cmd.Wait()
		return
	} else if *forever { //Step2: 被创建的子守护进程将重新进入此分支，因为daemon标志位已被删除。创建孙守护进程，每次孙守护进程退出，子守护进程将重启一个孙守护进程
		args := os.Args
		for { // 循环的目的是为了可以实现，当负责运行program的守护进程因故结束时能够将其重启
			loglogrus.Log.Infof("[Daemon] Forever Restart service Subprocess running in PID: %d PPID: %d\n", os.Getpid(), os.Getppid())
			cmd = SubProcess(StripSlice(args, "-"+FOREVER), false)
			cmd.Wait()
		}
	} else { // Step3: 被创建的孙进程重新进入此分支，因为daemon、forever标志位已被删除。孙守护进程创建新的自己的功能子进程，负责运行指定的program
		loglogrus.Log.Infof("[Daemon] Listening Subprocess running in PID: %d PPID: %d\n", os.Getpid(), os.Getppid())
		cmd = SubProcess([]string{*program}, true)
		cmd.Wait()
		return
	}
}
