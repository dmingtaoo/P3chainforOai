package main

import (
	"bufio"
	"os"
	loglogrus "p3Chain/log_logrus"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	root, _ := os.Getwd()
	pidFile := root + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "pid.log"
	file, err := os.OpenFile(pidFile, os.O_RDONLY, 0666)
	defer file.Close()
	if err != nil {
		loglogrus.Log.Errorf("[Daemon] The Dper Client hasn't running!\n")
		return
	}

	// 构造正则表达式匹配特定的日志消息格式
	regex := regexp.MustCompile(`\[Dper\] Dper Client running in PID: (\d+)\s*$`)

	// 逐行读取日志文件并匹配日志消息
	scanner := bufio.NewScanner(file)
	var pid string
	for scanner.Scan() {
		line := scanner.Text()
		if match := regex.FindStringSubmatch(line); match != nil {
			pid = match[1]
		}
	}
	if err := scanner.Err(); err != nil {
		loglogrus.Log.Errorf("[Daemon] Error in querying PID information, err: %v\n", err)
		return
	}

	// 输出提取到的PID
	if pid != "" {
		pidStr := strings.TrimSpace(pid)
		loglogrus.Log.Infof("[Daemon]  Dper Client is running in PID: %s\n", pidStr)

		pidInt, err := strconv.Atoi(pidStr)
		if err != nil {
			loglogrus.Log.Warnf("[Daemon] The PID is illegal: %s\n", pidStr)
			return
		}

		process, err := os.FindProcess(pidInt)
		if err != nil {
			loglogrus.Log.Warnf("[Daemon] Failed to find process with PID %d: %s\n", pidInt, err)
			os.Exit(1)
		}

		err = process.Kill()
		if err != nil {
			loglogrus.Log.Warnf("[Daemon] Failed to kill process with PID %d: %s\n", pidInt, err)
			os.Exit(1)
		}
		loglogrus.Log.Infof("[Daemon] PID :%d has been killed!\n", pidInt)
	} else {
		loglogrus.Log.Errorf("[Daemon] Failed to query relevant PID information!\n")
	}
}
